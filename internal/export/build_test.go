package export

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

func TestSheetGridLayout(t *testing.T) {
	s, _, settings := testSprite(t) // rows: default [f001 f002], blink [f003]
	if _, err := sprite.AddAnimation(&s, "empty"); err != nil {
		t.Fatalf("AddAnimation: %v", err)
	}
	frames, err := sprite.LoadFrames(s)
	if err != nil {
		t.Fatalf("LoadFrames: %v", err)
	}

	pngBytes, jsonBytes, err := SheetGrid(s, frames, settings, "eye.png")
	if err != nil {
		t.Fatalf("SheetGrid: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("png: %v", err)
	}
	// 2x2 frames; longest row has 2 frames; 2 non-empty rows.
	if b := img.Bounds(); b.Dx() != 4 || b.Dy() != 4 {
		t.Fatalf("sheet = %dx%d, want 4x4", b.Dx(), b.Dy())
	}

	var data gridSheetJSON
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		t.Fatalf("json: %v", err)
	}
	if data.Image != "eye.png" || data.FrameWidth != 2 || data.FrameHeight != 2 {
		t.Fatalf("header = %+v", data)
	}
	d := sprite.DefaultFrameDurationMS
	want := map[string][]gridFrameJSON{
		"default": {{0, 0, 2, 2, d}, {2, 0, 2, 2, d}},
		"blink":   {{0, 2, 2, 2, d}},
		"empty":   {},
	}
	for name, frames := range want {
		got := data.Animations[name]
		if len(got) != len(frames) {
			t.Fatalf("%s = %+v, want %+v", name, got, frames)
		}
		for i := range frames {
			if got[i] != frames[i] {
				t.Fatalf("%s[%d] = %+v, want %+v", name, i, got[i], frames[i])
			}
		}
	}
}

func TestBuildIsIdempotentAndCleansUpOnlyItsOwnFiles(t *testing.T) {
	s, _, settings := testSprite(t)
	other, err := sprite.NewSprite("Other", 2, 2, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	if _, err := sprite.AddFrame(&other, ""); err != nil {
		t.Fatalf("AddFrame: %v", err)
	}
	out := "assets/sprites"
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "hand-made.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	all, _ := sprite.Load()
	res, err := Build(all, out, settings, true)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(res.Written) != 4 {
		t.Fatalf("first build wrote %v, want 4 files", res.Written)
	}
	first, _ := os.ReadFile(filepath.Join(out, "eye.json"))

	res, err = Build(all, out, settings, true)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if len(res.Written) != 0 || len(res.Unchanged) != 4 {
		t.Fatalf("rebuild wrote %v / unchanged %v, want nothing rewritten", res.Written, res.Unchanged)
	}
	second, _ := os.ReadFile(filepath.Join(out, "eye.json"))
	if !bytes.Equal(first, second) {
		t.Fatal("rebuild changed eye.json")
	}

	// A partial build must not delete other sprites' outputs.
	if _, err := Build([]sprite.Sprite{s}, out, settings, false); err != nil {
		t.Fatalf("partial build: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "other.png")); err != nil {
		t.Fatal("partial build removed other.png")
	}

	// Deleting a sprite and doing a full build removes its outputs, but
	// never a file the build didn't write.
	if err := sprite.DeleteSprite(other.ID); err != nil {
		t.Fatalf("DeleteSprite: %v", err)
	}
	all, _ = sprite.Load()
	res, err = Build(all, out, settings, true)
	if err != nil {
		t.Fatalf("Build after delete: %v", err)
	}
	if len(res.Removed) != 2 {
		t.Fatalf("removed %v, want other.png + other.json", res.Removed)
	}
	for _, name := range []string{"other.png", "other.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still exists", name)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "hand-made.png")); err != nil {
		t.Fatal("build deleted a file it didn't create")
	}
}

func TestBuildRejectsSlugCollision(t *testing.T) {
	_, _, settings := testSprite(t)
	dup, err := sprite.NewSprite("eye", 2, 2, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	if _, err := sprite.AddFrame(&dup, ""); err != nil {
		t.Fatalf("AddFrame: %v", err)
	}
	all, _ := sprite.Load()
	if _, err := Build(all, "out", settings, true); err == nil {
		t.Fatal("expected an error for two sprites with the same slug")
	}
}

func TestBuildIntoSpriteSubfolders(t *testing.T) {
	_, _, settings := testSprite(t)
	m, err := sprite.NewSprite("mossling_30", 2, 2, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	if _, err := sprite.AddFrame(&m, ""); err != nil {
		t.Fatalf("AddFrame: %v", err)
	}
	out := "monsters_gen/"
	if _, err := sprite.UpdateSprite(m.ID, sprite.SpritePatch{Out: &out}); err != nil {
		t.Fatalf("UpdateSprite: %v", err)
	}

	all, _ := sprite.Load()
	res, err := Build(all, "assets/images", settings, true)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []string{"eye.json", "eye.png", "monsters_gen/mossling_30.json", "monsters_gen/mossling_30.png"}
	if len(res.Written) != len(want) {
		t.Fatalf("written = %v, want %v", res.Written, want)
	}
	for i := range want {
		if res.Written[i] != want[i] {
			t.Fatalf("written = %v, want %v", res.Written, want)
		}
	}
	var data gridSheetJSON
	raw, _ := os.ReadFile("assets/images/monsters_gen/mossling_30.json")
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("json: %v", err)
	}
	if data.Format != GridFormatID || data.Image != "mossling_30.png" {
		t.Fatalf("format/image = %q/%q", data.Format, data.Image)
	}

	// Moving the sprite to another subfolder removes the old outputs.
	root := ""
	if _, err := sprite.UpdateSprite(m.ID, sprite.SpritePatch{Out: &root}); err != nil {
		t.Fatalf("UpdateSprite: %v", err)
	}
	all, _ = sprite.Load()
	res, err = Build(all, "assets/images", settings, true)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(res.Removed) != 2 {
		t.Fatalf("removed = %v, want the two monsters_gen/ files", res.Removed)
	}
}

func TestBuildOverlayLayers(t *testing.T) {
	_, _, settings := testSprite(t) // "eye" is unused here
	guy, err := sprite.NewSprite("guy", 2, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	f, _ := sprite.AddFrame(&guy, "")
	if _, err := sprite.AddLayer(&guy, "Sword"); err != nil {
		t.Fatal(err)
	}
	sword := guy.Layers[0]
	sword.Overlay = true
	sword.Visible = false // hidden while editing must not drop it from the build
	layers := []sprite.LayerDef{sword, guy.Layers[1]}
	if _, err := sprite.UpdateSprite(guy.ID, sprite.SpritePatch{Layers: &layers}); err != nil {
		t.Fatal(err)
	}
	reloaded, _ := sprite.Find(guy.ID)
	fr, _ := sprite.FindFrame(*reloaded, f.ID)
	body := string(settings.Palette[0].Char)
	blade := string(settings.Palette[1].Char)
	fr.Layers[reloaded.Layers[1].ID] = [][]rune{[]rune(body + "."), []rune("..")} // body: top-left
	fr.Layers[sword.ID] = [][]rune{[]rune(".."), []rune("." + blade)}          // sword: bottom-right
	if err := fr.Save(*reloaded); err != nil {
		t.Fatal(err)
	}

	res, err := Build([]sprite.Sprite{*reloaded}, "out", settings, true)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []string{"guy.json", "guy.png", "guy.sword.json", "guy.sword.png"}
	if !reflect.DeepEqual(res.Written, want) {
		t.Fatalf("written = %v, want %v", res.Written, want)
	}

	opaque := func(name string, x, y int) bool {
		raw, _ := os.ReadFile("out/" + name)
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, a := img.At(x, y).RGBA()
		return a != 0
	}
	if !opaque("guy.png", 0, 0) || opaque("guy.png", 1, 1) {
		t.Fatal("guy.png should hold the body but not the sword")
	}
	if opaque("guy.sword.png", 0, 0) || !opaque("guy.sword.png", 1, 1) {
		t.Fatal("guy.sword.png should hold only the sword")
	}

	var base, over gridSheetJSON
	rawBase, _ := os.ReadFile("out/guy.json")
	rawOver, _ := os.ReadFile("out/guy.sword.json")
	_ = json.Unmarshal(rawBase, &base)
	_ = json.Unmarshal(rawOver, &over)
	if !reflect.DeepEqual(base.Overlays, []string{"sword"}) || base.OverlayOf != "" {
		t.Fatalf("base meta = overlays %v overlayOf %q", base.Overlays, base.OverlayOf)
	}
	if over.OverlayOf != "guy" || over.Layer != "sword" || over.Image != "guy.sword.png" {
		t.Fatalf("overlay meta = %+v", over)
	}
	if !reflect.DeepEqual(base.Animations, over.Animations) {
		t.Fatalf("overlay frames differ from the base's:\n%v\n%v", base.Animations, over.Animations)
	}
}

func TestBuildRejectsClashingOverlayKeys(t *testing.T) {
	_, _, settings := testSprite(t)
	s, _ := sprite.NewSprite("guy", 2, 2, "")
	_, _ = sprite.AddFrame(&s, "")
	_, _ = sprite.AddLayer(&s, "Big Axe")
	_, _ = sprite.AddLayer(&s, "big-axe")
	layers := append([]sprite.LayerDef(nil), s.Layers...)
	layers[0].Overlay, layers[1].Overlay = true, true
	if _, err := sprite.UpdateSprite(s.ID, sprite.SpritePatch{Layers: &layers}); err != nil {
		t.Fatal(err)
	}
	reloaded, _ := sprite.Find(s.ID)
	if _, err := Build([]sprite.Sprite{*reloaded}, "out", settings, true); err == nil {
		t.Fatal("expected an error for two overlays that both build as big_axe")
	}
}
