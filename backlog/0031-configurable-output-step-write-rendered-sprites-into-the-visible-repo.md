---
id: 0031
title: Configurable output step: write rendered sprites into the visible repo
status: todo
priority: low
tags: backend, export
---

User's own framing (from the same request that prompted #0030): "when we
implement an output step we'll add definitions of how sprites are saved to
the repo." Captured here as a placeholder for later — not scoped or designed
yet, deliberately, since it wasn't asked for beyond the mention.

Working interpretation (confirm with the user before building): `.pixel/` is
the git-friendly *editable source* (per #0030), separate from wherever a game
actually wants to consume the art — e.g. an `assets/sprites/` folder read by
a game engine. This ticket would let a project define one or more named
output targets (path + export format, reusing #0003/#0009's `internal/export`
registry) and a way to (re-)run them — on demand via CLI/UI, or perhaps
automatically after a save — so exported PNGs/sheets/gifs land where the game
actually expects them, tracked in git like any other generated build
artifact the user chooses to commit.

Not started. Revisit once there's a concrete need/shape in mind.
