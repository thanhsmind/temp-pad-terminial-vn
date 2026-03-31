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

Single-file application (`main.go`) — no packages, minimal dependencies (stdlib + `os/exec` for FFmpeg). Everything is in one file organized by sections:

- **Windows API bindings** — raw syscall to user32.dll, kernel32.dll, gdi32.dll, comctl32.dll, comdlg32.dll (no CGo, no external Win32 wrapper)
- **Config** — loads `config.json` (hotkey modifiers/key, window size/title) with defaults
- **Window Proc** — Win32 message loop with WM_HOTKEY, WM_COMMAND, WM_NOTIFY, WM_KEYDOWN handling
- **Tab Control** — SysTabControl32 with Tab 1 (VN Input) and Tab 2 (Video Convert)
- **Popup logic** — show/hide/toggle the input window, save/restore previous foreground window
- **Clipboard** — get/set clipboard via Win32 API, backup and restore after paste
- **SendInput** — simulates Ctrl+V keystrokes using raw byte arrays to handle 64-bit struct alignment
- **File dialogs** — GetOpenFileNameW/GetSaveFileNameW via comdlg32.dll for MP4 selection and output path
- **FFmpeg conversion** — runs FFmpeg in a goroutine with PostMessageW for progress updates, supports WebM VP9, MP4 H.265, and MP3 output

Key flows:
- **VN Input (Tab 1):** hotkey → `showPopup()` → user types → Ctrl+Enter or OK → `doOK()` → saves clipboard → sets clipboard to text → hides popup → restores focus → `simulateCtrlV()` → restores original clipboard after 800ms.
- **Video Convert (Tab 2):** select MP4 → choose format → Save As dialog → FFmpeg goroutine runs → PostMessageW sends progress → WM_CONVERT_PROGRESS updates progress bar → WM_CONVERT_DONE shows result.

## Configuration

`config.json` next to the executable. Default hotkey: `Ctrl+Shift+Space`. Supported modifiers: Ctrl, Shift, Alt, Win. Supported keys: A-Z, 0-9, Space, Enter, F1-F24.

## Important Notes

- `runtime.LockOSThread()` in `init()` is required — Win32 message loop must run on the OS thread that created the window
- Struct alignment matters for 64-bit Windows — `simulateCtrlV` uses raw `[40]byte` arrays instead of Go structs to match C INPUT layout exactly; `OPENFILENAMEW` struct also needs careful 64-bit alignment with explicit padding
- FFmpeg must run in a goroutine (not on UI thread) — use `PostMessageW` with custom WM_APP+N messages to send progress updates back to the UI thread safely
- Logging goes to `vn-input-helper.log` next to the executable
- Video Convert requires `ffmpeg.exe` placed alongside the app exe (gyan.dev "full" or BtbN "GPL" build for all codecs)
