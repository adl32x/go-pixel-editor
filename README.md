# pixel

A git-friendly pixel art editor: a Go web server plus a React canvas/timeline
UI (bundled with Bun), built the same way as
[go-backlog-cli](https://github.com/adl32x/go-backlog-cli) — plain-text
assets, no database, no manifest/index file.

## Why

Pixel art sprites end up as opaque binary blobs in most editors, which makes
them impossible to diff or merge in git. `pixel` stores every sprite as a
small set of plain-text files instead: one `sprite.md` per sprite (metadata:
canvas size, palette, layer stack, animation clips) and one small
palette-indexed text file per frame. Editing one pixel changes one line.
Inserting a frame in the middle of a walk cycle adds one file and one line —
nothing else moves.

## Quick start

```bash
make build        # builds the web UI (Bun) and the pixel binary
./bin/pixel new "Hero" --width=32 --height=32
./bin/pixel serve  # opens a browser tab with the canvas/timeline editor
```

## CLI

```
pixel                          List every sprite (default)
pixel new <name>                 --width= --height= --tags=
pixel show <id>                Print one sprite's sprite.md in full
pixel export <id>                --clip= --format=gif|sheet-json --out=
pixel serve                      --port= --no-open
pixel version / help
```

## On-disk format

```
sprites/
  0001-hero/
    sprite.md          # frontmatter + ## palette / ## layers / ## clips
    frames/
      f001.px           # one row per canvas row, one char per pixel
      f002.px
```

See `SKILL.md` for the full format grammar and package map.

## Export

Export format is chosen at export time, not fixed — pick whichever your
target engine wants:

- `--format=gif` — a single animated GIF.
- `--format=sheet-json` (default) — a zip of `sheet.png` (a horizontal sprite
  sheet) + `data.json` (frame rects/durations/tags, in an Aseprite-compatible
  schema that Phaser/Godot/Unity importers already understand).

New formats are just another `export.Format` implementation registered in
`internal/export`.
