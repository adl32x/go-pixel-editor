package importer

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/adl32x/go-pixel-editor/internal/export"
	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

var (
	black = color.NRGBA{0, 0, 0, 255}
	white = color.NRGBA{255, 255, 255, 255}
	gray  = color.NRGBA{120, 120, 120, 255} // not in the palette
)

// setup makes a project whose palette is black ('0') and white ('1'),
// building into assets/images.
func setup(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	settings := sprite.Settings{
		ActivePreset: "custom",
		BuildOut:     "assets/images",
		Palette: []sprite.PaletteEntry{
			{Char: "0", Color: "#000000"},
			{Char: "1", Color: "#ffffff"},
		},
	}
	if err := settings.Save(); err != nil {
		t.Fatal(err)
	}
}

// writeSheet writes a 4x4 image of 2x2 frames: row 0 = [black, white],
// row 1 = [white, empty], each frame filled solid.
func writeSheet(t *testing.T, p string) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	fill := func(x0, y0 int, c color.NRGBA) {
		for y := y0; y < y0+2; y++ {
			for x := x0; x < x0+2; x++ {
				img.SetNRGBA(x, y, c)
			}
		}
	}
	fill(0, 0, black)
	fill(2, 0, white)
	fill(0, 2, white)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func frameRows(t *testing.T, s sprite.Sprite, id string) []string {
	t.Helper()
	f, err := sprite.FindFrame(s, id)
	if err != nil || f == nil {
		t.Fatalf("FindFrame %s: %v", id, err)
	}
	var rows []string
	for _, r := range f.Layers[s.Layers[0].ID] {
		rows = append(rows, string(r))
	}
	return rows
}

func TestImportAsepriteArray(t *testing.T) {
	setup(t)
	writeSheet(t, "assets/images/characters/guy.png")
	json := `{ "frames": [
  { "filename": "guy (standing) 0.ase", "frame": { "x": 0, "y": 0, "w": 2, "h": 2 }, "duration": 1000 },
  { "filename": "guy (standing) 1.ase", "frame": { "x": 2, "y": 0, "w": 2, "h": 2 }, "duration": 1000 },
  { "filename": "guy (walking) 0.ase", "frame": { "x": 0, "y": 2, "w": 2, "h": 2 }, "duration": 250 }
 ], "meta": { "image": "guy.png" } }`
	if err := os.WriteFile("assets/images/characters/guy.json", []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Import("assets/images/characters/guy.json", Options{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	s, _ := sprite.Find(rep.Sprite.ID)
	if s.Name != "guy" || s.Out != "characters" || s.Width != 2 || s.Height != 2 {
		t.Fatalf("sprite = %s out=%q %dx%d", s.Name, s.Out, s.Width, s.Height)
	}
	if s.DurationMS != 1000 {
		t.Fatalf("default duration = %d, want the common 1000", s.DurationMS)
	}
	if len(s.Animations) != 2 || s.Animations[0].Name != "standing" || s.Animations[1].Name != "walking" {
		t.Fatalf("animations = %+v", s.Animations)
	}
	walk := s.Animations[1].Frames[0]
	if walk.DurationMS == nil || *walk.DurationMS != 250 {
		t.Fatalf("walking frame duration override = %v, want 250", walk.DurationMS)
	}
	if s.Animations[0].Frames[0].DurationMS != nil {
		t.Fatal("frames at the default duration shouldn't get an override")
	}
	if got := frameRows(t, *s, s.Animations[0].Frames[1].FrameID); got[0] != "11" {
		t.Fatalf("standing frame 2 = %v, want white", got)
	}
	if rep.InexactPixels != 0 {
		t.Fatalf("inexact = %d, want 0", rep.InexactPixels)
	}
}

func TestImportGridPNG(t *testing.T) {
	setup(t)
	writeSheet(t, "art/sheet.png")

	rep, err := Import("art/sheet.png", Options{Name: "tiles", FrameWidth: 2, FrameHeight: 2})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	s, _ := sprite.Find(rep.Sprite.ID)
	if s.Out != "" {
		t.Fatalf("out = %q, want none (art/ is outside build_out)", s.Out)
	}
	if len(s.Animations) != 2 || len(s.Animations[0].Frames) != 2 || len(s.Animations[1].Frames) != 1 {
		t.Fatalf("animations = %+v, want row1 x2, row2 x1 (trailing empty cell dropped)", s.Animations)
	}
	if got := frameRows(t, *s, s.Animations[0].Frames[0].FrameID); got[0] != "00" {
		t.Fatalf("row1 frame 1 = %v, want black", got)
	}
}

func TestImportSnapsOffPaletteColorsAndRejectsDuplicates(t *testing.T) {
	setup(t)
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, gray)
	f, _ := os.Create("one.png")
	_ = png.Encode(f, img)
	f.Close()

	rep, err := Import("one.png", Options{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rep.InexactPixels != 1 {
		t.Fatalf("inexact = %d, want 1", rep.InexactPixels)
	}
	if _, err := Import("one.png", Options{}); err == nil {
		t.Fatal("expected an error importing a second sprite with the same name")
	}
}

func TestFailedImportLeavesNoSprite(t *testing.T) {
	setup(t)
	writeSheet(t, "sheet.png")
	json := `{ "frames": [
  { "filename": "x (a) 0.ase", "frame": { "x": 0, "y": 0, "w": 2, "h": 2 }, "duration": 100 },
  { "filename": "x (a) 1.ase", "frame": { "x": 0, "y": 0, "w": 4, "h": 4 }, "duration": 100 }
 ], "meta": { "image": "sheet.png" } }`
	_ = os.WriteFile("sheet.json", []byte(json), 0o644)
	if _, err := Import("sheet.json", Options{}); err == nil {
		t.Fatal("expected an error for mixed frame sizes")
	}
	if all, _ := sprite.Load(); len(all) != 0 {
		t.Fatalf("left %d sprite(s) behind", len(all))
	}
}

func TestImportPixelSheetRoundTrip(t *testing.T) {
	setup(t)
	writeSheet(t, "src/sheet.png")
	orig, err := Import("src/sheet.png", Options{Name: "orig", FrameWidth: 2, FrameHeight: 2})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	// Name the rows so alphabetical order ("a" < "z") differs from sheet order.
	s, _ := sprite.Find(orig.Sprite.ID)
	for old, name := range map[string]string{"row1": "z_top", "row2": "a_bottom"} {
		n := name
		if _, err := sprite.UpdateAnimation(s, old, sprite.AnimationPatch{Name: &n}); err != nil {
			t.Fatal(err)
		}
	}
	all, _ := sprite.Load()
	settings, _ := sprite.LoadSettings()
	if _, err := export.Build(all, "assets/images", settings, true); err != nil {
		t.Fatalf("Build: %v", err)
	}

	rep, err := Import("assets/images/orig.json", Options{Name: "copy"})
	if err != nil {
		t.Fatalf("Import pixel-sheet: %v", err)
	}
	got, _ := sprite.Find(rep.Sprite.ID)
	if len(got.Animations) != 2 || got.Animations[0].Name != "z_top" || got.Animations[1].Name != "a_bottom" {
		t.Fatalf("animations = %+v, want z_top then a_bottom (sheet order)", got.Animations)
	}
	if got := frameRows(t, *got, got.Animations[0].Frames[1].FrameID); got[0] != "11" {
		t.Fatalf("z_top frame 2 = %v, want white", got)
	}
}
