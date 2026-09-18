---
id: 0021
title: Atomic sprite/frame writes (fix concurrent-write torn-read corruption)
status: done
priority: high
tags: backend, storage
x: 669.4826841512204
y: 439.4253661860942
---

`Sprite.Save()`/`Frame.Save()` used `os.WriteFile`, which opens with `O_TRUNC` and
writes in a separate step — not atomic. A concurrent request (e.g. a debounced
frame `PUT` landing close to a layer-panel metadata `PATCH`) could read
`sprite.md` mid-write and see it truncated to zero bytes, failing with
`parsing sprites/000X-*/sprite.md: empty file`.

Fixed with a `writeFileAtomic` helper (`internal/sprite/atomic.go`): write to a
temp file in the same directory, then `os.Rename` over the target — atomic on a
single filesystem, so a reader only ever sees the fully-old or fully-new content,
never a torn intermediate state. Used by both `Sprite.Save()` and `Frame.Save()`.

Added a permanent regression test, `TestConcurrentSavesNeverProduceEmptyFile`
(`internal/sprite/sprite_test.go`) — confirmed it reproduces the exact bug against
the old `os.WriteFile`-based code and passes clean under `go test -race` with the
fix.
