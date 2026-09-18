// Package export implements go-pixel-editor's pluggable export formats: a
// small registry of named Format implementations (currently "gif" and
// "sheet-json"), so the HTTP API and CLI can pick one at request/flag time
// instead of a single hardcoded encoder. Adding a new target format later
// (e.g. a TexturePacker JSON-hash variant) is just another Format
// implementation plus a Register() call.
package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"sort"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// Bundle is what a Format produces: one or more named files. A single-entry
// Bundle is served as-is with ContentType; a multi-entry Bundle is zipped
// (see Payload) so the HTTP layer never has to know a format's internal
// file layout.
type Bundle struct {
	Files       map[string][]byte
	ContentType string
}

// Payload returns the bytes and content type to actually send to a client:
// the single file's bytes verbatim if there's only one, or a zip of every
// file otherwise.
func (b Bundle) Payload() ([]byte, string, error) {
	if len(b.Files) == 0 {
		return nil, "", fmt.Errorf("export produced no files")
	}
	if len(b.Files) == 1 {
		for _, data := range b.Files {
			ct := b.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			return data, ct, nil
		}
	}

	names := make([]string, 0, len(b.Files))
	for name := range b.Files {
		names = append(names, name)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			return nil, "", err
		}
		if _, err := w.Write(b.Files[name]); err != nil {
			return nil, "", err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "application/zip", nil
}

// Format is one exportable output — a query-param/CLI-flag value (Name)
// plus the logic to render a sprite (optionally scoped to one clip) into a
// Bundle. settings resolves palette chars to colors (the project's single
// shared palette — see internal/sprite's settings.go).
type Format interface {
	Name() string
	Export(s sprite.Sprite, frames []sprite.Frame, clip *sprite.Clip, settings sprite.Settings) (Bundle, error)
}

var registry = map[string]Format{}

// Register adds a Format to the registry, keyed by its Name().
func Register(f Format) { registry[f.Name()] = f }

// Get looks up a registered Format by name.
func Get(name string) (Format, bool) {
	f, ok := registry[name]
	return f, ok
}

// Names returns every registered format name, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func init() {
	Register(gifFormat{})
	Register(sheetJSONFormat{})
}
