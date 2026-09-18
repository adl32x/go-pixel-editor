---
id: 0028
title: Multi-layer editing UI (add/delete/rename/reorder + active-layer drawing)
status: todo
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

