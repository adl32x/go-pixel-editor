---
id: 0022
title: macOS-style light UI redesign
status: done
priority: medium
tags: frontend, ui
x: 665.0704332043904
y: 798.1982836124889
---

User-requested: light theme by default, macOS look & feel. Rewrote
`web/src/index.css`: light window background (`#ececef`) with white rounded
"card" panels (workspace, timeline, playback, clip editor) matching the macOS
System Settings grouped-panel look, San Francisco system font stack, macOS
system-blue (`#007aff`) for selection/accent/focus states, subtle borders and
shadows, native-feeling buttons/inputs with focus rings, macOS-style
destructive-red delete badges — replacing the original dark theme.

Also covers the related "sprites default to a transparent background" request:
`defaultPixelColor`/`backgroundColor` fills disabled on the canvas (see #0020)
plus a CSS checkerboard behind the canvas and frame thumbnails, and
`isGridFixed`/`initAutoScale` added to the `<Dotting>` canvas.
