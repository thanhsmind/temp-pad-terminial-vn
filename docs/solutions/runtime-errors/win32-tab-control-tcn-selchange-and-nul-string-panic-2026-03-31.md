---
title: "Win32 Tab Control TCN_SELCHANGE wrong constant and StringToUTF16 NUL panic"
date: "2026-03-31"
category: runtime-errors
module: vn-input-helper
problem_type: runtime_error
component: tooling
symptoms:
  - "Tab switching reversed — clicking Video Convert showed VN Input controls and vice versa"
  - "PANIC crash: syscall: string with NUL passed to StringToUTF16 when clicking file chooser button"
  - "Controls (buttons, progress bar) overflowing outside visible window area"
root_cause: wrong_api
resolution_type: code_fix
severity: high
tags:
  - win32
  - syscall
  - tab-control
  - tcn-selchange
  - utf16
  - nul-byte
  - go
  - window-chrome
---

# Win32 Tab Control TCN_SELCHANGE wrong constant and StringToUTF16 NUL panic

## Problem

A Go Win32 application using raw syscall had three bugs after adding Tab Control (SysTabControl32) and file dialog features: reversed tab switching, a hard panic when opening file dialogs, and controls overflowing the visible window area.

## Symptoms

- **Tab switching reversed**: Clicking the "Video Convert" tab showed VN Input content, and clicking VN Input showed Video Convert content.
- **PANIC on file dialog**: App crashed with `"PANIC: syscall: string with NUL passed to StringToUTF16"` when clicking the file picker button.
- **Controls overflow**: Buttons and progress bar rendered beyond the visible window boundary, clipped by the window frame.

## What Didn't Work

- For the tab bug, the `switchTab()` function logic and `WM_NOTIFY` handler appeared correct on inspection — the show/hide logic for each tab's controls was properly structured. The issue was not in the switching logic itself but in *when* the notification fired.
- For the file dialog panic, the natural approach of passing a single string with embedded `\x00` characters to `syscall.StringToUTF16()` seemed like the obvious way to build NUL-separated filter strings, matching how it would be done in C.

## Solution

### Bug 1 — Tab switching: wrong notification constant

Before:
```go
TCN_SELCHANGE = -552 // WRONG — this is TCN_SELCHANGING
```

After:
```go
TCN_SELCHANGE = -551 // TCN_FIRST - 1 (correct)
```

### Bug 2 — File dialog panic: NUL bytes in StringToUTF16

Before:
```go
filter := syscall.StringToUTF16("MP4 Files (*.mp4)\x00*.mp4\x00All Files (*.*)\x00*.*\x00\x00")
// PANICS — StringToUTF16 does not accept embedded NUL bytes
```

After:
```go
func makeFilterUTF16(segments ...string) []uint16 {
    var result []uint16
    for _, s := range segments {
        u, _ := syscall.UTF16FromString(s)
        result = append(result, u...) // each call appends a NUL terminator
    }
    result = append(result, 0) // double-NUL terminator to end the filter list
    return result
}

// Usage:
filter := makeFilterUTF16("MP4 Files (*.mp4)", "*.mp4", "All Files (*.*)", "*.*")
```

### Bug 3 — Controls overflow: using window size instead of client area

Before:
```go
// Controls positioned using raw w, h (total window dimensions including chrome)
tabHwnd = createWindow(0, 0, w-16, h-8)
contentH = h - 50
```

After:
```go
clientW := w - 16   // subtract left + right border (~8px each)
clientH := h - 62   // subtract title bar (~31px) + top/bottom borders
tabHwnd = createWindow(0, 0, clientW, clientH)
contentH = clientH - 40
```

## Why This Works

**Bug 1**: Windows Tab Control notification codes are defined relative to `TCN_FIRST = -550`. `TCN_SELCHANGE` is `TCN_FIRST - 1 = -551` (fires *after* the selection changes), while `TCN_SELCHANGING` is `TCN_FIRST - 2 = -552` (fires *before* the change). Using `-552` meant `TCM_GETCURSEL` returned the *old* tab index, so the show/hide logic was applied to the wrong tab — producing the exact reversal symptom.

**Bug 2**: Go's `syscall.StringToUTF16()` explicitly panics if the input string contains NUL bytes (it treats NUL as a programming error, since Go strings are not NUL-terminated). Win32's `OPENFILENAME.lpstrFilter` requires NUL-separated segments with a double-NUL terminator. The fix builds the UTF16 slice segment by segment — `UTF16FromString()` processes one clean string and appends exactly one NUL terminator, which naturally serves as the separator between filter segments.

**Bug 3**: Win32 window dimensions (`w`, `h`) include non-client area — the title bar, borders, and frame. Controls are positioned within the *client area*, which is smaller. On a standard Windows setup with `WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU`, the title bar is roughly 31px and borders are roughly 8px on each side.

## Prevention

1. **Use authoritative references for Win32 constants**: Never guess or derive notification codes from memory. Cross-check against the Windows SDK headers (`CommCtrl.h`) or Microsoft documentation. Include the derivation as a comment: `TCN_SELCHANGE = -551 // TCN_FIRST - 1`.

2. **Wrap syscall string conversions for NUL-separated strings**: Create a helper function like `makeFilterUTF16()` upfront. Document that `syscall.StringToUTF16` panics on embedded NULs — this is a known Go footgun with Win32 interop. Use `syscall.UTF16FromString()` per segment instead.

3. **Use `GetClientRect` for layout calculations**: Query the actual client area dimensions at runtime rather than assuming fixed title bar and border sizes, which vary with DPI scaling, Windows theme, and window styles.

4. **Add source comments to Win32 constants**: For every constant pulled from Win32 headers, include a comment citing the header and derivation (e.g., `// CommCtrl.h: TCN_SELCHANGE = TCN_FIRST - 1`). This makes off-by-one errors immediately visible during review.

5. **Test on Windows early**: Tab switching and file dialog behavior can only be verified by running on Windows — cross-compilation alone won't catch these integration-level issues.

## Related Issues

- [docs/solutions/ui-bugs/fix-vn-input-helper-popup-sizing-2026-03-29.md](../ui-bugs/fix-vn-input-helper-popup-sizing-2026-03-29.md) — Earlier UI sizing fix in the same application (different root cause: hardcoded values too small)
- [docs/plans/2026-03-31-001-feat-mp4-converter-tab-plan.md](../../plans/2026-03-31-001-feat-mp4-converter-tab-plan.md) — Implementation plan that produced these bugs (noted OPENFILENAMEW alignment as a risk but missed the TCN_SELCHANGE constant and StringToUTF16 NUL issue)
