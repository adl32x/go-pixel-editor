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
	slug := s.Slug()
	pngBytes, jsonBytes, err := SheetGrid(s, frames, settings, slug+".png")
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Files: map[string][]byte{slug + ".png": pngBytes, slug + ".json": jsonBytes}}, nil
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
	// A map rather than a list so a game can look animations up by name;
	// encoding/json writes map keys sorted, which keeps output stable.
	Animations map[string][]gridFrameJSON `json:"animations"`
}

// SheetGrid renders s as one PNG — one row per non-empty animation, frames
// left to right, as wide as the longest row — plus JSON describing each
// animation's frame rects and hold durations. imageName is what the JSON's
// "image" field points at. The output is deterministic: the same sprite
// always yields byte-identical files.
func SheetGrid(s sprite.Sprite, frames []sprite.Frame, settings sprite.Settings, imageName string) ([]byte, []byte, error) {
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
