---
id: 0002
title: internal/sprite domain package: parse/save + tests
status: done
priority: high
tags: backend, storage
x: 660.2297287872134
y: -98.99536366145585
---

Domain package (no HTTP awareness), the `internal/backlog` equivalent. `sprite.go`
(Sprite struct, Load/Find/NormalizeID, frontmatter + `## palette`/`## layers`/`## clips`
parse/Save), `frame.go` (Frame struct, `frames/fNNN.px` row codec — one line per row,
one palette char per pixel, `.` = empty), `clip.go` (Clip/ClipEntry/LoopMode, one
frame-ID-per-line `frames:` lists so inserts stay 1-line diffs), `commands.go`
(NewSprite/Reslug/DeleteSprite/AddFrame/DeleteFrame/NewClip/UpdateClip/DeleteClip),
`convert.go` (PixelModifyItem/LayerProps <-> Frame, matching dotting's exact JSON shape,
empty pixel = `color: ""`).

IDs (sprite 4-digit, frame `fNNN`, layer `LN`) are scanned max+1, never renumbered.
Tests use `t.Chdir(t.TempDir())`, must include the plan's literal 4x4 worked example as
a fixture, a palette-overflow test (63rd color), and a "reorder only touches the moved
line" diff assertion.

See /Users/andreas/.claude/plans/giggly-noodling-sedgewick.md ("Storage format" section)
for the exact grammar.
