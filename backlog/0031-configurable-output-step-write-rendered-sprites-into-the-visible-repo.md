---
id: 0031
title: "pixel build: render every sprite to a sheet PNG + simple JSON in a given folder"
status: done
priority: medium
tags: backend, export, cli
---

`.pixel/` is the editable, git-friendly source; a game needs plain rendered
files in its own asset folder. `pixel build` is that output step.

Decisions (from the user):

1. **Simple custom JSON**, not the Aseprite schema (that stays available as
   the `sheet-json` export format).
2. **One sheet per sprite**, no packed atlas.
3. **The output folder must be given** — no default location. Set it once
   as `build_out:` in `.pixel/settings.md`, or pass `--out=DIR`; the command
   errors if neither is set.
4. **Custom engine**, so no engine-specific formats for now.

Shape:

- `pixel build [ids...] [--out=DIR]` writes `<out>/<slug>.png` and
  `<out>/<slug>.json` per sprite — plain files, not a zip. Two sprites
  sharing a slug is an error (rename one).
- Sheet layout mirrors the editor's frame grid: one PNG row per non-empty
  animation row, frames left to right, sheet width = the longest row.
- JSON:

  ```json
  {
    "name": "eye",
    "image": "eye.png",
    "frameWidth": 16,
    "frameHeight": 16,
    "animations": {
      "blink": [
        { "x": 0, "y": 0, "w": 16, "h": 16, "durationMs": 100 }
      ]
    }
  }
  ```

- Deterministic output: rebuilding unchanged art is byte-identical, and a
  file whose content didn't change isn't rewritten.
- A full build (no ids) removes outputs of sprites that no longer exist —
  only files a previous build wrote, tracked in `<out>/.pixel-build`.
- Also registered as a `sheet-grid` export format, so the per-sprite
  `pixel export` and the UI's export picker can produce the same pair.

Later, if wanted: a Build button in the UI, or rebuilding on save.
