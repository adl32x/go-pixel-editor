// Package sprite implements a plain-text, git-friendly pixel-art sprite
// format: one directory per sprite under .pixel/sprites/, with sprite
// metadata (canvas size, layer stack, animation rows) in a hand-parsed
// sprite.md frontmatter+sections file, and each frame as its own small
// palette-indexed text file under frames/. See settings.go for the
// project's single shared palette (every sprite's frames index into it, not
// a per-sprite palette), frame.go for the frame codec, and animation.go for
// the animation-row block format.
//
// Everything lives under a top-level .pixel/ dot-folder to keep a project's
// repo root clean — this is still fully git-tracked, plain-text, diffable
// data (the whole point of this tool), not a build cache or ignored
// directory; dot-prefixing is purely a "keep the repo root tidy" convention,
// the same reason tools like .vscode/ or .github/ use one.
//
//	.pixel/
//	  settings.md
//	  sprites/
//	    0001-eye/
//	      sprite.md
//	      frames/
//	        f001.px
//	        f002.px
package sprite

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Dir is the folder (relative to the current working directory) that holds
// sprite directories.
const Dir = ".pixel/sprites"

// paletteAlphabet is the ordered set of single-character palette codes a
// sprite can allocate. '.' is reserved (meaning "no pixel") and is never a
// member of this alphabet, which caps a sprite at 62 distinct colors — a
// deliberate simplicity tradeoff that keeps the frame codec fixed-width.
const paletteAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Sprite is a single pixel-art asset, backed by a sprite.md file plus a
// frames/ directory, both under one .pixel/sprites/<id>-<slug>/ directory.
type Sprite struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Width   int       `json:"width"`
	Height  int       `json:"height"`
	Tags    []string  `json:"tags"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	// DurationMS is how long each frame is held during playback unless the
	// frame overrides it (see AnimFrame.DurationMS).
	DurationMS int `json:"durationMs"`

	Layers     []LayerDef  `json:"layers"`
	Animations []Animation `json:"animations"`

	Path string `json:"-"`
}

// PaletteEntry maps a single-character code (from paletteAlphabet) to a CSS
// color string, exactly as used by the frame codec and by dotting's
// PixelModifyItem.Color. See settings.go — this is now the project's single
// shared palette, not a per-sprite one.
type PaletteEntry struct {
	Char  string `json:"char"`
	Color string `json:"color"`
}

// LayerDef describes one layer in the sprite's shared layer stack (every
// frame has a block for every layer). Order in Sprite.Layers is stack order,
// first = topmost.
type LayerDef struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Visible bool    `json:"visible"`
	Opacity float64 `json:"opacity"`
}

// Summary is the compact shape returned by the sprite list endpoint.
type Summary struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	Tags           []string `json:"tags"`
	FrameCount     int      `json:"frameCount"`
	AnimationNames []string `json:"animationNames"`
}

// Summary computes the list-view summary for s, which requires counting
// frame files on disk.
func (s Sprite) Summary() (Summary, error) {
	frames, err := LoadFrames(s)
	if err != nil {
		return Summary{}, err
	}
	names := make([]string, len(s.Animations))
	for i, a := range s.Animations {
		names[i] = a.Name
	}
	return Summary{
		ID: s.ID, Name: s.Name, Width: s.Width, Height: s.Height,
		Tags: s.Tags, FrameCount: len(frames), AnimationNames: names,
	}, nil
}

// NormalizeID left-pads a sprite id with zeros to the canonical 4-digit
// form, mirroring go-backlog-cli's ticket ID scheme.
func NormalizeID(id string) string {
	n, err := strconv.Atoi(strings.TrimSpace(id))
	if err != nil {
		return id
	}
	return fmt.Sprintf("%04d", n)
}

// Load reads every sprite in Dir. Returns an empty slice (not an error) if
// Dir doesn't exist yet.
func Load() ([]Sprite, error) {
	entries, err := os.ReadDir(Dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var sprites []Sprite
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(Dir, e.Name(), "sprite.md")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		s, err := parseSpriteFile(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		sprites = append(sprites, *s)
	}
	sort.Slice(sprites, func(i, j int) bool { return sprites[i].ID < sprites[j].ID })
	return sprites, nil
}

// Find loads every sprite and returns the one matching id (normalized to 4
// digits), or nil if not found.
func Find(id string) (*Sprite, error) {
	id = NormalizeID(id)
	sprites, err := Load()
	if err != nil {
		return nil, err
	}
	for i := range sprites {
		if sprites[i].ID == id {
			return &sprites[i], nil
		}
	}
	return nil, nil
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func slugify(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		out = "sprite"
	}
	return out
}

func splitTags(tags string) []string {
	out := []string{}
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// parseSpriteFile reads and parses one sprite.md file: a frontmatter block
// (same hand-rolled key:value scanner as go-backlog-cli's ticket.go) followed
// by "## layers" / "## animations" sections. Rows are then normalized
// against the frame files on disk (see normalizeAnimations).
func parseSpriteFile(path string) (*Sprite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	if !scanner.Scan() {
		return nil, fmt.Errorf("empty file")
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil, fmt.Errorf("missing frontmatter")
	}

	fields := map[string]string{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		sep := strings.Index(line, ":")
		if sep == -1 {
			continue
		}
		key := strings.TrimSpace(line[:sep])
		value := strings.TrimSpace(line[sep+1:])
		fields[key] = value
	}

	var rest []string
	for scanner.Scan() {
		rest = append(rest, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	width, _ := strconv.Atoi(fields["width"])
	height, _ := strconv.Atoi(fields["height"])
	duration, err := strconv.Atoi(fields["duration"])
	if err != nil || duration <= 0 {
		duration = DefaultFrameDurationMS
	}
	created, _ := time.Parse(time.RFC3339, fields["created"])
	updated, _ := time.Parse(time.RFC3339, fields["updated"])

	s := &Sprite{
		ID:         fields["id"],
		Name:       fields["name"],
		Width:      width,
		Height:     height,
		Tags:       splitTags(fields["tags"]),
		Created:    created,
		Updated:    updated,
		DurationMS: duration,
		Path:       path,
		Layers:     []LayerDef{},
		Animations: []Animation{},
	}

	if err := parseSections(rest, s); err != nil {
		return nil, err
	}
	onDisk, err := frameIDsOnDisk(*s)
	if err != nil {
		return nil, err
	}
	normalizeAnimations(s, onDisk)
	return s, nil
}

// parseSections walks the lines after the frontmatter block, dispatching
// into the "## layers" and "## animations" sections. Section and row
// headers must be unindented; frame-entry lines inside a row's "frames:"
// list must be indented — that distinction is how the scanner tells a
// section boundary from a list item.
//
// A legacy "## clips" section is read as "## animations": its per-clip
// "loop:"/"fps:" scalars are ignored, and any frame shared between clips
// ends up in the first one only (see normalizeAnimations).
func parseSections(lines []string, s *Sprite) error {
	section := ""
	var anim *Animation
	inFramesList := false

	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		indented := len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\t')

		if !indented && strings.HasPrefix(trimmed, "## ") {
			section = strings.TrimPrefix(trimmed, "## ")
			if section == "clips" {
				section = "animations"
			}
			anim = nil
			inFramesList = false
			continue
		}
		if section == "animations" && !indented && strings.HasPrefix(trimmed, "### ") {
			name := strings.TrimPrefix(trimmed, "### ")
			s.Animations = append(s.Animations, Animation{Name: name, Frames: []AnimFrame{}})
			anim = &s.Animations[len(s.Animations)-1]
			inFramesList = false
			continue
		}

		switch section {
		case "layers":
			if err := parseLayerLine(s, trimmed); err != nil {
				return err
			}
		case "animations":
			if anim == nil {
				return fmt.Errorf("animation content outside a ### block: %q", trimmed)
			}
			if indented {
				if !inFramesList {
					return fmt.Errorf("animation %s: frame entry outside frames: list: %q", anim.Name, trimmed)
				}
				if err := parseAnimFrameLine(anim, trimmed); err != nil {
					return err
				}
				continue
			}
			// Anything else unindented (legacy "loop:"/"fps:") ends the list
			// and is otherwise ignored.
			inFramesList = trimmed == "frames:"
		}
	}
	return nil
}

func parseLayerLine(s *Sprite, line string) error {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return fmt.Errorf("invalid layer line: %q", line)
	}
	ld := LayerDef{ID: fields[0], Name: fields[1], Visible: true, Opacity: 1}
	for _, kv := range fields[2:] {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		switch k {
		case "visible":
			ld.Visible = v == "true"
		case "opacity":
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				ld.Opacity = f
			}
		}
	}
	s.Layers = append(s.Layers, ld)
	return nil
}

// Save serializes the sprite's frontmatter + layers/animations sections
// back to s.Path, overwriting whatever is there. It is the single source of
// truth for the on-disk sprite.md format — every mutation (creation, patch,
// frame/animation commands that touch sprite-level state) goes through it rather
// than hand-patching lines.
func (s Sprite) Save() error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "frames"), 0o755); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintln(&b, "---")
	fmt.Fprintf(&b, "id: %s\n", s.ID)
	fmt.Fprintf(&b, "name: %s\n", s.Name)
	fmt.Fprintf(&b, "width: %d\n", s.Width)
	fmt.Fprintf(&b, "height: %d\n", s.Height)
	fmt.Fprintf(&b, "tags: %s\n", strings.Join(s.Tags, ", "))
	fmt.Fprintf(&b, "duration: %d\n", s.DurationMS)
	fmt.Fprintf(&b, "created: %s\n", s.Created.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "updated: %s\n", s.Updated.UTC().Format(time.RFC3339))
	fmt.Fprintln(&b, "---")

	if len(s.Layers) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "## layers")
		for _, l := range s.Layers {
			fmt.Fprintf(&b, "%s %s visible=%t opacity=%s\n", l.ID, l.Name, l.Visible, strconv.FormatFloat(l.Opacity, 'f', -1, 64))
		}
	}

	if len(s.Animations) > 0 {
		fmt.Fprintln(&b)
		formatAnimationsSection(&b, s.Animations)
	}

	return writeFileAtomic(s.Path, []byte(b.String()), 0o644)
}
