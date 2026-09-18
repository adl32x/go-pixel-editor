package palette

// pico8Preset returns the PICO-8 fantasy console's official 16-color
// palette. Registered as a general-purpose curated retro palette — there is
// no single canonical "SNES palette" the way there is for NES/PICO-8 (the
// SNES has a 15-bit, 32,768-color space, not a fixed hardware palette), so
// this stands in for that use case rather than an invented/approximated
// SNES-specific list.
func pico8Preset() Preset {
	return Preset{
		ID:   "pico8",
		Name: "PICO-8",
		Colors: []string{
			"#000000", "#1D2B53", "#7E2553", "#008751",
			"#AB5236", "#5F574F", "#C2C3C7", "#FFF1E8",
			"#FF004D", "#FFA300", "#FFEC27", "#00E436",
			"#29ADFF", "#83769C", "#FF77A8", "#FFCCAA",
		},
	}
}
