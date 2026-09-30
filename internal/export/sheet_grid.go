package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// sheetGridFormat is the `pixel build` output (#0031) as a registered
// format: a sheet laid out like the editor's frame grid plus a small custom
// JSON, rather than the Aseprite schema of sheet-json.
type sheetGridFormat struct{}

func (sheetGridFormat) Name() string { return "sheet-grid" }

// Export renders anim alone (one sheet row) or, when anim is nil, every
// animation row.
func (sheetGridFormat) Export(s sprite.Sprite, frames []sprite.Frame, anim *sprite.Animation, settings sprite.Settings) (Bundle, error) {
	if anim != nil {
		s.Animations = []sprite.Animation{*anim}
	}
	files, err := RenderSheets(s, frames, settings)
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Files: files}, nil
}

type gridFrameJSON struct {
	X          int `json:"x"`
	Y          int `json:"y"`
	W          int `json:"w"`
	H          int `json:"h"`
	DurationMS int `json:"durationMs"`
}

// GridFormatID identifies the sheet-grid JSON schema and its version, so a
// loader can tell it apart from other formats (e.g. Aseprite's) and later
// revisions of itself.
const GridFormatID = "pixel-sheet/1"

type gridSheetJSON struct {
	Format      string `json:"format"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	FrameWidth  int    `json:"frameWidth"`
	FrameHeight int    `json:"frameHeight"`
	// Overlays (on a sprite's own sheet) lists the visual keys of its
	// overlay layers, each built as <name>.<key>.png/.json.
	Overlays []string `json:"overlays,omitempty"`
	// OverlayOf and Layer (on an overlay's sheet) name the sprite sheet it
	// belongs on top of and the overlay's visual key.
	OverlayOf string `json:"overlayOf,omitempty"`
	Layer     string `json:"layer,omitempty"`
	// A map rather than a list so a game can look animations up by name;
	// encoding/json writes map keys sorted, which keeps output stable.
	Animations map[string][]gridFrameJSON `json:"animations"`
}

// sheetMeta is what distinguishes a sprite's own sheet from an overlay's.
type sheetMeta struct {
	overlays  []string
	overlayOf string
	layer     string
}

// RenderSheets renders s as the files `pixel build` writes, by base name:
// <slug>.png/.json from every non-overlay layer, plus <slug>.<key>.png/.json
// for each overlay layer (see sprite.LayerDef.Overlay). All sheets share one
// layout — same frames, same rects, same durations — so a game can draw an
// overlay's frame n exactly on top of the sprite's frame n.
//
// An overlay's sheet always contains its layer, even when it's hidden in
// the editor (hiding the sword while drawing the axe mustn't drop it from
// the build); hidden non-overlay layers stay out, as before.
func RenderSheets(s sprite.Sprite, frames []sprite.Frame, settings sprite.Settings) (map[string][]byte, error) {
	slug := s.Slug()
	base := s
	base.Layers = nil
	var overlays []sprite.LayerDef
	var keys []string
	seen := map[string]string{}
	for _, ld := range s.Layers {
		if !ld.Overlay {
			base.Layers = append(base.Layers, ld)
			continue
		}
		key := sprite.OverlayKey(ld.Name)
		if key == "" {
			return nil, fmt.Errorf("sprite %s: overlay layer %q needs a name with letters or digits", s.ID, ld.Name)
		}
		if other, ok := seen[key]; ok {
			return nil, fmt.Errorf("sprite %s: overlay layers %q and %q both build as %q; rename one", s.ID, other, ld.Name, key)
		}
		seen[key] = ld.Name
		overlays = append(overlays, ld)
		keys = append(keys, key)
	}

	files := map[string][]byte{}
	add := func(name string, sheet sprite.Sprite, meta sheetMeta) error {
		pngBytes, jsonBytes, err := sheetGrid(sheet, frames, settings, name+".png", meta)
		if err != nil {
			return err
		}
		files[name+".png"] = pngBytes
		files[name+".json"] = jsonBytes
		return nil
	}
	if err := add(slug, base, sheetMeta{overlays: keys}); err != nil {
		return nil, err
	}
	for i, ld := range overlays {
		ld.Visible = true
		layer := s
		layer.Layers = []sprite.LayerDef{ld}
		if err := add(slug+"."+keys[i], layer, sheetMeta{overlayOf: slug, layer: keys[i]}); err != nil {
			return nil, err
		}
	}
	return files, nil
}

// SheetGrid renders s as one PNG — one row per non-empty animation, frames
// left to right, as wide as the longest row — plus JSON describing each
// animation's frame rects and hold durations. imageName is what the JSON's
// "image" field points at. The output is deterministic: the same sprite
// always yields byte-identical files.
func SheetGrid(s sprite.Sprite, frames []sprite.Frame, settings sprite.Settings, imageName string) ([]byte, []byte, error) {
	return sheetGrid(s, frames, settings, imageName, sheetMeta{})
}

func sheetGrid(s sprite.Sprite, frames []sprite.Frame, settings sprite.Settings, imageName string, meta sheetMeta) ([]byte, []byte, error) {
	byID := make(map[string]sprite.Frame, len(frames))
	for _, f := range frames {
		byID[f.ID] = f
	}

	cols, rows := 0, 0
	for _, a := range s.Animations {
		if len(a.Frames) > 0 {
			rows++
			cols = max(cols, len(a.Frames))
		}
	}
	if rows == 0 {
		return nil, nil, fmt.Errorf("sprite %s has no frames", s.ID)
	}

	sheet := image.NewNRGBA(image.Rect(0, 0, cols*s.Width, rows*s.Height))
	data := gridSheetJSON{
		Format: GridFormatID, Name: s.Name, Image: imageName,
		FrameWidth: s.Width, FrameHeight: s.Height,
		Overlays: meta.overlays, OverlayOf: meta.overlayOf, Layer: meta.layer,
		Animations: map[string][]gridFrameJSON{},
	}

	row := 0
	for _, a := range s.Animations {
		out := []gridFrameJSON{}
		for col, af := range a.Frames {
			f, ok := byID[af.FrameID]
			if !ok {
				return nil, nil, fmt.Errorf("animation %q references unknown frame %s", a.Name, af.FrameID)
			}
			img, err := compositeFrame(s, f, settings)
			if err != nil {
				return nil, nil, err
			}
			x, y := col*s.Width, row*s.Height
			draw.Draw(sheet, image.Rect(x, y, x+s.Width, y+s.Height), img, image.Point{}, draw.Src)
			out = append(out, gridFrameJSON{
				X: x, Y: y, W: s.Width, H: s.Height,
				DurationMS: int(s.FrameDuration(af).Milliseconds()),
			})
		}
		data.Animations[a.Name] = out
		if len(a.Frames) > 0 {
			row++
		}
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, sheet); err != nil {
		return nil, nil, err
	}
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return pngBuf.Bytes(), append(jsonBytes, '\n'), nil
}
