package sprite

import "fmt"

// FlipFrames mirrors frames of s in place — horizontally (left↔right) or
// vertically (top↔bottom). layerID limits it to one layer; "" flips every
// layer. Frame ids must belong to s; nothing is written unless all do.
func FlipFrames(s Sprite, frameIDs []string, layerID string, horizontal bool) error {
	return transformFrames(s, frameIDs, layerID, func(grid [][]rune) ([][]rune, error) {
		return flipGrid(grid, horizontal), nil
	})
}

// RotateFrames turns frames of s by 90° — clockwise or counter-clockwise —
// around the canvas center; layerID as in FlipFrames. On a square canvas
// that's exact. On a non-square one the rotated content can stick out of
// the canvas (a w×h image becomes h×w), so it's refused if any drawn pixel
// would be cut off, rather than silently losing it. Nothing is written
// unless every frame and layer rotates cleanly.
func RotateFrames(s Sprite, frameIDs []string, layerID string, clockwise bool) error {
	return transformFrames(s, frameIDs, layerID, func(grid [][]rune) ([][]rune, error) {
		return rotateGrid(grid, s.Width, s.Height, clockwise)
	})
}

// transformFrames applies fn to the chosen layer(s) of every frame in
// frameIDs, all in memory first, and only then writes them — so an error
// on any frame (unknown id, a rotation that would crop) changes nothing.
func transformFrames(s Sprite, frameIDs []string, layerID string, fn func([][]rune) ([][]rune, error)) error {
	if len(frameIDs) == 0 {
		return fmt.Errorf("no frames to transform")
	}
	if layerID != "" {
		found := false
		for _, ld := range s.Layers {
			if ld.ID == layerID {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("no layer %q", layerID)
		}
	}

	frames := make([]*Frame, 0, len(frameIDs))
	for _, id := range frameIDs {
		f, err := FindFrame(s, id)
		if err != nil {
			return err
		}
		if f == nil {
			return fmt.Errorf("no frame %s", id)
		}
		for lid, grid := range f.Layers {
			if layerID != "" && lid != layerID {
				continue
			}
			out, err := fn(grid)
			if err != nil {
				return fmt.Errorf("frame %s: %w", id, err)
			}
			f.Layers[lid] = out
		}
		frames = append(frames, f)
	}

	for _, f := range frames {
		if err := f.Save(s); err != nil {
			return fmt.Errorf("frame %s: %w", f.ID, err)
		}
	}
	return nil
}

func flipGrid(grid [][]rune, horizontal bool) [][]rune {
	out := make([][]rune, len(grid))
	for r, row := range grid {
		src := row
		if !horizontal {
			src = grid[len(grid)-1-r]
		}
		cp := make([]rune, len(src))
		for c := range src {
			if horizontal {
				cp[c] = src[len(src)-1-c]
			} else {
				cp[c] = src[c]
			}
		}
		out[r] = cp
	}
	return out
}

// rotateGrid turns a w×h grid by 90°. The rotated h×w image is centered on
// the w×h canvas (offset (w-h)/2, (h-w)/2 — zero for a square), so it
// pivots around the middle; a drawn pixel landing outside is an error.
func rotateGrid(grid [][]rune, w, h int, clockwise bool) ([][]rune, error) {
	out := make([][]rune, h)
	for y := range out {
		row := make([]rune, w)
		for x := range row {
			row[x] = '.'
		}
		out[y] = row
	}
	ox, oy := (w-h)/2, (h-w)/2
	cut := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := grid[y][x]
			if ch == '.' {
				continue
			}
			// Position inside the rotated h×w image, then on the canvas.
			nx, ny := h-1-y, x // clockwise
			if !clockwise {
				nx, ny = y, w-1-x
			}
			nx, ny = nx+ox, ny+oy
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				cut++
				continue
			}
			out[ny][nx] = ch
		}
	}
	if cut > 0 {
		return nil, fmt.Errorf("rotating would cut off %d pixel(s) on this %dx%d canvas; resize it to a square first", cut, w, h)
	}
	return out, nil
}
