---
id: 0006
title: Dotting canvas integration + layer panel
status: done
priority: high
tags: frontend, canvas
x: 660.9469136277505
y: 633.712115764475
---

`DottingCanvas.tsx` wraps `<Dotting ref width height initLayers brushTool brushColor />`
(npm `dotting@2.1.18`, MIT — no built-in frame/animation concept, only static layers per
canvas). Converts the active frame's API response (`GET .../frames/{id}` ->
`LayerProps[]`) into dotting's `initLayers`, and on frame switch calls dotting's own
ref/hook data-swap (`useDotting`/`useData` — verify exact hook name against the
installed package's own `.d.ts`, don't guess) rather than remounting the component.
Local edits debounce (~400ms) and `PUT` back via `api.putFrame`.

`LayerPanel.tsx`: list sprite layers (name/visible/opacity), PATCHing sprite metadata on
change. `PaletteBar.tsx`: swatches from `Sprite.palette` + a color input feeding
`brushColor`.
