package sprite

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Frame is one frame of a sprite's animation: a palette-indexed pixel grid
// per sprite layer. Layers is keyed by LayerDef.ID; each grid is
// [height][width] runes, '.' meaning "no pixel".
type Frame struct {
	ID     string
	Layers map[string][][]rune
	Path   string
}

// framesDir returns the frames/ directory for a sprite (a sibling of its
// sprite.md).
func framesDir(s Sprite) string {
	return filepath.Join(filepath.Dir(s.Path), "frames")
}

// LoadFrames reads every frame of s, in filename order. It loads the
// project's Settings itself (see settings.go) since parsing a frame
// validates every pixel char against the current shared palette.
func LoadFrames(s Sprite) ([]Frame, error) {
	entries, err := os.ReadDir(framesDir(s))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".px") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	frames := make([]Frame, 0, len(names))
	for _, name := range names {
		f, err := parseFrameFile(filepath.Join(framesDir(s), name), s, settings)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		frames = append(frames, *f)
	}
	return frames, nil
}

// FindFrame loads one frame of s by id, or returns nil if it doesn't exist.
func FindFrame(s Sprite, id string) (*Frame, error) {
	path := filepath.Join(framesDir(s), id+".px")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}
	return parseFrameFile(path, s, settings)
}

// nextFrameID scans existing frames/*.px filenames for the highest numeric
// suffix and returns max+1. IDs are never reused, even after a delete.
func nextFrameID(s Sprite) (string, error) {
	entries, err := os.ReadDir(framesDir(s))
	if os.IsNotExist(err) {
		return "f001", nil
	}
	if err != nil {
		return "", err
	}
	max := 0
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".px")
		if !strings.HasPrefix(name, "f") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(name, "f"))
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return fmt.Sprintf("f%03d", max+1), nil
}

// parseFrameFile parses one frames/fNNN.px file: a "frame: <id>" header
// (cross-checked against the filename), followed by one "layer: <id>" block
// per sprite layer, each exactly s.Height rows of exactly s.Width
// characters. Every character must be '.' or a known palette char — an
// unrecognized character is treated as a corrupt file, not silently
// ignored.
func parseFrameFile(path string, s Sprite, settings Settings) (*Frame, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	idx := 0
	skipBlank := func() {
		for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
			idx++
		}
	}

	skipBlank()
	if idx >= len(lines) {
		return nil, fmt.Errorf("empty frame file")
	}
	header := strings.TrimSpace(lines[idx])
	id, ok := strings.CutPrefix(header, "frame: ")
	if !ok {
		return nil, fmt.Errorf("missing 'frame: <id>' header, got %q", header)
	}
	base := strings.TrimSuffix(filepath.Base(path), ".px")
	if id != base {
		return nil, fmt.Errorf("frame id %q does not match filename %q", id, base)
	}
	idx++

	f := &Frame{ID: id, Layers: map[string][][]rune{}, Path: path}
	seen := map[string]bool{}

	for {
		skipBlank()
		if idx >= len(lines) {
			break
		}
		layerHeader := strings.TrimSpace(lines[idx])
		layerID, ok := strings.CutPrefix(layerHeader, "layer: ")
		if !ok {
			return nil, fmt.Errorf("frame %s: expected 'layer: <id>' header, got %q", id, layerHeader)
		}
		idx++

		grid := make([][]rune, s.Height)
		for r := 0; r < s.Height; r++ {
			if idx >= len(lines) {
				return nil, fmt.Errorf("frame %s layer %s: truncated, expected %d rows", id, layerID, s.Height)
			}
			row := []rune(lines[idx])
			if len(row) != s.Width {
				return nil, fmt.Errorf("frame %s layer %s row %d: expected width %d, got %d", id, layerID, r, s.Width, len(row))
			}
			for c, ch := range row {
				if ch != '.' {
					if _, ok := settings.CharToColor(ch); !ok {
						return nil, fmt.Errorf("frame %s layer %s row %d col %d: unknown palette char %q", id, layerID, r, c, ch)
					}
				}
			}
			grid[r] = row
			idx++
		}
		f.Layers[layerID] = grid
		seen[layerID] = true
	}

	for _, ld := range s.Layers {
		if !seen[ld.ID] {
			return nil, fmt.Errorf("frame %s: missing layer %s", id, ld.ID)
		}
	}
	if len(seen) != len(s.Layers) {
		return nil, fmt.Errorf("frame %s: contains unknown layer blocks", id)
	}

	return f, nil
}

// Save serializes f into frames/<id>.px under s's directory, overwriting
// whatever is there. Layers are written in s.Layers order (the sprite's
// stack order) so every frame file has the same fixed shape. Since the
// project's palette (settings.go) is fixed rather than append-only, there is
// no longer any ordering requirement with saving sprite-level state first.
func (f Frame) Save(s Sprite) error {
	dir := framesDir(s)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "frame: %s\n\n", f.ID)
	for _, ld := range s.Layers {
		fmt.Fprintf(&b, "layer: %s\n", ld.ID)
		grid := f.Layers[ld.ID]
		for r := 0; r < s.Height; r++ {
			b.WriteString(string(grid[r]))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return writeFileAtomic(filepath.Join(dir, f.ID+".px"), []byte(b.String()), 0o644)
}
