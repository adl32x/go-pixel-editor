---
id: 0005
title: Web scaffold: Bun build/dev scripts, embed pipeline
status: done
priority: high
tags: frontend, infra
x: 665.4841759157657
y: 176.22936303685233
---

Bun is the bundler here (not Vite, unlike go-backlog-cli's reference setup — explicit
user choice). `web/index.html` as the entrypoint referencing `src/main.tsx`, `build.ts`
(`Bun.build({ entrypoints: ["./index.html"], outdir: "../internal/server/dist" })` —
feeds the Go binary's `go:embed all:dist`), `dev.ts` (`Bun.serve` dev server with HTML
import for hot reload + a `/api/*` fetch-proxy to `http://localhost:7777`, the dev-time
equivalent of Vite's `server.proxy`). `package.json` scripts: `dev` (`bun --hot dev.ts`),
`build` (`bun run build.ts`). Deps: `react`, `react-dom`, `dotting`; no state library
added up front.

See the plan's "Frontend" section for the exact script shapes.
