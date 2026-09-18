---
id: 0027
title: Single project-wide constrained palette (foundational — settings file + ColorToChar rework)
status: todo
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
