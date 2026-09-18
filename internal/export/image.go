package export

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// parseHexColor parses "#rrggbb" or "#rrggbbaa" (the only forms our own
// palette allocator and web UI color picker produce).
func parseHexColor(s string) (color.NRGBA, error) {
	if len(s) != 7 && len(s) != 9 {
		return color.NRGBA{}, fmt.Errorf("unsupported color format %q (want #rrggbb or #rrggbbaa)", s)
	}
	if s[0] != '#' {
		return color.NRGBA{}, fmt.Errorf("unsupported color format %q (want #rrggbb or #rrggbbaa)", s)
	}
	hexByte := func(hi, lo byte) (byte, error) {
		v, err := decodeHexPair(hi, lo)
		return v, err
	}
	r, err := hexByte(s[1], s[2])
	if err != nil {
		return color.NRGBA{}, err
	}
	g, err := hexByte(s[3], s[4])
	if err != nil {
		return color.NRGBA{}, err
	}
	b, err := hexByte(s[5], s[6])
	if err != nil {
		return color.NRGBA{}, err
	}
	a := byte(255)
	if len(s) == 9 {
		a, err = hexByte(s[7], s[8])
		if err != nil {
			return color.NRGBA{}, err
		}
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}

func decodeHexPair(hi, lo byte) (byte, error) {
	h, err := decodeHexDigit(hi)
	if err != nil {
		return 0, err
	}
	l, err := decodeHexDigit(lo)
	if err != nil {
		return 0, err
	}
	return h<<4 | l, nil
}

func decodeHexDigit(c byte) (byte, error) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', nil
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, nil
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, nil
	default:
		return 0, fmt.Errorf("invalid hex digit %q", c)
	}
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// compositeFrame flattens every visible sprite layer of f (bottom to top —
// Sprite.Layers is stored topmost-first, so this walks it in reverse) into
// one NRGBA image, honoring each layer's opacity. Colors that don't parse
// as #rrggbb(aa) are skipped rather than failing the whole export. settings
// resolves palette chars to colors (the project's single shared palette).
func compositeFrame(s sprite.Sprite, f sprite.Frame, settings sprite.Settings) (*image.NRGBA, error) {
	img := image.NewNRGBA(image.Rect(0, 0, s.Width, s.Height))
	for i := len(s.Layers) - 1; i >= 0; i-- {
		ld := s.Layers[i]
		if !ld.Visible {
			continue
		}
		grid, ok := f.Layers[ld.ID]
		if !ok {
			return nil, fmt.Errorf("frame %s: missing layer %s", f.ID, ld.ID)
		}

		layerImg := image.NewNRGBA(image.Rect(0, 0, s.Width, s.Height))
		for r := 0; r < s.Height; r++ {
			for c := 0; c < s.Width; c++ {
				ch := grid[r][c]
				if ch == '.' {
					continue
				}
				colorStr, ok := settings.CharToColor(ch)
				if !ok {
					return nil, fmt.Errorf("frame %s: unknown palette char %q", f.ID, ch)
				}
				col, err := parseHexColor(colorStr)
				if err != nil {
					continue
				}
				col.A = uint8(float64(col.A) * clamp01(ld.Opacity))
				layerImg.Set(c, r, col)
			}
		}
		draw.Draw(img, img.Bounds(), layerImg, image.Point{}, draw.Over)
	}
	return img, nil
}
