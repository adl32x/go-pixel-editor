---
id: 0010
title: PNG import to new sprite
status: done
priority: medium
tags: backend, import
x: -73.98605153400912
y: 205.62393301792352
---

`pixel import <sheet.json|image.png>` (internal/importer). Imports an
Aseprite json-array or pixel-sheet/1 sheet (keeping animation names and
durations) or a bare PNG, optionally split into a grid, as a new sprite.
Colors snap to the project's shared palette (there is no per-sprite palette
to allocate into anymore), with a warning count for inexact pixels. The
build subfolder is inferred from the source's location under build_out.
CLI only; no web upload control yet. See SKILL.md "Importing existing art".
