package sprite

import "fmt"

// PixelModifyItem mirrors dotting's PixelModifyItem type exactly:
// { rowIndex, columnIndex, color }, where an empty Color ("") means "no
// pixel" (verified against dotting's own CreateEmptySquareData helper).
type PixelModifyItem struct {
	RowIndex    int    `json:"rowIndex"`
	ColumnIndex int    `json:"columnIndex"`
	Color       string `json:"color"`
}

// LayerProps mirrors dotting's LayerProps type exactly: { id, data }, where
// data is a dense [height][width] grid of PixelModifyItem.
type LayerProps struct {
	ID   string              `json:"id"`
	Data [][]PixelModifyItem `json:"data"`
}

// ToLayerProps expands f's compact palette-indexed grid into dotting's
// dense LayerProps/PixelModifyItem shape, one LayerProps per sprite layer in
// s.Layers order.
func (f Frame) ToLayerProps(s Sprite) ([]LayerProps, error) {
	out := make([]LayerProps, 0, len(s.Layers))
	for _, ld := range s.Layers {
		grid, ok := f.Layers[ld.ID]
		if !ok {
			return nil, fmt.Errorf("frame %s: missing layer %s", f.ID, ld.ID)
		}
		data := make([][]PixelModifyItem, s.Height)
		for r := 0; r < s.Height; r++ {
			row := make([]PixelModifyItem, s.Width)
			for c := 0; c < s.Width; c++ {
				item := PixelModifyItem{RowIndex: r, ColumnIndex: c}
				if ch := grid[r][c]; ch != '.' {
					color, ok := s.CharToColor(ch)
					if !ok {
						return nil, fmt.Errorf("frame %s: unknown palette char %q", f.ID, ch)
					}
					item.Color = color
				}
				row[c] = item
			}
			data[r] = row
		}
		out = append(out, LayerProps{ID: ld.ID, Data: data})
	}
	return out, nil
}

// FrameFromLayerProps converts a browser edit (dotting's LayerProps[]) back
// into a compact Frame. It mutates s.Palette when it encounters a color
// that hasn't been seen before — callers MUST call s.Save() before
// Frame.Save() whenever that happens, so a crash never leaves a frame file
// referencing a palette char that isn't yet recorded in sprite.md.
func FrameFromLayerProps(s *Sprite, frameID string, layers []LayerProps) (Frame, error) {
	byID := make(map[string]LayerProps, len(layers))
	for _, l := range layers {
		byID[l.ID] = l
	}

	f := Frame{ID: frameID, Layers: map[string][][]rune{}}
	for _, ld := range s.Layers {
		lp, ok := byID[ld.ID]
		if !ok {
			return Frame{}, fmt.Errorf("missing data for layer %s", ld.ID)
		}
		if len(lp.Data) != s.Height {
			return Frame{}, fmt.Errorf("layer %s: expected %d rows, got %d", ld.ID, s.Height, len(lp.Data))
		}
		grid := make([][]rune, s.Height)
		for r, row := range lp.Data {
			if len(row) != s.Width {
				return Frame{}, fmt.Errorf("layer %s row %d: expected %d cols, got %d", ld.ID, r, s.Width, len(row))
			}
			gridRow := make([]rune, s.Width)
			for c, item := range row {
				if item.Color == "" {
					gridRow[c] = '.'
					continue
				}
				ch, err := s.ColorToChar(item.Color)
				if err != nil {
					return Frame{}, err
				}
				gridRow[c] = ch
			}
			grid[r] = gridRow
		}
		f.Layers[ld.ID] = grid
	}
	return f, nil
}
