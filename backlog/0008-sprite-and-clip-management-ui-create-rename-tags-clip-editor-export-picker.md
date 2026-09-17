---
id: 0008
title: Sprite and clip management UI (create/rename/tags, clip editor, export picker)
status: done
priority: medium
tags: frontend
---

`SpriteList.tsx` (list/create/select sprites), `SpriteMeta.tsx` (name/tags editor for
the active sprite), `ClipEditor.tsx` (create/rename clips, edit loop/fps, reorder frame
entries) plus an export picker: a `<select>` populated from `GET /api/export-formats`
and a download link/button hitting `GET /api/sprites/{id}/export?format=&clip=` (direct
`window.open`/link, no client-side blob handling needed).

Depends on #0004 (API) and #0005–#0007 (canvas/timeline scaffolding) being in place
first.
