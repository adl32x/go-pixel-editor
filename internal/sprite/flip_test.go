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
