---
id: 0004
title: HTTP API: sprites/frames/clips CRUD + export endpoint
status: done
priority: high
tags: backend, api
---

Thin HTTP layer over `internal/sprite` + `internal/export`, stdlib `net/http.ServeMux`
with Go 1.22+ method+pattern routes, no framework. Mirrors go-backlog-cli's
`writeJSON`/`writeError` ({"error": "..."} + 400) convention and its `go:embed all:dist`
static handler with SPA fallback + "web UI not built" message — ported near-verbatim.

Routes: `GET/POST /api/sprites`, `GET/PATCH/DELETE /api/sprites/{id}`,
`POST /api/sprites/{id}/frames`, `GET/PUT/DELETE .../frames/{frameId}` (DELETE is 409 if
referenced by a clip unless `?force=true`), `GET/POST /api/sprites/{id}/clips`,
`PATCH/DELETE .../clips/{name}`, `GET .../export.png?frame=`, `GET .../export?format=&clip=`
dispatching through the export registry, `GET /api/export-formats`.

`Run(args []string) error` supports `--port=` (default 7777) and `--no-open`. See the
plan's "HTTP API" table for the full route list.
