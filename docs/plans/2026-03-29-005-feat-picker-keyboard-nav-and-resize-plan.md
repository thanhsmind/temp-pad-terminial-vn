---
title: "feat: Add keyboard navigation to prompt picker and increase input window size"
type: feat
status: completed
date: 2026-03-29
---

# feat: Add keyboard navigation to prompt picker and increase input window size

## Overview

Two improvements to the VN Input Helper UI:
1. Enable keyboard-driven prompt selection — arrow keys to navigate the filtered list, Enter to insert — so users never need the mouse after Ctrl+P
2. Increase the main input window size (height ×3, width ×1.5) to give more room for composing text

## Problem Frame

The prompt picker currently requires mouse interaction (click or double-click) to select and insert a prompt. When the user opens the picker with Ctrl+P and types a search query, they must switch to the mouse to select from the filtered list. This breaks the keyboard-driven workflow. Additionally, the main input window (500×200) is too small for composing longer text.

## Requirements Trace

- R1. Arrow Down from search box moves selection/focus to the listbox
- R2. Arrow Up from the first listbox item returns focus to the search box
- R3. Enter on a selected listbox item inserts the prompt (same as double-click or "Chèn" button)
- R4. Arrow Up/Down within the listbox navigates items (native Win32 behavior when focused)
- R5. Escape in the picker closes the picker (not the main popup) — requires fixing the existing Esc handler which currently calls `hidePopup()` unconditionally
- R6. Main input window dimensions increase to 750×600 (width ×1.5, height ×3 from current 500×200)
- R7. After search filter updates, auto-select the first item in the list so Enter works immediately

## Scope Boundaries

- No changes to the prompt file format or loading logic
- No changes to the picker window dimensions (450×400 is adequate for the list)
- No config.json additions for picker — picker size stays hardcoded
- No DPI awareness changes (known gap, separate concern)

## Context & Research

### Relevant Code and Patterns

- **Message loop pre-dispatch pattern** (`main.go:1204-1241`): All keyboard shortcuts (Ctrl+Enter, Ctrl+P, Esc) are intercepted before `TranslateMessage`/`DispatchMessage`. This is the established pattern for handling keystrokes that span child controls.
- **Picker WndProc** (`main.go:613-708`): Handles `WM_CREATE`, `WM_COMMAND`, `WM_CLOSE`, `WM_DESTROY`. No `WM_KEYDOWN` handler exists.
- **`filterPrompts()`** (`main.go:710-743`): Clears and repopulates listbox. Does not auto-select any item after filtering.
- **`insertSelectedPrompt()`** (`main.go:745-773`): Reads `LB_GETCURSEL`, maps through `filteredIndices`, appends content.
- **VK constants** (`main.go:120-125`): Currently defines `VK_RETURN`, `VK_ESCAPE`, `VK_CONTROL`, `VK_V`, `VK_SPACE`, `VK_P`. Needs `VK_UP` (0x26) and `VK_DOWN` (0x28).
- **LB messages** (`main.go:151-158`): Has `LB_GETCURSEL`, `LB_ADDSTRING`, `LB_RESETCONTENT`. Needs `LB_SETCURSEL` (0x0186) and `LB_GETCOUNT` (0x018B).
- **Window sizing** (`config.json`): Currently `500×200`. Child controls in main window use relative positioning (`w-offset`, `h-offset`) so resizing just requires changing config values.

### Institutional Learnings

- Win32 controls use hardcoded pixel dimensions — relative positioning formulas keep layouts consistent when window size changes (from `docs/solutions/ui-bugs/fix-vn-input-helper-popup-sizing-2026-03-29.md`)

## Key Technical Decisions

- **Message loop interception over WndProc subclassing**: Arrow keys and Enter in child controls (EDIT, LISTBOX) are delivered to the child's WndProc, not the parent. Subclassing would require `SetWindowLongPtrW` with `GWLP_WNDPROC` for each control. The message loop pre-dispatch approach is simpler, already established in this codebase, and works across all child controls. We intercept keystrokes in the message loop by checking `m.Hwnd` against `pickerSearchHwnd` and `pickerListHwnd`.

- **Auto-select first item after filter**: After `filterPrompts()` repopulates the listbox, send `LB_SETCURSEL` with index 0. This makes Enter work immediately after typing a search query without needing to arrow down first.

- **Config change for window size**: Change `config.json` values to 750×600. No code changes needed — the main window's child controls already use relative positioning.

## Open Questions

### Resolved During Planning

- **Where to handle picker keyboard events?**: In the message loop (pre-dispatch), following the established pattern. Check `m.Hwnd` against picker control handles.
- **Should listbox get focus on Down arrow?**: Yes — `SetFocus` to listbox + `LB_SETCURSEL` to index 0 if no selection exists.
- **What happens with Enter when nothing is selected?**: `insertSelectedPrompt()` already checks `LB_GETCURSEL != LB_ERR` and returns early. No change needed.

### Deferred to Implementation

- **Exact behavior when arrowing up past item 0**: Need to test whether `LB_GETCURSEL` returns 0 reliably. If the current selection is 0 and user presses Up, move focus back to search box.

## Implementation Units

- [ ] **Unit 1: Add VK and LB constants**

**Goal:** Add missing Win32 constants needed for keyboard navigation

**Requirements:** R1, R2, R3, R4

**Dependencies:** None

**Files:**
- Modify: `main.go` (constants section, lines 77-167)

**Approach:**
- Add `VK_UP = 0x26`, `VK_DOWN = 0x28` to the VK constants block
- Add `LB_SETCURSEL = 0x0186`, `LB_GETCOUNT = 0x018B` to the LB constants block

**Patterns to follow:**
- Existing constant declarations at `main.go:120-158`

**Test scenarios:**
- Happy path: Constants compile without error and match Win32 API values

**Verification:**
- `go build` succeeds

- [ ] **Unit 2: Auto-select first item after search filter**

**Goal:** After filtering, automatically select the first listbox item so Enter works immediately

**Requirements:** R7

**Dependencies:** Unit 1

**Files:**
- Modify: `main.go` (`filterPrompts()` function, line 710)

**Approach:**
- At the end of `filterPrompts()`, after repopulating the listbox, if `len(filteredIndices) > 0`, send `LB_SETCURSEL` with index 0

**Patterns to follow:**
- Existing `pSendMessageW.Call` pattern for listbox messages in `filterPrompts()`

**Test scenarios:**
- Happy path: Type search text → first matching item is highlighted in the list
- Edge case: Search yields no results → no selection set, Enter does nothing
- Happy path: Clear search text → all items shown, first is selected

**Verification:**
- After typing in search box, the first filtered result is visually selected (highlighted)

- [ ] **Unit 3: Keyboard navigation in message loop**

**Goal:** Add arrow key and Enter handling for the picker in the message loop pre-dispatch section

**Requirements:** R1, R2, R3, R4

**Dependencies:** Unit 1, Unit 2

**Files:**
- Modify: `main.go` (message loop, lines 1204-1241)

**Approach:**
Add a new pre-dispatch block in the message loop (after the Ctrl+P block, before the Esc block) that checks if the picker is open (`pickerHwnd != 0`) and handles:

1. **Down arrow in search box** (`m.Message == WM_KEYDOWN && m.Hwnd == pickerSearchHwnd && m.WParam == VK_DOWN`):
   - Get listbox count via `LB_GETCOUNT`
   - If count > 0: `SetFocus` to listbox, `LB_SETCURSEL` to 0 if no current selection
   - `continue` to consume the keystroke

2. **Up arrow in listbox at first item** (`m.Message == WM_KEYDOWN && m.Hwnd == pickerListHwnd && m.WParam == VK_UP`):
   - Get current selection via `LB_GETCURSEL`
   - If selection is 0 or LB_ERR: `SetFocus` to search box, `continue`
   - Otherwise: let it fall through to normal dispatch (native listbox Up handling)

3. **Enter in listbox** (`m.Message == WM_KEYDOWN && m.Hwnd == pickerListHwnd && m.WParam == VK_RETURN`):
   - Call `insertSelectedPrompt()`
   - `continue` to consume the keystroke

4. **Escape when picker is open** (`m.Message == WM_KEYDOWN && m.WParam == VK_ESCAPE && pickerHwnd != 0`):
   - Call `pDestroyWindow` on `pickerHwnd` to close just the picker
   - `continue` to consume the keystroke (prevents existing Esc handler from hiding the main popup)

**Patterns to follow:**
- Existing Ctrl+P block (`main.go:1225-1231`): check `m.Message == WM_KEYDOWN`, check `m.WParam`, check handle, take action, `continue`
- Existing Ctrl+Enter block (`main.go:1219-1222`)

**Test scenarios:**
- Happy path: Open picker → type query → press Down → listbox gets focus with first item selected
- Happy path: In listbox → press Down/Up → items navigate (native behavior, no code needed)
- Happy path: In listbox with item selected → press Enter → prompt content inserted into main edit
- Edge case: In listbox at first item → press Up → focus returns to search box
- Edge case: In listbox with no items → Down arrow from search does nothing
- Edge case: Enter in listbox with no selection → nothing happens (existing guard in `insertSelectedPrompt`)
- Happy path: Esc while picker is open → picker closes, main popup stays visible
- Integration: Type query → Down → Enter → prompt inserted and picker closes, focus returns to main edit

**Verification:**
- Full keyboard flow works: Ctrl+P → type → Down → Enter → text appears in main edit control
- Arrow Up from first item returns to search box with cursor intact
- Mouse-based selection still works unchanged

- [ ] **Unit 4: Increase main input window size**

**Goal:** Make the input window larger — height ×3, width ×1.5

**Requirements:** R6

**Dependencies:** None (independent of Units 1-3)

**Files:**
- Modify: `config.json`

**Approach:**
- Change `"width": 500` → `"width": 750`
- Change `"height": 200` → `"height": 600`
- No code changes needed — main window child controls use relative positioning (`w-40`, `h-100`, `w-300`, `w-130`, `h-80`)

**Patterns to follow:**
- Existing relative layout in `wndProc` WM_CREATE (`main.go:817-860`)

**Test scenarios:**
- Happy path: Window opens at 750×600, centered on screen
- Happy path: Edit control fills most of the window area (width=710, height=500)
- Happy path: Buttons are positioned correctly at bottom of larger window
- Edge case: Window fits on a 1080p screen (750×600 < 1920×1080)

**Verification:**
- Main input window is visually larger with a spacious text editing area
- All buttons are visible and correctly positioned

## System-Wide Impact

- **Interaction graph:** The message loop handles keystrokes for both the main window and picker. New pre-dispatch checks must be guarded by `pickerHwnd != 0` to avoid interference when the picker is closed.
- **Error propagation:** No new error paths — `insertSelectedPrompt()` already handles edge cases.
- **State lifecycle risks:** `pickerListHwnd` and `pickerSearchHwnd` are cleared on `WM_DESTROY`. The message loop checks must handle the case where these are 0 (picker closed between keypress and dispatch).
- **Unchanged invariants:** Ctrl+Enter, Ctrl+P, Esc, and global hotkey behavior remain unchanged. Mouse-based selection in picker remains unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Arrow keys consumed by message loop may prevent native listbox navigation | Only intercept specific cases (Down in search, Up at item 0, Enter in list). All other keystrokes fall through to normal dispatch. |
| Picker handle becomes 0 between check and action | Guard all `pSendMessageW` / `pSetFocus` calls with handle != 0 checks, matching existing patterns. |
| Large window may not fit small screens | 750×600 fits comfortably on 1366×768 (minimum common resolution). Config is user-editable. |
