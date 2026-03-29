---
title: "feat: Add option to auto-start app on Windows login"
type: feat
status: completed
date: 2026-03-29
---

# feat: Add option to auto-start app on Windows login

## Overview

Add a `config.json` option (`auto_start`) that, when enabled, registers the app in the Windows Registry startup key so it launches automatically when the user logs in. When disabled, the registry entry is removed.

## Problem Frame

Users must manually launch the VN Input Helper each time they log in to Windows. Since this is a background utility meant to always be available, it should offer an auto-start option to remove this friction.

## Requirements Trace

- R1. New `auto_start` config field (boolean, default `false`) in `config.json`
- R2. When `auto_start: true`, the app registers itself in `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` on startup
- R3. When `auto_start: false` (or field absent), the app removes any existing registry entry for itself
- R4. Uses the current executable path so the registry entry stays correct even if the user moves the exe
- R5. Registry operations fail gracefully with logging — never crash the app

## Scope Boundaries

- Current-user startup only (`HKCU`), not machine-wide (`HKLM`) — no admin rights needed
- Config-file driven only — no system tray toggle or GUI checkbox for this setting
- No Windows Service or Task Scheduler approach
- No "first run" wizard or auto-detection of first launch

## Context & Research

### Relevant Code and Patterns

- `main.go` Config struct and `loadConfig()` — existing pattern for adding new config fields with defaults
- `main.go` Win32 API binding pattern — `syscall.NewLazyDLL` + `NewProc` for DLL function calls
- `os.Executable()` already used in `loadConfig()` to resolve exe directory — reuse for registry value
- Existing `logf()` function for error logging without crashing

### Institutional Learnings

- `docs/solutions/ui-bugs/fix-vn-input-helper-popup-sizing-2026-03-29.md` — confirms single-file architecture and config-driven approach

## Key Technical Decisions

- **Use `advapi32.dll` via syscall**: Consistent with the existing pattern of raw Win32 API calls via `syscall.NewLazyDLL`. No new dependencies. Go's `golang.org/x/sys/windows/registry` package would be cleaner but introduces an external dependency, which this project explicitly avoids (stdlib only).
- **Registry key: `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`**: Standard Windows auto-start location for current-user apps. No admin elevation required.
- **Registry value name: `VNInputHelper`**: Fixed string, not configurable. Simple and predictable.
- **Manage registry on every app start**: Check config and set/remove the registry entry each time the app launches. This keeps the registry in sync with the config without needing a separate "install" step. Slightly redundant on repeated starts but ensures correctness if the exe is moved.
- **Default `false`**: Opt-in behavior. The app should not auto-register itself without the user explicitly enabling it.

## Open Questions

### Resolved During Planning

- **Which registry hive?** `HKCU` — no admin rights needed, per-user scope is appropriate for a personal utility.
- **When to write the registry?** During app startup in `main()`, after `loadConfig()` but before the message loop. Simple and predictable.
- **What if registry write fails?** Log the error and continue. The app's primary function (hotkey → popup → paste) is independent of auto-start.

### Deferred to Implementation

- **Exact advapi32 function signatures and parameter types**: Will be determined during implementation by referencing Win32 API documentation. The functions needed are `RegOpenKeyExW`, `RegSetValueExW`, `RegDeleteValueW`, `RegCloseKey`.

## Implementation Units

- [x] **Unit 1: Add config field and advapi32 API bindings**

**Goal:** Extend the Config struct with `auto_start` field and add Windows Registry API bindings.

**Requirements:** R1

**Dependencies:** None

**Files:**
- Modify: `main.go`

**Approach:**
- Add `AutoStart bool` field with `json:"auto_start"` tag to the Config struct (top level, not nested)
- Default to `false` in `loadConfig()`
- Add `advapi32.dll` lazy DLL variable alongside existing `user32`, `kernel32`, `gdi32`
- Add proc variables for `RegOpenKeyExW`, `RegSetValueExW`, `RegDeleteValueW`, `RegCloseKey`
- Add constants: `HKEY_CURRENT_USER`, `KEY_SET_VALUE`, `KEY_WRITE`, `REG_SZ`, `ERROR_SUCCESS`

**Patterns to follow:**
- Existing `user32`/`kernel32`/`gdi32` DLL binding pattern at top of `main.go`
- Existing Config struct field pattern (nested struct with json tags)

**Test scenarios:**
- Happy path: `config.json` with `"auto_start": true` → `cfg.AutoStart` is `true` after `loadConfig()`
- Happy path: `config.json` with `"auto_start": false` → `cfg.AutoStart` is `false`
- Edge case: `config.json` without `auto_start` field → `cfg.AutoStart` defaults to `false`

**Verification:**
- Config struct has `AutoStart` field, `loadConfig()` logs the auto_start value
- advapi32 DLL procs compile without errors

- [x] **Unit 2: Implement registry set/remove functions**

**Goal:** Create functions to add or remove the app from the Windows startup registry key.

**Requirements:** R2, R3, R4, R5

**Dependencies:** Unit 1

**Files:**
- Modify: `main.go`

**Approach:**
- Create `setAutoStart()` function that:
  - Opens `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` with `RegOpenKeyExW`
  - Gets exe path via `os.Executable()`
  - Sets string value `VNInputHelper` = exe path using `RegSetValueExW` with `REG_SZ`
  - Closes key with `RegCloseKey`
  - Logs success or failure
- Create `removeAutoStart()` function that:
  - Opens same registry key
  - Deletes value `VNInputHelper` with `RegDeleteValueW`
  - Closes key
  - Logs success; ignores "value not found" errors gracefully
- Create `manageAutoStart()` orchestrator that checks `cfg.AutoStart` and calls the appropriate function

**Patterns to follow:**
- Existing Win32 API call pattern: `proc.Call(...)` with return value checking
- Existing `utf16Ptr()` helper for string conversion
- Existing `logf()` for error reporting

**Test scenarios:**
- Happy path: `setAutoStart()` with valid exe path → registry value created under `HKCU\...\Run` with key `VNInputHelper`
- Happy path: `removeAutoStart()` when entry exists → registry value deleted
- Edge case: `removeAutoStart()` when entry does not exist → no error, logs gracefully
- Error path: `RegOpenKeyExW` fails (e.g., corrupted registry) → error logged, app continues normally
- Integration: `manageAutoStart()` with `cfg.AutoStart=true` → calls `setAutoStart()`; with `false` → calls `removeAutoStart()`

**Verification:**
- After running with `auto_start: true`, check registry via `reg query "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v VNInputHelper` — value should be the exe path
- After running with `auto_start: false`, same query should show value not found
- App never crashes regardless of registry operation outcome

- [x] **Unit 3: Wire into main() and update config documentation**

**Goal:** Call `manageAutoStart()` during app startup and document the new config option.

**Requirements:** R1, R2, R3

**Dependencies:** Unit 2

**Files:**
- Modify: `main.go`
- Modify: `config.json` (example/documentation)

**Approach:**
- In `main()`, call `manageAutoStart()` after `loadConfig()` and before the hotkey registration
- Log the auto_start config value in the existing config log line
- Add `"auto_start": false` to any example `config.json` if one exists, or document it in comments

**Patterns to follow:**
- Existing `main()` initialization sequence: `initLog()` → `loadConfig()` → [new: `manageAutoStart()`] → hotkey registration → message loop

**Test scenarios:**
- Happy path: App starts with `auto_start: true` in config → log shows "Auto-start enabled, registry updated" → registry entry present
- Happy path: App starts with `auto_start: false` → log shows "Auto-start disabled, registry cleaned" → no registry entry
- Happy path: App starts without `auto_start` in config → treats as `false`, no registry entry
- Integration: Full app startup sequence completes without errors regardless of auto_start setting

**Verification:**
- App starts and registers/unregisters correctly based on config
- Log file shows auto-start management activity
- Startup notification message could optionally mention auto-start status (deferred — not required)

## System-Wide Impact

- **Interaction graph:** `manageAutoStart()` runs once during `main()` init, before the message loop. No interaction with hotkey, popup, or clipboard logic.
- **Error propagation:** Registry failures are logged and swallowed — they never affect the main app flow.
- **State lifecycle risks:** If the user moves the exe after a `true` registration, the registry entry becomes stale. The next app launch from the new location will update the registry to the correct path. If the user deletes the app without running it with `false` first, a stale registry entry remains (Windows silently ignores startup entries that point to missing executables).
- **Unchanged invariants:** Hotkey registration, popup behavior, clipboard handling, and paste simulation are completely unaffected.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Stale registry entry if app is deleted without cleanup | Windows silently ignores missing startup entries; user can manually remove via `msconfig` or Settings |
| Registry write permissions denied in restricted environments | Graceful failure with logging; HKCU writes are allowed for standard users in normal Windows configurations |
| Exe path contains special characters or Unicode | `utf16Ptr()` already handles Unicode; `os.Executable()` returns the resolved path |

## Sources & References

- Related code: `main.go` — Config struct, loadConfig(), Win32 API binding pattern
- Windows API: `RegOpenKeyExW`, `RegSetValueExW`, `RegDeleteValueW` from `advapi32.dll`
- Registry path: `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
