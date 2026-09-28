package sprite

import (
	"fmt"
	"testing"
)

// A worst-ish case: every pixel of a 64x64 frame is an off-palette color
// matched against a full 62-color palette.
func BenchmarkColorToChar64x64(b *testing.B) {
	var s Settings
	for i := 0; i < 62; i++ {
		s.Palette = append(s.Palette, PaletteEntry{Char: string(paletteAlphabet[i]), Color: fmt.Sprintf("#%02x%02x%02x", i*4, 255-i*4, i*2)})
	}
	colors := make([]string, 64*64)
	for i := range colors {
		colors[i] = fmt.Sprintf("#%02x%02x%02x", i%251, (i*7)%253, (i*13)%255)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, c := range colors {
			_, _ = s.ColorToChar(c)
		}
	}
}

// The drawing case: every pixel is already a palette color.
func BenchmarkColorToCharExact64x64(b *testing.B) {
	var s Settings
	for i := 0; i < 62; i++ {
		s.Palette = append(s.Palette, PaletteEntry{Char: string(paletteAlphabet[i]), Color: fmt.Sprintf("#%02x%02x%02x", i*4, 255-i*4, i*2)})
	}
	colors := make([]string, 64*64)
	for i := range colors {
		colors[i] = s.Palette[i%62].Color
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, c := range colors {
			_, _ = s.ColorToChar(c)
		}
	}
}
