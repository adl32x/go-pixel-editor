---
id: 0028
title: Multi-layer editing UI (add/delete/rename/reorder + active-layer drawing)
status: done
priority: high
tags: backend, frontend, canvas
x: -78.33849600822748
y: 587.4294054684892
---

User request: real multi-layer editing — e.g. a player sprite split into
head/torso/clothes/weapon layers, each independently toggleable/reorderable.

**The data model and file format already fully support this** — no format
change needed. `Sprite.Layers []LayerDef` (id/name/visible/opacity) and each
frame's `.px` file already stores one `layer: <id>` block per sprite layer
(`internal/sprite/frame.go`), and `LayerPanel.tsx` already renders a
visibility checkbox + opacity slider per layer. What's actually missing is
everything needed to *use* more than the one default layer every sprite is
born with:

1. **Backend has no safe way to add/remove a layer.** `UpdateSprite` only
   accepts a wholesale `SpritePatch.Layers` replace. Naively appending a new
   `LayerDef` that way would immediately break every existing frame: both
   `parseFrameFile` (`frame.go:171`) and `Frame.ToLayerProps`
   (`convert.go:29`) hard-error with `"missing layer %s"` the moment a
   sprite declares a layer that isn't present as a block in a given frame
   file. Need dedicated `AddLayer`/`DeleteLayer`/`RenameLayer` commands in
   `internal/sprite/commands.go` that keep every frame in sync — mirroring
   the discipline `AddFrame`/`DeleteFrame` already apply to frames-vs-clips:
   adding a layer means retrofitting a blank (`.`-filled) block onto every
   existing frame file; deleting one means stripping that layer's block from
   every frame file. Reordering doesn't touch frame files at all (only stack
   order in `sprite.md`), so that can keep going through the existing
   `SpritePatch.Layers` whole-replace path.
2. **New HTTP endpoints**: `POST .../sprites/{id}/layers`, `DELETE
   .../layers/{layerId}`, `PATCH .../layers/{layerId}` (rename/opacity/
   visibility) alongside the existing whole-sprite `PATCH` for reordering.
3. **Frontend has no layer CRUD UI at all** — `LayerPanel.tsx` only lists
   existing layers. Needs add/delete/rename controls and drag-or-buttons
   reordering (dotting's ref already exposes `changeLayerPosition`/
   `reorderLayersByIds` for this, currently unused).
4. **No "active layer" concept exists yet.** Nothing in `DottingCanvas.tsx`
   or `App.tsx` ever calls dotting's `setCurrentLayer` ref method, so which
   layer a stroke actually lands on with more than one layer present is
   currently unverified/likely just whatever dotting defaults to. Needs a
   layer-select UI (radio/click-to-activate in `LayerPanel.tsx`) wired to
   `setCurrentLayer`.

Given how many real bugs the *existing* single-layer dotting integration
turned up under live testing (#0020), this should get the same live-browser
verification treatment before being called done, not just type-checked —
specifically: draw on layer A, switch to layer B, draw there too, confirm
both layers' strokes land in the right `layer:` block in the saved `.px`
file and neither overwrites the other.

---

**Implemented.** Backend: `sprite.AddLayer`/`DeleteLayer`
(`internal/sprite/commands.go`) follow the same frames-first,
`sprite.md`-last write ordering as `RemapPalette` — a crash mid-operation
leaves at worst an orphaned blank/stale block, never a sprite that
references a layer no frame has. `AddLayer` retrofits a blank (`.`-filled)
block onto every existing frame and inserts the new layer at the top of the
stack; `DeleteLayer` strips that layer's block from every frame and refuses
to remove a sprite's last remaining layer. New endpoints: `POST
/api/sprites/{id}/layers`, `DELETE /api/sprites/{id}/layers/{layerId}`.
Rename/opacity/visibility/reorder deliberately did **not** get their own
endpoint (unlike the ticket's suggested `PATCH .../layers/{layerId}`) —
none of them touch frame files, so they keep going through the existing
whole-sprite `PATCH /api/sprites/{id}` (`SpritePatch.Layers` replace),
avoiding a second code path for the same data.

Frontend: `LayerPanel.tsx` rewritten with full CRUD — click-to-select
active layer, inline rename, opacity slider, up/down reorder, delete (with
a `layers.length <= 1` disabled guard mirroring the backend's own refusal),
and an add-layer form. `DottingCanvas.tsx` gained an `activeLayerId` prop
driving dotting's `setCurrentLayer` ref method, and a real (not
hypothetical) stale-closure bug fix: its internal `currentLayersArray()`
was iterating the mount-time `initLayers` closure rather than the current
layer set, which would have silently dropped any newly-added/removed
layer's data from every save once layer CRUD existed to trigger the
mismatch.

**Two real bugs surfaced by live-browser verification** (not caught by
`tsc`/unit tests, per the bar #0020 set):

1. Adding a layer threw an uncaught `"Layer not found"` from dotting's
   `setCurrentLayer` — `handleAddLayer` called `setActiveLayerId(newId)`
   before `reloadActiveFrame()` had pushed the new layer into dotting's own
   internal layer set, so the id genuinely didn't exist yet from dotting's
   point of view. Reordering to update layers before switching the active
   id fixes the immediate case; `DottingCanvas` also now guards the
   `setCurrentLayer` call against `layerIdsRef` so any future ordering slip
   is a silent no-op instead of an uncaught exception that aborts the rest
   of React's effect flush.
2. Deleting a layer crashed dotting itself (`TypeError: Cannot read
   properties of undefined (reading 'interactionLayer')` in `renderAll`) —
   confirming dotting's known "can't be repurposed via `setLayers()`"
   limitation (already documented in `DottingCanvas.tsx` for grid-shape
   changes) also applies to a changed *layer count*, not just changed
   width/height. Fixed the same way the app already handles a sprite
   switch: `<DottingCanvas>`'s `key` in `App.tsx` now includes the sprite's
   layer-id list, so add/delete forces a full remount with correct
   `initLayers` from the start rather than mutating a live editor instance;
   `handleAddLayer`/`handleDeleteLayer` fetch fresh frame data directly
   instead of going through the imperative `reloadActiveFrame()` path for
   this reason.

**Verified live in the browser**: created a sprite, drew on the default
`base` layer, added a second layer (`setCurrentLayer` correctly switched
dotting's active layer, confirmed via the emitted `dataChange` event's
`layerId`), drew a second stroke — confirmed via the saved `.px` file that
the new stroke landed in the new layer's block and `base`'s block was
byte-for-byte unchanged. Switched back to `base`, drew a third stroke,
confirmed it landed in `base`'s block without touching the other layer.
Deleted the second layer through the UI, confirmed the canvas re-rendered
showing only `base`'s content and the frame file's second `layer:` block
was cleanly stripped, with no crash pre- or post-fix comparison for both
bugs above.

Full test suite (`go test ./... -race`) green, `gofmt -l .` clean, `tsc
--noEmit` and `bun run build` clean.
