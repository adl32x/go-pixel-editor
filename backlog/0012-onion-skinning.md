---
id: 0012
title: Onion-skinning
status: todo
priority: low
tags: frontend, animation
---

Show a ghosted overlay of the previous/next frame(s) while editing the active frame in
`DottingCanvas`, to help draw smooth animations. Likely implemented as a semi-transparent
canvas layer composited from adjacent frames' `LayerProps` (via `Frame.ToLayerProps` on
neighboring frame IDs from the active clip), toggled on/off from `PlaybackControls` or a
new small control. Depends on #0006/#0007 existing first.
