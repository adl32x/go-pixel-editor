package server

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// Undo/redo for the editor: before any request that changes a sprite, the
// server snapshots that sprite's whole directory (sprite.md + frames/*.px —
// a few KB of text), and undo restores the previous snapshot. Working at the
// file level means every kind of edit — a stroke (one autosave), a resize,
// a frame/animation/layer change, a rename — is undoable the same way,
// without each operation needing its own inverse.
//
// History lives in memory, per sprite, for the lifetime of `pixel serve`;
// anything older is what git is for.

// maxHistory caps each sprite's undo stack.
const maxHistory = 100

// historyHeader carries a sprite's undo/redo depth ("undo=3;redo=1") on
// every response to a sprite-changing request, so the editor can enable
// its buttons without asking separately.
const historyHeader = "X-Pixel-History"

// snapshot is a sprite directory's full contents: its directory name
// (which changes on rename) and every file in it, by slash-separated
// relative path.
type snapshot struct {
	dir   string
	files map[string][]byte
}

func (a snapshot) equal(b snapshot) bool {
	if a.dir != b.dir || len(a.files) != len(b.files) {
		return false
	}
	for p, data := range a.files {
		if other, ok := b.files[p]; !ok || !bytes.Equal(data, other) {
			return false
		}
	}
	return true
}

// changedFrames lists the frame ids whose file differs between a and b
// (including frames that exist in only one of them), sorted.
func changedFrames(a, b snapshot) []string {
	seen := map[string]bool{}
	check := func(x, y snapshot) {
		for p, data := range x.files {
			id, ok := strings.CutPrefix(p, "frames/")
			if !ok || !strings.HasSuffix(id, ".px") {
				continue
			}
			if other, ok := y.files[p]; !ok || !bytes.Equal(data, other) {
				seen[strings.TrimSuffix(id, ".px")] = true
			}
		}
	}
	check(a, b)
	check(b, a)
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func spriteDir(id string) (string, error) {
	s, err := sprite.Find(id)
	if err != nil {
		return "", err
	}
	if s == nil {
		return "", fmt.Errorf("no sprite with id %s", sprite.NormalizeID(id))
	}
	return filepath.Dir(s.Path), nil
}

func takeSnapshot(id string) (snapshot, error) {
	dir, err := spriteDir(id)
	if err != nil {
		return snapshot{}, err
	}
	snap := snapshot{dir: filepath.Base(dir), files: map[string][]byte{}}
	err = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		snap.files[filepath.ToSlash(rel)] = data
		return nil
	})
	return snap, err
}

// restoreSnapshot replaces sprite id's directory with snap: written to a
// temporary sibling first, then swapped in, so a failure midway leaves the
// current directory untouched.
func restoreSnapshot(id string, snap snapshot) error {
	current, err := spriteDir(id)
	if err != nil {
		return err
	}
	// Next to .pixel/sprites/, not inside it: sprite.Load scans that folder
	// and would briefly see two copies of this sprite.
	tmp := filepath.Join(filepath.Dir(sprite.Dir), ".undo-"+sprite.NormalizeID(id))
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	for rel, data := range snap.files {
		p := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			return err
		}
	}
	if err := os.RemoveAll(current); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(sprite.Dir, snap.dir))
}

type spriteHistory struct {
	undo, redo []snapshot
}

// history is the server's undo/redo state. mu also serializes every
// sprite-changing request, so a snapshot always pairs with exactly the
// change it precedes.
var history = struct {
	mu       sync.Mutex
	bySprite map[string]*spriteHistory
}{bySprite: map[string]*spriteHistory{}}

func historyFor(id string) *spriteHistory {
	id = sprite.NormalizeID(id)
	h := history.bySprite[id]
	if h == nil {
		h = &spriteHistory{}
		history.bySprite[id] = h
	}
	return h
}

func setHistoryHeader(w http.ResponseWriter, id string) {
	h := historyFor(id)
	w.Header().Set(historyHeader, fmt.Sprintf("undo=%d;redo=%d", len(h.undo), len(h.redo)))
}

// bufferedResponse holds a handler's response so the history header can
// be added after the handler has run (headers can't change once written).
type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedResponse) Header() http.Header         { return b.header }
func (b *bufferedResponse) Write(p []byte) (int, error) { return b.body.Write(p) }
func (b *bufferedResponse) WriteHeader(status int)      { b.status = status }

// tracked wraps a handler that changes the sprite named by the {id} path
// value: a successful request that actually changed its files becomes one
// undo step (and clears the redo stack, as in any editor).
func tracked(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		history.mu.Lock()
		defer history.mu.Unlock()

		before, beforeErr := takeSnapshot(id)
		buf := &bufferedResponse{header: http.Header{}, status: http.StatusOK}
		h(buf, r)

		if beforeErr == nil && buf.status < 300 {
			if after, err := takeSnapshot(id); err == nil && !before.equal(after) {
				hist := historyFor(id)
				hist.undo = append(hist.undo, before)
				if len(hist.undo) > maxHistory {
					hist.undo = hist.undo[len(hist.undo)-maxHistory:]
				}
				hist.redo = nil
			}
		}

		for k, v := range buf.header {
			w.Header()[k] = v
		}
		setHistoryHeader(w, id)
		w.WriteHeader(buf.status)
		_, _ = w.Write(buf.body.Bytes())
	}
}

type undoResponse struct {
	Sprite spriteResponse `json:"sprite"`
	// ChangedFrames are the frames the step touched, so the editor can
	// jump to one when the frame on screen wasn't affected.
	ChangedFrames []string `json:"changedFrames"`
}

func handleUndo(w http.ResponseWriter, r *http.Request) { stepHistory(w, r, true) }
func handleRedo(w http.ResponseWriter, r *http.Request) { stepHistory(w, r, false) }

// stepHistory moves one step back (undo) or forward (redo): the current
// state goes onto the opposite stack, and the popped snapshot is restored.
func stepHistory(w http.ResponseWriter, r *http.Request, undo bool) {
	id := r.PathValue("id")
	history.mu.Lock()
	defer history.mu.Unlock()

	hist := historyFor(id)
	from, to := &hist.undo, &hist.redo
	if !undo {
		from, to = to, from
	}
	if len(*from) == 0 {
		setHistoryHeader(w, id)
		writeError(w, http.StatusConflict, fmt.Errorf("nothing to %s", map[bool]string{true: "undo", false: "redo"}[undo]))
		return
	}

	current, err := takeSnapshot(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	target := (*from)[len(*from)-1]
	if err := restoreSnapshot(id, target); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	*from = (*from)[:len(*from)-1]
	*to = append(*to, current)

	s, err := sprite.Find(id)
	if err != nil || s == nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("reloading sprite %s after %s: %v", id, map[bool]string{true: "undo", false: "redo"}[undo], err))
		return
	}
	ids := s.OrderedFrameIDs()
	if ids == nil {
		ids = []string{}
	}
	setHistoryHeader(w, id)
	writeJSON(w, undoResponse{
		Sprite:        spriteResponse{Sprite: *s, FrameIDs: ids},
		ChangedFrames: changedFrames(current, target),
	})
}
