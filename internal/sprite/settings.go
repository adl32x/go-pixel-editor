package sprite

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adl32x/go-pixel-editor/internal/palette"
)

// SettingsPath is the project-global settings file, alongside Dir
// (.pixel/sprites/) under the .pixel/ dot-folder. Unlike a sprite's own
// metadata, there is exactly one of these per project.
const SettingsPath = ".pixel/settings.md"

// Settings holds the project's single shared palette. Every sprite's frame
// files index into this same ordered color list by character position (see
// paletteAlphabet) — there is no more per-sprite palette.
type Settings struct {
	ActivePreset string         `json:"activePreset"`
	Palette      []PaletteEntry `json:"palette"`
	Updated      time.Time      `json:"updated"`
}

// LoadSettings reads SettingsPath, creating it — defaulting to the first
// registered preset (palette.Names()[0]) — if it doesn't exist yet.
func LoadSettings() (Settings, error) {
	data, err := os.ReadFile(SettingsPath)
	if os.IsNotExist(err) {
		return newDefaultSettings()
	}
	if err != nil {
		return Settings{}, err
	}
	return parseSettings(data)
}

func newDefaultSettings() (Settings, error) {
	names := palette.Names()
	if len(names) == 0 {
		return Settings{}, fmt.Errorf("no palette presets registered")
	}
	p, _ := palette.Get(names[0])
	s := Settings{
		ActivePreset: p.ID,
		Palette:      paletteEntriesFromColors(p.Colors),
		Updated:      time.Now().UTC(),
	}
	if err := s.Save(); err != nil {
		return Settings{}, err
	}
	return s, nil
}

// paletteEntriesFromColors assigns each color the next paletteAlphabet
// character, in order. Colors beyond the 62-character cap are silently
// truncated — see #0018 for lifting this via a 2-char encoding; until then,
// a preset with more than 62 colors (e.g. a 256-color VGA palette) is only
// partially usable.
func paletteEntriesFromColors(colors []string) []PaletteEntry {
	entries := make([]PaletteEntry, 0, len(colors))
	for i, c := range colors {
		if i >= len(paletteAlphabet) {
			break
		}
		entries = append(entries, PaletteEntry{Char: string(paletteAlphabet[i]), Color: c})
	}
	return entries
}

// parseSettings parses SettingsPath's content: a frontmatter block
// (`active_preset`, `updated`) followed by a "## palette" section identical
// in shape to a sprite's old per-sprite one.
func parseSettings(data []byte) (Settings, error) {
	lines := strings.Split(string(data), "\n")
	idx := 0
	for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
		idx++
	}
	if idx >= len(lines) || strings.TrimSpace(lines[idx]) != "---" {
		return Settings{}, fmt.Errorf("missing frontmatter")
	}
	idx++

	fields := map[string]string{}
	for idx < len(lines) {
		line := lines[idx]
		idx++
		if strings.TrimSpace(line) == "---" {
			break
		}
		sep := strings.Index(line, ":")
		if sep == -1 {
			continue
		}
		fields[strings.TrimSpace(line[:sep])] = strings.TrimSpace(line[sep+1:])
	}

	updated, _ := time.Parse(time.RFC3339, fields["updated"])
	s := Settings{ActivePreset: fields["active_preset"], Updated: updated}

	for ; idx < len(lines); idx++ {
		trimmed := strings.TrimSpace(lines[idx])
		if trimmed == "" || trimmed == "## palette" {
			continue
		}
		sep := strings.Index(trimmed, " = ")
		if sep == -1 {
			return Settings{}, fmt.Errorf("invalid palette line: %q", trimmed)
		}
		char := trimmed[:sep]
		color := trimmed[sep+3:]
		if len([]rune(char)) != 1 {
			return Settings{}, fmt.Errorf("invalid palette char: %q", char)
		}
		s.Palette = append(s.Palette, PaletteEntry{Char: char, Color: color})
	}
	return s, nil
}

// Save serializes Settings to SettingsPath atomically.
func (s Settings) Save() error {
	if err := os.MkdirAll(filepath.Dir(SettingsPath), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintln(&b, "---")
	fmt.Fprintf(&b, "active_preset: %s\n", s.ActivePreset)
	fmt.Fprintf(&b, "updated: %s\n", s.Updated.UTC().Format(time.RFC3339))
	fmt.Fprintln(&b, "---")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## palette")
	for _, p := range s.Palette {
		fmt.Fprintf(&b, "%s = %s\n", p.Char, p.Color)
	}
	return writeFileAtomic(SettingsPath, []byte(b.String()), 0o644)
}

// ColorToChar resolves color to a palette character: an exact match if the
// project palette contains it, otherwise the nearest color by RGB distance
// (see colorDistance). Unlike the old per-sprite append-only scheme, the
// palette itself is fixed here — drawing an out-of-palette color never adds
// a new entry, it always snaps to whatever's closest, so this never fails as
// long as the palette isn't empty.
func (s Settings) ColorToChar(color string) (rune, error) {
	if len(s.Palette) == 0 {
		return 0, fmt.Errorf("no palette loaded")
	}
	best := 0
	bestDist := -1
	for i, p := range s.Palette {
		if p.Color == color {
			return firstRune(p.Char), nil
		}
		d := colorDistance(color, p.Color)
		if bestDist == -1 || d < bestDist {
			bestDist = d
			best = i
		}
	}
	return firstRune(s.Palette[best].Char), nil
}

// CharToColor is the reverse lookup used when expanding a frame's compact
// grid back into full pixel data.
func (s Settings) CharToColor(ch rune) (string, bool) {
	for _, p := range s.Palette {
		if firstRune(p.Char) == ch {
			return p.Color, true
		}
	}
	return "", false
}

// colorDistance is squared Euclidean distance in RGB space between two
// "#rrggbb" colors — simple, no new dependency, good enough for a first cut
// (see #0026 for a possible future perceptual-distance refinement). A
// malformed color sorts as maximally distant rather than erroring, so a
// single bad palette entry can't break every lookup.
func colorDistance(a, b string) int {
	ar, ag, ab, aok := parseHexColor(a)
	br, bg, bb, bok := parseHexColor(b)
	if !aok || !bok {
		return 1 << 30
	}
	dr, dg, db := ar-br, ag-bg, ab-bb
	return dr*dr + dg*dg + db*db
}

func parseHexColor(s string) (r, g, b int, ok bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	rv, err1 := strconv.ParseInt(s[0:2], 16, 32)
	gv, err2 := strconv.ParseInt(s[2:4], 16, 32)
	bv, err3 := strconv.ParseInt(s[4:6], 16, 32)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return int(rv), int(gv), int(bv), true
}
