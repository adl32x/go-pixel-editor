package sprite

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// nextSpriteID scans existing .pixel/sprites/<id>-<slug> directories for the
// highest numeric id and returns max+1, zero-padded to 4 digits.
func nextSpriteID() (string, error) {
	entries, err := os.ReadDir(Dir)
	if os.IsNotExist(err) {
		return "0001", nil
	}
	if err != nil {
		return "", err
	}
	max := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := strings.SplitN(e.Name(), "-", 2)[0]
		n, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return fmt.Sprintf("%04d", max+1), nil
}

// NewSprite creates a new sprite with one default layer ("L1"/"base") and
// one empty animation row ("default"), and persists it immediately.
func NewSprite(name string, width, height int, tags string) (Sprite, error) {
	if width <= 0 {
		width = 16
	}
	if height <= 0 {
		height = 16
	}
	if name == "" {
		name = "(untitled)"
	}
	id, err := nextSpriteID()
	if err != nil {
		return Sprite{}, err
	}
	now := time.Now().UTC()
	s := Sprite{
		ID: id, Name: name, Width: width, Height: height,
		Tags:    splitTags(tags),
		Created: now, Updated: now,
		DurationMS: DefaultFrameDurationMS,
		Layers:     []LayerDef{{ID: "L1", Name: "base", Visible: true, Opacity: 1}},
		Animations: []Animation{{Name: "default", Frames: []AnimFrame{}}},
	}
	s.Path = filepath.Join(Dir, id+"-"+slugify(name), "sprite.md")
	if err := s.Save(); err != nil {
		return Sprite{}, err
	}
	return s, nil
}

// Reslug renames a sprite's directory to match its current Name, leaving ID
// untouched. It is a no-op if the directory already matches.
func Reslug(s Sprite) (Sprite, error) {
	oldDir := filepath.Dir(s.Path)
	newDir := filepath.Join(Dir, s.ID+"-"+slugify(s.Name))
	if oldDir == newDir {
		return s, nil
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return s, err
	}
	s.Path = filepath.Join(newDir, "sprite.md")
	return s, nil
}

// DeleteSprite removes a sprite and all of its frames.
func DeleteSprite(id string) error {
	s, err := Find(id)
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("no sprite with id %s", NormalizeID(id))
	}
	return os.RemoveAll(filepath.Dir(s.Path))
}

// SpritePatch carries pointer-optional fields for partial sprite metadata
// updates, mirroring go-backlog-cli's TicketPatch: only non-nil fields
// change.
type SpritePatch struct {
	Name       *string     `json:"name,omitempty"`
	Tags       *[]string   `json:"tags,omitempty"`
	Layers     *[]LayerDef `json:"layers,omitempty"`
	DurationMS *int        `json:"durationMs,omitempty"`
}

// UpdateSprite applies patch to the sprite matching id, reslugging its
// directory if the name changed, and saves the result.
func UpdateSprite(id string, patch SpritePatch) (Sprite, error) {
	s, err := Find(id)
	if err != nil {
		return Sprite{}, err
	}
	if s == nil {
		return Sprite{}, fmt.Errorf("no sprite with id %s", NormalizeID(id))
	}

	renamed := false
	if patch.Name != nil && *patch.Name != s.Name {
		s.Name = *patch.Name
		renamed = true
	}
	if patch.Tags != nil {
		s.Tags = *patch.Tags
	}
	if patch.Layers != nil {
		s.Layers = *patch.Layers
	}
	if patch.DurationMS != nil {
		if *patch.DurationMS <= 0 {
			return Sprite{}, fmt.Errorf("duration must be positive, got %d", *patch.DurationMS)
		}
		s.DurationMS = *patch.DurationMS
	}
	s.Updated = time.Now().UTC()

	if renamed {
		ns, err := Reslug(*s)
		if err != nil {
			return Sprite{}, err
		}
		*s = ns
	}
	if err := s.Save(); err != nil {
		return Sprite{}, err
	}
	return *s, nil
}

// nextLayerID scans s.Layers for the highest numeric suffix after "L" and
// returns max+1, e.g. "L3" if the highest existing is "L2". Mirrors
// nextFrameID's max+1-over-existing scheme: never renumbered, never reused
// from a currently-declared layer (though, like frame IDs, an ID freed by
// deleting the current max could in principle be reused by a later add —
// see frame_test.go's TestAddFrameIDNeverFillsAGap for why that's an
// accepted, documented limitation of this scheme rather than a bug).
func nextLayerID(s Sprite) string {
	max := 0
	for _, ld := range s.Layers {
		n, err := strconv.Atoi(strings.TrimPrefix(ld.ID, "L"))
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return fmt.Sprintf("L%d", max+1)
}

// AddLayer adds a new layer to s, as the new topmost layer (prepended —
// stack order is first=topmost, see LayerDef), and retrofits a blank
// (all-'.') block for it onto every one of s's existing frames.
//
// Order matters here to keep the failure window as small as possible: every
// frame is rewritten *before* s.Save() persists the new layer to sprite.md.
// If this is interrupted partway through, only the frames actually
// retrofitted so far end up with an "unknown layer block" (since sprite.md
// on disk still doesn't declare the new layer) until the operation
// completes or is retried — the alternative order (save sprite.md first)
// would instead make *every* frame of the sprite immediately unloadable
// ("missing layer") the moment any single frame's rewrite failed, which is
// worse. Same ordering RemapPalette (#0026) already uses for the same
// reason: rewrite content first, commit the metadata describing it last.
func AddLayer(s *Sprite, name string) (LayerDef, error) {
	frames, err := LoadFrames(*s)
	if err != nil {
		return LayerDef{}, err
	}
	if name == "" {
		name = "Layer"
	}

	newLayer := LayerDef{ID: nextLayerID(*s), Name: name, Visible: true, Opacity: 1}
	s.Layers = append([]LayerDef{newLayer}, s.Layers...)

	blank := make([][]rune, s.Height)
	for r := range blank {
		row := make([]rune, s.Width)
		for c := range row {
			row[c] = '.'
		}
		blank[r] = row
	}

	for _, f := range frames {
		gridCopy := make([][]rune, len(blank))
		for r, row := range blank {
			gridCopy[r] = append([]rune(nil), row...)
		}
		f.Layers[newLayer.ID] = gridCopy
		if err := f.Save(*s); err != nil {
			return LayerDef{}, fmt.Errorf("frame %s: %w", f.ID, err)
		}
	}

	if err := s.Save(); err != nil {
		return LayerDef{}, err
	}
	return newLayer, nil
}

// DeleteLayer removes layerID from s and strips its block from every
// existing frame. Refuses to delete a sprite's last remaining layer — a
// sprite with zero layers has nothing for a frame to declare.
//
// Same frames-first, sprite.md-last ordering as AddLayer, for the same
// reason (minimize the blast radius of an interrupted operation).
func DeleteLayer(s *Sprite, layerID string) error {
	if len(s.Layers) <= 1 {
		return fmt.Errorf("cannot delete a sprite's last remaining layer")
	}
	found := false
	for _, ld := range s.Layers {
		if ld.ID == layerID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no layer %q", layerID)
	}

	frames, err := LoadFrames(*s)
	if err != nil {
		return err
	}

	kept := make([]LayerDef, 0, len(s.Layers)-1)
	for _, ld := range s.Layers {
		if ld.ID != layerID {
			kept = append(kept, ld)
		}
	}
	s.Layers = kept

	for _, f := range frames {
		delete(f.Layers, layerID)
		if err := f.Save(*s); err != nil {
			return fmt.Errorf("frame %s: %w", f.ID, err)
		}
	}

	return s.Save()
}

// AddFrame creates a new blank frame (every layer filled with '.') with the
// next never-reused frame id, appends it to the end of the named animation
// row (the last row when animation is "", creating a "default" row if the
// sprite has none), and saves both.
//
// The frame file is written before sprite.md: if interrupted in between,
// the orphaned frame is simply adopted into the first row on the next load
// (see normalizeAnimations) rather than leaving a dangling reference.
func AddFrame(s *Sprite, animation string) (Frame, error) {
	idx := len(s.Animations) - 1
	if animation != "" {
		idx = s.FindAnimation(animation)
		if idx < 0 {
			return Frame{}, fmt.Errorf("no animation named %q", animation)
		}
	}
	if idx < 0 {
		s.Animations = append(s.Animations, Animation{Name: "default", Frames: []AnimFrame{}})
		idx = 0
	}

	id, err := nextFrameID(*s)
	if err != nil {
		return Frame{}, err
	}
	f := Frame{ID: id, Layers: map[string][][]rune{}}
	for _, ld := range s.Layers {
		grid := make([][]rune, s.Height)
		for r := range grid {
			row := make([]rune, s.Width)
			for c := range row {
				row[c] = '.'
			}
			grid[r] = row
		}
		f.Layers[ld.ID] = grid
	}
	if err := f.Save(*s); err != nil {
		return Frame{}, err
	}
	s.Animations[idx].Frames = append(s.Animations[idx].Frames, AnimFrame{FrameID: id})
	if err := s.Save(); err != nil {
		return Frame{}, err
	}
	return f, nil
}

// DeleteFrame removes a frame file and its cell in whichever animation row
// holds it.
func DeleteFrame(s *Sprite, frameID string) error {
	path := filepath.Join(framesDir(*s), frameID+".px")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("no frame %s", frameID)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if i := s.AnimationOf(frameID); i >= 0 {
		kept := []AnimFrame{}
		for _, f := range s.Animations[i].Frames {
			if f.FrameID != frameID {
				kept = append(kept, f)
			}
		}
		s.Animations[i].Frames = kept
	}
	return s.Save()
}

func validateAnimationName(s Sprite, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("animation name must not be empty")
	}
	if strings.ContainsAny(name, "\r\n") || name != strings.TrimSpace(name) {
		return fmt.Errorf("invalid animation name %q", name)
	}
	if s.FindAnimation(name) >= 0 {
		return fmt.Errorf("animation %q already exists", name)
	}
	return nil
}

// AddAnimation appends a new, empty animation row to s and saves it.
func AddAnimation(s *Sprite, name string) (Animation, error) {
	if err := validateAnimationName(*s, name); err != nil {
		return Animation{}, err
	}
	a := Animation{Name: name, Frames: []AnimFrame{}}
	s.Animations = append(s.Animations, a)
	if err := s.Save(); err != nil {
		return Animation{}, err
	}
	return a, nil
}

// AnimationPatch renames a row and/or replaces its frame list wholesale.
// A new Frames list may only reorder the row's existing frames or change
// their duration overrides — it must contain exactly the same frame ids, so
// the "every frame in exactly one row" invariant can't be broken by a
// patch. Frames are created and removed via AddFrame/DeleteFrame instead.
type AnimationPatch struct {
	Name   *string      `json:"name,omitempty"`
	Frames *[]AnimFrame `json:"frames,omitempty"`
}

// UpdateAnimation applies patch to the named row of s and saves the result.
func UpdateAnimation(s *Sprite, name string, patch AnimationPatch) (Animation, error) {
	i := s.FindAnimation(name)
	if i < 0 {
		return Animation{}, fmt.Errorf("no animation named %q", name)
	}
	if patch.Name != nil && *patch.Name != name {
		if err := validateAnimationName(*s, *patch.Name); err != nil {
			return Animation{}, err
		}
		s.Animations[i].Name = *patch.Name
	}
	if patch.Frames != nil {
		current := map[string]bool{}
		for _, f := range s.Animations[i].Frames {
			current[f.FrameID] = true
		}
		seen := map[string]bool{}
		for _, f := range *patch.Frames {
			if !current[f.FrameID] || seen[f.FrameID] {
				return Animation{}, fmt.Errorf("frames must be a reordering of the row's existing frames (bad entry %q)", f.FrameID)
			}
			if f.DurationMS != nil && *f.DurationMS <= 0 {
				return Animation{}, fmt.Errorf("frame %s: duration must be positive", f.FrameID)
			}
			seen[f.FrameID] = true
		}
		if len(seen) != len(current) {
			return Animation{}, fmt.Errorf("frames must include every frame already in the row")
		}
		s.Animations[i].Frames = append([]AnimFrame{}, *patch.Frames...)
	}
	if err := s.Save(); err != nil {
		return Animation{}, err
	}
	return s.Animations[i], nil
}

// DeleteAnimation removes the named row from s along with every frame file
// in it, then saves s. Same frames-first, sprite.md-last ordering as
// AddLayer: an interrupted delete leaves at worst a reference to a missing
// frame, which the next load drops.
func DeleteAnimation(s *Sprite, name string) error {
	i := s.FindAnimation(name)
	if i < 0 {
		return fmt.Errorf("no animation named %q", name)
	}
	for _, f := range s.Animations[i].Frames {
		err := os.Remove(filepath.Join(framesDir(*s), f.FrameID+".px"))
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("frame %s: %w", f.FrameID, err)
		}
	}
	s.Animations = append(s.Animations[:i], s.Animations[i+1:]...)
	return s.Save()
}
