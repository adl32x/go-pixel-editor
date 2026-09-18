---
id: 0025
title: Palette editor settings page
status: done
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

The settings page needs: a preset picker (endpoint already exists — `GET
/api/palette-presets` lists `{id, name, colorCount}`, `GET
/api/palette-presets/{id}` returns one preset's full colors, per #0024/#0027
— select one → triggers #0026's remap), a swatch grid of the current active
palette (`GET /api/palette`, also already implemented) with add/remove/
reorder and a native `<input type=color>` for editing/adding entries, and a
clear "this will remap existing sprite colors to the nearest match" warning
before committing a change, since #0026 makes this a lossy operation on every
sprite in the project. There's no `PATCH`/write endpoint for the active
palette yet — only the read side (`GET /api/palette`) landed as part of
#0027; this ticket needs to add that write path (and it should very likely
call straight into #0026's remap rather than just overwriting the palette and
leaving every sprite's existing pixels pointing at stale/wrong colors).

**Already done as part of #0027**, so out of scope here: `PaletteBar.tsx` now
reads the shared project palette (`GET /api/palette`, fetched once in
`App.tsx`) instead of a per-sprite one — every sprite's palette bar already
shows the same list. **Still open**: the free `<input type=color>` picker is
still there; since the backend snaps unconditionally on save, this doesn't
corrupt anything, but there's no preview of the snap before it happens — worth
deciding here whether to restrict it to swatches only or add a live preview.

Depends on #0024 (something to list, done) and #0027 (somewhere to persist
the selection, done) — both prerequisites are in place now. #0026 (remap)
should land alongside or before this so the "change palette" action in the UI
has a working backend behind it from day one rather than a page that edits
state nothing else reacts to.

---

**Implemented.** `App.tsx` gained a `view` state (`"sprites" | "settings"`)
with a small tab switcher in the sidebar (`.app-nav`) — no router library, as
planned. `web/src/settings/PaletteSettings.tsx` is the new page:

- Preset buttons (from `GET /api/palette-presets`), highlighting the active
  one; clicking applies immediately via `PUT /api/palette` (see #0026/#0027).
- A swatch grid of the current palette with per-swatch move-left/move-right/
  remove controls and a native `<input type=color>` per swatch for editing,
  plus "+ Color" to append and "Apply custom palette" to commit the whole
  edited list (only enabled once it actually differs from what's loaded).
- Both actions go through `window.confirm()` first — "every sprite's existing
  pixels will be remapped" — matching the `confirm()` pattern already used
  elsewhere in this codebase (`Timeline.tsx`'s delete-frame confirmation),
  not a new pattern introduced for this.

Also handled: switching the palette while a sprite/frame is open needs the
already-mounted `<DottingCanvas>` to reflect the remap immediately, not just
on next reload — `App.tsx`'s `handleSettingsChanged` re-fetches the active
frame and pushes it through the same imperative `canvasRef.current
.loadLayers()` path frame-switching already uses (dotting only reads
`initLayers` at mount, so just updating state wouldn't reach an
already-mounted canvas).

**Free `<input type=color>` picker**: kept, not restricted to swatches — this
was the open question from before, resolved by leaving it as-is since the
backend snaps unconditionally regardless of what's submitted, so it can't
produce an inconsistent state, just possibly a surprising snap.

Verified live in the browser end-to-end (not just type-checked): switched
NES → PICO-8 (palette bar and Settings grid both updated), drew a PICO-8-blue
pixel, switched back to NES, and confirmed via the API/on-disk file that the
pixel's color genuinely changed to NES's nearest match (`#3cbcfc`, not left
as stale `#29ADFF`) — and that the already-open canvas re-rendered with the
new color without a page reload.

One process note for whoever tests this next: `PaletteSettings`'s buttons
call `window.confirm()`, which blocks all further page JS until a human
dismisses it — automated `computer` tool clicks on them will appear to do
nothing (the dialog isn't visible to screenshots) without a human present to
click through it. Confirmed via `javascript_tool` that the page wasn't
actually frozen, then temporarily stubbed `window.confirm` in the live page's
own runtime (`window.confirm = () => true`) to click through it during
testing — safe since it only affects that loaded session's memory and resets
on reload, never touches shipped code.
