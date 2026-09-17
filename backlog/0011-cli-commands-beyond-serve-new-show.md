---
id: 0011
title: CLI commands beyond serve (new, show)
status: done
priority: medium
tags: cli
---

The initial pass's `main.go` covers `list`/`new`/`show`/`export`/`serve`/`version`/`help`
at a basic level (hand-rolled dispatch, no cobra, mirroring go-backlog-cli's `main.go`).
This ticket is for hardening/extending those beyond a minimal first cut once real usage
surfaces gaps — e.g. richer `list` filtering by tag, `pixel delete <id>`, batch export of
every clip in a sprite, `--tags=` editing via CLI. No TUI planned (tracked as a
possible future ticket only if a concrete use case shows up — pixel editing needs the
canvas, so the browser UI is the primary interactive surface, unlike go-backlog-cli's
list-browsable tickets).
