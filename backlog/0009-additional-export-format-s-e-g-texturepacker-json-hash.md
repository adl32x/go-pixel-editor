---
id: 0009
title: Additional export format(s) (e.g. TexturePacker JSON-hash)
status: todo
priority: medium
tags: backend, export
---

The user wants export format configurable "depending on the game I'm targeting" — #0003
ships `gif` and an Aseprite-style `sheet-json` as the defaults (already broadly
importable by Phaser 3/Godot/Unity). This ticket is for adding further target-specific
formats as the need arises, e.g. a plain TexturePacker JSON-hash variant, or an
engine-specific manifest (Godot `.tres`, Unity sprite metadata) — each is just a new
`export.Format` implementation + `Register()` call in `internal/export/`, no changes to
the HTTP/CLI layer (see the plan's "Configurable export formats" section). Not scoped to
a specific format yet — pick one when a concrete target engine is known.
