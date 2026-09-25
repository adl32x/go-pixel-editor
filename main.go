// Command pixel is a git-friendly pixel-art editor: a CLI plus a
// browser-based canvas/timeline editor (`pixel serve`). It operates on a
// `.pixel/sprites/` directory in the current working directory — run it from
// inside whichever git repo you want to track sprites for.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/adl32x/go-pixel-editor/internal/export"
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
    --format=gif|sheet-json        (default: sheet-json)
    --out=<path>                   (default: .pixel/sprites/<id>-<slug>/export.<ext>)
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
