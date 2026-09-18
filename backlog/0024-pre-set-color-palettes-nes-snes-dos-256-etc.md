---
id: 0024
title: Pre-set color palettes (NES, SNES, DOS 256, etc.)
status: done
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

**Implemented as**: `internal/palette` (`palette.go` registry, `nes.go`,
`pico8.go`), tested in `palette_test.go`. Shipped two presets:
- **NES**: 55 deduplicated colors (Lospec's "Nintendo Entertainment System"
  listing), confirmed accurate against the source rather than from memory.
- **PICO-8** (16 colors, official palette) in place of "SNES" — as flagged
  above, there's no single canonical SNES palette to cite, so this stands in
  as the curated general-purpose retro palette rather than an invented
  SNES-specific list. Named `pico8`, not `snes`, so it isn't mislabeled.

**DOS/VGA 256 deferred, not shipped**: it would exceed the 62-color cap (see
#0018) entirely — 256 > 62, not partially usable like a borderline case — so
shipping a preset that mostly can't be selected felt worse than not shipping
it yet. Add it once #0018's 2-char encoding lands.

Registration order is explicit (`palette.go`'s own `init()` calls
`Register(nesPreset())` then `Register(pico8Preset())`) rather than relying on
Go's per-file init-order convention, since `Names()[0]` (NES) is what a new
project defaults to per #0027.
