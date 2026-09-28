// Command pixel is a git-friendly pixel-art editor: a CLI plus a
// browser-based canvas/timeline editor (`pixel serve`). It operates on a
// `.pixel/sprites/` directory in the current working directory — run it from
// inside whichever git repo you want to track sprites for.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adl32x/go-pixel-editor/internal/export"
	"github.com/adl32x/go-pixel-editor/internal/importer"
	"github.com/adl32x/go-pixel-editor/internal/server"
	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	args := os.Args[1:]
	command := "list"
	if len(args) > 0 {
		command = strings.ToLower(args[0])
		args = args[1:]
	}

	var err error
	switch command {
	case "list":
		err = runList()
	case "new":
		err = runNew(args)
	case "show":
		err = runShow(args)
	case "export":
		err = runExport(args)
	case "build":
		err = runBuild(args)
	case "import":
		err = runImport(args)
	case "resize":
		err = runResize(args)
	case "serve":
		err = server.Run(args)
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		fmt.Println("pixel", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`pixel — git-friendly pixel art editor

Usage:
  pixel <command>

Commands:
  list                          List every sprite (default)
  new <name>                    Create a sprite
    --width=N --height=N          (default: 16x16)
    --tags=a,b,c                  (default: none)
  show <id>                     Print one sprite's sprite.md in full
  export <id>                   Export a sprite
    --animation=<name>             (default: every animation row, in order)
    --format=gif|sheet-json|sheet-grid (default: sheet-json)
    --out=<path>                   (default: .pixel/sprites/<id>-<slug>/export.<ext>)
  build [id...]                 Render sprites to <slug>.png + <slug>.json
    --out=<dir>                    (default: build_out in .pixel/settings.md;
                                    one of the two is required)
  import <sheet.json|image.png> Create a sprite from existing art
    --name=<name>                  (default: the file's base name)
    --out=<subfolder>              (default: its folder under build_out)
    --frame-width=N --frame-height=N  split a bare PNG into a grid
                                   (default: the whole image is one frame)
  resize <id>                   Change a sprite's canvas size (every frame)
    --width=N --height=N           (default: unchanged)
    --anchor=<pos>                 part that stays put: center (default),
                                   top-left, top, top-right, left, right,
                                   bottom-left, bottom, bottom-right.
                                   Growing adds transparent pixels;
                                   shrinking cuts off what doesn't fit.
  serve                          Open the browser-based canvas/timeline editor
    --port=NNNN                    (default: 7788)
    --no-open                      Don't launch the browser automatically
  version                       Print the pixel version
  help                          Show this help message`)
}

func runList() error {
	sprites, err := sprite.Load()
	if err != nil {
		return err
	}
	if len(sprites) == 0 {
		fmt.Println("No sprites yet. Create one with: pixel new <name>")
		return nil
	}
	for _, s := range sprites {
		sum, err := s.Summary()
		if err != nil {
			return err
		}
		fmt.Printf("%s  %-20s %3dx%-3d  %2d frame(s)  %2d animation(s)  %s\n",
			sum.ID, sum.Name, sum.Width, sum.Height, sum.FrameCount, len(sum.AnimationNames), strings.Join(sum.Tags, ","))
	}
	return nil
}

func runNew(args []string) error {
	var positional []string
	width, height, tags := 0, 0, ""

	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--width="):
			width, _ = strconv.Atoi(strings.TrimPrefix(a, "--width="))
		case strings.HasPrefix(a, "--height="):
			height, _ = strconv.Atoi(strings.TrimPrefix(a, "--height="))
		case strings.HasPrefix(a, "--tags="):
			tags = strings.TrimPrefix(a, "--tags=")
		default:
			positional = append(positional, a)
		}
	}

	name := strings.Join(positional, " ")
	s, err := sprite.NewSprite(name, width, height, tags)
	if err != nil {
		return err
	}
	fmt.Println("Created", s.Path)
	return nil
}

func runShow(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pixel show <id>")
	}
	s, err := sprite.Find(args[0])
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("no sprite with id %s", sprite.NormalizeID(args[0]))
	}
	content, err := os.ReadFile(s.Path)
	if err != nil {
		return err
	}
	fmt.Print(string(content))
	return nil
}

func runExport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pixel export <id> [--animation=] [--format=] [--out=]")
	}
	id := args[0]
	formatName, animName, out := "sheet-json", "", ""
	for _, a := range args[1:] {
		switch {
		case strings.HasPrefix(a, "--format="):
			formatName = strings.TrimPrefix(a, "--format=")
		case strings.HasPrefix(a, "--animation="):
			animName = strings.TrimPrefix(a, "--animation=")
		case strings.HasPrefix(a, "--out="):
			out = strings.TrimPrefix(a, "--out=")
		}
	}

	s, err := sprite.Find(id)
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("no sprite with id %s", sprite.NormalizeID(id))
	}

	format, ok := export.Get(formatName)
	if !ok {
		return fmt.Errorf("unknown export format %q (available: %s)", formatName, strings.Join(export.Names(), ", "))
	}

	var anim *sprite.Animation
	if animName != "" {
		i := s.FindAnimation(animName)
		if i < 0 {
			return fmt.Errorf("no animation named %q", animName)
		}
		anim = &s.Animations[i]
	}

	frames, err := sprite.LoadFrames(*s)
	if err != nil {
		return err
	}
	settings, err := sprite.LoadSettings()
	if err != nil {
		return err
	}
	bundle, err := format.Export(*s, frames, anim, settings)
	if err != nil {
		return err
	}
	data, contentType, err := bundle.Payload()
	if err != nil {
		return err
	}

	if out == "" {
		out = defaultExportPath(*s, formatName, contentType)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	fmt.Println("Wrote", out)
	return nil
}

// runImport creates a sprite from existing art (#0010) — an Aseprite
// json-array or pixel-sheet/1 JSON with its PNG, or a bare PNG.
func runImport(args []string) error {
	var opts importer.Options
	var src string
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--name="):
			opts.Name = strings.TrimPrefix(a, "--name=")
		case strings.HasPrefix(a, "--out="):
			opts.Out = strings.TrimPrefix(a, "--out=")
		case strings.HasPrefix(a, "--frame-width="):
			opts.FrameWidth, _ = strconv.Atoi(strings.TrimPrefix(a, "--frame-width="))
		case strings.HasPrefix(a, "--frame-height="):
			opts.FrameHeight, _ = strconv.Atoi(strings.TrimPrefix(a, "--frame-height="))
		default:
			src = a
		}
	}
	if src == "" {
		return fmt.Errorf("usage: pixel import <sheet.json|image.png> [--name=] [--out=] [--frame-width= --frame-height=]")
	}
	rep, err := importer.Import(src, opts)
	if err != nil {
		return err
	}
	s := rep.Sprite
	names := make([]string, len(s.Animations))
	for i, a := range s.Animations {
		names[i] = fmt.Sprintf("%s (%d)", a.Name, len(a.Frames))
	}
	fmt.Printf("Created %s: %dx%d, %d frame(s), animations: %s\n", s.Path, s.Width, s.Height, rep.Frames, strings.Join(names, ", "))
	if s.Out != "" {
		fmt.Println("Builds into subfolder", s.Out)
	}
	if rep.InexactPixels > 0 {
		fmt.Printf("warning: %d pixel(s) weren't palette colors and were snapped to the nearest one\n", rep.InexactPixels)
	}
	if rep.TranslucentPixels > 0 {
		fmt.Printf("warning: %d pixel(s) had partial alpha (>= 50%% kept opaque, the rest dropped)\n", rep.TranslucentPixels)
	}
	return nil
}

// runResize changes a sprite's canvas size (see sprite.ResizeSprite).
func runResize(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pixel resize <id> [--width=N] [--height=N] [--anchor=center]")
	}
	s, err := sprite.Find(args[0])
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("no sprite with id %s", sprite.NormalizeID(args[0]))
	}
	width, height, anchor := s.Width, s.Height, "center"
	for _, a := range args[1:] {
		switch {
		case strings.HasPrefix(a, "--width="):
			if width, err = strconv.Atoi(strings.TrimPrefix(a, "--width=")); err != nil {
				return fmt.Errorf("invalid --width: %w", err)
			}
		case strings.HasPrefix(a, "--height="):
			if height, err = strconv.Atoi(strings.TrimPrefix(a, "--height=")); err != nil {
				return fmt.Errorf("invalid --height: %w", err)
			}
		case strings.HasPrefix(a, "--anchor="):
			anchor = strings.TrimPrefix(a, "--anchor=")
		default:
			return fmt.Errorf("unknown flag %q", a)
		}
	}
	oldW, oldH := s.Width, s.Height
	if err := sprite.ResizeSprite(s, width, height, anchor); err != nil {
		return err
	}
	fmt.Printf("Resized %s from %dx%d to %dx%d (anchor %s)\n", s.ID, oldW, oldH, s.Width, s.Height, anchor)
	return nil
}

// runBuild renders sprites into the project's build folder (#0031). With no
// ids it builds every sprite and also removes outputs of deleted sprites.
func runBuild(args []string) error {
	out := ""
	var ids []string
	for _, a := range args {
		if strings.HasPrefix(a, "--out=") {
			out = strings.TrimPrefix(a, "--out=")
		} else {
			ids = append(ids, a)
		}
	}
	res, err := export.BuildProject(out, ids)
	if err != nil {
		return err
	}
	for _, n := range res.Written {
		fmt.Println("wrote  ", filepath.Join(res.Out, n))
	}
	for _, n := range res.Removed {
		fmt.Println("removed", filepath.Join(res.Out, n))
	}
	for _, id := range res.Skipped {
		fmt.Printf("skipped sprite %s (no frames)\n", id)
	}
	fmt.Printf("%d written, %d unchanged, %d removed\n", len(res.Written), len(res.Unchanged), len(res.Removed))
	return nil
}

func defaultExportPath(s sprite.Sprite, formatName, contentType string) string {
	ext := ".bin"
	switch contentType {
	case "image/gif":
		ext = ".gif"
	case "image/png":
		ext = ".png"
	case "application/zip":
		ext = ".zip"
	}
	dir := s.Path[:len(s.Path)-len("sprite.md")]
	return dir + formatName + ext
}
