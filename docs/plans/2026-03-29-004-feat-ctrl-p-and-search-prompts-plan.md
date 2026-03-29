---
title: "feat: Add Ctrl+P shortcut and search filter to prompt picker"
type: feat
status: completed
date: 2026-03-29
---

# feat: Add Ctrl+P shortcut and search filter to prompt picker

## Overview

Add Ctrl+P keyboard shortcut to open the prompt picker while the popup is visible, and add a search/filter EDIT control at the top of the picker window for fulltext filtering of prompts by name, description, and content.

## Problem Frame

Users must click the "Prompts" button to access templates. A keyboard shortcut (Ctrl+P) is faster. Additionally, when the prompt list grows, users need a way to quickly filter/search for the relevant prompt without scrolling.

## Requirements Trace

- R1. Ctrl+P while the main popup is visible opens the prompt picker (same as clicking the "Prompts" button)
- R2. Ctrl+P should not interfere with normal typing when the popup is hidden
- R3. Search EDIT control at the top of the picker window
- R4. Typing in the search box filters the LISTBOX in real-time, matching against prompt name, description, and content (case-insensitive)
- R5. Clearing the search box restores the full list
- R6. Search is simple substring match — no regex, no fuzzy matching

## Scope Boundaries

- No new global hotkey — Ctrl+P only works within the message loop when the popup is visible
- No fuzzy/ranked search — simple case-insensitive substring match
- No changes to prompt file format or loading logic

## Context & Research

### Relevant Code and Patterns

- `main.go` message loop (line ~1150): existing pattern for intercepting keystrokes — `isCtrlEnter()` checks `WM_KEYDOWN` + `GetKeyState(VK_CONTROL)`, same pattern reusable for Ctrl+P
- `main.go` `showPromptPicker()`: existing function to call
- `main.go` `pickerWndProc` WM_CREATE: creates LISTBOX and buttons — add EDIT control above LISTBOX
- `main.go` `loadedPrompts` slice: the data source for filtering
- Win32 `EN_CHANGE` notification (via WM_COMMAND): fires when EDIT text changes — use for real-time filter
- Win32 `LB_RESETCONTENT` + `LB_ADDSTRING`: clear and repopulate LISTBOX on filter change

## Key Technical Decisions

- **Ctrl+P detection in the message loop**: Same approach as Ctrl+Enter detection. Check `WM_KEYDOWN` with `wParam == VK_P` and `GetKeyState(VK_CONTROL) & 0x8000`. Only trigger when `popupVisible` is true and `pickerHwnd` is 0 (picker not already open).
- **VK_P constant = 0x50**: Standard Windows virtual key code for 'P'.
- **Search EDIT at top of picker, LISTBOX shifted down**: The search EDIT takes ~30px at the top. LISTBOX Y position shifts from 10 to 45. Picker height stays 400.
- **Filter by repopulating LISTBOX**: On each `EN_CHANGE`, clear the LISTBOX and re-add only matching prompts. Maintain a `filteredIndices` slice mapping LISTBOX positions back to `loadedPrompts` indices for correct selection.
- **Case-insensitive substring match**: Use `strings.Contains(strings.ToLower(target), strings.ToLower(query))` on name + description + content.

## Implementation Units

- [ ] **Unit 1: Add Ctrl+P shortcut to open prompt picker**

**Goal:** Pressing Ctrl+P while the popup is visible opens the prompt picker.

**Requirements:** R1, R2

**Dependencies:** None

**Files:**
- Modify: `main.go`

**Approach:**
- Add `VK_P = 0x50` constant
- In the message loop (after the Ctrl+Enter check), add a Ctrl+P check: `WM_KEYDOWN`, `wParam == VK_P`, `GetKeyState(VK_CONTROL) & 0x8000 != 0`, `popupVisible`, `pickerHwnd == 0`
- Call `showPromptPicker()` and `continue`

**Patterns to follow:**
- `isCtrlEnter()` function pattern and its usage in the message loop

**Test scenarios:**
- Happy path: Popup visible → Ctrl+P → prompt picker opens
- Edge case: Popup hidden → Ctrl+P → nothing happens (no crash, no picker)
- Edge case: Picker already open → Ctrl+P → nothing happens (doesn't open duplicate)
- Happy path: After picker closes → Ctrl+P opens it again

**Verification:**
- Ctrl+P opens prompt picker when popup is visible
- Normal 'P' key typing still works in the edit box (Ctrl must be held)

- [ ] **Unit 2: Add search EDIT and filter logic to prompt picker**

**Goal:** Add a search box at the top of the picker that filters the LISTBOX in real-time.

**Requirements:** R3, R4, R5, R6

**Dependencies:** Unit 1 (or existing prompt picker)

**Files:**
- Modify: `main.go`

**Approach:**
- Add `IDC_PICKER_SEARCH = 2004` constant and `EN_CHANGE = 0x0300` constant
- Add `pickerSearchHwnd syscall.Handle` and `filteredIndices []int` to picker state variables
- In `pickerWndProc` WM_CREATE:
  - Create search EDIT control at (10, 10) size (410, 25) with placeholder-like behavior
  - Shift LISTBOX Y from 10 to 45, reduce height from 300 to 265
  - Shift buttons Y from 320 to 320 (unchanged — still fits)
  - Apply font to search EDIT
- In `pickerWndProc` WM_COMMAND:
  - Detect `EN_CHANGE` from `IDC_PICKER_SEARCH` (id == IDC_PICKER_SEARCH && notif == EN_CHANGE >> 16 portion)
  - Call new `filterPrompts()` function
- `filterPrompts()`:
  - Get text from search EDIT via `WM_GETTEXTLENGTH` + `WM_GETTEXT`
  - Clear LISTBOX via `LB_RESETCONTENT` message
  - Reset `filteredIndices` slice
  - For each prompt in `loadedPrompts`, check if lowered query is a substring of lowered (name + " " + description + " " + content)
  - If match, add to LISTBOX via `LB_ADDSTRING` and append original index to `filteredIndices`
  - If query is empty, show all prompts (full reset)
- Update `insertSelectedPrompt()`: use `filteredIndices[sel]` instead of `sel` directly to map LISTBOX selection back to the correct prompt in `loadedPrompts`
- Set focus to search EDIT after picker window is shown (so user can start typing immediately)

**Patterns to follow:**
- `getEditText()` pattern for reading text from an EDIT control
- Existing LISTBOX population loop in `pickerWndProc` WM_CREATE
- `WM_SETFONT` application to new controls

**Test scenarios:**
- Happy path: Open picker → type "code" → only prompts containing "code" in name/description/content shown
- Happy path: Clear search box → full list restored
- Happy path: Search with match → select filtered item → correct prompt content inserted
- Edge case: Search with no matches → LISTBOX empty, selecting nothing does nothing
- Edge case: Search is case-insensitive — "CODE" matches "Code Review"
- Edge case: Search matches content (not just name) — a prompt whose name doesn't match but content contains the query still appears
- Integration: Ctrl+P → picker opens → type search → select → content inserted correctly into edit box

**Verification:**
- Search box appears at top of picker, focused on open
- Typing filters the list in real-time
- Selecting a filtered item inserts the correct prompt content
- Empty search shows all prompts

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Ctrl+P might conflict with some IME combinations | Only triggers on `WM_KEYDOWN` with Ctrl state, which IME typically doesn't produce |
| `filteredIndices` mapping could be wrong if list is repopulated during selection | `filterPrompts()` rebuilds `filteredIndices` atomically before any selection can happen |
