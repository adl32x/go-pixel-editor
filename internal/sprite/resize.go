package sprite

import (
	"fmt"
	"strings"
)

// MaxCanvasSize caps a sprite's width and height; pixel art beyond this is
// far outside what the editor (or the one-char-per-pixel frame format) is
// meant for.
const MaxCanvasSize = 1024

// Anchors are the valid ResizeSprite anchor names: which part of the old
// canvas stays put. "top-left" keeps the top-left corner fixed (the canvas
// grows or is cut on the right and bottom), "center" grows/cuts evenly,
// "bottom" keeps the bottom edge (a character's feet) in place.
var Anchors = []string{
	"top-left", "top", "top-right",
	"left", "center", "right",
	"bottom-left", "bottom", "bottom-right",
}

// anchorOffset is where the old canvas's top-left lands in the new one,
// along one axis: 0 (start), half the size change (middle), or the whole
// size change (end). Negative when shrinking — that part is cut off.
func anchorOffset(oldSize, newSize int, pos string) int {
	switch pos {
	case "middle":
		return (newSize - oldSize) / 2
	case "end":
		return newSize - oldSize
	default:
		return 0
	}
}

// anchorAxes splits an anchor name into its horizontal and vertical
// positions ("start", "middle", "end").
func anchorAxes(anchor string) (h, v string, err error) {
	if anchor == "" {
		anchor = "center"
	}
	valid := false
	for _, a := range Anchors {
		if a == anchor {
			valid = true
		}
	}
	if !valid {
		return "", "", fmt.Errorf("invalid anchor %q (want one of %s)", anchor, strings.Join(Anchors, ", "))
	}
	h, v = "middle", "middle"
	if strings.Contains(anchor, "left") {
		h = "start"
	} else if strings.Contains(anchor, "right") {
		h = "end"
	}
	if strings.HasPrefix(anchor, "top") {
		v = "start"
	} else if strings.HasPrefix(anchor, "bottom") {
		v = "end"
	}
	return h, v, nil
}

// ResizeSprite changes s's canvas to width x height, rewriting every layer
// of every frame. Growing adds transparent ('.') pixels; shrinking simply
// cuts off whatever falls outside the new canvas — destructive, by design.
// anchor (see Anchors; "" = "center") picks which part of the old canvas
// stays in place.
//
// Frames are rewritten before sprite.md, like AddLayer: if interrupted,
// only the frames already rewritten are unreadable until it's rerun,
// rather than every frame at once.
func ResizeSprite(s *Sprite, width, height int, anchor string) error {
	if width < 1 || height < 1 || width > MaxCanvasSize || height > MaxCanvasSize {
		return fmt.Errorf("canvas size must be between 1x1 and %dx%d, got %dx%d", MaxCanvasSize, MaxCanvasSize, width, height)
	}
	h, v, err := anchorAxes(anchor)
	if err != nil {
		return err
	}
	if width == s.Width && height == s.Height {
		return nil
	}
	frames, err := LoadFrames(*s)
	if err != nil {
		return err
	}

	dx := anchorOffset(s.Width, width, h)
	dy := anchorOffset(s.Height, height, v)
	resized := *s
	resized.Width, resized.Height = width, height

	for _, f := range frames {
		for id, grid := range f.Layers {
			out := make([][]rune, height)
			for y := range out {
				row := make([]rune, width)
				for x := range row {
					row[x] = '.'
					if oy, ox := y-dy, x-dx; oy >= 0 && oy < s.Height && ox >= 0 && ox < s.Width {
						row[x] = grid[oy][ox]
					}
				}
				out[y] = row
			}
			f.Layers[id] = out
		}
		if err := f.Save(resized); err != nil {
			return fmt.Errorf("frame %s: %w", f.ID, err)
		}
	}

	s.Width, s.Height = width, height
	return s.Save()
}
