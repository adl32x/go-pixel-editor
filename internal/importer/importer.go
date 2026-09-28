// Package importer turns existing sprite art into a new sprite (#0010):
// either a sheet PNG plus the JSON describing it — an Aseprite json-array
// export or our own pixel-sheet/1 — which keeps animation names and frame
// durations, or a bare PNG split into a grid of equal frames, one animation
// row per grid row.
//
// Colors are mapped onto the project's shared palette (nearest match, like
// drawing does); the Report says how many pixels weren't exact matches.
package importer

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png" // register the PNG decoder
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// Options tweak an import. Zero values mean "work it out".
type Options struct {
	// Name of the new sprite; defaults to the file's base name.
	Name string
	// Out is the sprite's build subfolder; defaults to the source file's
	// folder relative to the project's build_out, when it lies inside it.
	Out string
	// FrameWidth/FrameHeight split a bare PNG into a grid; default is the
	// whole image as one frame. Ignored for JSON sources.
	FrameWidth, FrameHeight int
}

// Report describes what an import did beyond creating the sprite.
type Report struct {
	Sprite sprite.Sprite
	Frames int
	// InexactPixels were not exactly a palette color and were snapped to
	// the nearest one.
	InexactPixels int
	// TranslucentPixels had partial alpha; >= 50% became opaque, the rest
	// empty, since palette colors have no alpha.
	TranslucentPixels int
}

// frameSpec is one source frame: where it is in the image and which
// animation row it belongs to.
type frameSpec struct {
	anim       string
	rect       image.Rectangle
	durationMS int // 0 = sprite default
}

// Import creates a new sprite from src (a .json sheet description or a
// .png image) and returns what it did.
func Import(src string, opts Options) (Report, error) {
	var (
		imagePath string
		frames    []frameSpec
		err       error
	)
	if strings.EqualFold(filepath.Ext(src), ".json") {
		imagePath, frames, err = readSheetJSON(src)
		if err != nil {
			return Report{}, err
		}
	} else {
		imagePath = src
	}

	img, err := decodePNG(imagePath)
	if err != nil {
		return Report{}, err
	}
	if frames == nil {
		frames, err = gridFrames(img.Bounds(), opts.FrameWidth, opts.FrameHeight)
		if err != nil {
			return Report{}, err
		}
		frames = dropEmptyTrailingFrames(img, frames)
	}
	if len(frames) == 0 {
		return Report{}, fmt.Errorf("%s: no frames to import", src)
	}
	w, h := frames[0].rect.Dx(), frames[0].rect.Dy()
	for _, f := range frames {
		if f.rect.Dx() != w || f.rect.Dy() != h {
			return Report{}, fmt.Errorf("%s: frames differ in size (%dx%d vs %dx%d); a sprite has one frame size", src, w, h, f.rect.Dx(), f.rect.Dy())
		}
		if !f.rect.In(img.Bounds()) {
			return Report{}, fmt.Errorf("%s: frame %v lies outside the %v image", src, f.rect, img.Bounds())
		}
	}

	name := opts.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	}
	existing, err := sprite.Load()
	if err != nil {
		return Report{}, err
	}
	for _, s := range existing {
		if s.Name == name {
			return Report{}, fmt.Errorf("a sprite named %q already exists (%s); pass --name=", name, s.ID)
		}
	}

	settings, err := sprite.LoadSettings()
	if err != nil {
		return Report{}, err
	}
	out := opts.Out
	if out == "" {
		out = inferOut(settings.BuildOut, imagePath)
	}
	if out, err = sprite.CleanOutDir(out); err != nil {
		return Report{}, err
	}

	return create(name, out, img, frames, w, h, settings)
}

// create writes the new sprite: animation rows in first-seen order, the
// most common frame duration as the sprite default, and overrides only for
// frames that differ from it.
func create(name, out string, img image.Image, frames []frameSpec, w, h int, settings sprite.Settings) (rep Report, err error) {
	s, err := sprite.NewSprite(name, w, h, "")
	if err != nil {
		return Report{}, err
	}
	// Never leave a half-imported sprite behind.
	defer func() {
		if err != nil {
			_ = sprite.DeleteSprite(s.ID)
		}
	}()
	s.Out = out
	s.DurationMS = mostCommonDuration(frames)
	s.Animations = nil

	for _, spec := range frames {
		if s.FindAnimation(spec.anim) < 0 {
			if _, err := sprite.AddAnimation(&s, spec.anim); err != nil {
				return Report{}, err
			}
		}
		f, err := sprite.AddFrame(&s, spec.anim)
		if err != nil {
			return Report{}, err
		}
		grid := f.Layers[s.Layers[0].ID]
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := color.NRGBAModel.Convert(img.At(spec.rect.Min.X+x, spec.rect.Min.Y+y)).(color.NRGBA)
				if c.A != 0 && c.A != 255 {
					rep.TranslucentPixels++
				}
				if c.A < 128 {
					continue // stays '.'
				}
				hex := fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
				ch, err := settings.ColorToChar(hex)
				if err != nil {
					return Report{}, err
				}
				if exact, _ := settings.CharToColor(ch); !strings.EqualFold(exact, hex) {
					rep.InexactPixels++
				}
				grid[y][x] = ch
			}
		}
		if err := f.Save(s); err != nil {
			return Report{}, err
		}

		row := s.FindAnimation(spec.anim)
		last := &s.Animations[row].Frames[len(s.Animations[row].Frames)-1]
		if spec.durationMS > 0 && spec.durationMS != s.DurationMS {
			ms := spec.durationMS
			last.DurationMS = &ms
		}
		rep.Frames++
	}

	if err := s.Save(); err != nil {
		return Report{}, err
	}
	rep.Sprite = s
	return rep, nil
}

func decodePNG(p string) (image.Image, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return img, nil
}

// aseStateRe pulls the animation name out of an Aseprite frame filename:
// "guy (walking) 1.ase" -> "walking" (a --split-layers export).
var aseStateRe = regexp.MustCompile(`\((.*?)\)`)

type rectJSON struct{ X, Y, W, H int }

func (r rectJSON) rect() image.Rectangle { return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H) }

// readSheetJSON reads a pixel-sheet/1 or Aseprite json-array description.
// The returned image path is resolved relative to the JSON file.
func readSheetJSON(src string) (string, []frameSpec, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return "", nil, err
	}
	var probe struct {
		Format     string `json:"format"`
		Image      string `json:"image"`
		Animations map[string][]struct {
			rectJSON
			DurationMS int `json:"durationMs"`
		} `json:"animations"`
		Frames json.RawMessage `json:"frames"`
		Meta   struct {
			Image string `json:"image"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return "", nil, fmt.Errorf("%s: %w", src, err)
	}
	dir := filepath.Dir(src)

	if probe.Format == "pixel-sheet/1" {
		// encoding/json loses the object's key order, so rows are ordered by
		// where they sit in the sheet (top to bottom).
		names := make([]string, 0, len(probe.Animations))
		for n := range probe.Animations {
			names = append(names, n)
		}
		firstY := func(n string) int {
			if fs := probe.Animations[n]; len(fs) > 0 {
				return fs[0].Y
			}
			return 1 << 30
		}
		sort.Slice(names, func(i, j int) bool {
			if firstY(names[i]) != firstY(names[j]) {
				return firstY(names[i]) < firstY(names[j])
			}
			return names[i] < names[j]
		})
		var frames []frameSpec
		for _, n := range names {
			for _, f := range probe.Animations[n] {
				frames = append(frames, frameSpec{anim: n, rect: f.rect(), durationMS: f.DurationMS})
			}
		}
		return filepath.Join(dir, probe.Image), frames, nil
	}

	var aseFrames []struct {
		Filename string   `json:"filename"`
		Frame    rectJSON `json:"frame"`
		Duration int      `json:"duration"`
	}
	if len(probe.Frames) == 0 || json.Unmarshal(probe.Frames, &aseFrames) != nil || probe.Meta.Image == "" {
		return "", nil, fmt.Errorf("%s: not a pixel-sheet/1 or Aseprite json-array sheet (json-hash exports aren't supported; re-export with --format json-array)", src)
	}
	frames := make([]frameSpec, 0, len(aseFrames))
	for _, f := range aseFrames {
		m := aseStateRe.FindStringSubmatch(f.Filename)
		if m == nil {
			return "", nil, fmt.Errorf("%s: frame %q has no \"(animation)\" token; export with --split-layers", src, f.Filename)
		}
		frames = append(frames, frameSpec{anim: m[1], rect: f.Frame.rect(), durationMS: f.Duration})
	}
	return filepath.Join(dir, probe.Meta.Image), frames, nil
}

// gridFrames splits bounds into fw x fh cells, row by row; each grid row
// becomes an animation ("default" if there's only one, else "row1", …).
func gridFrames(bounds image.Rectangle, fw, fh int) ([]frameSpec, error) {
	if fw <= 0 {
		fw = bounds.Dx()
	}
	if fh <= 0 {
		fh = bounds.Dy()
	}
	if bounds.Dx()%fw != 0 || bounds.Dy()%fh != 0 {
		return nil, fmt.Errorf("image is %dx%d, not a whole number of %dx%d frames", bounds.Dx(), bounds.Dy(), fw, fh)
	}
	rows, cols := bounds.Dy()/fh, bounds.Dx()/fw
	var frames []frameSpec
	for r := 0; r < rows; r++ {
		anim := "default"
		if rows > 1 {
			anim = fmt.Sprintf("row%d", r+1)
		}
		for c := 0; c < cols; c++ {
			x, y := bounds.Min.X+c*fw, bounds.Min.Y+r*fh
			frames = append(frames, frameSpec{anim: anim, rect: image.Rect(x, y, x+fw, y+fh)})
		}
	}
	return frames, nil
}

// dropEmptyTrailingFrames removes fully transparent cells at the end of
// each grid row — the padding a sheet gets when its rows differ in length.
func dropEmptyTrailingFrames(img image.Image, frames []frameSpec) []frameSpec {
	empty := func(r image.Rectangle) bool {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
					return false
				}
			}
		}
		return true
	}
	var out []frameSpec
	for i := 0; i < len(frames); {
		j := i
		for j < len(frames) && frames[j].anim == frames[i].anim {
			j++
		}
		end := j
		for end > i+1 && empty(frames[end-1].rect) {
			end--
		}
		out = append(out, frames[i:end]...)
		i = j
	}
	return out
}

// mostCommonDuration picks the sprite-wide default: the duration most
// frames use (ties go to the shorter one), or the editor default when the
// source has none.
func mostCommonDuration(frames []frameSpec) int {
	counts := map[int]int{}
	for _, f := range frames {
		if f.durationMS > 0 {
			counts[f.durationMS]++
		}
	}
	best, bestN := sprite.DefaultFrameDurationMS, 0
	for d, n := range counts {
		if n > bestN || (n == bestN && d < best) {
			best, bestN = d, n
		}
	}
	return best
}

// inferOut returns imagePath's folder relative to buildOut ("characters"
// for assets/images/characters/guy.png with build_out assets/images), or ""
// if it isn't inside buildOut.
func inferOut(buildOut, imagePath string) string {
	if buildOut == "" {
		return ""
	}
	rel, err := filepath.Rel(filepath.Clean(buildOut), filepath.Dir(filepath.Clean(imagePath)))
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return ""
	}
	return path.Clean(rel)
}
