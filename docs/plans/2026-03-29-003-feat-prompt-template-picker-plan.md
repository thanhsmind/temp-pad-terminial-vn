---
title: "feat: Add prompt template picker with markdown-based prompt files"
type: feat
status: completed
date: 2026-03-29
---

# feat: Add prompt template picker with markdown-based prompt files

## Overview

Add a "Prompts" button to the popup window that opens a selection dialog listing prompt templates loaded from `prompts/*.md` files. Each prompt file uses YAML-like frontmatter (name, description) with the prompt content below. Selecting a prompt inserts its full content into the text edit box.

## Problem Frame

Users frequently type the same prompt patterns. Currently they must type or paste from an external source each time. A built-in prompt picker lets users maintain a library of reusable templates and insert them with two clicks.

## Requirements Trace

- R1. A "Prompts" button in the main popup window, alongside OK and Cancel
- R2. Clicking the button opens a selection dialog showing all available prompts
- R3. Each prompt displays its name (and optionally description) in the list
- R4. Selecting a prompt and confirming inserts the prompt content into the edit box
- R5. Prompts are loaded from `prompts/*.md` files next to the executable
- R6. Prompt file format: YAML-like frontmatter with `name` and `description` fields, content below the closing `---`
- R7. If no prompts directory or no `.md` files exist, the button is disabled or shows a message
- R8. Inserting a prompt appends to existing text (does not replace) — if the edit box has content, insert at the end with a newline separator

## Scope Boundaries

- No prompt editing, creation, or deletion from within the app
- No prompt categories, folders, search, or filtering
- No config.json settings for prompts (directory is fixed as `prompts/`)
- No recursive subdirectory scanning — only top-level `prompts/*.md`
- The prompt picker dialog is modal — user must close it before returning to the main edit box

## Context & Research

### Relevant Code and Patterns

- `main.go` WM_CREATE handler — pattern for creating child controls (BUTTON, EDIT) with `CreateWindowExW`
- `main.go` WM_COMMAND handler — pattern for handling button clicks by control ID
- `main.go` `loadConfig()` — pattern for resolving paths relative to the executable via `os.Executable()`
- `main.go` `utf16Ptr()` helper — used for all Win32 string parameters
- `main.go` button layout — OK at `(w-300, h-80)` 160x35, Cancel at `(w-130, h-80)` 120x35

### Institutional Learnings

- `docs/solutions/ui-bugs/fix-vn-input-helper-popup-sizing-2026-03-29.md` — Win32 buttons don't auto-size; must calculate width for label text

## Key Technical Decisions

- **Use a LISTBOX in a secondary popup window**: Consistent with the existing Win32 approach. A LISTBOX provides native keyboard navigation (arrow keys, Enter to select). The secondary window is created as a popup with `WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU`, similar to the main window but smaller and modal-like (brought to foreground, main popup hidden or stays behind).
- **Simple frontmatter parser instead of YAML library**: The format only needs `name` and `description` string fields. A line-by-line parser that looks for `---` delimiters and `key: value` pairs is simpler and avoids external dependencies. Matches the project's no-dependency philosophy.
- **Prompts directory next to executable**: Same pattern as `config.json` resolution — use `os.Executable()` to find the directory. Predictable and consistent.
- **Append rather than replace**: When inserting a prompt into an edit box that already has text, append with a newline. This lets users combine multiple prompts or add a prompt after typing some context.
- **"Prompts" button positioned left of OK button**: Natural reading order — Prompts (action) → OK (confirm) → Cancel (dismiss). Button width ~100px for label "Prompts".
- **Prompt picker as a separate registered window class**: Keeps the WndProc clean. The picker window has its own message handling for LISTBOX selection (LBN_DBLCLK for double-click select, an OK button for single-click + confirm).

## Open Questions

### Resolved During Planning

- **What happens if the prompts directory doesn't exist?** Show a message box: "No prompts found. Create a `prompts/` folder next to the exe and add `.md` files." This is simpler than disabling the button (which requires tracking state at creation time).
- **How to handle malformed prompt files?** Skip files that fail to parse (no `---` frontmatter). Log a warning. Don't crash.
- **Should the picker be a child of the main window or an independent window?** Independent top-level window (like the main popup itself) — simpler Win32 management, avoids child-window Z-order issues.
- **What encoding for prompt files?** UTF-8, consistent with Go's default file reading. The frontmatter parser reads bytes and converts to string. Win32 display uses UTF-16 conversion via existing `utf16Ptr()`.

### Deferred to Implementation

- **Exact LISTBOX Win32 style flags**: Will be determined during implementation. Likely `LBS_NOTIFY|LBS_NOINTEGRALHEIGHT|WS_VSCROLL`.
- **Exact window size for the picker dialog**: Will be tuned visually. Starting estimate: 400x350.
- **Whether to show description in the listbox or just the name**: Start with name only in the LISTBOX items. Description can be shown in a separate STATIC label when an item is highlighted (deferred — nice to have, not required for v1).

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
User flow:
  1. User presses hotkey → main popup appears (existing)
  2. User clicks "Prompts" button
  3. App scans prompts/*.md, parses frontmatter
  4. If no prompts found → show message box, return
  5. If prompts found → create picker window with LISTBOX
  6. LISTBOX populated with prompt names
  7. User double-clicks or clicks OK → selected prompt content inserted into edit box
  8. Picker window closes, focus returns to edit box

Prompt file format (prompts/example.md):
  ---
  name: Code Review Request
  description: Template for requesting a code review
  ---
  Please review the following code changes:

  **Context:** [describe what changed]
  **Files:** [list modified files]
  **Focus areas:** [what to look for]
```

## Implementation Units

- [ ] **Unit 1: Prompt file format and parser**

**Goal:** Define the prompt file format and implement loading/parsing of prompt files from the `prompts/` directory.

**Requirements:** R5, R6

**Dependencies:** None

**Files:**
- Modify: `main.go`
- Create: `prompts/example.md` (sample prompt file)

**Approach:**
- Define a `Prompt` struct with `Name`, `Description`, `Content` string fields and `FilePath` for debugging
- Implement `loadPrompts()` that:
  - Resolves `prompts/` directory relative to exe path (same pattern as config.json)
  - Lists `*.md` files using `os.ReadDir()`
  - For each file, calls `parsePromptFile(path)` which reads the file, finds `---` delimiters, extracts `name:` and `description:` values, and captures everything after the closing `---` as content
- Frontmatter parser: split on lines, find opening `---` (first line or first non-empty line), read `key: value` pairs until closing `---`, remainder is content. Trim whitespace from values and content.
- Return `[]Prompt` slice. Skip files that fail to parse. Log warnings for skipped files.

**Patterns to follow:**
- `loadConfig()` for exe-relative path resolution
- `logf()` for error reporting

**Test scenarios:**
- Happy path: `prompts/` directory with 2 valid .md files → returns 2 Prompt structs with correct Name, Description, Content
- Happy path: Prompt file with multi-line content → Content preserves all lines after closing `---`
- Edge case: `prompts/` directory doesn't exist → returns empty slice, no error
- Edge case: Empty `prompts/` directory → returns empty slice
- Edge case: File without `---` frontmatter → skipped, logged as warning
- Edge case: File with frontmatter but no `name:` field → skipped or uses filename as fallback name
- Edge case: File with extra whitespace around `key: value` → values trimmed correctly
- Error path: File read permission denied → skipped, logged

**Verification:**
- `loadPrompts()` returns correctly parsed prompts from sample files
- Malformed files are skipped without crashing

- [ ] **Unit 2: Add "Prompts" button to main popup**

**Goal:** Add a "Prompts" button to the main popup window layout, positioned left of the OK button.

**Requirements:** R1

**Dependencies:** Unit 1 (prompt struct must exist for the click handler reference, but the handler itself is Unit 3)

**Files:**
- Modify: `main.go`

**Approach:**
- Add `IDC_PROMPTS_BTN = 1004` constant
- Add `promptsBtn syscall.Handle` to app state variables
- In WM_CREATE handler, create the "Prompts" button:
  - Position: left of OK button. With current layout (w=600): Prompts at `(10, h-80)` width 100, OK at `(w-300, h-80)` width 160, Cancel at `(w-130, h-80)` width 120
  - Style: `WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON`
- Apply font to new button (same `hFont` as other controls)
- In WM_COMMAND handler, add `case IDC_PROMPTS_BTN: showPromptPicker()` (function implemented in Unit 3)

**Patterns to follow:**
- Existing OK and Cancel button creation in WM_CREATE
- Existing WM_COMMAND handler switch statement

**Test scenarios:**
- Happy path: Build and run → "Prompts" button visible in popup, properly positioned, no overlap with OK/Cancel
- Happy path: Button label "Prompts" fully visible at 100px width
- Integration: Clicking the button triggers `IDC_PROMPTS_BTN` in WM_COMMAND

**Verification:**
- Popup shows 3 buttons: Prompts (left), OK (right-center), Cancel (right)
- No button overlap, consistent styling

- [ ] **Unit 3: Prompt picker dialog with LISTBOX**

**Goal:** Implement the prompt picker dialog window that shows a LISTBOX of available prompts and handles selection.

**Requirements:** R2, R3, R4, R7, R8

**Dependencies:** Unit 1, Unit 2

**Files:**
- Modify: `main.go`

**Approach:**
- Implement `showPromptPicker()`:
  - Call `loadPrompts()` to get available prompts
  - If empty, show message box ("No prompts found...") and return
  - Store loaded prompts in a package-level variable for access during picker WndProc
  - Create a new top-level window (register a second window class `VNPromptPicker` if not already registered, or reuse a simpler approach)
  - Alternative simpler approach: Use the main window's WndProc with a state flag, or create the picker as a set of child controls in a new popup. The simplest Win32 approach for a modal-like picker:
    - Create a new top-level window with `WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU` and `WS_EX_TOPMOST`
    - Add a LISTBOX child control filled with prompt names via `LB_ADDSTRING` messages
    - Add "Insert" and "Cancel" buttons below the LISTBOX
    - Register a separate WndProc via a new window class for clean message handling
  - On "Insert" click or LISTBOX double-click (LBN_DBLCLK notification):
    - Get selected index via `LB_GETCURSEL`
    - Look up the prompt content from the stored slice
    - Get current edit text, append newline + prompt content (or just set if edit is empty)
    - Set edit text via `WM_SETTEXT`
    - Destroy/hide picker window
    - Set focus back to edit control
  - On "Cancel" or WM_CLOSE: destroy/hide picker window

**Patterns to follow:**
- Main window creation pattern (RegisterClassExW + CreateWindowExW) for the picker window class
- WM_COMMAND handling pattern for button clicks and LISTBOX notifications
- `getEditText()` and `WM_SETTEXT` pattern for reading/writing edit content

**Test scenarios:**
- Happy path: Click "Prompts" with 3 prompt files → picker window appears with 3 items in LISTBOX
- Happy path: Double-click a prompt → content inserted into edit box, picker closes
- Happy path: Select prompt + click "Insert" → same behavior as double-click
- Happy path: Edit box has existing text "Hello" → selecting prompt appends "\n[prompt content]"
- Happy path: Edit box is empty → selecting prompt sets content directly (no leading newline)
- Edge case: Click "Prompts" with no prompts directory → message box shown, no picker window
- Edge case: Click "Cancel" in picker → picker closes, edit box unchanged
- Edge case: Close picker via window X button → same as Cancel
- Integration: Full flow — hotkey → popup → click Prompts → select template → content in edit → Ctrl+Enter → text pasted to target app

**Verification:**
- Picker window appears centered, shows all loaded prompts
- Selection inserts correct prompt content into edit box
- Picker closes cleanly, focus returns to edit control
- Empty prompts directory shows helpful message

- [ ] **Unit 4: Create sample prompt files**

**Goal:** Provide example prompt files so the feature is immediately usable and demonstrates the file format.

**Requirements:** R5, R6

**Dependencies:** Unit 1

**Files:**
- Create: `prompts/code-review.md`
- Create: `prompts/bug-report.md`

**Approach:**
- Create 2 sample prompt files with proper frontmatter format
- Content should be useful Vietnamese-context templates (since this is a Vietnamese input helper)
- Keep content short but representative

**Patterns to follow:**
- The prompt file format defined in the High-Level Technical Design section

**Test scenarios:**
- Happy path: Both files parse correctly with `loadPrompts()`
- Happy path: Names and descriptions appear correctly in the picker

**Verification:**
- Sample files exist and are parseable
- Running the app shows them in the prompt picker

## System-Wide Impact

- **Interaction graph:** The "Prompts" button click triggers `showPromptPicker()` which reads files from disk and creates a new window. The picker modifies the edit control text via `WM_SETTEXT`. This is a leaf-node addition — no interaction with hotkey, clipboard, or paste logic.
- **Error propagation:** File I/O errors (missing directory, unreadable files) are logged and result in empty prompt list or skipped files. Never crash.
- **State lifecycle risks:** The loaded prompts slice is transient — loaded on each button click, not cached. This means users can add/edit prompt files while the app is running and see changes on next click. No stale state risk.
- **Unchanged invariants:** Hotkey registration, popup show/hide, OK/Cancel behavior, clipboard paste, and auto-start are completely unaffected. The edit control's existing WM_KEYDOWN handling (Ctrl+Enter, Esc) continues to work after prompt insertion.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| New window class registration could fail | Log error and show message box; feature degrades gracefully (button does nothing) |
| Large prompt files could be slow to load | Prompts are loaded on-click, not at startup. Even 100 files would be fast for simple file reads |
| LISTBOX doesn't show full prompt names if too long | LISTBOX has horizontal scroll; names should be kept reasonable by the user |
| Second window Z-order issues with the main popup | Picker uses `WS_EX_TOPMOST` and is brought to foreground with `SetForegroundWindow` |

## Sources & References

- Related code: `main.go` — WM_CREATE handler, WM_COMMAND handler, button creation pattern
- Win32 API: LISTBOX control, `LB_ADDSTRING`, `LB_GETCURSEL`, `LBN_DBLCLK` notification
- File format reference: Claude Code skill files (YAML frontmatter + markdown content)
