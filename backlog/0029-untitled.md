---
id: 0029
title: Tool selector UI (eraser, select+move, and dotting's other built-in tools)
status: todo
priority: high
tags: frontend, canvas, tools
x: -90.86471251405526
y: 736.6978092387594
---

User request: eraser and select(+move) tools, beyond the pen the app currently
offers.

**Dotting already implements a full toolbox — we've just never exposed a
switcher for it.** `BrushTool` (`node_modules/dotting/build/components/Canvas/
types.d.ts`) has ten values: `DOT`, `ERASER`, `PAINT_BUCKET`, `SELECT`, `NONE`,
`LINE`, `RECTANGLE`, `RECTANGLE_FILLED`, `ELLIPSE`, `ELLIPSE_FILLED`. Per
dotting's own docs, `SELECT` already covers "select *and move/resize* the
selected region" as one built-in tool — exactly what the user asked for,
no extra work needed on that front once it's selectable.

The plumbing is already there too: `DottingCanvas.tsx` already accepts and
forwards a `brushTool` prop straight into `<Dotting brushTool={brushTool} />`.
The entire gap is in `App.tsx:160`, which hardcodes
`brushTool={BrushTool.DOT}` with no state and no UI to change it.

Scope: add `brushColor`-style state (`useState<BrushTool>`) in `App.tsx` and a
small toolbar component (new, e.g. `web/src/canvas/Toolbar.tsx`) with a button
per tool, highlighting the active one — same visual language as the existing
palette swatches/selected-list-item pattern in `index.css`. MVP is the two
tools explicitly requested (pen/`DOT` already default, `ERASER`, `SELECT`),
but since a real switcher needs to exist either way, it costs little extra to
include the rest of the enum (`PAINT_BUCKET`, `LINE`, `RECTANGLE`(`_FILLED`),
`ELLIPSE`(`_FILLED`)) rather than hardcoding just two more special cases —
worth doing since they come essentially free.

Given how much live-browser testing (#0020) turned up for even the single
`DOT` tool's dotting integration, this should get the same treatment before
being called done — specifically confirm `ERASER` actually clears pixels
(writes `color: ""`, not just visually blanks them) and that a `SELECT`+move
edit round-trips through `PUT .../frames/{id}` correctly (moved pixels end up
in their new position in the saved `.px` file, not duplicated or left behind
at the old position).
