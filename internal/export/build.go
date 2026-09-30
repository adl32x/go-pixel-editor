package export

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// BuildManifest is the file, inside a build's output folder, listing every
// file a previous build wrote there — so a later full build can remove
// outputs of deleted sprites without ever touching files it didn't create.
const BuildManifest = ".pixel-build"

// BuildResult reports what Build did. Paths are relative to the build
// folder, with forward slashes.
type BuildResult struct {
	Out       string   `json:"out"`       // the build folder itself
	Written   []string `json:"written"`   // new or changed
	Unchanged []string `json:"unchanged"` // identical content already on disk, left alone
	Removed   []string `json:"removed"`   // stale outputs of sprites that no longer exist
	Skipped   []string `json:"skipped"`   // sprite ids with no frames to render
}

// ErrNoBuildOut is returned by BuildProject when neither an override nor
// the project's build_out setting names a build folder.
var ErrNoBuildOut = fmt.Errorf("no output folder: set build_out in %s (or the editor's Settings page), or pass --out=<dir>", sprite.SettingsPath)

// BuildProject is `pixel build` as a single call, shared by the CLI and the
// editor's Build button: it resolves the build folder (outOverride, else
// Settings.BuildOut), picks the sprites (ids, or every sprite), and builds.
func BuildProject(outOverride string, ids []string) (BuildResult, error) {
	settings, err := sprite.LoadSettings()
	if err != nil {
		return BuildResult{}, err
	}
	out := outOverride
	if out == "" {
		out = settings.BuildOut
	}
	if out == "" {
		return BuildResult{}, ErrNoBuildOut
	}

	sprites, err := sprite.Load()
	if err != nil {
		return BuildResult{}, err
	}
	if len(ids) > 0 {
		sprites = nil
		for _, id := range ids {
			s, err := sprite.Find(id)
			if err != nil {
				return BuildResult{}, err
			}
			if s == nil {
				return BuildResult{}, fmt.Errorf("no sprite with id %s", sprite.NormalizeID(id))
			}
			sprites = append(sprites, *s)
		}
	}
	return Build(sprites, out, settings, len(ids) == 0)
}

// outputBase is the manifest-relative path, minus extension, of s's build
// output: <s.Out>/<slug>.
func outputBase(s sprite.Sprite) (string, error) {
	sub, err := sprite.CleanOutDir(s.Out)
	if err != nil {
		return "", fmt.Errorf("sprite %s: %w", s.ID, err)
	}
	return path.Join(sub, s.Slug()), nil
}

// Build renders each sprite into outDir/<sprite.Out>/ as <slug>.png +
// <slug>.json, plus <slug>.<key>.png/.json per overlay layer (see
// RenderSheets). full means sprites is every sprite in the
// project, which makes it safe to delete previously built files that
// weren't produced this time; a partial build only adds to the manifest.
func Build(sprites []sprite.Sprite, outDir string, settings sprite.Settings, full bool) (BuildResult, error) {
	res := BuildResult{Out: outDir, Written: []string{}, Unchanged: []string{}, Removed: []string{}, Skipped: []string{}}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return res, err
	}

	byBase := map[string]string{}
	for _, s := range sprites {
		base, err := outputBase(s)
		if err != nil {
			return res, err
		}
		if other, ok := byBase[base]; ok {
			return res, fmt.Errorf("sprites %s and %s would both build to %q; rename one or give it another output folder", other, s.ID, base)
		}
		byBase[base] = s.ID
	}

	previous, err := readManifest(outDir)
	if err != nil {
		return res, err
	}
	produced := map[string]bool{}

	for _, s := range sprites {
		frames, err := sprite.LoadFrames(s)
		if err != nil {
			return res, fmt.Errorf("sprite %s: %w", s.ID, err)
		}
		if len(s.OrderedFrameIDs()) == 0 {
			res.Skipped = append(res.Skipped, s.ID)
			continue
		}
		base, _ := outputBase(s)
		sheets, err := RenderSheets(s, frames, settings)
		if err != nil {
			return res, fmt.Errorf("sprite %s: %w", s.ID, err)
		}
		if err := os.MkdirAll(filepath.Join(outDir, filepath.FromSlash(path.Dir(base))), 0o755); err != nil {
			return res, err
		}
		for file, data := range sheets {
			name := path.Join(path.Dir(base), file)
			changed, err := writeIfChanged(filepath.Join(outDir, filepath.FromSlash(name)), data)
			if err != nil {
				return res, err
			}
			if changed {
				res.Written = append(res.Written, name)
			} else {
				res.Unchanged = append(res.Unchanged, name)
			}
			produced[name] = true
		}
	}

	manifest := produced
	if full {
		for name := range previous {
			if produced[name] {
				continue
			}
			err := os.Remove(filepath.Join(outDir, filepath.FromSlash(name)))
			if err != nil && !os.IsNotExist(err) {
				return res, err
			}
			res.Removed = append(res.Removed, name)
		}
	} else {
		for name := range previous {
			manifest[name] = true
		}
	}
	if err := writeManifest(outDir, manifest); err != nil {
		return res, err
	}

	sort.Strings(res.Written)
	sort.Strings(res.Unchanged)
	sort.Strings(res.Removed)
	return res, nil
}

// writeIfChanged writes data to path unless the file already holds exactly
// that content, so rebuilding unchanged art doesn't touch mtimes.
func writeIfChanged(path string, data []byte) (bool, error) {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return false, nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return false, err
	}
	return true, os.Rename(tmp, path)
}

func readManifest(outDir string) (map[string]bool, error) {
	data, err := os.ReadFile(filepath.Join(outDir, BuildManifest))
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || line == BuildManifest {
			continue
		}
		// Never let a hand-edited manifest point a delete outside the
		// build folder.
		if clean, err := sprite.CleanOutDir(line); err != nil || clean != line {
			continue
		}
		names[line] = true
	}
	return names, nil
}

func writeManifest(outDir string, names map[string]bool) error {
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	var b strings.Builder
	b.WriteString("# Files written by `pixel build`. A full build deletes entries whose sprite is gone.\n")
	for _, n := range sorted {
		b.WriteString(n + "\n")
	}
	_, err := writeIfChanged(filepath.Join(outDir, BuildManifest), []byte(b.String()))
	return err
}
