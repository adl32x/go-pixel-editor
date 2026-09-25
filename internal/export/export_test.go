package export

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

func testSprite(t *testing.T) (sprite.Sprite, []sprite.Frame, sprite.Settings) {
	t.Helper()
	t.Chdir(t.TempDir())

	settings, err := sprite.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}

	s, err := sprite.NewSprite("Eye", 2, 2, "")
	if err != nil {
		t.Fatalf("NewSprite: %v", err)
	}
	var frames []sprite.Frame
	for i := 0; i < 2; i++ {
		f, err := sprite.AddFrame(&s, "")
		if err != nil {
			t.Fatalf("AddFrame: %v", err)
		}
		layerProps, err := f.ToLayerProps(s, settings)
		if err != nil {
			t.Fatalf("ToLayerProps: %v", err)
		}
		layerProps[0].Data[0][0].Color = settings.Palette[0].Color
		nf, err := sprite.FrameFromLayerProps(s, f.ID, layerProps, settings)
		if err != nil {
			t.Fatalf("FrameFromLayerProps: %v", err)
		}
		if err := nf.Save(s); err != nil {
			t.Fatalf("Frame.Save: %v", err)
		}
		frames = append(frames, nf)
	}

	if _, err := sprite.AddAnimation(&s, "blink"); err != nil {
		t.Fatalf("AddAnimation: %v", err)
	}
	if _, err := sprite.AddFrame(&s, "blink"); err != nil {
		t.Fatalf("AddFrame: %v", err)
	}

	reloaded, err := sprite.Find(s.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	return *reloaded, frames, settings
}

func TestGIFExport(t *testing.T) {
	s, frames, settings := testSprite(t)
	f, ok := Get("gif")
	if !ok {
		t.Fatal("gif format not registered")
	}
	bundle, err := f.Export(s, frames, &s.Animations[0], settings)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	data, ct, err := bundle.Payload()
	if err != nil {
		t.Fatalf("Payload: %v", err)
	}
	if ct != "image/gif" {
		t.Fatalf("content type = %q, want image/gif", ct)
	}
	if len(data) < 6 || string(data[:3]) != "GIF" {
		t.Fatalf("output doesn't look like a GIF: %q...", data[:min(len(data), 16)])
	}
}

func TestSheetJSONExport(t *testing.T) {
	s, frames, settings := testSprite(t)
	f, ok := Get("sheet-json")
	if !ok {
		t.Fatal("sheet-json format not registered")
	}
	bundle, err := f.Export(s, frames, &s.Animations[0], settings)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if _, ok := bundle.Files["sheet.png"]; !ok {
		t.Fatal("bundle missing sheet.png")
	}
	if _, ok := bundle.Files["data.json"]; !ok {
		t.Fatal("bundle missing data.json")
	}
	data, ct, err := bundle.Payload()
	if err != nil {
		t.Fatalf("Payload: %v", err)
	}
	if ct != "application/zip" {
		t.Fatalf("content type = %q, want application/zip", ct)
	}
	if len(data) < 4 || string(data[:2]) != "PK" {
		t.Fatalf("output doesn't look like a zip: %q...", data[:min(len(data), 16)])
	}
}

func TestSheetJSONTagsEveryRowInGridOrder(t *testing.T) {
	s, frames, settings := testSprite(t)
	all, err := sprite.LoadFrames(s)
	if err != nil {
		t.Fatalf("LoadFrames: %v", err)
	}
	if len(all) != len(frames)+1 {
		t.Fatalf("frames on disk = %d, want %d", len(all), len(frames)+1)
	}
	f, _ := Get("sheet-json")
	bundle, err := f.Export(s, all, nil, settings)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	var data sheetDataJSON
	if err := json.Unmarshal(bundle.Files["data.json"], &data); err != nil {
		t.Fatalf("data.json: %v", err)
	}
	want := []frameTagJSON{
		{Name: "default", From: 0, To: 1, Direction: "forward"},
		{Name: "blink", From: 2, To: 2, Direction: "forward"},
	}
	if !reflect.DeepEqual(data.Meta.FrameTags, want) {
		t.Fatalf("frameTags = %+v, want %+v", data.Meta.FrameTags, want)
	}
	if d := data.Frames["f001"].Duration; d != sprite.DefaultFrameDurationMS {
		t.Fatalf("f001 duration = %d, want sprite default %d", d, sprite.DefaultFrameDurationMS)
	}
}
