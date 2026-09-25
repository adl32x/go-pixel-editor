package sprite

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DefaultFrameDurationMS is a new sprite's Sprite.DurationMS: how long each
// frame is held during playback unless the frame overrides it.
const DefaultFrameDurationMS = 100

// AnimFrame is one cell of an animation row: a frame reference plus an
// optional per-frame duration override (in milliseconds). When nil, the
// sprite's DurationMS applies.
type AnimFrame struct {
	FrameID    string `json:"frameId"`
	DurationMS *int   `json:"durationMs,omitempty"`
}

// Animation is one named row of a sprite's frame grid. Every frame file of
// a sprite belongs to exactly one animation (see normalizeAnimations), so
// the rows together are also the sprite's frame manifest and ordering.
type Animation struct {
	Name   string      `json:"name"`
	Frames []AnimFrame `json:"frames"`
}

// FrameDuration returns how long f should be held during playback, falling
// back to the sprite-wide default when f has no override.
func (s Sprite) FrameDuration(f AnimFrame) time.Duration {
	ms := s.DurationMS
	if f.DurationMS != nil {
		ms = *f.DurationMS
	}
	if ms <= 0 {
		ms = DefaultFrameDurationMS
	}
	return time.Duration(ms) * time.Millisecond
}

// FindAnimation returns the index of the animation named name, or -1.
func (s Sprite) FindAnimation(name string) int {
	for i, a := range s.Animations {
		if a.Name == name {
			return i
		}
	}
	return -1
}

// AnimationOf returns the index of the animation containing frameID, or -1.
func (s Sprite) AnimationOf(frameID string) int {
	for i, a := range s.Animations {
		for _, f := range a.Frames {
			if f.FrameID == frameID {
				return i
			}
		}
	}
	return -1
}

// OrderedFrameIDs returns every frame id in grid order: row by row, left to
// right.
func (s Sprite) OrderedFrameIDs() []string {
	var ids []string
	for _, a := range s.Animations {
		for _, f := range a.Frames {
			ids = append(ids, f.FrameID)
		}
	}
	return ids
}

// normalizeAnimations enforces the "every frame file is in exactly one
// row" invariant against the frame ids actually on disk: references to
// missing frames and repeat references are dropped, and frames no row
// mentions are appended to the first row (created if there is none). This
// is also what migrates a legacy "## clips" sprite, whose clips could share
// or omit frames, into the grid model.
func normalizeAnimations(s *Sprite, onDisk []string) {
	exists := make(map[string]bool, len(onDisk))
	for _, id := range onDisk {
		exists[id] = true
	}
	seen := map[string]bool{}
	for i := range s.Animations {
		kept := []AnimFrame{}
		for _, f := range s.Animations[i].Frames {
			if exists[f.FrameID] && !seen[f.FrameID] {
				kept = append(kept, f)
				seen[f.FrameID] = true
			}
		}
		s.Animations[i].Frames = kept
	}
	for _, id := range onDisk {
		if seen[id] {
			continue
		}
		if len(s.Animations) == 0 {
			s.Animations = append(s.Animations, Animation{Name: "default", Frames: []AnimFrame{}})
		}
		s.Animations[0].Frames = append(s.Animations[0].Frames, AnimFrame{FrameID: id})
	}
}

// parseAnimFrameLine parses one indented "fNNN [@ms]" line of a row's
// frames: list.
func parseAnimFrameLine(a *Animation, line string) error {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return fmt.Errorf("animation %s: empty frame entry", a.Name)
	}
	f := AnimFrame{FrameID: fields[0]}
	if len(fields) > 1 && strings.HasPrefix(fields[1], "@") {
		ms, err := strconv.Atoi(strings.TrimPrefix(fields[1], "@"))
		if err != nil {
			return fmt.Errorf("animation %s: invalid duration override %q", a.Name, fields[1])
		}
		f.DurationMS = &ms
	}
	a.Frames = append(a.Frames, f)
	return nil
}

// formatAnimationsSection writes the "## animations" block: repeated
// "### <name>" rows, each with a "frames:" list of ONE frame id per line
// (never a single space-separated line) so inserting a frame in the middle
// of an animation is a clean single-line diff instead of rewriting a whole
// line.
func formatAnimationsSection(b *strings.Builder, anims []Animation) {
	fmt.Fprintln(b, "## animations")
	for _, a := range anims {
		fmt.Fprintf(b, "### %s\n", a.Name)
		fmt.Fprintln(b, "frames:")
		for _, f := range a.Frames {
			if f.DurationMS != nil {
				fmt.Fprintf(b, "  %s @%d\n", f.FrameID, *f.DurationMS)
			} else {
				fmt.Fprintf(b, "  %s\n", f.FrameID)
			}
		}
	}
}
