---
id: 0024
title: Pre-set color palettes (NES, SNES, DOS 256, etc.)
status: todo
priority: medium
tags: backend, palette, link:0024-0027
x: -348.96975110159474
y: 302.3321584091729
---

User request: ship built-in preset palettes (NES, SNES, DOS 256/VGA, etc.) as
options for the project's single shared palette (#0027 — every sprite in a
project now draws from the same one fixed palette, not a color-by-color
append-only list per sprite as today).

Add a new `internal/palette` package: a registry of built-in presets, each a
named, ordered list of hex colors (`type Preset struct { ID, Name string; Colors
[]string }`, `Register`/`Get`/`Names` mirroring `internal/export`'s registry
shape). Concrete presets to ship:
- **NES**: the classic 64-slot NES palette (many duplicate/unused slots in the
  real hardware table — use a curated ~54-color deduplicated list, several
  public-domain references exist).
- **DOS/VGA 256**: the standard default 256-color VGA palette — a well-known
  fixed table, straightforward to hardcode.
- **SNES**: the SNES has a 15-bit (32,768 color) space, not a fixed palette, so
  "the SNES palette" isn't a single canonical list the way NES/VGA are. Ship a
  curated, commonly-used SNES-style palette (or substitute a well-known
  general-purpose pixel-art palette like PICO-8 or DB32 if a citable SNES-specific
  one isn't available) rather than trying to enumerate the full color space —
  flag this substitution to the user when implementing.

Expose `GET /api/palettes` listing `{id, name, colorCount}` for each registered
preset (not the full color lists, to keep the list endpoint light — a separate
`GET /api/palettes/{id}` can return the full color array). Depended on by #0027
(the foundational settings/palette rework — needs a preset to default to) and
transitively #0025 (settings page needs something to list) and #0026 (remap
needs a target palette to switch *to*).

**Note**: the DOS/VGA 256 preset has more colors than a sprite's per-sprite
palette can currently encode (62-char cap — see #0018). #0018 needs to land
before or alongside this for the DOS-256 preset to actually be usable, not just
listed.
