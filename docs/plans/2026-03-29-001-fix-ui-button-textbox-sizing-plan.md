---
title: "fix: Resize buttons and text input box to fit content"
type: fix
status: completed
date: 2026-03-29
---

# fix: Resize buttons and text input box to fit content

## Overview

Increase the default window size, widen the OK/Cancel buttons so their labels ("OK (Ctrl+Enter)", "Huy (Esc)") are fully visible, and enlarge the multiline text input area to give more typing space.

## Problem Frame

The current popup window is 500x200 pixels. The OK button label "OK (Ctrl+Enter)" is truncated at 100px width. The text input area (460x100) is small for composing Vietnamese text. Users need buttons that show their full labels and a larger editing area.

## Requirements Trace

- R1. OK button must be wide enough to display "OK (Ctrl+Enter)" without truncation
- R2. Cancel button must be wide enough to display "Huy (Esc)" without truncation
- R3. Text input box should be significantly taller to provide more editing space
- R4. Window should be resized proportionally to accommodate larger controls
- R5. Popup should remain centered on screen

## Scope Boundaries

- No changes to hotkey behavior, clipboard logic, or paste mechanism
- No changes to font or font size
- No dynamic/auto-sizing — just better static defaults

## Context & Research

### Relevant Code and Patterns

Current layout in `main.go` `wndProc` WM_CREATE handler (lines 366-411):
- Window default: `Width=500, Height=200` (config defaults, line 263-264)
- Edit control: position `(10, 10)`, size `(w-40, h-100)` = `460x100`
- OK button: position `(w-240, h-80)`, size `100x35` — label "OK (Ctrl+Enter)" truncated
- Cancel button: position `(w-130, h-80)`, size `100x35` — label "Huy (Esc)" fits but tight

Layout is computed from `cfg.Window.Width` and `cfg.Window.Height` in WM_CREATE, so changing the defaults and button sizes in one place propagates correctly through `showPopup()` centering logic.

## Key Technical Decisions

- **Increase default window to 600x350**: Provides enough room for wider buttons and a much taller edit area (~200px vs current ~100px). Still reasonable for most screens.
- **OK button width 160px, Cancel button width 120px**: "OK (Ctrl+Enter)" needs ~150px at the current 20px Segoe UI font. 160px gives breathing room. "Huy (Esc)" is shorter, 120px is comfortable.
- **Button height stays 35px**: Current height is fine for a single line of text.

## Implementation Units

- [x] **Unit 1: Update default window size and control dimensions**

**Goal:** Make buttons fully show their text labels and enlarge the text input area.

**Requirements:** R1, R2, R3, R4, R5

**Dependencies:** None

**Files:**
- Modify: `main.go`

**Approach:**
- Change default `cfg.Window.Width` from 500 to 600 and `cfg.Window.Height` from 200 to 350
- In WM_CREATE: adjust OK button width from 100 to 160, Cancel button width from 100 to 120
- Recalculate button X positions so they remain right-aligned with proper spacing:
  - OK button X: `w - 300` (was `w-240`)
  - Cancel button X: `w - 130` (stays same — 120px button + 10px right margin)
- Edit control size formula `(w-40, h-100)` already scales with the new window dimensions — no change needed
- Button Y positions use `h-80` which still works — buttons sit near bottom

**Patterns to follow:**
- Same direct pixel arithmetic used in current WM_CREATE handler

**Test scenarios:**
- Happy path: Build and run — OK button shows full "OK (Ctrl+Enter)" text, Cancel button shows full "Huy (Esc)" text
- Happy path: Text input area is visibly larger with more lines visible
- Happy path: Popup appears centered on screen
- Edge case: Custom config.json with smaller width/height still produces a functional layout (buttons may overlap but app doesn't crash)

**Verification:**
- Visual inspection: both button labels fully visible, no truncation
- Text input area is approximately double the previous height
- Window centers correctly on screen

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Button position arithmetic off by a few pixels | Simple to verify visually and adjust |
| Larger window may feel big on small screens | 600x350 is still modest; users can override via config.json |
