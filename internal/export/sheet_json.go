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

type sheetJSONFormat struct{}

func (sheetJSONFormat) Name() string { return "sheet-json" }

type frameRectJSON struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type frameEntryJSON struct {
	Frame    frameRectJSON `json:"frame"`
	Duration int           `json:"duration"`
}

type frameTagJSON struct {
	Name      string `json:"name"`
	From      int    `json:"from"`
	To        int    `json:"to"`
	Direction string `json:"direction"`
}

type sheetSizeJSON struct {
	W int `json:"w"`
	H int `json:"h"`
}

type sheetMetaJSON struct {
	Image     string         `json:"image"`
	Size      sheetSizeJSON  `json:"size"`
	FrameTags []frameTagJSON `json:"frameTags"`
}

type sheetDataJSON struct {
	Frames map[string]frameEntryJSON `json:"frames"`
	Meta   sheetMetaJSON             `json:"meta"`
}

// Export packs anim's frames (or every row back to back, in grid order,
// when anim is nil) into one horizontal sprite-sheet PNG, plus an
// Aseprite-style data.json describing each frame's rect/duration and, in
// "frameTags", one tag per exported animation row — so common engine
// importers (e.g. Phaser's load.aseprite) can consume the pair directly.
func (sheetJSONFormat) Export(s sprite.Sprite, frames []sprite.Frame, anim *sprite.Animation, settings sprite.Settings) (Bundle, error) {
	steps, err := sequence(s, frames, anim)
	if err != nil {
		return Bundle{}, err
	}
	if len(steps) == 0 {
		return Bundle{}, fmt.Errorf("no frames to export")
	}

	sheet := image.NewNRGBA(image.Rect(0, 0, s.Width*len(steps), s.Height))
	data := sheetDataJSON{Frames: map[string]frameEntryJSON{}}
	data.Meta.Image = "sheet.png"

	for i, st := range steps {
		img, err := compositeFrame(s, st.frame, settings)
		if err != nil {
			return Bundle{}, err
		}
		x := i * s.Width
		draw.Draw(sheet, image.Rect(x, 0, x+s.Width, s.Height), img, image.Point{}, draw.Src)
		data.Frames[st.frame.ID] = frameEntryJSON{
			Frame:    frameRectJSON{X: x, Y: 0, W: s.Width, H: s.Height},
			Duration: st.durationMS,
		}
	}
	data.Meta.Size = sheetSizeJSON{W: sheet.Bounds().Dx(), H: sheet.Bounds().Dy()}

	rows := s.Animations
	if anim != nil {
		rows = []sprite.Animation{*anim}
	}
	from := 0
	for _, a := range rows {
		if len(a.Frames) == 0 {
			continue
		}
		to := from + len(a.Frames) - 1
		data.Meta.FrameTags = append(data.Meta.FrameTags, frameTagJSON{
			Name: a.Name, From: from, To: to, Direction: "forward",
		})
		from = to + 1
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, sheet); err != nil {
		return Bundle{}, err
	}
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return Bundle{}, err
	}

	return Bundle{
		Files: map[string][]byte{
			"sheet.png": pngBuf.Bytes(),
			"data.json": jsonBytes,
		},
	}, nil
}
