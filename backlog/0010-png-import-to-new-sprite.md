---
id: 0010
title: PNG import to new sprite
status: todo
priority: medium
tags: backend, import
x: -73.98605153400912
y: 205.62393301792352
---

Import an existing PNG (e.g. reference art, or a sprite sheet exported from another
tool) and convert it into a new sprite: decode with stdlib `image/png`, quantize/map
pixels onto a new or existing palette (allocating chars via `Sprite.ColorToChar`,
subject to the 62-color cap — see #0002's palette-overflow behavior), and write out as
a single-frame sprite via the normal `Frame.Save` path. Exposed via CLI
(`pixel import <file.png> --name=`) and optionally a web upload control. Not scoped for
the initial pass.
