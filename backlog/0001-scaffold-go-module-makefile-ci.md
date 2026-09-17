---
id: 0001
title: Scaffold Go module + Makefile + CI
status: done
priority: high
tags: infra
---

`go.mod` (module `github.com/adl32x/go-pixel-editor`, go 1.26, zero direct deps —
stdlib only), `main.go` hand-rolled subcommand dispatch mirroring go-backlog-cli's
style (no cobra/flag pkg), `Makefile` (web-build/build/install/dist/clean targets,
cross-compile PLATFORMS, VERSION via `git describe`, no install-skill target),
`.gitignore`, `.github/workflows/{ci.yml,release.yml}` (go vet/build/test job +
bun install/build job; tag-triggered `make dist` release job).

See /Users/andreas/.claude/plans/giggly-noodling-sedgewick.md for full spec.
