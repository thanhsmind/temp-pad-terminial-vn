# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VN Input Helper — a Windows-only Go application that provides a popup text input window for typing Vietnamese in terminals/apps that don't handle Vietnamese IME well. It runs in the background, listens for a global hotkey, shows a popup with a native Win32 EDIT control (which supports IME), then pastes the typed text into the previously focused window via simulated Ctrl+V.

## Build

Requires Go 1.21+. Windows only (uses Win32 API via syscall).

```bash
# Production build (no console window)
go build -ldflags="-H windowsgui -s -w" -o vn-input-helper.exe main.go

# Debug build (with console window for stdout)
go build -o vn-input-helper-debug.exe main.go
```

Or run `build.bat` on Windows.

## Architecture

Single-file application (`main.go`, ~770 lines) — no packages, no dependencies beyond stdlib. Everything is in one file organized by sections:

- **Windows API bindings** — raw syscall to user32.dll, kernel32.dll, gdi32.dll (no CGo, no external Win32 wrapper)
- **Config** — loads `config.json` (hotkey modifiers/key, window size/title) with defaults
- **Window Proc** — Win32 message loop with WM_HOTKEY, WM_COMMAND, WM_KEYDOWN handling
- **Popup logic** — show/hide/toggle the input window, save/restore previous foreground window
- **Clipboard** — get/set clipboard via Win32 API, backup and restore after paste
- **SendInput** — simulates Ctrl+V keystrokes using raw byte arrays to handle 64-bit struct alignment

Key flow: hotkey → `showPopup()` → user types → Ctrl+Enter or OK → `doOK()` → saves clipboard → sets clipboard to text → hides popup → restores focus → `simulateCtrlV()` → restores original clipboard after 800ms.

## Configuration

`config.json` next to the executable. Default hotkey: `Ctrl+Shift+Space`. Supported modifiers: Ctrl, Shift, Alt, Win. Supported keys: A-Z, 0-9, Space, Enter, F1-F24.

## Important Notes

- `runtime.LockOSThread()` in `init()` is required — Win32 message loop must run on the OS thread that created the window
- Struct alignment matters for 64-bit Windows — `simulateCtrlV` uses raw `[40]byte` arrays instead of Go structs to match C INPUT layout exactly
- Logging goes to `vn-input-helper.log` next to the executable
