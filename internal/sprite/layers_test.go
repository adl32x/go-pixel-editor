package sprite

import "testing"

// TestAddLayerRetrofitsExistingFrames covers the core correctness concern:
// a layer added after frames already exist must not make those frames
// unloadable (the "missing layer" hard-error both parseFrameFile and
// ToLayerProps would otherwise raise), and the new layer must actually
// become the topmost (first) layer.
func TestAddLayerRetrofitsExistingFrames(t *testing.T) {
	s, _, _ := setupFourByFour(t)
	f1, err := AddFrame(&s, "")
	if err != nil {
		t.Fatalf("AddFrame: %v", err)
	}
	f2, err := AddFrame(&s, "")
	if err != nil {
		t.Fatalf("AddFrame #2: %v", err)
	}

	newLayer, err := AddLayer(&s, "Clothes")
	if err != nil {
		t.Fatalf("AddLayer: %v", err)
	}
	if newLayer.Name != "Clothes" {
		t.Fatalf("new layer name = %q, want Clothes", newLayer.Name)
	}
	if len(s.Layers) != 2 || s.Layers[0].ID != newLayer.ID {
		t.Fatalf("expected new layer prepended (topmost): %+v", s.Layers)
	}

	reloaded, err := Find(s.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(reloaded.Layers) != 2 {
		t.Fatalf("reloaded sprite has %d layers, want 2", len(reloaded.Layers))
	}

	for _, id := range []string{f1.ID, f2.ID} {
		frame, err := FindFrame(*reloaded, id)
		if err != nil {
			t.Fatalf("FindFrame %s after AddLayer: %v", id, err)
		}
		if frame == nil {
			t.Fatalf("frame %s vanished after AddLayer", id)
		}
		grid, ok := frame.Layers[newLayer.ID]
		if !ok {
			t.Fatalf("frame %s missing retrofitted layer block %s", id, newLayer.ID)
		}
		for r, row := range grid {
			for c, ch := range row {
				if ch != '.' {
					t.Fatalf("frame %s new layer (%d,%d) = %q, want '.' (blank)", id, r, c, ch)
				}
			}
		}
	}
}

// TestDeleteLayerStripsFrameBlocks confirms the reverse: deleting a layer
// removes its block from every frame (so a re-add-with-same-name wouldn't
// resurrect stale pixel data) without disturbing the other layer's data.
func TestDeleteLayerStripsFrameBlocks(t *testing.T) {
	s, _, _ := setupFourByFour(t)
	f, err := AddFrame(&s, "")
	if err != nil {
		t.Fatalf("AddFrame: %v", err)
	}
	newLayer, err := AddLayer(&s, "Clothes")
	if err != nil {
		t.Fatalf("AddLayer: %v", err)
	}

	if err := DeleteLayer(&s, newLayer.ID); err != nil {
		t.Fatalf("DeleteLayer: %v", err)
	}
	if len(s.Layers) != 1 {
		t.Fatalf("expected 1 layer after delete, got %d: %+v", len(s.Layers), s.Layers)
	}

	reloaded, err := Find(s.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	frame, err := FindFrame(*reloaded, f.ID)
	if err != nil {
		t.Fatalf("FindFrame after DeleteLayer: %v", err)
	}
	if _, ok := frame.Layers[newLayer.ID]; ok {
		t.Fatalf("deleted layer's block still present in frame file")
	}
	if _, ok := frame.Layers["L1"]; !ok {
		t.Fatalf("original layer L1's block was removed along with the deleted one")
	}
}

// TestDeleteLayerRefusesLastLayer guards the "a sprite always has at least
// one layer" invariant every frame file's fixed shape depends on.
func TestDeleteLayerRefusesLastLayer(t *testing.T) {
	s, _, _ := setupFourByFour(t)
	if err := DeleteLayer(&s, "L1"); err == nil {
		t.Fatal("expected an error deleting a sprite's only layer, got nil")
	}
	if len(s.Layers) != 1 {
		t.Fatalf("layer count changed despite refused delete: %+v", s.Layers)
	}
}

// TestNextLayerIDNeverFillsAGap mirrors TestAddFrameIDNeverFillsAGap: max+1
// over currently-declared layers, so deleting a layer that ISN'T the
// current max never causes a later add to fill that now-empty gap. (Like
// frame IDs, this scheme can't prevent reusing the ID of whatever the
// single highest layer was at the moment *it's* deleted, since there's no
// separate persisted counter — see setupFourByFour's sibling frame-ID test
// for the same accepted limitation.)
func TestNextLayerIDNeverFillsAGap(t *testing.T) {
	s, _, _ := setupFourByFour(t) // starts with L1

	l2, err := AddLayer(&s, "B")
	if err != nil {
		t.Fatalf("AddLayer B: %v", err)
	}
	l3, err := AddLayer(&s, "C")
	if err != nil {
		t.Fatalf("AddLayer C: %v", err)
	}
	if l2.ID != "L2" || l3.ID != "L3" {
		t.Fatalf("layer ids = %s, %s, want L2, L3", l2.ID, l3.ID)
	}

	// Delete the middle layer (not the current max) and confirm the next
	// AddLayer continues from the true max, rather than filling the gap
	// left at L2.
	if err := DeleteLayer(&s, l2.ID); err != nil {
		t.Fatalf("DeleteLayer L2: %v", err)
	}
	l4, err := AddLayer(&s, "D")
	if err != nil {
		t.Fatalf("AddLayer D: %v", err)
	}
	if l4.ID != "L4" {
		t.Fatalf("id after deleting a middle layer = %s, want L4 (must not fill the L2 gap)", l4.ID)
	}
}
