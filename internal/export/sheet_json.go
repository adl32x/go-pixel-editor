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

// Export packs frames (a clip's entries in order, or every frame in file
// order when clip is nil) into one horizontal sprite-sheet PNG, plus an
// Aseprite-style data.json describing each frame's rect/duration and, in
// "frameTags", the sprite's clip(s) — so common engine importers (e.g.
// Phaser's load.aseprite) can consume the pair directly.
func (sheetJSONFormat) Export(s sprite.Sprite, frames []sprite.Frame, clip *sprite.Clip) (Bundle, error) {
	if len(frames) == 0 {
		return Bundle{}, fmt.Errorf("no frames to export")
	}

	sheet := image.NewNRGBA(image.Rect(0, 0, s.Width*len(frames), s.Height))
	data := sheetDataJSON{Frames: map[string]frameEntryJSON{}}
	data.Meta.Image = "sheet.png"

	frameIndex := make(map[string]int, len(frames))
	for i, f := range frames {
		frameIndex[f.ID] = i

		img, err := compositeFrame(s, f)
		if err != nil {
			return Bundle{}, err
		}
		x := i * s.Width
		draw.Draw(sheet, image.Rect(x, 0, x+s.Width, s.Height), img, image.Point{}, draw.Src)

		duration := 100
		if clip != nil {
			for _, e := range clip.Entries {
				if e.FrameID == f.ID {
					duration = int(clip.FrameDuration(e).Milliseconds())
					break
				}
			}
		}
		data.Frames[f.ID] = frameEntryJSON{
			Frame:    frameRectJSON{X: x, Y: 0, W: s.Width, H: s.Height},
			Duration: duration,
		}
	}
	data.Meta.Size = sheetSizeJSON{W: sheet.Bounds().Dx(), H: sheet.Bounds().Dy()}

	tags := s.Clips
	if clip != nil {
		tags = []sprite.Clip{*clip}
	}
	for _, c := range tags {
		if len(c.Entries) == 0 {
			continue
		}
		from, fromOK := frameIndex[c.Entries[0].FrameID]
		to, toOK := frameIndex[c.Entries[len(c.Entries)-1].FrameID]
		if !fromOK || !toOK {
			continue
		}
		direction := "forward"
		if c.Loop == sprite.LoopPingpong {
			direction = "pingpong"
		}
		data.Meta.FrameTags = append(data.Meta.FrameTags, frameTagJSON{
			Name: c.Name, From: from, To: to, Direction: direction,
		})
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
