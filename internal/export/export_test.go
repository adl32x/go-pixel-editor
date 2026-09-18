package export

import (
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
		f, err := sprite.AddFrame(s)
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

	if _, err := sprite.NewClip(&s, "blink", sprite.LoopForward, 4); err != nil {
		t.Fatalf("NewClip: %v", err)
	}
	entries := []sprite.ClipEntry{{FrameID: frames[0].ID}, {FrameID: frames[1].ID}}
	if _, err := sprite.UpdateClip(&s, "blink", sprite.ClipPatch{Entries: &entries}); err != nil {
		t.Fatalf("UpdateClip: %v", err)
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
	bundle, err := f.Export(s, frames, &s.Clips[0], settings)
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
	bundle, err := f.Export(s, frames, &s.Clips[0], settings)
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
