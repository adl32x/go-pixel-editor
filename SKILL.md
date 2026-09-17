---
name: pixel-editor
description: Git-friendly pixel art editor. Use to view, create, or export sprites (plain-text sprite.md + frames/*.px) in a repo's sprites/ folder, or to launch the browser-based canvas/timeline editor.
---

# Skill: pixel editor

Sprites are plain-text files in a `sprites/` directory **inside the current
working directory** — always `cd` into the target repo first. There is no
global/shared sprite store; each repo has its own.

## On-disk format

One directory per sprite: `sprites/NNNN-slug/`.

```
sprites/0001-eye/
  sprite.md
  frames/
    f001.px
    f002.px
```

`sprite.md` — frontmatter (same hand-rolled `key: value` scanner as
go-backlog-cli's tickets, no YAML lib) plus `## palette` / `## layers` /
`## clips` sections:

```markdown
---
id: 0001
name: Eye
width: 4
height: 4
tags: ui, eye
created: 2026-09-17T09:00:00Z
updated: 2026-09-17T09:00:00Z
---

## palette
0 = #000000
1 = #ffffff

## layers
L1 base visible=true opacity=1

## clips
### blink
loop: forward
fps: 2
frames:
  f001
  f002
```

- `## palette`: one `<char> = <color>` line per color, append-only (a char's
  color never changes once written). Alphabet is `0-9A-Za-z` — a sprite caps
  out at 62 distinct colors.
- `## layers`: one line per layer, stack order (first line = topmost):
  `<id> <name> visible=<bool> opacity=<float>`.
- `## clips`: repeated `### <name>` blocks. `loop:` is `none`, `forward`, or
  `pingpong`; `fps:` is the default frame rate; `frames:` is followed by
  **one frame id per line** (never a single space-separated line — that's
  what keeps inserting a frame a clean single-line `git diff`), optionally
  `  f008 @250` to override that entry's duration in milliseconds.

`frames/fNNN.px` — one line per canvas row, one character per pixel, `.`
meaning "no pixel", resolved through the sprite's palette:

```
frame: f001

layer: L1
....
.10.
.01.
....
```

Every sprite layer must appear as its own `layer: <id>` block of exactly
`height` rows of exactly `width` characters. IDs (sprite/frame/layer) are
never renumbered — they're derived by scanning existing files/lines for
`max+1`, the same scheme as go-backlog-cli's ticket IDs.

## Usage

```bash
pixel                            # list every sprite
pixel new "<name>" --width= --height= --tags=
pixel show <id>                  # print one sprite's sprite.md in full
pixel export <id> --clip= --format=gif|sheet-json --out=
pixel serve --port= --no-open    # browser-based canvas/timeline editor
pixel version / help
```

## Browser editor (`pixel serve`)

Starts a localhost HTTP server and opens a browser tab: a
[dotting](https://github.com/hunkim98/dotting)-based pixel canvas plus a
custom frame timeline and playback controls (dotting itself has no notion of
frames/animation — only static layers — so the timeline, playback loop, and
clip model are this project's own code, built on top of dotting's per-frame
canvas). Create a sprite, draw pixels, add frames, group frames into a named
clip with an fps/loop mode, and export.

## Configurable export

Export format is chosen per-request, not fixed in code — `internal/export`
is a small registry of `Format` implementations:

- `gif` — a single animated GIF (loop mode maps to the GIF's Netscape loop
  extension; `pingpong` clips are expanded into an explicit forward+reverse
  frame sequence since GIF itself can't bounce).
- `sheet-json` (default) — `sheet.png` (a horizontal sprite sheet) +
  `data.json` in an Aseprite-compatible schema (frame rects/durations,
  `frameTags` for clips) — importable by common engines (e.g. Phaser's
  `load.aseprite`) without a bespoke schema.

Adding a new target format is a new `export.Format` implementation plus a
`Register()` call in `internal/export/export.go`'s `init()` — no HTTP/CLI
changes needed.

## Implementation

- `main.go` — subcommand dispatch (`list`, `new`, `show`, `export`, `serve`).
- `internal/sprite/sprite.go` — `Sprite` struct, frontmatter+sections
  parse/`Save`, `Load`/`Find`, palette char allocation.
- `internal/sprite/frame.go` — `Frame` struct, the `.px` row codec,
  `LoadFrames`/`FindFrame`/`Frame.Save`.
- `internal/sprite/clip.go` — `Clip`/`ClipEntry`/`LoopMode`, the `## clips`
  block parse/format.
- `internal/sprite/commands.go` — `NewSprite`, `Reslug`, `AddFrame`,
  `DeleteFrame`, `NewClip`, `UpdateClip`, patch types.
- `internal/sprite/convert.go` — `LayerProps`/`PixelModifyItem` (dotting's
  exact JSON shape) ⇄ `Frame` conversion.
- `internal/export/` — the pluggable `Format` registry, `gif.go`,
  `sheet_json.go`, plus `image.go` (shared layer-compositing helper).
- `internal/server/server.go` — HTTP API + embedded static file server
  (`go:embed all:dist`).
- `web/` — the `pixel serve` frontend: Bun (bundler + package manager,
  **not** Vite) + React + TypeScript + dotting, built to
  `internal/server/dist` and embedded via `go:embed`.

## Build & install

```bash
make          # build, install binary to ~/.local/bin
make build    # bun-build the web UI into internal/server/dist, then build bin/pixel
make web-build # just (re)build the web UI, e.g. after editing web/src
make install  # build + copy binary to ~/.local/bin
```
