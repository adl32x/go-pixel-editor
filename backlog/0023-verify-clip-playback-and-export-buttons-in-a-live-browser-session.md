---
id: 0023
title: Verify clip playback and export buttons in a live browser session
status: todo
priority: medium
tags: frontend, canvas
x: -67.47943329609672
y: 299.72520471841733
---

Carved out of #0014, which covered drawing/saving/reload persistence but not
this. Still needs a real browser pass: create a clip with 2+ frame entries,
press play in `PlaybackControls`, confirm it actually steps through frames at
the set fps/loop mode (not just that the `setInterval` logic looks right in
`PlaybackControls.tsx`), and click the export format `<select>` + download
link/button in `ClipEditor.tsx` to confirm a real `gif`/`sheet-json` download
happens from the UI (the export *endpoints* were already curl-tested during
backend scaffolding — this is specifically about the UI wiring).
