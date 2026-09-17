package export

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

type gifFormat struct{}

func (gifFormat) Name() string { return "gif" }

type gifStep struct {
	frame sprite.Frame
	delay int // 1/100s units, per image/gif's Delay field
}

// Export renders frames (a clip's entries in order, or every frame in file
// order when clip is nil) as a single animated GIF.
func (gifFormat) Export(s sprite.Sprite, frames []sprite.Frame, clip *sprite.Clip) (Bundle, error) {
	steps, err := gifSteps(frames, clip)
	if err != nil {
		return Bundle{}, err
	}
	if len(steps) == 0 {
		return Bundle{}, fmt.Errorf("no frames to export")
	}

	pal := buildPalette(s)

	g := &gif.GIF{LoopCount: gifLoopCount(clip)}
	for _, st := range steps {
		img, err := compositeFrame(s, st.frame)
		if err != nil {
			return Bundle{}, err
		}
		paletted := image.NewPaletted(img.Bounds(), pal)
		draw.Draw(paletted, paletted.Bounds(), img, image.Point{}, draw.Src)
		g.Image = append(g.Image, paletted)
		g.Delay = append(g.Delay, st.delay)
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return Bundle{}, err
	}
	return Bundle{Files: map[string][]byte{"animation.gif": buf.Bytes()}, ContentType: "image/gif"}, nil
}

// gifSteps expands a clip's entries (or, with no clip, every frame at a
// flat 100ms) into playback steps. Ping-pong clips are expanded into an
// explicit forward+reverse sequence, since GIF itself has no notion of
// bouncing — it only ever plays its frame list in order and loops from the
// start.
func gifSteps(frames []sprite.Frame, clip *sprite.Clip) ([]gifStep, error) {
	if clip == nil {
		steps := make([]gifStep, len(frames))
		for i, f := range frames {
			steps[i] = gifStep{frame: f, delay: 10}
		}
		return steps, nil
	}

	byID := make(map[string]sprite.Frame, len(frames))
	for _, f := range frames {
		byID[f.ID] = f
	}

	var steps []gifStep
	for _, e := range clip.Entries {
		f, ok := byID[e.FrameID]
		if !ok {
			return nil, fmt.Errorf("clip %q references unknown frame %s", clip.Name, e.FrameID)
		}
		delay := int(clip.FrameDuration(e).Milliseconds() / 10)
		if delay < 1 {
			delay = 1
		}
		steps = append(steps, gifStep{frame: f, delay: delay})
	}

	if clip.Loop == sprite.LoopPingpong && len(steps) > 2 {
		for i := len(steps) - 2; i > 0; i-- {
			steps = append(steps, steps[i])
		}
	}
	return steps, nil
}

// gifLoopCount maps a clip's LoopMode to image/gif's LoopCount semantics:
// 0 means "loop forever" (the Netscape looping-extension convention), a
// negative value omits the extension entirely, which plays the GIF exactly
// once.
func gifLoopCount(clip *sprite.Clip) int {
	if clip != nil && clip.Loop == sprite.LoopNone {
		return -1
	}
	return 0
}

// buildPalette turns a sprite's own palette into a GIF color.Palette, with
// a transparent entry at index 0 for empty pixels.
func buildPalette(s sprite.Sprite) color.Palette {
	pal := make(color.Palette, 0, len(s.Palette)+1)
	pal = append(pal, color.NRGBA{})
	for _, p := range s.Palette {
		c, err := parseHexColor(p.Color)
		if err != nil {
			continue
		}
		pal = append(pal, c)
	}
	return pal
}
