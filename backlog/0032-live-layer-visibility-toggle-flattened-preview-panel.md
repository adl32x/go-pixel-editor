---
id: 0032
title: Live layer visibility toggle + flattened preview panel
status: done
priority: high
tags: frontend, canvas, layers
---

User report: toggling a layer's visibility checkbox in `LayerPanel` didn't
appear to do anything — the layer still showed on the canvas regardless.
Asked whether dotting even supports layering, and if not, requested an
alternative way to see the true composited result.

**Root cause**: `LayerPanel`'s checkbox only ever patched `LayerDef.Visible`
into sprite metadata server-side (`PATCH /api/sprites/{id}`) — correctly
used by the Go export/thumbnail compositor
(`internal/export/image.go:compositeFrame`, which already honors both
`Visible` and `Opacity`) — but `DottingCanvas.tsx` never told the live
dotting editor about it. Dotting *does* support real multi-layer editing
(verified in #0028: independent per-layer data, correctly composited on
the live canvas) and *does* support toggling a layer's visibility via ref
methods (`showLayer`/`hideLayer`) — they were simply never called.

`Opacity`, however, has **zero support anywhere in dotting** (confirmed:
no reference to "opacity" exists in its entire bundled source). It can
never be previewed on the live canvas without pre-flattening layers before
handing them to dotting, which would break independent per-layer editing —
not a worthwhile trade.

**Implemented**:

1. `DottingCanvas` gained a `layers: LayerDef[]` prop and an effect that
   calls dotting's `showLayer`/`hideLayer` per layer whenever visibility
   changes — guarded against `layerIdsRef` the same way the existing
   `setCurrentLayer` effect is (see #0028), since dotting throws
   synchronously ("Layer not found") for an id it doesn't know about yet,
   e.g. mid layer-add/delete.
2. New `PreviewPanel.tsx`: a small live-updating `<img>` pointed at the
   existing `GET .../export.png` endpoint — the same Go compositor real
   exports use, so it's the one place that already correctly honors both
   visibility and opacity. `api.exportFramePngUrl` gained an optional
   `version` param purely to cache-bust the URL (spriteId/frameId alone
   don't change on a pixel edit or a visibility/opacity/palette change).
   `App.tsx` bumps this version after every save, layer change, add/delete,
   and palette remap.
3. Swapped the plain visibility checkbox for a 👁️/🚫 toggle button (the
   user asked whether an eye icon made more sense) — same click semantics,
   dimmed+grayscaled when hidden for an at-a-glance state beyond the emoji
   alone.

**Verified live in the browser**: created a sprite, drew on the default
layer, added a second layer, drew a distinct mark on it — confirmed both
marks compose correctly on the canvas and in the preview. Hid the second
layer: confirmed via `getImageData` on `dotting-data-canvas` that its pixel
was actually removed from the rendered bitmap (not just visually masked),
confirmed the preview panel updated to show only the remaining layer's
mark, and confirmed showing it again restored both. (A stray gray square
briefly seen at the hidden pixel's location during this was dotting's own
hover-cursor indicator on the separate interaction canvas, confirmed by
moving the mouse elsewhere and watching it clear — unrelated to layer data.)

Full test suite (`go test ./... -race`) green, `gofmt -l .` clean, `tsc
--noEmit` and `bun run build` clean.
