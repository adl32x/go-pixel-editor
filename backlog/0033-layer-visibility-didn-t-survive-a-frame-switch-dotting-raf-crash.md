---
id: 0033
title: Layer visibility didn't survive a frame switch (+ dotting rAF crash)
status: done
priority: high
tags: frontend, canvas, layers
---

User report (immediate follow-up to #0032): "If I toggle the layer and switch
frames, it doesn't seem to obey the toggles correctly. Eg. if I toggle the
top-most layer off and switch to a second frame, I can still see the first
frame's top layer."

**Root cause of the reported bug**: dotting's `setLayers()` — called on
every frame switch via `DottingCanvas.loadLayers()` — constructs entirely
new internal layer objects from the given `{id, data}` pairs alone, with no
visibility flag to preserve. Every layer silently resets to visible, and
`setLayers()` separately resets dotting's "current layer" to whichever id
is first in the array, regardless of what's actually selected. Neither of
`DottingCanvas`'s own effects (for `layers` metadata, for `activeLayerId`)
re-fire on a frame switch, since a frame switch changes pixel *data*, not
either of those settings — so both resets went uncorrected.

Fixed in `DottingCanvas.tsx`: `loadLayers()` now re-applies both right
after calling dotting's `setLayers()` — visibility via the same
`applyLayerVisibility` helper the reactive effect uses, and the active
layer via `setCurrentLayer` — reading both from refs (`layersPropRef`,
`activeLayerIdRef`) rather than the imperative handle's closed-over props,
since `useImperativeHandle`'s dependency array is `[]`.

**A second, deeper bug found while reproducing this**: switching frames at
all — with or without touching visibility — threw an uncaught `TypeError:
Cannot read properties of undefined (reading 'interactionLayer')` from
inside dotting's own `renderAll`, roughly one animation frame after every
single call to `setLayers()`. Traced to dotting's own source
(`node_modules/dotting/build/index.esm.js`, end of `Editor.setLayers`):

```js
this.lastRenderAllCall = requestAnimationFrame(this.renderAll);
```

A bare unbound method reference — `requestAnimationFrame` invokes its
callback with no receiver, so `this` was `undefined` inside `renderAll`
every time. This is unconditional and pre-existing (present since #0028
shipped frame switching via `setLayers`, unrelated to today's visibility
work) — confirmed by reproducing it on a sprite with no layer-visibility
interaction at all, just two plain frame switches.

It's silent in a production build (`go run . serve`, embedded `dist/`) —
`renderAll`'s early-thrown exception doesn't stop the canvas from having
already rendered correctly via the rest of `setLayers()`, so nothing
visibly breaks — but it's loud in Bun's dev server (`bun --hot dev.ts`),
which shows a full-screen "Bun v1.4.2 Runtime Error" overlay on every
frame switch.

**Fixed at the source** with a persistent patch via `bun patch dotting` +
`bun patch --commit`, changing the bare reference to `() =>
this.renderAll()` in both `build/index.esm.js` and `build/index.js` (the
package exposes both an ESM and a CJS entry point — patched identically).
Saved as `web/patches/dotting@2.1.18.patch`, wired into
`web/package.json`'s `patchedDependencies` and `web/bun.lock` — verified
the patch reapplies correctly after a full `rm -rf node_modules/dotting &&
bun install`, so it isn't lost for anyone else running this repo.

**Verified live in the browser**, both bugs:

- Drew on a "Top" layer on frame 1, drew a different mark on frame 2, hid
  "Top" while on frame 2, switched to frame 1 — confirmed via
  `dotting-data-canvas` `getImageData` that frame 1's "Top" pixels were
  genuinely excluded from the render (not just visually masked), and that
  re-showing it immediately restored them.
- Cycled between frames repeatedly (patched dotting) with the browser
  console open — zero `interactionLayer` exceptions, versus reproducing the
  crash unconditionally beforehand on a plain frame switch with no
  visibility interaction at all.

Full test suite (`go test ./... -race`) green, `gofmt -l .` clean, `tsc
--noEmit` and `bun run build` clean.
