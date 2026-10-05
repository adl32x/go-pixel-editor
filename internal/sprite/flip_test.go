package sprite

import "testing"

func TestFlipFrames(t *testing.T) {
	s, f := resizeFixture(t) // 4x4, frame rows "0..1" "...." "...." "1..0"
	f.Layers["L1"] = [][]rune{
		[]rune("01.."),
		[]rune("...."),
		[]rune("...."),
		[]rune("...1"),
	}
	if err := f.Save(s); err != nil {
		t.Fatal(err)
	}

	if err := FlipFrames(s, []string{f.ID}, "", true); err != nil {
		t.Fatalf("horizontal: %v", err)
	}
	if got := rowsOf(t, s, f.ID); got != "..10|....|....|1..." {
		t.Fatalf("after horizontal flip = %s", got)
	}

	if err := FlipFrames(s, []string{f.ID}, "", false); err != nil {
		t.Fatalf("vertical: %v", err)
	}
	if got := rowsOf(t, s, f.ID); got != "1...|....|....|..10" {
		t.Fatalf("after vertical flip = %s", got)
	}
}

func TestFlipOnlyOneLayerAcrossFrames(t *testing.T) {
	s, f1 := resizeFixture(t)
	if _, err := AddLayer(&s, "sword"); err != nil {
		t.Fatal(err)
	}
	f2, _ := AddFrame(&s, "")
	for _, id := range []string{f1.ID, f2.ID} {
		fr, _ := FindFrame(s, id)
		fr.Layers["L1"] = [][]rune{[]rune("0..."), []rune("...."), []rune("...."), []rune("....")}
		fr.Layers["L2"] = [][]rune{[]rune("1..."), []rune("...."), []rune("...."), []rune("....")}
		if err := fr.Save(s); err != nil {
			t.Fatal(err)
		}
	}

	if err := FlipFrames(s, []string{f1.ID, f2.ID}, "L2", true); err != nil {
		t.Fatalf("FlipFrames: %v", err)
	}
	for _, id := range []string{f1.ID, f2.ID} {
		fr, _ := FindFrame(s, id)
		if got := string(fr.Layers["L2"][0]); got != "...1" {
			t.Fatalf("frame %s sword row 0 = %q, want flipped", id, got)
		}
		if got := string(fr.Layers["L1"][0]); got != "0..." {
			t.Fatalf("frame %s base row 0 = %q, want untouched", id, got)
		}
	}
}

func TestFlipRejectsUnknownFrameOrLayerWithoutWriting(t *testing.T) {
	s, f := resizeFixture(t)
	before := rowsOf(t, s, f.ID)
	if err := FlipFrames(s, []string{f.ID, "f999"}, "", true); err == nil {
		t.Fatal("expected an error for an unknown frame")
	}
	if err := FlipFrames(s, []string{f.ID}, "L9", true); err == nil {
		t.Fatal("expected an error for an unknown layer")
	}
	if got := rowsOf(t, s, f.ID); got != before {
		t.Fatalf("a rejected flip changed the frame: %s", got)
	}
}

func TestRotateFramesSquare(t *testing.T) {
	s, f := resizeFixture(t) // 4x4
	f.Layers["L1"] = [][]rune{
		[]rune("01.."),
		[]rune("...."),
		[]rune("...."),
		[]rune("...."),
	}
	if err := f.Save(s); err != nil {
		t.Fatal(err)
	}
	if err := RotateFrames(s, []string{f.ID}, "", true); err != nil {
		t.Fatalf("clockwise: %v", err)
	}
	// The top row turns into the right column, read top to bottom.
	if got := rowsOf(t, s, f.ID); got != "...0|...1|....|...." {
		t.Fatalf("after clockwise = %s", got)
	}
	if err := RotateFrames(s, []string{f.ID}, "", false); err != nil {
		t.Fatalf("counter-clockwise: %v", err)
	}
	if got := rowsOf(t, s, f.ID); got != "01..|....|....|...." {
		t.Fatalf("clockwise then counter-clockwise should be a no-op, got %s", got)
	}
	for i := 0; i < 4; i++ {
		if err := RotateFrames(s, []string{f.ID}, "", true); err != nil {
			t.Fatal(err)
		}
	}
	if got := rowsOf(t, s, f.ID); got != "01..|....|....|...." {
		t.Fatalf("four turns should be a no-op, got %s", got)
	}
}

func TestRotateNonSquareCentersOrRefuses(t *testing.T) {
	s, f := resizeFixture(t)
	if err := ResizeSprite(&s, 6, 4, "top-left"); err != nil { // 6 wide, 4 tall
		t.Fatal(err)
	}
	// A 2-pixel vertical bar in the middle fits after turning sideways.
	f.Layers["L1"] = [][]rune{
		[]rune("......"),
		[]rune("..0..."),
		[]rune("..1..."),
		[]rune("......"),
	}
	if err := f.Save(s); err != nil {
		t.Fatal(err)
	}
	if err := RotateFrames(s, []string{f.ID}, "", true); err != nil {
		t.Fatalf("rotate a centered bar: %v", err)
	}
	// The bar sat half a pixel left of center (column 2 of 6); turned
	// clockwise, left becomes up, so it lands half a pixel above center.
	if got := rowsOf(t, s, f.ID); got != "......|..10..|......|......" {
		t.Fatalf("after clockwise = %s", got)
	}

	// Something spanning the full width can't fit once it's vertical.
	wide, _ := FindFrame(s, f.ID)
	wide.Layers["L1"] = [][]rune{
		[]rune("......"),
		[]rune("000000"),
		[]rune("......"),
		[]rune("......"),
	}
	if err := wide.Save(s); err != nil {
		t.Fatal(err)
	}
	before := rowsOf(t, s, f.ID)
	if err := RotateFrames(s, []string{f.ID}, "", true); err == nil {
		t.Fatal("expected an error: a 6-wide line can't stand up in a 4-tall canvas")
	}
	if got := rowsOf(t, s, f.ID); got != before {
		t.Fatalf("a refused rotation changed the frame: %s", got)
	}
}
