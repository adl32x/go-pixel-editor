---
id: 0025
title: Palette editor settings page
status: todo
priority: medium
tags: frontend, palette
x: -342.56388234931234
y: 94.69356532808439
---

User request: a separate settings page (not per-sprite) where the user can
modify the *project's* active palette — pick a different built-in preset
(#0024), or add/remove/reorder individual colors.

`App.tsx` currently has no routing/multi-page concept at all — one view, a
sprite selected or not. Simplest addition: a small view-switcher (e.g. a
"Sprites" / "Settings" tab in the sidebar toggling which panel renders) rather
than pulling in a router library, consistent with the "no new deps unless truly
needed" approach used elsewhere in this project.

The settings page needs: a preset picker (`GET /api/palettes`, select one →
triggers #0026's remap), a swatch grid of the current active palette (from
#0027's settings file) with add/remove/reorder and a native `<input
type=color>` for editing/adding entries, and a clear "this will remap existing
sprite colors to the nearest match" warning before committing a change, since
#0026 makes this a lossy operation on every sprite in the project.

Related but separate frontend change worth doing alongside this: `PaletteBar.tsx`
currently reads `sprite.palette` (per-sprite). Since palettes are now
project-wide and shared (#0027), every sprite's palette bar should show the
*same* list, and — since the whole point is a constrained palette — brush color
selection should probably be limited to swatches from it rather than the
current free `<input type=color>` (or if that stays for convenience, drawn
colors should visibly snap to the nearest palette entry per #0027, not silently
diverge from what's shown).

Depends on #0024 (something to list) and #0027 (somewhere to persist the
selection) landing first; #0026 (remap) should land alongside or before this so
the "change palette" action in the UI has a working backend behind it from day
one rather than a page that edits state nothing else reacts to.
