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

// Export renders anim's frames (or every row back to back when anim is nil)
// as a single looping animated GIF.
func (gifFormat) Export(s sprite.Sprite, frames []sprite.Frame, anim *sprite.Animation, settings sprite.Settings) (Bundle, error) {
	steps, err := sequence(s, frames, anim)
	if err != nil {
		return Bundle{}, err
	}
	if len(steps) == 0 {
		return Bundle{}, fmt.Errorf("no frames to export")
	}

	pal := buildPalette(settings)

	g := &gif.GIF{LoopCount: 0} // 0 = loop forever
	for _, st := range steps {
		img, err := compositeFrame(s, st.frame, settings)
		if err != nil {
			return Bundle{}, err
		}
		paletted := image.NewPaletted(img.Bounds(), pal)
		draw.Draw(paletted, paletted.Bounds(), img, image.Point{}, draw.Src)
		delay := st.durationMS / 10 // image/gif's Delay is in 1/100s
		if delay < 1 {
			delay = 1
		}
		g.Image = append(g.Image, paletted)
		g.Delay = append(g.Delay, delay)
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return Bundle{}, err
	}
	return Bundle{Files: map[string][]byte{"animation.gif": buf.Bytes()}, ContentType: "image/gif"}, nil
}

// buildPalette turns the project's shared palette into a GIF color.Palette,
// with a transparent entry at index 0 for empty pixels.
func buildPalette(settings sprite.Settings) color.Palette {
	pal := make(color.Palette, 0, len(settings.Palette)+1)
	pal = append(pal, color.NRGBA{})
	for _, p := range settings.Palette {
		c, err := parseHexColor(p.Color)
		if err != nil {
			continue
		}
		pal = append(pal, c)
	}
	return pal
}
