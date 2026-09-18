---
id: 0020
title: Fix dotting integration bugs found during live browser verification
status: done
priority: high
tags: frontend, canvas
x: 660.8631769426337
y: 350.46930851740046
---

Live browser testing (see #0014) surfaced several real bugs no amount of
type-checking or curl-testing the API would have caught, all in `DottingCanvas.tsx`:

1. **Broken ref API**: `DottingRef.getLayersAsArray()`/`getLayers()` unconditionally
   return `undefined` at runtime in the installed `dotting@2.1.18`, contradicting
   their own `.d.ts` — confirmed by instrumenting and watching it live. Fixed by
   reconstructing current pixel data from the `dataChange` event's own payload
   (`denseGridFromDottingData`) instead of ever calling back into the ref for it.
2. **Destructive save-on-open**: that event also fires once on mount as dotting
   syncs from `initLayers` (`isLocalChange: false`) with an *empty* payload.
   Saving unconditionally on every event meant simply reopening a sprite with
   already-drawn pixels silently overwrote them with a blank frame. Fixed by
   gating the debounced save on `params.isLocalChange`.
3. **StrictMode-only crash**: `TypeError: Cannot read properties of null
   (reading 'removeEventListener')` — dotting can tear down its internal editor
   before our effect cleanup runs (React 18 StrictMode's dev-only double
   mount/unmount). Fixed with a try/catch around `removeDataChangeListener`.
4. **Debounce silently cancelled**: `handleCanvasChange` wasn't memoized in
   `App.tsx`, so `DottingCanvas`'s listener/debounce-timer effect (keyed on that
   callback's identity) tore down and rebuilt on almost any unrelated re-render,
   cancelling in-flight saves before the 400ms timeout fired. Fixed with
   `useCallback` plus keeping the latest callback in a ref so the subscription
   effect itself only depends on mount/unmount.
5. **Transparency**: dotting has two separate opaque fills — `backgroundColor`
   (CSS, outside the grid) and `defaultPixelColor` (canvas fillStyle, painted
   under every layer inside the grid on every render, default `#fff`). Both
   needed disabling (`backgroundColor="transparent"`, `defaultPixelColor=""` —
   empty string is falsy so dotting's own check skips the fill outright, more
   certain than trusting a canvas's handling of the string `"transparent"`) plus
   a CSS checkerboard behind the canvas and frame thumbnails.
6. **Grid could grow past the sprite's fixed size**: `isGridFixed` added — the
   backend's width/height are fixed at sprite creation, so an extendable grid
   would have broken the save round-trip.
7. A single `left_click` doesn't reliably register as a draw with dotting's
   interaction layer — needs an actual drag gesture. Noted for anyone testing
   manually or writing browser-driven tests later.

Verified end-to-end: draw → canvas re-renders → edit persists to
`sprites/*/frames/*.px` → reload the page → reopen the sprite → drawing is
still there, unmodified.
