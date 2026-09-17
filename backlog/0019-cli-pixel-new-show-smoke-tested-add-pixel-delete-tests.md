---
id: 0019
title: Add pixel delete <id> command + CLI tests
status: todo
priority: low
tags: cli
---

`pixel new`/`pixel show`/`pixel export`/`pixel serve` are implemented in main.go
and smoke-tested manually (curl + CLI runs during initial scaffolding). Still
missing: a `pixel delete <id>` command (internal/sprite.DeleteSprite already
exists, just needs a CLI subcommand wired to it) and any *_test.go coverage for
main.go's argument parsing (currently only internal/sprite and internal/export
have tests).
