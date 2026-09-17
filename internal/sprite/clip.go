package sprite

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LoopMode is how a clip repeats during playback.
type LoopMode int

const (
	// LoopNone plays the clip once and holds the last frame.
	LoopNone LoopMode = iota
	// LoopForward repeats the entry list from the start.
	LoopForward
	// LoopPingpong bounces back and forth across the entry list.
	LoopPingpong
)

func (l LoopMode) String() string {
	switch l {
	case LoopNone:
		return "none"
	case LoopPingpong:
		return "pingpong"
	default:
		return "forward"
	}
}

// ParseLoopMode parses the on-disk/JSON string form of a LoopMode.
func ParseLoopMode(s string) (LoopMode, error) {
	switch s {
	case "none":
		return LoopNone, nil
	case "forward", "":
		return LoopForward, nil
	case "pingpong":
		return LoopPingpong, nil
	default:
		return LoopForward, fmt.Errorf("invalid loop mode: %q", s)
	}
}

// MarshalJSON encodes a LoopMode as its string form ("none"/"forward"/"pingpong").
func (l LoopMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// UnmarshalJSON decodes a LoopMode from its string form.
func (l *LoopMode) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	lm, err := ParseLoopMode(s)
	if err != nil {
		return err
	}
	*l = lm
	return nil
}

// ClipEntry is one step of a clip's playback sequence: a frame reference
// plus an optional per-entry duration override (in milliseconds). When nil,
// the duration is 1000/Clip.FPS.
type ClipEntry struct {
	FrameID    string `json:"frameId"`
	DurationMS *int   `json:"durationMs,omitempty"`
}

// Clip is a named, ordered animation sequence over a subset of a sprite's
// frames.
type Clip struct {
	Name    string      `json:"name"`
	Loop    LoopMode    `json:"loop"`
	FPS     float64     `json:"fps"`
	Entries []ClipEntry `json:"entries"`
}

// FrameDuration returns how long entry should be held during playback.
func (c Clip) FrameDuration(entry ClipEntry) time.Duration {
	if entry.DurationMS != nil {
		return time.Duration(*entry.DurationMS) * time.Millisecond
	}
	fps := c.FPS
	if fps <= 0 {
		fps = 1
	}
	return time.Duration(1000.0 / fps * float64(time.Millisecond))
}

func parseClipScalarLine(c *Clip, line string) error {
	key, val, ok := strings.Cut(line, ":")
	if !ok {
		return fmt.Errorf("clip %s: invalid line %q", c.Name, line)
	}
	key = strings.TrimSpace(key)
	val = strings.TrimSpace(val)
	switch key {
	case "loop":
		lm, err := ParseLoopMode(val)
		if err != nil {
			return fmt.Errorf("clip %s: %w", c.Name, err)
		}
		c.Loop = lm
	case "fps":
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("clip %s: invalid fps %q", c.Name, val)
		}
		c.FPS = f
	}
	return nil
}

func parseClipEntryLine(c *Clip, line string) error {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return fmt.Errorf("clip %s: empty frame entry", c.Name)
	}
	entry := ClipEntry{FrameID: fields[0]}
	if len(fields) > 1 && strings.HasPrefix(fields[1], "@") {
		ms, err := strconv.Atoi(strings.TrimPrefix(fields[1], "@"))
		if err != nil {
			return fmt.Errorf("clip %s: invalid duration override %q", c.Name, fields[1])
		}
		entry.DurationMS = &ms
	}
	c.Entries = append(c.Entries, entry)
	return nil
}

// formatClipsSection writes the "## clips" block: repeated "### <name>"
// clips, each with unindented "loop:"/"fps:" scalars and a "frames:" list
// of ONE frame id per line (never a single space-separated line) so
// inserting a frame in the middle of an animation is a clean single-line
// diff instead of rewriting a whole line.
func formatClipsSection(b *strings.Builder, clips []Clip) {
	fmt.Fprintln(b, "## clips")
	for _, c := range clips {
		fmt.Fprintf(b, "### %s\n", c.Name)
		fmt.Fprintf(b, "loop: %s\n", c.Loop.String())
		fmt.Fprintf(b, "fps: %s\n", strconv.FormatFloat(c.FPS, 'f', -1, 64))
		fmt.Fprintln(b, "frames:")
		for _, e := range c.Entries {
			if e.DurationMS != nil {
				fmt.Fprintf(b, "  %s @%d\n", e.FrameID, *e.DurationMS)
			} else {
				fmt.Fprintf(b, "  %s\n", e.FrameID)
			}
		}
	}
}
