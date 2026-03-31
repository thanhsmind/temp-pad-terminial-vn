---
title: "feat: Add MP4 converter tab with WebM VP9, H.265, and MP3 output"
type: feat
status: completed
date: 2026-03-31
origin: docs/brainstorms/2026-03-31-mp4-converter-tab-requirements.md
---

# feat: Add MP4 converter tab with WebM VP9, H.265, and MP3 output

## Overview

Add a second tab to the existing popup window that allows users to select an MP4 file and convert it to WebM VP9 (best quality), MP4 H.265/HEVC (best quality), or MP3 (320kbps) using a bundled FFmpeg binary. Conversion runs asynchronously with a real-time progress bar. Output location is chosen via Save As dialog before conversion begins.

## Problem Frame

Users need local video/audio conversion without relying on online tools. The VN Input Helper already runs persistently, making it a convenient host for this utility feature. (see origin: docs/brainstorms/2026-03-31-mp4-converter-tab-requirements.md)

## Requirements Trace

- R1. Win32 Tab Control: Tab 1 = VN Input, Tab 2 = Video Convert
- R2. Tab 2 UI: file picker button, filename display, convert buttons
- R3. Convert MP4 -> WebM VP9 (best quality, CRF 18, libvpx-vp9)
- R4. Convert MP4 -> MP4 H.265 (best quality, CRF 18, libx265, preset slow)
- R5. Convert MP4 -> MP3 (extract audio, libmp3lame 320kbps CBR)
- R6. Each convert option is a separate button
- R7. FFmpeg.exe placed alongside the app exe
- R8. Error message when ffmpeg.exe not found
- R9. Progress bar with percent (parse FFmpeg stderr), marquee fallback
- R10. Save As dialog before conversion starts
- R11. "Completed" status on success; error message + partial file cleanup on failure

## Scope Boundaries

- Only MP4 input, only 3 output formats
- No batch conversion
- No user-configurable quality settings (fixed presets)
- FFmpeg must be manually placed alongside exe, not auto-downloaded
- No impact on existing VN Input (Tab 1) functionality

## Context & Research

### Relevant Code and Patterns

- `main.go:24-71` — DLL/proc declarations pattern (add comctl32.dll, comdlg32.dll)
- `main.go:77-173` — Constants section (add tab control, progress bar, file dialog, WM_NOTIFY constants)
- `main.go:556-564` — Prompt picker window as pattern for separate window management
- `main.go:824-883` — WM_CREATE handler for main window (controls creation pattern)
- `main.go:892-916` — WM_COMMAND handler (add WM_NOTIFY for tab control)
- `main.go:1232-1318` — Message loop (add converter-specific key handling if needed)
- The app uses raw syscall, no CGo, no external packages. New code must follow this pattern.

### External References

- FFmpeg VP9 encoding: `libvpx-vp9 -crf 18 -b:v 0 -cpu-used 0 -row-mt 1 -c:a libopus -b:a 192k`
- FFmpeg H.265 encoding: `libx265 -crf 18 -preset slow -tag:v hvc1 -c:a aac -b:a 256k -movflags +faststart`
- FFmpeg MP3 extraction: `-vn -c:a libmp3lame -b:a 320k`
- FFmpeg progress: parse `Duration:` from initial stderr, then `time=HH:MM:SS.ss` from ongoing output. Alternative: use `-progress pipe:1` for machine-readable key=value output on stdout
- Windows builds: gyan.dev "full" or BtbN "GPL" builds include libx265, libvpx, libopus, libmp3lame

## Key Technical Decisions

- **comctl32.dll required for Tab Control and Progress Bar**: Must call `InitCommonControlsEx` with `ICC_TAB_CLASSES | ICC_PROGRESS_CLASS` before creating SysTabControl32 or msctls_progress32. Currently not loaded in the codebase.
- **comdlg32.dll required for file dialogs**: `GetOpenFileNameW` for MP4 selection, `GetSaveFileNameW` for output path. Requires defining the `OPENFILENAMEW` struct (aligned for 64-bit).
- **Tab switching via WM_NOTIFY/TCN_SELCHANGE**: Win32 Tab Control does not auto-show/hide children. Maintain two slices of HWNDs (tab1Controls, tab2Controls) and toggle visibility on tab change.
- **FFmpeg in goroutine + PostMessageW for progress**: FFmpeg process runs in a background goroutine via `os/exec`. Progress updates sent to UI thread via `PostMessageW` with custom message `WM_APP+1`. UI thread handles the custom message in wndProc to update progress bar and label. This avoids blocking the message loop and unsafe cross-thread Win32 calls.
- **`-progress pipe:1` for clean progress parsing**: Instead of parsing raw stderr, use FFmpeg's `-progress pipe:1` flag which outputs structured key=value pairs to stdout. Parse `out_time_ms` and compare to total duration obtained from a quick `ffprobe` or initial FFmpeg stderr scan.
- **Save As dialog before conversion**: User picks output location first, FFmpeg writes directly there. Avoids temp file management. If conversion fails, delete the partial file (see origin decision).
- **FFmpeg command presets** (resolved from research):
  - WebM VP9: `-c:v libvpx-vp9 -crf 18 -b:v 0 -cpu-used 1 -row-mt 1 -c:a libopus -b:a 192k` (cpu-used 1 for reasonable speed; 0 is extremely slow)
  - H.265: `-c:v libx265 -crf 18 -preset slow -pix_fmt yuv420p -tag:v hvc1 -c:a aac -b:a 256k -movflags +faststart`
  - MP3: `-vn -c:a libmp3lame -b:a 320k`

## Open Questions

### Resolved During Planning

- **FFmpeg CRF values**: CRF 18 for both VP9 and H.265 (visually lossless range). For VP9, must include `-b:v 0` to enable pure CRF mode.
- **libx265 availability**: Standard FFmpeg Windows "full"/"GPL" builds include it. Document that user must use a full build, not LGPL-only.
- **Progress parsing approach**: Use `-progress pipe:1` for structured output instead of raw stderr parsing. Cleaner and more reliable.
- **Save As timing**: Before conversion (origin decision). FFmpeg writes directly to user-chosen path.

### Deferred to Implementation

- Exact pixel layout/sizing for Tab 2 controls (depends on window dimensions and testing)
- Whether `OPENFILENAMEW` struct requires additional padding on 64-bit (test at runtime)
- Handling of FFmpeg processes that hang or take extremely long (cancel button behavior)

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
User Flow:
  [Hotkey] -> showPopup() -> Tab Control visible
  [Click Tab 2] -> WM_NOTIFY/TCN_SELCHANGE -> hide tab1Controls, show tab2Controls
  [Click "Chon file MP4"] -> GetOpenFileNameW -> display filename in static label
  [Click "WebM VP9" / "H.265" / "MP3"] -> GetSaveFileNameW -> launch goroutine

Conversion Goroutine:
  1. Locate ffmpeg.exe next to app exe
  2. Build command args based on selected format
  3. Start exec.Command with stdout pipe (for -progress pipe:1)
  4. Parse "out_time_ms" from stdout, get total duration from stderr
  5. PostMessageW(mainHwnd, WM_APP+1, progressPercent, 0) periodically
  6. On completion: PostMessageW(mainHwnd, WM_APP+2, success, 0)
  7. On failure: PostMessageW(mainHwnd, WM_APP+2, error, 0) + delete partial file

UI Thread (wndProc):
  WM_APP+1 -> update progress bar position + percent label
  WM_APP+2 -> show completion message or error, reset UI state
```

## Implementation Units

- [x] **Unit 1: Add comctl32.dll and comdlg32.dll bindings**

  **Goal:** Load the two new DLLs and declare all procs needed for tab control, progress bar, and file dialogs.

  **Requirements:** R1, R2, R7, R9, R10

  **Dependencies:** None

  **Files:**
  - Modify: `main.go`

  **Approach:**
  - Add `comctl32` and `comdlg32` to the DLL declarations section alongside existing user32/kernel32/gdi32/advapi32
  - Declare procs: `InitCommonControlsEx`, `GetOpenFileNameW`, `GetSaveFileNameW`
  - Add constants: `ICC_TAB_CLASSES`, `ICC_PROGRESS_CLASS`, `TCM_INSERTITEM`, `TCM_GETCURSEL`, `TCN_SELCHANGE`, `WM_NOTIFY`, `WM_APP`, `PBM_SETPOS`, `PBM_SETRANGE`, `PBM_SETMARQUEE`, `PBS_MARQUEE`, `PBS_SMOOTH`, `OFN_FILEMUSTEXIST`, `OFN_OVERWRITEPROMPT`, `OFN_PATHMUSTEXIST`
  - Define `OPENFILENAMEW` struct (careful with 64-bit alignment: the struct has pointer fields that affect padding)
  - Define `INITCOMMONCONTROLSEX` struct
  - Define `TCITEMW` struct for tab items
  - Define `NMHDR` struct for WM_NOTIFY
  - Add new control IDs: `IDC_TAB_CTRL`, `IDC_CONV_FILE_BTN`, `IDC_CONV_FILE_LABEL`, `IDC_CONV_WEBM_BTN`, `IDC_CONV_H265_BTN`, `IDC_CONV_MP3_BTN`, `IDC_CONV_PROGRESS`, `IDC_CONV_STATUS`
  - Call `InitCommonControlsEx` early in `main()` before window creation

  **Patterns to follow:**
  - Existing DLL/proc declaration pattern at `main.go:24-71`
  - Existing struct definitions at `main.go:179-216`
  - Existing constant blocks at `main.go:77-173`

  **Test scenarios:**
  - Happy path: App starts successfully with comctl32/comdlg32 initialized, no crashes
  - Error path: If InitCommonControlsEx fails, app should log error and show message box

  **Verification:**
  - App compiles and starts without errors
  - Log shows comctl32 initialization success

- [x] **Unit 2: Add Tab Control to main window**

  **Goal:** Replace the flat main window layout with a Tab Control containing two tabs. Tab 1 shows existing VN Input controls, Tab 2 shows converter controls.

  **Requirements:** R1, R2, R6

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `main.go`

  **Approach:**
  - In `WM_CREATE`: create SysTabControl32 as first child, sized to fill most of the window
  - Insert two tabs via `TCM_INSERTITEM`: "VN Input" and "Video Convert"
  - Create Tab 1 controls (existing edit, OK, Cancel, Prompts buttons) as children — same as current code but with adjusted Y positions to account for tab header height (~25px)
  - Create Tab 2 controls: "Chon file MP4" button, static label for filename, three convert buttons (WebM VP9, H.265, MP3), progress bar (msctls_progress32), status label
  - Store HWNDs in two slices: `tab1Controls []syscall.Handle` and `tab2Controls []syscall.Handle`
  - Add `switchTab(index int)` function that shows/hides the correct set of controls
  - Add `WM_NOTIFY` handler in `wndProc` to detect `TCN_SELCHANGE` and call `switchTab`
  - Initially show Tab 1, hide Tab 2

  **Patterns to follow:**
  - Existing control creation in `WM_CREATE` at `main.go:826-882`
  - Existing WM_COMMAND handler pattern at `main.go:892-902`

  **Test scenarios:**
  - Happy path: Window shows with tab headers "VN Input" and "Video Convert"
  - Happy path: Clicking Tab 1 shows VN Input controls, hides converter controls
  - Happy path: Clicking Tab 2 shows converter controls, hides VN Input controls
  - Happy path: Existing VN Input functionality (type, Ctrl+Enter, Esc) works identically on Tab 1
  - Edge case: Switching tabs rapidly does not cause visual glitches or orphaned controls

  **Verification:**
  - Both tabs are visible and clickable
  - Controls switch correctly between tabs
  - All existing VN Input features continue to work on Tab 1

- [x] **Unit 3: Implement file selection dialog (Open File)**

  **Goal:** Allow user to select an MP4 file via standard Windows Open File dialog and display the selected filename.

  **Requirements:** R2

  **Dependencies:** Unit 2

  **Files:**
  - Modify: `main.go`

  **Approach:**
  - Add handler for `IDC_CONV_FILE_BTN` click in `WM_COMMAND`
  - Populate `OPENFILENAMEW` struct with filter `"MP4 Files (*.mp4)\0*.mp4\0All Files (*.*)\0*.*\0"` and `OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST`
  - Call `GetOpenFileNameW` — returns TRUE if file selected
  - Store selected file path in a global `var selectedMP4 string`
  - Update the static label (`IDC_CONV_FILE_LABEL`) with the basename of selected file
  - Enable convert buttons only when a file is selected

  **Patterns to follow:**
  - Existing button click handling in `WM_COMMAND` at `main.go:892-902`

  **Test scenarios:**
  - Happy path: Click "Chon file" -> dialog opens filtered to *.mp4 -> select file -> filename displayed
  - Happy path: Cancel dialog -> no file selected, label unchanged
  - Edge case: File path with Vietnamese/Unicode characters displays correctly

  **Verification:**
  - Selected MP4 path is stored and displayed correctly
  - Convert buttons respond to file selection state

- [x] **Unit 4: Implement Save As dialog**

  **Goal:** Open a Save As dialog before conversion, letting user choose output location with appropriate file extension.

  **Requirements:** R10

  **Dependencies:** Unit 3

  **Files:**
  - Modify: `main.go`

  **Approach:**
  - Create `showSaveDialog(defaultName string, filterExt string) string` function
  - Populate `OPENFILENAMEW` with `OFN_OVERWRITEPROMPT | OFN_PATHMUSTEXIST`
  - Set filter based on output format: `.webm` for VP9, `.mp4` for H.265, `.mp3` for MP3
  - Set default filename to `basename(selectedMP4)` with new extension
  - Returns empty string if user cancels

  **Patterns to follow:**
  - Same OPENFILENAMEW struct usage as Unit 3

  **Test scenarios:**
  - Happy path: Dialog opens with correct default filename and extension filter
  - Happy path: User picks location, path returned correctly
  - Happy path: Cancel dialog -> returns empty string, conversion does not start
  - Edge case: File path with spaces and Unicode works correctly

  **Verification:**
  - Save As dialog opens with correct defaults for each format
  - Returned path has correct extension

- [x] **Unit 5: Implement FFmpeg conversion engine**

  **Goal:** Run FFmpeg in a background goroutine with progress reporting back to the UI thread via PostMessageW.

  **Requirements:** R3, R4, R5, R7, R8, R9, R11

  **Dependencies:** Unit 4

  **Files:**
  - Modify: `main.go`

  **Approach:**
  - Add `pPostMessageW = user32.NewProc("PostMessageW")` to DLL proc declarations
  - Add `import "os/exec"` and `import "strconv"` and `import "regexp"`
  - Define custom messages: `WM_CONVERT_PROGRESS = WM_APP + 1`, `WM_CONVERT_DONE = WM_APP + 2`
  - Add `var converting bool` state flag to prevent concurrent conversions
  - Create `locateFFmpeg() string` — looks for ffmpeg.exe next to the app exe. Returns empty if not found.
  - Create `startConversion(format string)` — main entry point from button click:
    1. Check `converting` flag (prevent double-click)
    2. Validate `selectedMP4` is set
    3. Call `locateFFmpeg()`, show error if missing (R8)
    4. Show Save As dialog with format-appropriate extension
    5. If user cancels Save As, return
    6. Set `converting = true`, disable convert buttons, set progress bar to 0%
    7. Launch `go runFFmpeg(ffmpegPath, inputPath, outputPath, format)`
  - Create `runFFmpeg(ffmpegPath, input, output, format string)`:
    1. Build command args based on format:
       - "webm": `-i input -c:v libvpx-vp9 -crf 18 -b:v 0 -cpu-used 1 -row-mt 1 -c:a libopus -b:a 192k -progress pipe:1 -y output`
       - "h265": `-i input -c:v libx265 -crf 18 -preset slow -pix_fmt yuv420p -tag:v hvc1 -c:a aac -b:a 256k -movflags +faststart -progress pipe:1 -y output`
       - "mp3": `-i input -vn -c:a libmp3lame -b:a 320k -progress pipe:1 -y output`
    2. Create `exec.Command`, pipe stdout for progress, pipe stderr for duration
    3. Start the process
    4. Parse total duration from early stderr output (`Duration: HH:MM:SS.ss`)
    5. Continuously read stdout lines, parse `out_time_ms=` values
    6. Calculate percent = out_time_ms / total_duration_ms * 100
    7. Call `PostMessageW(mainHwnd, WM_CONVERT_PROGRESS, percent, 0)`
    8. Wait for process to complete
    9. If error: delete partial output file, `PostMessageW(mainHwnd, WM_CONVERT_DONE, 0, errorCode)`
    10. If success: `PostMessageW(mainHwnd, WM_CONVERT_DONE, 1, 0)`
  - Handle custom messages in `wndProc`:
    - `WM_CONVERT_PROGRESS`: update progress bar position (`PBM_SETPOS`) and status label
    - `WM_CONVERT_DONE`: if success show "Hoan thanh!", if error show error message. Re-enable convert buttons, set `converting = false`
  - If duration is unknown (N/A), set progress bar to `PBS_MARQUEE` style and skip percent display

  **Patterns to follow:**
  - Existing goroutine pattern at `main.go:1130-1136` (clipboard restore)
  - Existing message handling pattern in `wndProc`

  **Test scenarios:**
  - Happy path: Convert MP4 to WebM -> progress bar updates -> file saved correctly
  - Happy path: Convert MP4 to H.265 -> progress bar updates -> file saved correctly
  - Happy path: Convert MP4 to MP3 -> progress bar updates -> file saved correctly
  - Error path: ffmpeg.exe not found -> clear error message shown
  - Error path: No file selected -> convert buttons do nothing / show message
  - Error path: FFmpeg process fails mid-conversion -> error message + partial file deleted
  - Error path: User double-clicks convert while conversion is running -> ignored
  - Edge case: Input file with no duration metadata -> marquee progress bar mode
  - Integration: Convert while Tab 1 VN Input is used (switch tabs during conversion) -> conversion continues, progress updates when switching back to Tab 2

  **Verification:**
  - All three conversion formats produce valid output files
  - Progress bar updates in real-time during conversion
  - UI remains responsive during conversion (not frozen)
  - Error cases produce clear messages and clean up partial files

- [x] **Unit 6: Update build and documentation**

  **Goal:** Update build script and config documentation for the new feature.

  **Requirements:** R7

  **Dependencies:** Unit 5

  **Files:**
  - Modify: `build.bat`
  - Modify: `config.json` (if any new config fields)
  - Modify: `README.md`

  **Approach:**
  - Update `build.bat` with a note about placing ffmpeg.exe next to the built exe
  - Update README with: new tab feature, required ffmpeg.exe placement, supported conversions
  - No new config fields needed (presets are fixed)

  **Test scenarios:**
  - Happy path: Build completes successfully with `go build` command
  - Happy path: README accurately describes the new feature

  **Verification:**
  - Build produces working exe
  - Documentation matches actual behavior

## System-Wide Impact

- **Interaction graph:** The new Tab Control changes the main window's WM_CREATE and adds WM_NOTIFY handling. All existing WM_COMMAND handlers remain unchanged. The prompt picker (Ctrl+P) should only work when Tab 1 is active.
- **Error propagation:** FFmpeg errors propagate from goroutine to UI thread via PostMessageW. Partial output files are cleaned up on error.
- **State lifecycle risks:** `converting` flag prevents concurrent conversions. Tab switching during conversion must not interfere with progress updates. The clipboard save/restore flow (Tab 1) and FFmpeg conversion (Tab 2) are independent and do not share state.
- **API surface parity:** The hotkey (Ctrl+Shift+Space) still shows the popup. Tab state could be remembered between popup show/hide cycles or always default to Tab 1.
- **Unchanged invariants:** All Tab 1 functionality (text input, Ctrl+Enter paste, Esc hide, Ctrl+P prompts, clipboard backup/restore) must remain identical. The prompt picker window is unaffected.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| OPENFILENAMEW struct alignment on 64-bit | Test thoroughly; the struct has pointer fields that create padding. Use `unsafe.Sizeof` verification in debug builds |
| FFmpeg build missing libx265 or libvpx | Document requirement for "full" or "GPL" build. Show specific error if codec not found (FFmpeg exits with error mentioning the codec) |
| VP9 encoding with cpu-used 0 is extremely slow | Use cpu-used 1 instead (still high quality, much faster). Document that conversion may take a while for large files |
| Progress bar not updating smoothly | PostMessageW is safe for cross-thread calls. Buffer progress updates to avoid flooding the message queue (e.g., update at most every 500ms) |
| Single file grows to ~2000+ lines | Acceptable for this project's single-file architecture. Use clear section comments as already done |
| comctl32.dll InitCommonControlsEx may affect existing controls | Call it once at startup before any window creation. This should not affect standard EDIT/BUTTON controls |

## Sources & References

- **Origin document:** [docs/brainstorms/2026-03-31-mp4-converter-tab-requirements.md](docs/brainstorms/2026-03-31-mp4-converter-tab-requirements.md)
- FFmpeg VP9 encoding guide: recommended CRF 18, `-b:v 0`, cpu-used 0-1
- FFmpeg H.265 encoding: CRF 18, preset slow, `-tag:v hvc1` for compatibility
- FFmpeg `-progress pipe:1` for machine-readable progress output
- Windows FFmpeg builds: gyan.dev full or BtbN GPL builds include all required codecs
