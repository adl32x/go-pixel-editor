package sprite

import (
	"fmt"
	"time"
)

// RemapPalette is the only supported way to change the project's palette.
// Changing Settings.Palette directly (e.g. just overwriting the file) would
// silently reinterpret every existing frame's pixel chars against colors
// they were never drawn with, since a char is just a position in whatever
// palette happens to be active — there is no independent record of what
// color a char "really" meant. So instead: load every sprite's every frame
// under the *old* palette, work out the nearest new-palette color for each
// old color in use, rewrite the affected frames' chars through that
// mapping, and only then save the new Settings.
//
// This is a bulk, project-wide, lossy operation — multiple old colors can
// collapse onto the same nearest new entry — by design: the user asked for
// "closest colors for all sprites when making changes to the palette".
func RemapPalette(activePreset string, colors []string) (Settings, error) {
	oldSettings, err := LoadSettings()
	if err != nil {
		return Settings{}, err
	}
	newSettings := Settings{
		ActivePreset: activePreset,
		Palette:      paletteEntriesFromColors(colors),
		Updated:      time.Now().UTC(),
	}
	if len(newSettings.Palette) == 0 {
		return Settings{}, fmt.Errorf("palette must have at least one color")
	}

	// old char -> new char, built once from the (small, <=62-entry) palettes
	// rather than per-pixel, so remapping stays cheap even for large sprites.
	charTable := make(map[rune]rune, len(oldSettings.Palette))
	for _, p := range oldSettings.Palette {
		newCh, err := newSettings.ColorToChar(p.Color)
		if err != nil {
			return Settings{}, err
		}
		charTable[firstRune(p.Char)] = newCh
	}

	sprites, err := Load()
	if err != nil {
		return Settings{}, err
	}
	for _, s := range sprites {
		frames, err := LoadFrames(s)
		if err != nil {
			return Settings{}, fmt.Errorf("sprite %s: %w", s.ID, err)
		}
		for _, f := range frames {
			changed := false
			for _, grid := range f.Layers {
				for r, row := range grid {
					for c, ch := range row {
						if ch == '.' {
							continue
						}
						if newCh, ok := charTable[ch]; ok && newCh != ch {
							grid[r][c] = newCh
							changed = true
						}
					}
				}
			}
			if changed {
				if err := f.Save(s); err != nil {
					return Settings{}, fmt.Errorf("sprite %s frame %s: %w", s.ID, f.ID, err)
				}
			}
		}
	}

	if err := newSettings.Save(); err != nil {
		return Settings{}, err
	}
	return newSettings, nil
}
