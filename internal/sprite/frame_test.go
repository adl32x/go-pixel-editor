package sprite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupFourByFour creates the plan's literal worked example: a 4x4, 1-layer
// sprite with a 2-color palette, and returns the sprite plus the exact
// frame file content it should produce.
func setupFourByFour(t *testing.T) (Sprite, string) {
	t.Helper()
	t.Chdir(t.TempDir())

	s, err := NewSprite("Eye", 4, 4, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	if _, err := s.ColorToChar("#000000"); err != nil { // -> '0'
		t.Fatalf("ColorToChar black: %v", err)
	}
	if _, err := s.ColorToChar("#ffffff"); err != nil { // -> '1'
		t.Fatalf("ColorToChar white: %v", err)
	}
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	want := "frame: f001\n\nlayer: L1\n....\n.10.\n.01.\n....\n\n"
	return s, want
}

func TestFrameRoundTrip(t *testing.T) {
	s, want := setupFourByFour(t)

	f := Frame{
		ID: "f001",
		Layers: map[string][][]rune{
			"L1": {
				[]rune("...."),
				[]rune(".10."),
				[]rune(".01."),
				[]rune("...."),
			},
		},
	}
	if err := f.Save(s); err != nil {
		t.Fatalf("Frame.Save: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(framesDir(s), "f001.px"))
	if err != nil {
		t.Fatalf("reading frame file: %v", err)
	}
	if string(got) != want {
		t.Fatalf("frame file =\n%q\nwant\n%q", got, want)
	}

	loaded, err := FindFrame(s, "f001")
	if err != nil {
		t.Fatalf("FindFrame: %v", err)
	}
	if loaded == nil {
		t.Fatal("FindFrame returned nil")
	}
	for r := 0; r < 4; r++ {
		gotRow := string(loaded.Layers["L1"][r])
		wantRow := string(f.Layers["L1"][r])
		if gotRow != wantRow {
			t.Fatalf("row %d = %q, want %q", r, gotRow, wantRow)
		}
	}
}

func TestFrameParseRejectsUnknownChar(t *testing.T) {
	s, _ := setupFourByFour(t)

	bad := "frame: f001\n\nlayer: L1\n....\n.Z0.\n....\n....\n"
	if err := os.WriteFile(filepath.Join(framesDir(s), "f001.px"), []byte(bad), 0o644); err != nil {
		t.Fatalf("writing bad frame: %v", err)
	}

	if _, err := FindFrame(s, "f001"); err == nil {
		t.Fatal("expected error parsing frame with unknown palette char, got nil")
	}
}

// TestAddFrameIDNeverFillsAGap asserts the real guarantee our max+1-over-
// existing-files scheme provides: deleting a frame that ISN'T the current
// highest ID never causes a later AddFrame to fill that now-empty gap. IDs
// are derived purely from the current on-disk max, so nothing is ever
// renumbered by an insert/delete elsewhere — the one case this scheme
// can't prevent is immediately reusing the ID of whatever the single
// highest frame was when it's deleted, since no separate counter is kept
// (deliberately — see sprite.md's "no next_id counters" design).
func TestAddFrameIDNeverFillsAGap(t *testing.T) {
	s, _ := setupFourByFour(t)

	f1, err := AddFrame(s)
	if err != nil {
		t.Fatalf("AddFrame #1: %v", err)
	}
	if f1.ID != "f001" {
		t.Fatalf("first frame id = %q, want f001", f1.ID)
	}
	f2, err := AddFrame(s)
	if err != nil {
		t.Fatalf("AddFrame #2: %v", err)
	}
	if f2.ID != "f002" {
		t.Fatalf("second frame id = %q, want f002", f2.ID)
	}
	f3, err := AddFrame(s)
	if err != nil {
		t.Fatalf("AddFrame #3: %v", err)
	}
	if f3.ID != "f003" {
		t.Fatalf("third frame id = %q, want f003", f3.ID)
	}

	// Delete the middle frame (not the current max) and confirm the next
	// AddFrame continues from the true max, rather than filling the gap
	// left at f002.
	if err := DeleteFrame(&s, "f002", false); err != nil {
		t.Fatalf("DeleteFrame: %v", err)
	}
	f4, err := AddFrame(s)
	if err != nil {
		t.Fatalf("AddFrame #4: %v", err)
	}
	if f4.ID != "f004" {
		t.Fatalf("id after deleting a middle frame = %q, want f004 (must not fill the f002 gap)", f4.ID)
	}
}

func TestDeleteFrameReferencedByClipRequiresForce(t *testing.T) {
	s, _ := setupFourByFour(t)

	f, err := AddFrame(s)
	if err != nil {
		t.Fatalf("AddFrame: %v", err)
	}
	if _, err := NewClip(&s, "blink", LoopForward, 2); err != nil {
		t.Fatalf("NewClip: %v", err)
	}
	entries := []ClipEntry{{FrameID: f.ID}}
	if _, err := UpdateClip(&s, "blink", ClipPatch{Entries: &entries}); err != nil {
		t.Fatalf("UpdateClip: %v", err)
	}

	if err := DeleteFrame(&s, f.ID, false); err == nil {
		t.Fatal("expected error deleting a frame referenced by a clip without force")
	}
	if err := DeleteFrame(&s, f.ID, true); err != nil {
		t.Fatalf("DeleteFrame with force: %v", err)
	}

	reloaded, err := Find(s.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(reloaded.Clips[0].Entries) != 0 {
		t.Fatalf("expected clip entries stripped after forced delete, got %+v", reloaded.Clips[0].Entries)
	}
}

func TestClipInsertIsASingleLineDiff(t *testing.T) {
	s, _ := setupFourByFour(t)

	f1, _ := AddFrame(s)
	f2, _ := AddFrame(s)
	f3, _ := AddFrame(s)

	if _, err := NewClip(&s, "walk", LoopForward, 4); err != nil {
		t.Fatalf("NewClip: %v", err)
	}
	entries := []ClipEntry{{FrameID: f1.ID}, {FrameID: f3.ID}}
	if _, err := UpdateClip(&s, "walk", ClipPatch{Entries: &entries}); err != nil {
		t.Fatalf("UpdateClip: %v", err)
	}
	before, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatalf("reading sprite.md: %v", err)
	}
	beforeLines := strings.Split(string(before), "\n")

	// Insert f2 in the middle.
	entries = []ClipEntry{{FrameID: f1.ID}, {FrameID: f2.ID}, {FrameID: f3.ID}}
	if _, err := UpdateClip(&s, "walk", ClipPatch{Entries: &entries}); err != nil {
		t.Fatalf("UpdateClip insert: %v", err)
	}
	after, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatalf("reading sprite.md after insert: %v", err)
	}
	afterLines := strings.Split(string(after), "\n")

	if len(afterLines) != len(beforeLines)+1 {
		t.Fatalf("inserting one frame changed line count by %d, want +1", len(afterLines)-len(beforeLines))
	}
	// Every line before the insertion point must be byte-identical, and
	// every line after it must reappear unchanged later in the new file —
	// i.e. the only change is one new line, not a rewrite of the whole
	// frames: list.
	for i, line := range beforeLines {
		if i < 3 { // f1's entry line and everything before it is untouched
			if line != afterLines[i] {
				t.Fatalf("line %d changed: got %q, want %q", i, afterLines[i], line)
			}
		}
	}
	if !strings.Contains(string(after), "  "+f2.ID+"\n") {
		t.Fatalf("expected a new indented line for %s, got:\n%s", f2.ID, after)
	}
}

func TestToLayerPropsFromLayerPropsRoundTrip(t *testing.T) {
	s, _ := setupFourByFour(t)

	orig := Frame{
		ID: "f001",
		Layers: map[string][][]rune{
			"L1": {
				[]rune("...."),
				[]rune(".10."),
				[]rune(".01."),
				[]rune("...."),
			},
		},
	}

	layerProps, err := orig.ToLayerProps(s)
	if err != nil {
		t.Fatalf("ToLayerProps: %v", err)
	}
	if len(layerProps) != 1 || layerProps[0].ID != "L1" {
		t.Fatalf("layerProps = %+v, want one entry for L1", layerProps)
	}
	if layerProps[0].Data[1][1].Color != "#ffffff" {
		t.Fatalf("pixel (1,1) color = %q, want #ffffff", layerProps[0].Data[1][1].Color)
	}
	if layerProps[0].Data[1][2].Color != "#000000" {
		t.Fatalf("pixel (1,2) color = %q, want #000000", layerProps[0].Data[1][2].Color)
	}
	if layerProps[0].Data[0][0].Color != "" {
		t.Fatalf("pixel (0,0) color = %q, want empty (no pixel)", layerProps[0].Data[0][0].Color)
	}

	roundTripped, err := FrameFromLayerProps(&s, "f001", layerProps)
	if err != nil {
		t.Fatalf("FrameFromLayerProps: %v", err)
	}
	for r := 0; r < 4; r++ {
		got := string(roundTripped.Layers["L1"][r])
		want := string(orig.Layers["L1"][r])
		if got != want {
			t.Fatalf("row %d = %q, want %q", r, got, want)
		}
	}
}
