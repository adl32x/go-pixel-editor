package sprite

import (
	"fmt"
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

func TestPaletteOverflow(t *testing.T) {
	t.Chdir(t.TempDir())

	s, err := NewSprite("Palette Test", 2, 2, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}

	for i := 0; i < 62; i++ {
		if _, err := s.ColorToChar(fmt.Sprintf("#%06x", i+1)); err != nil {
			t.Fatalf("ColorToChar #%d: unexpected error: %v", i, err)
		}
	}
	if len(s.Palette) != 62 {
		t.Fatalf("palette len = %d, want 62", len(s.Palette))
	}

	// A color already in the palette must resolve to its existing char,
	// not consume a new slot.
	first, err := s.ColorToChar("#000001")
	if err != nil {
		t.Fatalf("re-lookup of existing color failed: %v", err)
	}
	if first != rune(paletteAlphabet[0]) {
		t.Fatalf("re-lookup char = %q, want %q", first, paletteAlphabet[0])
	}
	if len(s.Palette) != 62 {
		t.Fatalf("palette len after re-lookup = %d, want unchanged 62", len(s.Palette))
	}

	if _, err := s.ColorToChar("#ffffff"); err == nil {
		t.Fatalf("expected overflow error allocating a 63rd color, got nil")
	}
}
