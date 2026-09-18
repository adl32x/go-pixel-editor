// Package palette implements go-pixel-editor's built-in preset color
// palettes (NES, PICO-8, ...) — a small named registry, mirroring
// internal/export's Format registry shape, so the project's single active
// palette (internal/sprite.Settings) can default to and switch between them.
package palette

import "sort"

// Preset is one built-in named palette: an ordered list of "#rrggbb" colors.
// Order matters — it becomes palette-character position when a preset is
// loaded into a project's Settings (see internal/sprite.LoadSettings).
type Preset struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Colors []string `json:"colors"`
}

var registry = map[string]Preset{}
var order []string // registration order; order[0] is the default preset

// Register adds a Preset to the registry, keyed by its ID. The first Preset
// ever registered becomes the default (see Names).
func Register(p Preset) {
	if _, exists := registry[p.ID]; !exists {
		order = append(order, p.ID)
	}
	registry[p.ID] = p
}

// Get looks up a registered Preset by ID.
func Get(id string) (Preset, bool) {
	p, ok := registry[id]
	return p, ok
}

// Names returns every registered preset ID in registration order — Names()[0]
// is the default a new project's Settings falls back to.
func Names() []string {
	out := make([]string, len(order))
	copy(out, order)
	return out
}

// List returns every registered Preset, sorted by ID (for a stable API
// listing order independent of registration order).
func List() []Preset {
	out := make([]Preset, 0, len(registry))
	for _, id := range order {
		out = append(out, registry[id])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// init registers the built-in presets in one explicit, obviously-ordered
// place — deliberately not relying on Go's per-file init-order convention
// (lexical file name order is "encouraged" by the spec for build tools, not
// guaranteed), since Names()[0]/this registration order determines which
// preset a new project defaults to.
func init() {
	Register(nesPreset())
	Register(pico8Preset())
}
