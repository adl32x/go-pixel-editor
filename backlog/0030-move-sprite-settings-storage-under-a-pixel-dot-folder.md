---
id: 0030
title: Move sprite/settings storage under a .pixel/ dot-folder
status: done
priority: high
tags: backend, storage
---

User request: keep the repo root clean — `sprites/` and `pixel.settings.md`
should live under a dot-folder, `.pixel/`, rather than top-level, similar to
how other tools keep their own state out of the way (`.vscode/`, `.git/`).

Scope is small: `internal/sprite.Dir` becomes `.pixel/sprites` and
`internal/sprite.SettingsPath` becomes `.pixel/settings.md` (dropping the now-
redundant `pixel.` prefix since it's already namespaced by the dot-folder).
`os.MkdirAll` already creates intermediate directories, so no other code
changes needed — just the two constants, plus doc comments/README/SKILL.md
that describe the old top-level layout.

**Important — this is explicitly still git-tracked, not a build
cache/gitignored directory.** Git-friendly diffable sprite storage is this
whole project's founding premise (see the original plan); moving it under a
dot-folder is purely about tidying the repo root, matching a convention many
dev tools already use for their own config/state, not about making the data
ephemeral. Nothing changes in `.gitignore` for `.pixel/` itself.

Separately, the user mentioned a future "output step" that defines how
rendered sprites get saved into the wider repo (see #0031) — that's about
where *exported* artifacts (PNGs, sheets) land for actual game use, distinct
from `.pixel/` which remains the git-friendly editable source of truth.

---

**Implemented** exactly as scoped — `Dir = ".pixel/sprites"`, `SettingsPath =
".pixel/settings.md"`, doc comments/README/SKILL.md updated. One real gap
`os.MkdirAll` alone didn't cover: `Settings.Save()` had never needed to
create a parent directory before (its old path, `pixel.settings.md`, always
had `.` — the cwd, always present — as its parent), so it was missing the
`os.MkdirAll(filepath.Dir(SettingsPath), 0o755)` call `Sprite.Save()` already
had; added it, since a fresh project's very first `Settings.Save()` (from
`LoadSettings()`'s default-creation path) now needs `.pixel/` itself created
first. Verified: `pixel new` on an empty directory correctly creates
`.pixel/sprites/0001-.../sprite.md`.

Also fixed while touching related test infrastructure: `pixel serve`'s
default port changed from 7777 to **7788** (separate user request, made in
the same session) specifically because 7777 collides with go-backlog-cli's
own `backlog serve` default — discovered the hard way when testing this very
change (see the port-conflict note; nothing in `.pixel/`'s own design
required this, it's an unrelated but related fix bundled into the same
verification pass).
