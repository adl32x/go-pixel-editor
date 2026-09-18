package palette

import "testing"

func TestBuiltinPresetsRegistered(t *testing.T) {
	names := Names()
	if len(names) < 2 {
		t.Fatalf("Names() = %v, want at least nes and pico8", names)
	}
	if names[0] != "nes" {
		t.Fatalf("Names()[0] = %q, want %q (the default preset)", names[0], "nes")
	}

	for _, id := range []string{"nes", "pico8"} {
		p, ok := Get(id)
		if !ok {
			t.Fatalf("Get(%q) not found", id)
		}
		if len(p.Colors) == 0 {
			t.Fatalf("preset %q has no colors", id)
		}
		if len(p.Colors) > 62 {
			t.Fatalf("preset %q has %d colors, exceeds the 62-char palette-alphabet cap (see internal/sprite paletteAlphabet)", id, len(p.Colors))
		}
		for _, c := range p.Colors {
			if len(c) != 7 || c[0] != '#' {
				t.Fatalf("preset %q has malformed color %q, want #rrggbb", id, c)
			}
		}
	}
}

func TestListSortedByID(t *testing.T) {
	list := List()
	for i := 1; i < len(list); i++ {
		if list[i-1].ID >= list[i].ID {
			t.Fatalf("List() not sorted by ID: %q before %q", list[i-1].ID, list[i].ID)
		}
	}
}
