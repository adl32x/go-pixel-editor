package sprite

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestNewSpriteLoadFind(t *testing.T) {
	t.Chdir(t.TempDir())

	s, err := NewSprite("Eye", 4, 4, "ui, eye")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	if s.ID != "0001" {
		t.Fatalf("ID = %q, want 0001", s.ID)
	}
	if len(s.Layers) != 1 || s.Layers[0].ID != "L1" {
		t.Fatalf("expected one default layer L1, got %+v", s.Layers)
	}

	s2, err := NewSprite("Cloud", 8, 8, "")
	if err != nil {
		t.Fatalf("NewSprite #2: %v", err)
	}
	if s2.ID != "0002" {
		t.Fatalf("ID = %q, want 0002 (never reused/renumbered)", s2.ID)
	}

	all, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("Load: got %d sprites, want 2", len(all))
	}

	found, err := Find("1")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found == nil || found.Name != "Eye" {
		t.Fatalf("Find(1) = %+v, want Eye", found)
	}
	if len(found.Tags) != 2 || found.Tags[0] != "ui" || found.Tags[1] != "eye" {
		t.Fatalf("Tags = %v, want [ui eye]", found.Tags)
	}

	missing, err := Find("9999")
	if err != nil {
		t.Fatalf("Find(9999): %v", err)
	}
	if missing != nil {
		t.Fatalf("Find(9999) = %+v, want nil", missing)
	}
}

func TestUpdateSpriteReslugsOnRename(t *testing.T) {
	t.Chdir(t.TempDir())

	s, err := NewSprite("Eye", 4, 4, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	oldDir := s.Path

	newName := "Cat Eye"
	updated, err := UpdateSprite(s.ID, SpritePatch{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateSprite: %v", err)
	}
	if updated.ID != s.ID {
		t.Fatalf("ID changed on rename: got %q, want %q", updated.ID, s.ID)
	}
	if updated.Path == oldDir {
		t.Fatalf("expected directory to be reslugged, path unchanged: %q", updated.Path)
	}

	found, err := Find(s.ID)
	if err != nil {
		t.Fatalf("Find after rename: %v", err)
	}
	if found == nil || found.Name != newName {
		t.Fatalf("Find after rename = %+v, want Name %q", found, newName)
	}
}

// TestSettingsColorToCharExactAndNearest covers the project's single fixed
// palette (see settings.go): an exact color match resolves to its char
// without changing the palette, and an out-of-palette color snaps to the
// nearest entry by RGB distance rather than allocating a new slot or
// erroring — a real behavior change from the old per-sprite append-only
// scheme, where a brand new color always grew the palette.
func TestSettingsColorToCharExactAndNearest(t *testing.T) {
	settings := Settings{
		Palette: []PaletteEntry{
			{Char: "0", Color: "#000000"},
			{Char: "1", Color: "#ff0000"}, // pure red
			{Char: "2", Color: "#0000ff"}, // pure blue
		},
	}

	ch, err := settings.ColorToChar("#0000ff")
	if err != nil {
		t.Fatalf("exact match: %v", err)
	}
	if ch != '2' {
		t.Fatalf("exact match char = %q, want '2'", ch)
	}

	// #e00000 (224,0,0) is much closer to pure red #ff0000 (distance 31²)
	// than to black (distance 224²) or blue.
	ch, err = settings.ColorToChar("#e00000")
	if err != nil {
		t.Fatalf("nearest match: %v", err)
	}
	if ch != '1' {
		t.Fatalf("nearest match char = %q, want '1' (red)", ch)
	}
	if len(settings.Palette) != 3 {
		t.Fatalf("palette len = %d, want unchanged 3 (snapping must not allocate)", len(settings.Palette))
	}
}

// TestPaletteEntriesFromColorsTruncatesAt62 guards the interaction with
// #0018: a preset with more colors than the single-char alphabet can
// address (e.g. a 256-color VGA palette) must be truncated, not overflow
// paletteAlphabet or panic.
func TestPaletteEntriesFromColorsTruncatesAt62(t *testing.T) {
	colors := make([]string, 100)
	for i := range colors {
		colors[i] = fmt.Sprintf("#%06x", i+1)
	}
	entries := paletteEntriesFromColors(colors)
	if len(entries) != len(paletteAlphabet) {
		t.Fatalf("truncated palette len = %d, want %d", len(entries), len(paletteAlphabet))
	}
	if entries[0].Color != colors[0] || entries[0].Char != "0" {
		t.Fatalf("first entry = %+v, want char '0' color %s", entries[0], colors[0])
	}
}

// TestLoadSettingsDefaultsToFirstPreset covers the on-disk default-creation
// path: a fresh project with no pixel.settings.md yet gets one created from
// the first registered palette preset.
func TestLoadSettingsDefaultsToFirstPreset(t *testing.T) {
	t.Chdir(t.TempDir())

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if settings.ActivePreset == "" {
		t.Fatal("expected a non-empty default ActivePreset")
	}
	if len(settings.Palette) == 0 {
		t.Fatal("expected a non-empty default palette")
	}
	if _, err := os.Stat(SettingsPath); err != nil {
		t.Fatalf("expected %s to be created on first load: %v", SettingsPath, err)
	}

	// A second load must read the same file back, not recreate a new
	// default (which would be indistinguishable here, but reloading the
	// exact same preset id and colors confirms Save/parse round-trip).
	reloaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("second LoadSettings: %v", err)
	}
	if reloaded.ActivePreset != settings.ActivePreset {
		t.Fatalf("ActivePreset changed across reload: %q vs %q", reloaded.ActivePreset, settings.ActivePreset)
	}
	if len(reloaded.Palette) != len(settings.Palette) {
		t.Fatalf("palette len changed across reload: %d vs %d", len(reloaded.Palette), len(settings.Palette))
	}
}

// TestConcurrentSavesNeverProduceEmptyFile guards against a real bug: two
// concurrent HTTP handlers (e.g. a debounced frame PUT and a layer-panel
// PATCH) each call Sprite.Save() around the same time. A plain
// os.WriteFile opens with O_TRUNC and writes in a separate step, so a
// concurrent Find() can observe the file mid-write, truncated to zero
// bytes, and fail with "empty file". Save must write atomically (temp
// file + rename) so a reader only ever sees a complete file.
func TestConcurrentSavesNeverProduceEmptyFile(t *testing.T) {
	t.Chdir(t.TempDir())
	s, err := NewSprite("Race", 8, 8, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			got, err := Find(s.ID)
			if err != nil {
				t.Errorf("Find during concurrent Save: %v", err)
				return
			}
			if got == nil {
				t.Errorf("Find returned nil sprite mid-write")
				return
			}
		}
	}()

	var writers sync.WaitGroup
	for i := 0; i < 4; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for j := 0; j < 200; j++ {
				if err := s.Save(); err != nil {
					t.Errorf("Save: %v", err)
					return
				}
			}
		}()
	}
	writers.Wait()
	close(stop)
	wg.Wait()
}
