---
id: 0003
title: internal/export registry + gif/sheet-json formats + tests
status: done
priority: high
tags: backend, export
x: 663.8156529898988
y: -13.014358332043916
---

Pluggable export so output format is configurable per target game/engine, not fixed in
code. `Format` interface (`Name() string`, `Export(sprite.Sprite, []sprite.Frame,
*sprite.Clip) (Bundle, error)`) + `Register`/`Get`/`Names` registry, stdlib only
(`image/gif`, `image/png`, `image/draw`, `archive/zip`, `encoding/json`).

Built-ins: `gif` (animated GIF via `gif.EncodeAll`, delay/loop from the clip) and
`sheet-json` (default — grid PNG of composited frames + Aseprite-style `data.json`,
zipped together) — see the plan's "Configurable export formats" section for the exact
JSON schema. Adding a new target format later is just another `Format` impl +
`Register()` call, no HTTP/CLI changes (tracked separately as #0009 for e.g.
TexturePacker JSON-hash).

Tests: GIF round-trips through `image/gif.DecodeAll` with correct frame count/delays;
sheet-json zip contains both files and frame rects don't overlap and sum to sheet size.
