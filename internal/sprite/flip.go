package sprite

import "fmt"

// FlipFrames mirrors frames of s in place — horizontally (left↔right) or
// vertically (top↔bottom). layerID limits it to one layer; "" flips every
// layer. Frame ids must belong to s; nothing is written unless all do.
func FlipFrames(s Sprite, frameIDs []string, layerID string, horizontal bool) error {
	if len(frameIDs) == 0 {
		return fmt.Errorf("no frames to flip")
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
		frames = append(frames, f)
	}

	for _, f := range frames {
		for id, grid := range f.Layers {
			if layerID != "" && id != layerID {
				continue
			}
			f.Layers[id] = flipGrid(grid, horizontal)
		}
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
