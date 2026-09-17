---
id: 0014
title: Verify dotting canvas integration live in a browser
status: todo
priority: high
tags: frontend, canvas
---

Static verification is done: `DottingCanvas.tsx`'s use of `DottingRef` (`setLayers`,
`getLayersAsArray`, `add/removeDataChangeListener`) was checked against the real
installed `node_modules/dotting/build/components/Dotting.d.ts` and matches exactly;
`bunx tsc --noEmit` and `bun run build` both pass; the Go API was smoke-tested
end-to-end with curl (create sprite, add frame, paint pixels via PUT, create/patch
a clip, export both `gif` and `sheet-json`) and `pixel serve` correctly serves the
real built `index.html` + JS/CSS chunks.

Not done: an actual live browser session (draw a pixel with the mouse, watch the
canvas re-render, play a clip and watch it animate) — no Chrome extension was
connected in the environment this was built in. Do this manually: `make build &&
./bin/pixel serve`, then create a sprite, draw a few pixels, add 2-3 frames, make a
clip, hit play, and try both export formats from the UI.
