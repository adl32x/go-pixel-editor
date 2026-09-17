// Package server implements `pixel serve`: a localhost HTTP API over the
// sprites/ directory plus the embedded web/dist SPA (a React + dotting
// pixel canvas with a custom frame/timeline/playback layer).
package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/adl32x/go-pixel-editor/internal/export"
	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// Run starts the server, blocking until it exits (or an error occurs).
// args supports --port=NNNN (default 7777) and --no-open (skip launching
// the browser).
func Run(args []string) error {
	port := "7777"
	open := true
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--port="):
			port = strings.TrimPrefix(a, "--port=")
		case a == "--no-open":
			open = false
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/sprites", handleListSprites)
	mux.HandleFunc("POST /api/sprites", handleCreateSprite)
	mux.HandleFunc("GET /api/sprites/{id}", handleGetSprite)
	mux.HandleFunc("PATCH /api/sprites/{id}", handlePatchSprite)
	mux.HandleFunc("DELETE /api/sprites/{id}", handleDeleteSprite)

	mux.HandleFunc("POST /api/sprites/{id}/frames", handleAddFrame)
	mux.HandleFunc("GET /api/sprites/{id}/frames/{frameId}", handleGetFrame)
	mux.HandleFunc("PUT /api/sprites/{id}/frames/{frameId}", handlePutFrame)
	mux.HandleFunc("DELETE /api/sprites/{id}/frames/{frameId}", handleDeleteFrame)

	mux.HandleFunc("GET /api/sprites/{id}/clips", handleListClips)
	mux.HandleFunc("POST /api/sprites/{id}/clips", handleCreateClip)
	mux.HandleFunc("PATCH /api/sprites/{id}/clips/{name}", handlePatchClip)
	mux.HandleFunc("DELETE /api/sprites/{id}/clips/{name}", handleDeleteClip)

	mux.HandleFunc("GET /api/sprites/{id}/export.png", handleExportPNG)
	mux.HandleFunc("GET /api/sprites/{id}/export", handleExport)
	mux.HandleFunc("GET /api/export-formats", handleExportFormats)

	mux.Handle("/", staticHandler())

	url := fmt.Sprintf("http://localhost:%s", port)
	fmt.Println("pixel serve listening on", url)
	if open {
		openBrowser(url)
	}
	return http.ListenAndServe(":"+port, mux)
}

// spriteResponse adds the frame id list to a Sprite for the HTTP layer.
// Frame existence has no manifest in sprite.md (see internal/sprite's
// design doc comment) — it's computed at read time by scanning frames/*.px,
// same as Sprite.Summary does for FrameCount, so the frontend's timeline
// knows which frames to fetch without a separate list-frames round trip.
type spriteResponse struct {
	sprite.Sprite
	FrameIDs []string `json:"frameIds"`
}

func toSpriteResponse(s sprite.Sprite) (spriteResponse, error) {
	frames, err := sprite.LoadFrames(s)
	if err != nil {
		return spriteResponse{}, err
	}
	ids := make([]string, len(frames))
	for i, f := range frames {
		ids[i] = f.ID
	}
	return spriteResponse{Sprite: s, FrameIDs: ids}, nil
}

func writeSprite(w http.ResponseWriter, s sprite.Sprite) {
	resp, err := toSpriteResponse(s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, resp)
}

func findSprite(w http.ResponseWriter, id string) *sprite.Sprite {
	s, err := sprite.Find(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return nil
	}
	if s == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("no sprite with id %s", sprite.NormalizeID(id)))
		return nil
	}
	return s
}

func handleListSprites(w http.ResponseWriter, r *http.Request) {
	sprites, err := sprite.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	summaries := make([]sprite.Summary, 0, len(sprites))
	for _, s := range sprites {
		sum, err := s.Summary()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		summaries = append(summaries, sum)
	}
	writeJSON(w, summaries)
}

type createSpriteRequest struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Tags   string `json:"tags"`
}

func handleCreateSprite(w http.ResponseWriter, r *http.Request) {
	var req createSpriteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s, err := sprite.NewSprite(req.Name, req.Width, req.Height, req.Tags)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeSprite(w, s)
}

func handleGetSprite(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	writeSprite(w, *s)
}

func handlePatchSprite(w http.ResponseWriter, r *http.Request) {
	var patch sprite.SpritePatch
	if err := decodeJSON(r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s, err := sprite.UpdateSprite(r.PathValue("id"), patch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeSprite(w, s)
}

func handleDeleteSprite(w http.ResponseWriter, r *http.Request) {
	if err := sprite.DeleteSprite(r.PathValue("id")); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleAddFrame(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	f, err := sprite.AddFrame(*s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]string{"frameId": f.ID})
}

func handleGetFrame(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	f, err := sprite.FindFrame(*s, r.PathValue("frameId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if f == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("no frame %s", r.PathValue("frameId")))
		return
	}
	layers, err := f.ToLayerProps(*s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, layers)
}

func handlePutFrame(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	var layers []sprite.LayerProps
	if err := decodeJSON(r, &layers); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	frameID := r.PathValue("frameId")
	f, err := sprite.FrameFromLayerProps(s, frameID, layers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Sprite.Save() must run first: FrameFromLayerProps may have allocated
	// new palette chars, and a frame file must never reference a char that
	// isn't yet recorded in sprite.md.
	if err := s.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := f.Save(*s); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func handleDeleteFrame(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	force := r.URL.Query().Get("force") == "true"
	if err := sprite.DeleteFrame(s, r.PathValue("frameId"), force); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleListClips(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	writeJSON(w, s.Clips)
}

type createClipRequest struct {
	Name string  `json:"name"`
	Loop string  `json:"loop"`
	FPS  float64 `json:"fps"`
}

func handleCreateClip(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	var req createClipRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	loop, err := sprite.ParseLoopMode(req.Loop)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	c, err := sprite.NewClip(s, req.Name, loop, req.FPS)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, c)
}

func handlePatchClip(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	var patch sprite.ClipPatch
	if err := decodeJSON(r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	c, err := sprite.UpdateClip(s, r.PathValue("name"), patch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, c)
}

func handleDeleteClip(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	if err := sprite.DeleteClip(s, r.PathValue("name")); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleExportPNG(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	frames, err := sprite.LoadFrames(*s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(frames) == 0 {
		writeError(w, http.StatusNotFound, fmt.Errorf("sprite %s has no frames", s.ID))
		return
	}
	frame := frames[0]
	if id := r.URL.Query().Get("frame"); id != "" {
		found := false
		for _, f := range frames {
			if f.ID == id {
				frame = f
				found = true
				break
			}
		}
		if !found {
			writeError(w, http.StatusNotFound, fmt.Errorf("no frame %s", id))
			return
		}
	}
	png, err := export.FramePNG(*s, frame)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	formatName := r.URL.Query().Get("format")
	if formatName == "" {
		formatName = "sheet-json"
	}
	format, ok := export.Get(formatName)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown export format %q (available: %s)", formatName, strings.Join(export.Names(), ", ")))
		return
	}

	var clip *sprite.Clip
	if name := r.URL.Query().Get("clip"); name != "" {
		for i := range s.Clips {
			if s.Clips[i].Name == name {
				clip = &s.Clips[i]
				break
			}
		}
		if clip == nil {
			writeError(w, http.StatusNotFound, fmt.Errorf("no clip named %q", name))
			return
		}
	}

	frames, err := sprite.LoadFrames(*s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	bundle, err := format.Export(*s, frames, clip)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	data, contentType, err := bundle.Payload()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}

type exportFormatInfo struct {
	Name string `json:"name"`
}

func handleExportFormats(w http.ResponseWriter, r *http.Request) {
	names := export.Names()
	out := make([]exportFormatInfo, len(names))
	for i, n := range names {
		out[i] = exportFormatInfo{Name: n}
	}
	writeJSON(w, out)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// decodeJSON decodes r's body into v, turning the raw io.EOF a JSON decoder
// returns for a request with no body at all into a message that actually
// says so, instead of leaking "EOF" verbatim into an API error response.
func decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		if err == io.EOF {
			return fmt.Errorf("request body is required")
		}
		return err
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

const notBuiltMessage = "web UI not built: run `cd web && bun install && bun run build` (or `make build`)"

// staticHandler serves the embedded SPA build, falling back to index.html
// for any path that isn't a real asset (client-side routing).
func staticHandler() http.Handler {
	notBuilt := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, notBuiltMessage, http.StatusNotFound)
	})

	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return notBuilt
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return notBuilt
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(sub, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
			r = cloneWithPath(r, "/")
		}
		fileServer.ServeHTTP(w, r)
	})
}

func cloneWithPath(r *http.Request, path string) *http.Request {
	r2 := r.Clone(r.Context())
	r2.URL.Path = path
	return r2
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

//go:embed all:dist
var distFS embed.FS
