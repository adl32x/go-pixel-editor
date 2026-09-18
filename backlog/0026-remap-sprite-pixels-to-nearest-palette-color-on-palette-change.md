---
id: 0026
title: Remap sprite pixels to nearest palette color on palette change
status: done
priority: medium
tags: backend, palette
x: -336.3278071956675
y: 2.5409889936038397
---

User request: when the project's single shared palette changes (preset switch
or a manual color edit/removal via #0025), the backend should find the closest
color for every sprite so existing artwork retargets to the new palette rather
than breaking or going untouched.

**Decided** (see #0027): projects have exactly one shared palette, and a
`.px` frame's chars index directly into it (positionally) — there is no more
per-sprite `## palette` block to separately update. That changes what this
ticket actually has to do, compared to the per-sprite-palette design it was
originally scoped against: **this is now a real pixel-data rewrite across
every frame in the project**, not a metadata-only tweak.

Mechanics: given the old palette (list of colors, index = char) and the new
one, build an old-char → new-char table by, for each old index's color, finding
the nearest color in the new palette (see distance metric below) and taking
*that* color's index in the new list. Then walk every sprite's every
`frames/*.px` file and rewrite each pixel char through this table (an in-place
char substitution per the existing `.px` row format — cheap per file, but
there could be many files across a large project, so batch it and don't hold
every sprite in memory at once). Each rewritten frame file goes through the
existing atomic `writeFileAtomic` (#0021).

Nearest-color matching: start with plain Euclidean distance in RGB
(`(r1-r2)² + (g1-g2)² + (b1-b2)²`, sum minimized) — simple, no new dependency,
good enough for a first cut. Note perceptual distance (e.g. weighted RGB or a
CIE Lab conversion) as a possible future refinement if plain RGB produces
visibly bad matches, but don't over-engineer it up front.

**Already exists, reuse it**: `internal/sprite/settings.go` (#0027) has
exactly this distance function already — `colorDistance`/`parseHexColor`,
currently unexported and used by `Settings.ColorToChar`'s snap-to-nearest
behavior. Either export it or add a small package-level helper that takes two
palettes and returns the old-index→new-index table this ticket needs, rather
than re-deriving the same RGB-distance math a second time.

This is a bulk, project-wide, somewhat lossy operation (multiple old colors
can collapse onto the same nearest new entry) — #0025's UI should warn clearly
before triggering it, and it's worth considering whether the API should
support a dry-run/preview response (old→new color pairs) before committing to
the actual rewrite pass.

Depends on #0024 (a palette to remap *to*) and #0027 (the single-palette
architecture this rewrite operates within — must land first, not in parallel);
#0025 is the UI trigger for this but this ticket is backend-only.

---

**Implemented** as `internal/sprite/remap.go`'s `RemapPalette(activePreset
string, colors []string) (Settings, error)` — matches the mechanics described
above closely: loads every sprite's every frame under the *old* (still
on-disk) settings, builds an old-char→new-char table via `Settings
.ColorToChar`'s existing nearest-match logic (kept as a method, not
extracted into a standalone two-palette helper — `RemapPalette` just
constructs the new `Settings` first and calls its own `ColorToChar` per old
color, which turned out simpler than threading two palettes through a
separate function), rewrites only the frames that actually contain a
changed char (confirmed via `TestRemapPaletteLeavesUntouchedFramesAlone` that
untouched frames aren't rewritten), and only saves the new `Settings` last —
after every frame rewrite succeeds — so a reader never sees a palette that
doesn't match what's on disk yet.

No dry-run/preview endpoint was added — out of scope for this pass; #0025's
UI warns via `confirm()` before calling it, which was judged sufficient for
now given the existing codebase's precedent (`Timeline.tsx`'s delete
confirmation) rather than adding new API surface for a preview.

Two tests added (`internal/sprite/remap_test.go`):
`TestRemapPaletteRewritesExistingFrames` — the core case, using a palette
whose color-to-position order is deliberately shuffled between old and new,
so a bug that left chars alone (instead of genuinely remapping by color)
would be caught; and `TestRemapPaletteLeavesUntouchedFramesAlone` — a blank
frame's file must not be rewritten if it has nothing to remap. Both pass
under `go test -race`. Also verified live end-to-end through the browser
(see #0025's done-note) with a real preset switch on a drawn sprite.
