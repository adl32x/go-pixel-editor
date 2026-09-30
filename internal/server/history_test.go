package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// testServer starts the real routes over a fresh project with one 2x2
// sprite (one frame), and resets undo history between tests.
func testServer(t *testing.T) (*httptest.Server, sprite.Sprite) {
	t.Helper()
	t.Chdir(t.TempDir())
	history.bySprite = map[string]*spriteHistory{}
	settings := sprite.Settings{ActivePreset: "test", Palette: []sprite.PaletteEntry{
		{Char: "0", Color: "#000000"}, {Char: "1", Color: "#ffffff"},
	}}
	if err := settings.Save(); err != nil {
		t.Fatal(err)
	}
	s, err := sprite.NewSprite("Eye", 2, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sprite.AddFrame(&s, ""); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newMux())
	t.Cleanup(srv.Close)
	return srv, s
}

func call(t *testing.T, srv *httptest.Server, method, path, body string) (*http.Response, []byte) {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return res, data
}

// paint PUTs frame f001 with pixel (0,0) set to color ("" = empty).
func paint(t *testing.T, srv *httptest.Server, id, color string) *http.Response {
	t.Helper()
	cell := func(r, c int, col string) string {
		return `{"rowIndex":` + string(rune('0'+r)) + `,"columnIndex":` + string(rune('0'+c)) + `,"color":"` + col + `"}`
	}
	body := `[{"id":"L1","data":[[` + cell(0, 0, color) + `,` + cell(0, 1, "") + `],[` + cell(1, 0, "") + `,` + cell(1, 1, "") + `]]}]`
	res, _ := call(t, srv, "PUT", "/api/sprites/"+id+"/frames/f001", body)
	if res.StatusCode != 200 {
		t.Fatalf("PUT frame: %d", res.StatusCode)
	}
	return res
}

func firstPixel(t *testing.T, s sprite.Sprite) string {
	t.Helper()
	cur, _ := sprite.Find(s.ID)
	f, err := sprite.FindFrame(*cur, "f001")
	if err != nil || f == nil {
		t.Fatalf("FindFrame: %v", err)
	}
	return string(f.Layers["L1"][0][0])
}

func TestUndoRedoStrokes(t *testing.T) {
	srv, s := testServer(t)

	if res := paint(t, srv, s.ID, "#000000"); res.Header.Get(historyHeader) != "undo=1;redo=0" {
		t.Fatalf("history after one stroke = %q", res.Header.Get(historyHeader))
	}
	paint(t, srv, s.ID, "#ffffff")
	if got := firstPixel(t, s); got != "1" {
		t.Fatalf("pixel = %q, want 1", got)
	}

	res, body := call(t, srv, "POST", "/api/sprites/"+s.ID+"/undo", "")
	if res.StatusCode != 200 || res.Header.Get(historyHeader) != "undo=1;redo=1" {
		t.Fatalf("undo: %d %q", res.StatusCode, res.Header.Get(historyHeader))
	}
	var ur struct {
		ChangedFrames []string `json:"changedFrames"`
	}
	_ = json.Unmarshal(body, &ur)
	if len(ur.ChangedFrames) != 1 || ur.ChangedFrames[0] != "f001" {
		t.Fatalf("changedFrames = %v, want [f001]", ur.ChangedFrames)
	}
	if got := firstPixel(t, s); got != "0" {
		t.Fatalf("after undo pixel = %q, want 0", got)
	}

	call(t, srv, "POST", "/api/sprites/"+s.ID+"/undo", "")
	if got := firstPixel(t, s); got != "." {
		t.Fatalf("after 2nd undo pixel = %q, want empty", got)
	}
	if res, _ := call(t, srv, "POST", "/api/sprites/"+s.ID+"/undo", ""); res.StatusCode != http.StatusConflict {
		t.Fatalf("undo past the start = %d, want 409", res.StatusCode)
	}

	call(t, srv, "POST", "/api/sprites/"+s.ID+"/redo", "")
	if got := firstPixel(t, s); got != "0" {
		t.Fatalf("after redo pixel = %q, want 0", got)
	}

	// A new edit clears the redo stack.
	if res := paint(t, srv, s.ID, ""); res.Header.Get(historyHeader) != "undo=2;redo=0" {
		t.Fatalf("history after a new edit = %q, want undo=2;redo=0", res.Header.Get(historyHeader))
	}
}

func TestNoOpSaveIsNotAnUndoStep(t *testing.T) {
	srv, s := testServer(t)
	paint(t, srv, s.ID, "#000000")
	if res := paint(t, srv, s.ID, "#000000"); res.Header.Get(historyHeader) != "undo=1;redo=0" {
		t.Fatalf("history after an identical save = %q, want undo=1", res.Header.Get(historyHeader))
	}
}

func TestUndoResizeAndRename(t *testing.T) {
	srv, s := testServer(t)
	paint(t, srv, s.ID, "#ffffff")

	call(t, srv, "POST", "/api/sprites/"+s.ID+"/resize", `{"width":4,"height":3,"anchor":"top-left"}`)
	call(t, srv, "PATCH", "/api/sprites/"+s.ID, `{"name":"Big Eye"}`)
	if _, err := os.Stat(filepath.Join(sprite.Dir, s.ID+"-big-eye")); err != nil {
		t.Fatalf("rename didn't reslug: %v", err)
	}

	call(t, srv, "POST", "/api/sprites/"+s.ID+"/undo", "") // undo rename
	if _, err := os.Stat(filepath.Join(sprite.Dir, s.ID+"-eye")); err != nil {
		t.Fatalf("undoing the rename didn't restore the directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sprite.Dir, s.ID+"-big-eye")); !os.IsNotExist(err) {
		t.Fatal("undoing the rename left the renamed directory behind")
	}

	call(t, srv, "POST", "/api/sprites/"+s.ID+"/undo", "") // undo resize
	cur, _ := sprite.Find(s.ID)
	if cur.Width != 2 || cur.Height != 2 || firstPixel(t, s) != "1" {
		t.Fatalf("after undoing resize: %dx%d pixel %q", cur.Width, cur.Height, firstPixel(t, s))
	}
	if entries, _ := os.ReadDir(filepath.Dir(sprite.Dir)); len(entries) != 2 { // settings.md + sprites/
		t.Fatalf("leftover temp files next to sprites/: %v", entries)
	}
}

func TestFailedRequestIsNotAnUndoStep(t *testing.T) {
	srv, s := testServer(t)
	res, _ := call(t, srv, "POST", "/api/sprites/"+s.ID+"/resize", `{"width":0,"height":3}`)
	if res.StatusCode != http.StatusBadRequest || res.Header.Get(historyHeader) != "undo=0;redo=0" {
		t.Fatalf("bad resize: %d %q", res.StatusCode, res.Header.Get(historyHeader))
	}
}
