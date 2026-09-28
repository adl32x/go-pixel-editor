package sprite

import (
	"math"
	"testing"
)

// Reference pairs from Sharma, Wu & Dalal (2005), the paper that pins down
// the CIEDE2000 formula's edge cases (hue wrap-around, neutral colors).
func TestCIEDE2000ReferencePairs(t *testing.T) {
	cases := []struct {
		a, b lab
		want float64
	}{
		{lab{50, 2.6772, -79.7751}, lab{50, 0, -82.7485}, 2.0425},
		{lab{50, 0, 0}, lab{50, -1, 2}, 2.3669},
		{lab{50, 2.5, 0}, lab{73, 25, -18}, 27.1492},
		{lab{60.2574, -34.0099, 36.2677}, lab{60.4626, -34.1751, 39.4387}, 1.2644},
		{lab{2.0776, 0.0795, -1.1350}, lab{0.9033, -0.0636, -0.5514}, 0.9082},
	}
	for _, c := range cases {
		if got := ciede2000(c.a, c.b); math.Abs(got-c.want) > 1e-4 {
			t.Errorf("ΔE(%v, %v) = %.4f, want %.4f", c.a, c.b, got, c.want)
		}
		if got := ciede2000(c.b, c.a); math.Abs(got-c.want) > 1e-4 {
			t.Errorf("ΔE is not symmetric for %v, %v: %.4f", c.a, c.b, got)
		}
	}
}

func TestRGBToLab(t *testing.T) {
	for hex, want := range map[string]lab{
		"#ffffff": {100, 0, 0},
		"#000000": {0, 0, 0},
		"#ff0000": {53.2408, 80.0925, 67.2032},
	} {
		got, ok := hexToLab(hex)
		if !ok || math.Abs(got.L-want.L) > 0.01 || math.Abs(got.A-want.A) > 0.01 || math.Abs(got.B-want.B) > 0.01 {
			t.Errorf("Lab(%s) = %+v, want %+v", hex, got, want)
		}
	}
}

// The case that motivated switching away from RGB distance: the subfive
// logo's teal against the game palette's greens. RGB distance picks the
// muddy grey-green #546756; the eye (and CIEDE2000) picks the sage.
func TestColorToCharIsPerceptual(t *testing.T) {
	settings := Settings{Palette: []PaletteEntry{
		{Char: "K", Color: "#546756"},
		{Char: "L", Color: "#89a477"},
		{Char: "M", Color: "#44702d"},
	}}
	ch, err := settings.ColorToChar("#37946e")
	if err != nil {
		t.Fatal(err)
	}
	if ch != 'L' {
		t.Fatalf("#37946e -> %q, want 'L' (#89a477)", ch)
	}
	if ch, _ := settings.ColorToChar("#89A477"); ch != 'L' {
		t.Fatalf("exact match should ignore hex case, got %q", ch)
	}
}
