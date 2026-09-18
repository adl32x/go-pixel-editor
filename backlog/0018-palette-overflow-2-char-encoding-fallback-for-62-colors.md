---
id: 0018
title: Palette overflow: 2-char encoding fallback for >62 colors
status: todo
priority: medium
tags: backend, storage
x: -55.94708759721733
y: 75.02209102402128
---

The palette (`internal/sprite/sprite.go`, `ColorToChar`) is capped at 62
distinct colors — one alphabet character (`0-9A-Za-z`) per color, by design, to
keep the `.px` frame format fixed-width and trivially parseable (see the plan's
"Storage format" section for the original rationale). `ColorToChar` returns an
explicit error past 62 rather than silently switching encodings. Since #0027,
this cap applies to the single project-wide palette (shared by every sprite)
rather than drifting independently per sprite, but the cap itself is unchanged.

**Bumped from low to medium priority**: #0024 proposes shipping a DOS/VGA
256-color preset, which directly collides with this 62-color cap — a project
using that preset couldn't represent more than 62 of its 256 colors as things
stand. This ticket (a `frames/*.px2` variant, or similar, using 2 characters
per pixel — fixed-width still, just base-4096-ish instead of base-62) needs to
land before or alongside #0024's DOS-256 preset actually being usable end to
end, not just as a background nice-to-have.
