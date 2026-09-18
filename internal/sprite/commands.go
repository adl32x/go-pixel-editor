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
// persists it immediately.
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
		Layers: []LayerDef{{ID: "L1", Name: "base", Visible: true, Opacity: 1}},
		Clips:  []Clip{},
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
	Name   *string     `json:"name,omitempty"`
	Tags   *[]string   `json:"tags,omitempty"`
	Layers *[]LayerDef `json:"layers,omitempty"`
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

// AddFrame creates a new blank frame (every layer filled with '.') for s,
// with the next never-reused frame id, and saves it.
func AddFrame(s Sprite) (Frame, error) {
	id, err := nextFrameID(s)
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
	if err := f.Save(s); err != nil {
		return Frame{}, err
	}
	return f, nil
}

// DeleteFrame removes a frame file. If the frame is referenced by any clip,
// the delete is refused unless force is true, in which case the references
// are also stripped from those clips (and s is re-saved) so no clip is left
// pointing at a nonexistent frame.
func DeleteFrame(s *Sprite, frameID string, force bool) error {
	referenced := false
	for _, c := range s.Clips {
		for _, e := range c.Entries {
			if e.FrameID == frameID {
				referenced = true
			}
		}
	}
	if referenced && !force {
		return fmt.Errorf("frame %s is referenced by a clip; delete with force=true to remove it there too", frameID)
	}
	if referenced {
		for i := range s.Clips {
			kept := s.Clips[i].Entries[:0]
			for _, e := range s.Clips[i].Entries {
				if e.FrameID != frameID {
					kept = append(kept, e)
				}
			}
			s.Clips[i].Entries = kept
		}
		if err := s.Save(); err != nil {
			return err
		}
	}
	return os.Remove(filepath.Join(framesDir(*s), frameID+".px"))
}

// NewClip creates a named clip on s and saves it. Returns an error if a
// clip with that name already exists.
func NewClip(s *Sprite, name string, loop LoopMode, fps float64) (Clip, error) {
	for _, c := range s.Clips {
		if c.Name == name {
			return Clip{}, fmt.Errorf("clip %q already exists", name)
		}
	}
	if fps <= 0 {
		fps = 12
	}
	c := Clip{Name: name, Loop: loop, FPS: fps, Entries: []ClipEntry{}}
	s.Clips = append(s.Clips, c)
	if err := s.Save(); err != nil {
		return Clip{}, err
	}
	return c, nil
}

// ClipPatch carries pointer-optional/whole-slice-replace fields for a clip
// update: Entries is replaced wholesale rather than spliced, matching
// go-backlog-cli's "coarse patch, let the caller build the new value"
// style.
type ClipPatch struct {
	Loop    *LoopMode    `json:"loop,omitempty"`
	FPS     *float64     `json:"fps,omitempty"`
	Entries *[]ClipEntry `json:"entries,omitempty"`
}

// UpdateClip applies patch to the named clip on s and saves the result.
func UpdateClip(s *Sprite, name string, patch ClipPatch) (Clip, error) {
	for i := range s.Clips {
		if s.Clips[i].Name != name {
			continue
		}
		if patch.Loop != nil {
			s.Clips[i].Loop = *patch.Loop
		}
		if patch.FPS != nil {
			s.Clips[i].FPS = *patch.FPS
		}
		if patch.Entries != nil {
			s.Clips[i].Entries = *patch.Entries
		}
		if err := s.Save(); err != nil {
			return Clip{}, err
		}
		return s.Clips[i], nil
	}
	return Clip{}, fmt.Errorf("no clip named %q", name)
}

// DeleteClip removes the named clip from s and saves the result. Frame
// files themselves are untouched — a clip is just a named view over them.
func DeleteClip(s *Sprite, name string) error {
	for i, c := range s.Clips {
		if c.Name == name {
			s.Clips = append(s.Clips[:i], s.Clips[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("no clip named %q", name)
}
