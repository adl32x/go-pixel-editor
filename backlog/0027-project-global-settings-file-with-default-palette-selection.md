---
id: 0027
title: Single project-wide constrained palette (foundational — settings file + ColorToChar rework)
status: done
priority: high
tags: backend, palette, link:0024-0027
x: -367.063532961104
y: 571.8251242370802
---

**Decided** (was an open question, now resolved): projects have exactly ONE
palette, shared by every sprite — not each sprite freely allocating its own
colors as it's drawn (today's behavior). This is foundational: #0024 (presets),
#0025 (settings page), and #0026 (remap) all depend on this landing first.

**Storage format change**: today, each sprite's own `## palette` block in
`sprite.md` is an independent append-only list (`Sprite.Palette`,
`Sprite.ColorToChar` in `internal/sprite/sprite.go`) — a `.px` frame's chars
index into *that sprite's own* list. Going forward, a `.px` char instead
indexes into the **one project-wide palette**, so the per-sprite `## palette`
block goes away entirely (this is a breaking change to the on-disk format —
acceptable at this stage since the project has no real user data yet, but
worth flagging as a compatibility break if that stops being true).

New top-level file, sibling to `sprites/` and `backlog/` — matching the
project's plain-text/hand-parsed-frontmatter philosophy, no new config
library, e.g. `pixel.settings.md`:

```
---
active_preset: nes
updated: 2026-09-17T12:00:00Z
---

## palette
0 = #000000
1 = #ffffff
...
```

(`active_preset` is informational/for display — the `## palette` block below
it is the actual authoritative, editable, ordered color list every sprite's
frames index into; a fully custom edit just means this block no longer matches
any named preset exactly.)

**Behavior change to `ColorToChar`**: currently allocates a new char for any
unseen color, erroring past 62 (see #0018). Once the palette is a fixed,
user-controlled list rather than something that grows as you draw, drawing an
out-of-palette color should **snap to the nearest existing palette entry**
(reuse whatever distance function #0026 introduces) rather than allocate or
error. Whether the frontend should also *restrict* the brush color picker to
palette swatches only (removing free `<input type=color>` picking, or keeping
it but snapping visibly on save so the user sees what actually got drawn) is
worth deciding when this is implemented — snapping silently could be
surprising otherwise.

A new small `internal/settings` package (or `internal/palette`, shared with
#0024) owns parse/save: `Load()` returns the first registered preset's colors
(#0024's `palette.Names()[0]`) and writes the file if it doesn't exist yet;
`Save()` persists a new selection/edit. Loaded once at `pixel serve` startup
and after any change from #0025's settings page. `internal/sprite`'s
`ColorToChar`/`CharToColor` move from `Sprite` methods to operating against
this loaded global palette instead. Depends on #0024 existing first (needs a
preset to default to).

---

**Implemented.** `internal/sprite/settings.go`: `Settings{ActivePreset,
Palette, Updated}`, `LoadSettings()` (creates `pixel.settings.md` from
`palette.Names()[0]` on first run), `Settings.Save()` (atomic, via the
existing `writeFileAtomic`). `ColorToChar` moved off `Sprite` onto `Settings`
and now genuinely never fails or grows the palette — exact match first, else
nearest by squared RGB Euclidean distance (`colorDistance`) — resolving the
open question decisively: the palette is fixed, drawing an unlisted color
always snaps rather than allocating.

Every call site that touched the old per-sprite palette was threaded onto the
loaded `Settings` instead: `Frame.ToLayerProps`/`FrameFromLayerProps` (now a
value receiver, not `*Sprite` — it no longer mutates anything),
`parseFrameFile`/`LoadFrames`/`FindFrame` (validate pixel chars against the
project palette, not a per-sprite one), and every export path
(`compositeFrame`, `buildPalette`, the `Format` interface itself gained a
`settings sprite.Settings` parameter). `internal/server` loads `Settings`
fresh per request (`sprite.LoadSettings()`, same directory-as-database
philosophy as `sprite.Find`/`sprite.Load` — no in-memory cache to invalidate).

New endpoints: `GET /api/palette` (active settings), `GET /api/palette-presets`
(list, from #0024), `GET /api/palette-presets/{id}` (one preset's full colors)
— the latter two aren't consumed by the frontend yet (that's #0025's job), but
`GET /api/palette` is: `App.tsx` fetches it once and feeds `PaletteBar`, which
previously read the now-removed `sprite.palette`.

**Left as-is, not addressed here**: `PaletteBar.tsx`'s free `<input
type=color>` picker still exists — since the backend snaps unconditionally on
save regardless of what's submitted, this doesn't corrupt anything, but a user
picking an arbitrary color gets silently snapped to the nearest palette entry
with no preview before the round-trip. Worth revisiting as part of #0025.

**Verified live in the browser**, not just type-checked/unit-tested (the
project's established bar per #0020): created a sprite, confirmed the
`PaletteBar` shows the full 55-color NES default, drew with a NES swatch,
confirmed the `.px` file stored the right char, reloaded the page and
reopened the sprite, confirmed the drawing was intact. Also drove the snap
behavior directly through the live HTTP API with an off-palette color
(`#f9c700` → snapped to NES's `#f8b800`).

Full test suite (`go test ./... -race`) green across `internal/sprite`,
`internal/export`, and the new `internal/palette`; `tsc --noEmit` and
`bun run build` clean on the frontend.
