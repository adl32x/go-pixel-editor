---
id: 0029
title: Tool selector UI (eraser, select+move, and dotting's other built-in tools)
status: done
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

---

**Implemented.** New `web/src/canvas/Toolbar.tsx`: a button per tool, same
selected-state visual language as `PaletteBar`/`LayerPanel`. `App.tsx` gained
`brushTool` state (mirroring the existing `brushColor` pattern) threaded into
`<DottingCanvas>`, which already forwarded the prop — no changes needed there.
Included 9 of the enum's 10 values; left out `BrushTool.NONE` deliberately
(not by oversight) — it's dotting's "nothing selected, just pan" mode, and
panning is already reachable via middle-click/scroll regardless of the active
tool, so it isn't a "tool" a user would deliberately pick from a toolbar.

**Verified live in the browser**, per the ticket's own bar:

- **Eraser**: drew two adjacent pixels with the pen, erased one with the
  eraser, confirmed via the saved `.px` file that the erased cell became `.`
  (not just visually cleared) while its neighbor was untouched.
- **Select + move**: drew a single pixel, drew a selection box around it,
  then dragged from inside the selection to a distant empty area — confirmed
  the `.px` file showed exactly one pixel, at the new position, with the old
  position back to `.` (no duplication, nothing left behind).
- **Fill** (smoke test beyond the ticket's explicit list, since flood-fill is
  a meaningfully different commit path than a stroke): flood-filled an empty
  16x16 canvas from a single click, confirmed the saved `.px` file was fully
  colored.
- Clicked through the remaining tools (Line, Rect, Rect filled, Ellipse,
  Ellipse filled) to confirm each selects cleanly with no crash.

**Incidental fix, found via this testing, unrelated to the tool selector
itself**: switching the selected sprite left `SpriteMeta`'s name/tags fields
showing the *previous* sprite's values — its `useState` only reads `sprite`
props on first mount, and nothing remounted it on a sprite switch, so a blur
right after switching could silently rename the wrong sprite. Fixed with
`key={sprite.id}` on `<SpriteMeta>` in `App.tsx`, the same remount-on-identity-
change pattern already used for `<DottingCanvas>`. Verified: switching sprites
now immediately shows the newly-selected one's name.

Full test suite (`go test ./... -race`) green, `gofmt -l .` clean, `tsc
--noEmit` and `bun run build` clean.
