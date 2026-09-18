package sprite

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRemapPaletteRewritesExistingFrames covers the core guarantee: after
// switching the project's palette, an existing frame's pixel chars still
// resolve to the same (or nearest) visual color under the *new* palette,
// because the chars themselves were rewritten — not just left pointing at
// whatever the new palette happens to have at that same position.
func TestRemapPaletteRewritesExistingFrames(t *testing.T) {
	t.Chdir(t.TempDir())

	// Old palette: char '0' = black, char '1' = pure red.
	oldSettings := Settings{
		ActivePreset: "old",
		Palette: []PaletteEntry{
			{Char: "0", Color: "#000000"},
			{Char: "1", Color: "#ff0000"},
		},
	}
	if err := oldSettings.Save(); err != nil {
		t.Fatalf("oldSettings.Save: %v", err)
	}

	s, err := NewSprite("Eye", 2, 2, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	f, err := AddFrame(s)
	if err != nil {
		t.Fatalf("AddFrame: %v", err)
	}
	// Draw: (0,0) black, (0,1) red.
	f.Layers["L1"][0][0] = '0'
	f.Layers["L1"][0][1] = '1'
	if err := f.Save(s); err != nil {
		t.Fatalf("Frame.Save: %v", err)
	}

	// New palette: deliberately different order — char '0' = white, char
	// '1' = black, char '2' = a red-ish color. If remap just left chars
	// alone, (0,0) would silently become white and (0,1) would become
	// black — wrong. It must instead rewrite chars so the *colors* still
	// make sense: black should now be char '1', red should snap to '2'.
	newColors := []string{"#ffffff", "#000000", "#e00000"}
	if _, err := RemapPalette("new", newColors); err != nil {
		t.Fatalf("RemapPalette: %v", err)
	}

	newSettings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after remap: %v", err)
	}
	if newSettings.ActivePreset != "new" {
		t.Fatalf("ActivePreset = %q, want %q", newSettings.ActivePreset, "new")
	}

	reloadedSprite, err := Find(s.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	reloadedFrame, err := FindFrame(*reloadedSprite, f.ID)
	if err != nil {
		t.Fatalf("FindFrame after remap: %v", err)
	}
	grid := reloadedFrame.Layers["L1"]

	blackColor, _ := newSettings.CharToColor(grid[0][0])
	if blackColor != "#000000" {
		t.Fatalf("(0,0) resolves to %q, want #000000 (black must stay black)", blackColor)
	}
	redColor, _ := newSettings.CharToColor(grid[0][1])
	if redColor != "#e00000" {
		t.Fatalf("(0,1) resolves to %q, want #e00000 (red must snap to the new red)", redColor)
	}
}

// TestRemapPaletteLeavesUntouchedFramesAlone confirms a frame using only
// colors that map to themselves (same char, e.g. an unused palette slot or
// a genuinely unchanged entry) isn't needlessly rewritten.
func TestRemapPaletteLeavesUntouchedFramesAlone(t *testing.T) {
	t.Chdir(t.TempDir())

	oldSettings := Settings{
		ActivePreset: "old",
		Palette:      []PaletteEntry{{Char: "0", Color: "#000000"}},
	}
	if err := oldSettings.Save(); err != nil {
		t.Fatalf("oldSettings.Save: %v", err)
	}

	s, err := NewSprite("Eye", 2, 2, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	f, err := AddFrame(s)
	if err != nil {
		t.Fatalf("AddFrame: %v", err)
	}
	path := filepath.Join(framesDir(s), f.ID+".px")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before remap: %v", err)
	}

	// New palette keeps black at char '0' first, unrelated colors after —
	// the blank frame (all '.') has nothing to rewrite.
	if _, err := RemapPalette("new", []string{"#000000", "#123456"}); err != nil {
		t.Fatalf("RemapPalette: %v", err)
	}

	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after remap: %v", err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("frame file was rewritten despite having no non-'.' pixels to remap")
	}
}
