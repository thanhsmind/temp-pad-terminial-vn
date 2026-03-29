---
title: Fix truncated button labels and small input area in VN Input Helper popup
date: 2026-03-29
category: ui-bugs
module: vn-input-helper
problem_type: ui_bug
component: tooling
symptoms:
  - OK button label "OK (Ctrl+Enter)" truncated at 100px width
  - Cancel button label "Huy (Esc)" barely fits at 100px width
  - Text input area too small (460x100px) for composing Vietnamese text
root_cause: config_error
resolution_type: config_change
severity: low
tags:
  - win32
  - gui-layout
  - button-sizing
  - vietnamese-input
---

# Fix truncated button labels and small input area in VN Input Helper popup

## Problem

The popup window's buttons were too narrow to display their full label text, and the text input area was cramped for composing Vietnamese text. Default window size of 500x200px was insufficient for the UI content.

## Symptoms

- OK button label "OK (Ctrl+Enter)" was visually truncated because the button was only 100px wide
- Cancel button label "Huy (Esc)" barely fit at 100px wide
- Text input area was only 460x100px, leaving very little vertical room for multi-line Vietnamese composition
- Overall window (500x200) felt cramped for its purpose

## What Didn't Work

N/A -- this was a straightforward sizing fix identified by inspecting the hardcoded layout values in the WM_CREATE handler and the default config in `loadConfig`.

## Solution

Changed 6 numeric constants in `main.go`:

**Default window dimensions in `loadConfig`:**
- `cfg.Window.Width`: 500 -> 600
- `cfg.Window.Height`: 200 -> 350

**Button sizes in the WM_CREATE handler:**
- OK button width: 100 -> 160, X position formula: `w-240` -> `w-300`
- Cancel button width: 100 -> 120 (X position `w-130` unchanged)

**Edit control:** No changes needed -- it uses the formula `(w-40, h-100)` which auto-scales with window dimensions.

**Resulting layout at w=600, h=350:**

| Control | Position | Size | Right edge |
|---------|----------|------|-----------|
| Edit | (10, 10) | 560x250 | 570 |
| OK button | (300, 270) | 160x35 | 460 |
| Cancel button | (470, 270) | 120x35 | 590 |

10px gap between buttons, 10px right margin from window edge.

## Why This Works

Win32 buttons do not auto-size to fit their text -- they render at exactly the pixel dimensions specified in the `CreateWindowExW` call. The root cause was hardcoded button widths (100px) that were too narrow for the actual label text. Widening the OK button to 160px fits "OK (Ctrl+Enter)" comfortably. The larger window (600x350) gives the edit control substantially more composing area (560x250 vs 460x100). The relative positioning formulas (based on `w` and `h`) ensure the layout remains consistent if the user customizes window size via `config.json`.

## Prevention

- **Size buttons for their content:** When setting button dimensions in Win32, account for the label text length -- Win32 buttons don't auto-size like web/CSS buttons do.
- **Test with actual labels:** After changing button text, verify it fits within the allocated dimensions.
- **Consider named constants:** Define margin/gap/minimum-width constants so spatial relationships are explicit and auditable.
- **DPI awareness:** On high-DPI displays, hardcoded pixel values may still be too small. A future improvement could scale layout values by the system DPI factor.

## Related Issues

- Plan: `docs/plans/2026-03-29-001-fix-ui-button-textbox-sizing-plan.md`
