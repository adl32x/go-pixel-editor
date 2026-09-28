package sprite

import (
	"strings"
	"testing"
)

// resizeFixture is a 4x4 sprite with one frame whose pixels spell out
// their position, so a test can see exactly what moved where.
func resizeFixture(t *testing.T) (Sprite, Frame) {
	t.Helper()
	s, _, _ := setupFourByFour(t)
	f, err := AddFrame(&s, "")
	if err != nil {
		t.Fatal(err)
	}
	f.Layers["L1"] = [][]rune{
		[]rune("0..1"),
		[]rune("...."),
		[]rune("...."),
		[]rune("1..0"),
	}
	if err := f.Save(s); err != nil {
		t.Fatal(err)
	}
	return s, f
}

func rowsOf(t *testing.T, s Sprite, id string) string {
	t.Helper()
	f, err := FindFrame(s, id)
	if err != nil || f == nil {
		t.Fatalf("FindFrame: %v", err)
	}
	var rows []string
	for _, r := range f.Layers["L1"] {
		rows = append(rows, string(r))
	}
	return strings.Join(rows, "|")
}

func TestResizeEnlargePadsWithTransparent(t *testing.T) {
	s, f := resizeFixture(t)
	if err := ResizeSprite(&s, 6, 5, "top-left"); err != nil {
		t.Fatalf("ResizeSprite: %v", err)
	}
	reloaded, _ := Find(s.ID)
	if reloaded.Width != 6 || reloaded.Height != 5 {
		t.Fatalf("size = %dx%d, want 6x5", reloaded.Width, reloaded.Height)
	}
	want := "0..1..|......|......|1..0..|......"
	if got := rowsOf(t, *reloaded, f.ID); got != want {
		t.Fatalf("rows = %s, want %s", got, want)
	}
}

func TestResizeEnlargeCentered(t *testing.T) {
	s, f := resizeFixture(t)
	if err := ResizeSprite(&s, 6, 6, "center"); err != nil {
		t.Fatalf("ResizeSprite: %v", err)
	}
	want := "......|.0..1.|......|......|.1..0.|......"
	if got := rowsOf(t, s, f.ID); got != want {
		t.Fatalf("rows = %s, want %s", got, want)
	}
}

func TestResizeShrinkCrops(t *testing.T) {
	s, f := resizeFixture(t)
	// Keep the bottom-right: the top row and left column are cut off.
	if err := ResizeSprite(&s, 3, 3, "bottom-right"); err != nil {
		t.Fatalf("ResizeSprite: %v", err)
	}
	want := "...|...|..0"
	if got := rowsOf(t, s, f.ID); got != want {
		t.Fatalf("rows = %s, want %s", got, want)
	}
}

func TestResizeEveryLayerAndFrame(t *testing.T) {
	s, _ := resizeFixture(t)
	if _, err := AddLayer(&s, "top"); err != nil {
		t.Fatal(err)
	}
	f2, _ := AddFrame(&s, "")
	if err := ResizeSprite(&s, 2, 8, "top"); err != nil {
		t.Fatalf("ResizeSprite: %v", err)
	}
	frames, err := LoadFrames(s) // fails if any layer of any frame has the wrong shape
	if err != nil {
		t.Fatalf("LoadFrames after resize: %v", err)
	}
	if len(frames) != 2 || frames[1].ID != f2.ID {
		t.Fatalf("frames = %d", len(frames))
	}
}

func TestResizeRejectsBadInput(t *testing.T) {
	s, _ := resizeFixture(t)
	for _, c := range []struct {
		w, h   int
		anchor string
	}{{0, 4, ""}, {4, -1, ""}, {MaxCanvasSize + 1, 4, ""}, {4, 4, "middle-ish"}} {
		if err := ResizeSprite(&s, c.w, c.h, c.anchor); err == nil {
			t.Errorf("ResizeSprite(%d, %d, %q): expected an error", c.w, c.h, c.anchor)
		}
	}
	if s.Width != 4 || s.Height != 4 {
		t.Fatalf("a rejected resize changed the sprite to %dx%d", s.Width, s.Height)
	}
}
