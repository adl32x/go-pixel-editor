package sprite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupFourByFour creates the plan's literal worked example: a 4x4, 1-layer
// sprite plus a project-wide 2-color Settings palette (black='0',
// white='1'), and returns the sprite, settings, and the exact frame file
// content it should produce.
func setupFourByFour(t *testing.T) (Sprite, Settings, string) {
	t.Helper()
	t.Chdir(t.TempDir())

	s, err := NewSprite("Eye", 4, 4, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	settings := Settings{
		ActivePreset: "test",
		Palette: []PaletteEntry{
			{Char: "0", Color: "#000000"},
			{Char: "1", Color: "#ffffff"},
		},
	}
	if err := settings.Save(); err != nil {
		t.Fatalf("Settings.Save: %v", err)
	}

	want := "frame: f001\n\nlayer: L1\n....\n.10.\n.01.\n....\n\n"
	return s, settings, want
}

func TestFrameRoundTrip(t *testing.T) {
	s, _, want := setupFourByFour(t)

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
	s, _, _ := setupFourByFour(t)

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
	s, _, _ := setupFourByFour(t)

	f1, err := AddFrame(&s, "")
	if err != nil {
		t.Fatalf("AddFrame #1: %v", err)
	}
	if f1.ID != "f001" {
		t.Fatalf("first frame id = %q, want f001", f1.ID)
	}
	f2, err := AddFrame(&s, "")
	if err != nil {
		t.Fatalf("AddFrame #2: %v", err)
	}
	if f2.ID != "f002" {
		t.Fatalf("second frame id = %q, want f002", f2.ID)
	}
	f3, err := AddFrame(&s, "")
	if err != nil {
		t.Fatalf("AddFrame #3: %v", err)
	}
	if f3.ID != "f003" {
		t.Fatalf("third frame id = %q, want f003", f3.ID)
	}

	// Delete the middle frame (not the current max) and confirm the next
	// AddFrame continues from the true max, rather than filling the gap
	// left at f002.
	if err := DeleteFrame(&s, "f002"); err != nil {
		t.Fatalf("DeleteFrame: %v", err)
	}
	f4, err := AddFrame(&s, "")
	if err != nil {
		t.Fatalf("AddFrame #4: %v", err)
	}
	if f4.ID != "f004" {
		t.Fatalf("id after deleting a middle frame = %q, want f004 (must not fill the f002 gap)", f4.ID)
	}
}

func TestDeleteFrameRemovesItFromItsRow(t *testing.T) {
	s, _, _ := setupFourByFour(t)

	f1, _ := AddFrame(&s, "")
	f2, _ := AddFrame(&s, "")
	if err := DeleteFrame(&s, f1.ID); err != nil {
		t.Fatalf("DeleteFrame: %v", err)
	}

	reloaded, err := Find(s.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	got := reloaded.OrderedFrameIDs()
	if len(got) != 1 || got[0] != f2.ID {
		t.Fatalf("frames after delete = %v, want [%s]", got, f2.ID)
	}
}

func TestAnimationInsertIsASingleLineDiff(t *testing.T) {
	s, _, _ := setupFourByFour(t)

	f1, _ := AddFrame(&s, "")
	f2, _ := AddFrame(&s, "")
	f3, _ := AddFrame(&s, "")

	frames := []AnimFrame{{FrameID: f1.ID}, {FrameID: f3.ID}, {FrameID: f2.ID}}
	if _, err := UpdateAnimation(&s, "default", AnimationPatch{Frames: &frames}); err != nil {
		t.Fatalf("UpdateAnimation: %v", err)
	}
	before, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatalf("reading sprite.md: %v", err)
	}

	// Move f2 between f1 and f3: one line moves, nothing else is rewritten.
	frames = []AnimFrame{{FrameID: f1.ID}, {FrameID: f2.ID}, {FrameID: f3.ID}}
	if _, err := UpdateAnimation(&s, "default", AnimationPatch{Frames: &frames}); err != nil {
		t.Fatalf("UpdateAnimation reorder: %v", err)
	}
	after, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatalf("reading sprite.md after reorder: %v", err)
	}
	if len(strings.Split(string(after), "\n")) != len(strings.Split(string(before), "\n")) {
		t.Fatal("reordering changed the line count")
	}
	if !strings.Contains(string(after), "frames:\n  "+f1.ID+"\n  "+f2.ID+"\n  "+f3.ID+"\n") {
		t.Fatalf("expected one frame id per indented line, got:\n%s", after)
	}
}

func TestUpdateAnimationRejectsForeignFrames(t *testing.T) {
	s, _, _ := setupFourByFour(t)

	f1, _ := AddFrame(&s, "")
	if _, err := AddAnimation(&s, "walk"); err != nil {
		t.Fatalf("AddAnimation: %v", err)
	}
	f2, _ := AddFrame(&s, "walk")

	steal := []AnimFrame{{FrameID: f1.ID}, {FrameID: f2.ID}}
	if _, err := UpdateAnimation(&s, "default", AnimationPatch{Frames: &steal}); err == nil {
		t.Fatal("expected error pulling another row's frame into a row")
	}
	drop := []AnimFrame{}
	if _, err := UpdateAnimation(&s, "default", AnimationPatch{Frames: &drop}); err == nil {
		t.Fatal("expected error dropping a frame from a row via patch")
	}
	name := "walk"
	if _, err := UpdateAnimation(&s, "default", AnimationPatch{Name: &name}); err == nil {
		t.Fatal("expected error renaming a row to an existing name")
	}
}

func TestFrameDurationOverride(t *testing.T) {
	s, _, _ := setupFourByFour(t)

	f1, _ := AddFrame(&s, "")
	f2, _ := AddFrame(&s, "")
	d := 250
	if _, err := UpdateSprite(s.ID, SpritePatch{DurationMS: &d}); err != nil {
		t.Fatalf("UpdateSprite: %v", err)
	}
	reloaded, _ := Find(s.ID)
	override := 40
	frames := []AnimFrame{{FrameID: f1.ID}, {FrameID: f2.ID, DurationMS: &override}}
	if _, err := UpdateAnimation(reloaded, "default", AnimationPatch{Frames: &frames}); err != nil {
		t.Fatalf("UpdateAnimation: %v", err)
	}

	reloaded, _ = Find(s.ID)
	a := reloaded.Animations[0]
	if got := reloaded.FrameDuration(a.Frames[0]).Milliseconds(); got != 250 {
		t.Fatalf("f1 duration = %dms, want sprite default 250", got)
	}
	if got := reloaded.FrameDuration(a.Frames[1]).Milliseconds(); got != 40 {
		t.Fatalf("f2 duration = %dms, want override 40", got)
	}
}

func TestDeleteAnimationDeletesItsFrames(t *testing.T) {
	s, _, _ := setupFourByFour(t)

	keep, _ := AddFrame(&s, "")
	if _, err := AddAnimation(&s, "walk"); err != nil {
		t.Fatalf("AddAnimation: %v", err)
	}
	gone, _ := AddFrame(&s, "walk")
	if err := DeleteAnimation(&s, "walk"); err != nil {
		t.Fatalf("DeleteAnimation: %v", err)
	}
	if f, _ := FindFrame(s, gone.ID); f != nil {
		t.Fatalf("frame %s still on disk after deleting its row", gone.ID)
	}
	reloaded, _ := Find(s.ID)
	if ids := reloaded.OrderedFrameIDs(); len(ids) != 1 || ids[0] != keep.ID {
		t.Fatalf("frames = %v, want [%s]", ids, keep.ID)
	}
}

func TestLegacyClipsAreMigratedToRows(t *testing.T) {
	s, _, _ := setupFourByFour(t)
	f1, _ := AddFrame(&s, "")
	f2, _ := AddFrame(&s, "")
	f3, _ := AddFrame(&s, "")

	// Legacy format: clips could share frames, miss frames, and carry
	// loop/fps scalars. f3 is in no clip; f1 is in both.
	legacy := "---\nid: " + s.ID + "\nname: Eye\nwidth: 4\nheight: 4\ntags: \n---\n\n" +
		"## layers\nL1 base visible=true opacity=1\n\n" +
		"## clips\n### blink\nloop: pingpong\nfps: 4\nframes:\n  " + f2.ID + " @50\n  " + f1.ID + "\n" +
		"### look\nloop: forward\nfps: 12\nframes:\n  " + f1.ID + "\n  f099\n"
	if err := os.WriteFile(s.Path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("writing legacy sprite.md: %v", err)
	}

	got, err := Find(s.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.DurationMS != DefaultFrameDurationMS {
		t.Fatalf("duration = %d, want default %d", got.DurationMS, DefaultFrameDurationMS)
	}
	if len(got.Animations) != 2 {
		t.Fatalf("animations = %+v, want 2 rows", got.Animations)
	}
	blink := got.Animations[0]
	if len(blink.Frames) != 3 || blink.Frames[0].FrameID != f2.ID || *blink.Frames[0].DurationMS != 50 ||
		blink.Frames[1].FrameID != f1.ID || blink.Frames[2].FrameID != f3.ID {
		t.Fatalf("blink = %+v, want [%s@50 %s %s]", blink.Frames, f2.ID, f1.ID, f3.ID)
	}
	if look := got.Animations[1]; len(look.Frames) != 0 {
		t.Fatalf("look = %+v, want empty (f001 already in blink, f099 missing)", look.Frames)
	}
}

func TestToLayerPropsFromLayerPropsRoundTrip(t *testing.T) {
	s, settings, _ := setupFourByFour(t)

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

	layerProps, err := orig.ToLayerProps(s, settings)
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

	roundTripped, err := FrameFromLayerProps(s, "f001", layerProps, settings)
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
