---
id: 0014
title: Verify dotting canvas integration live in a browser
status: done
priority: high
tags: frontend, canvas
x: 662.5870783843509
y: 261.51325084870666
---

Done via a live Chrome session once the extension became available. This surfaced
and fixed several real bugs that static verification/type-checking couldn't catch —
see #0021 for the full account (dotting's `getLayersAsArray`/`getLayers` silently
returning `undefined` at runtime despite their type declarations, a destructive
bug where reopening a sprite wiped its saved pixels, a StrictMode-only crash, and
the transparency fills). Confirmed working end-to-end: draw with the mouse (a
drag, specifically — a plain click doesn't register with dotting's interaction
layer), canvas re-renders live, the edit persists to `sprites/*/frames/*.px`,
and — critically — reloading the page and reopening the same sprite shows the
drawing intact rather than a blank frame.

Not covered by this pass: actually pressing play on a clip and watching it animate
in the timeline, and clicking the export buttons in the UI (vs. curl-testing the
export endpoints directly, which was done during backend scaffolding). Carved out
as #0020 since it's a distinct, still-open piece of verification.
