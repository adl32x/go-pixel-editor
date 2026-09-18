---
id: 0007
title: Timeline/playback UI (frame list, scrubber, fps, loop mode)
status: done
priority: high
tags: frontend, animation
x: 665.9672075115102
y: 538.245356438004
---

Fully custom (dotting has no animation concept at all). `Timeline.tsx`: horizontal frame
strip for the active sprite with add/duplicate/delete and click-to-select-active-frame
(drives `DottingCanvas`'s frame swap). `PlaybackControls.tsx`: play/pause, fps, loop-mode
(`none`/`forward`/`pingpong`) select, clip selector — a `setInterval` loop steps through
the active clip's `entries`, respecting each entry's `durationMs` override or
`1000/fps`, updating the shared active-frame state each tick.

State lives in `state/spriteStore.ts` (plain module state + `useSyncExternalStore`, no
external state library) so canvas/timeline/playback controls all read the same
active-sprite/frame/clip/playback state without prop-drilling.
