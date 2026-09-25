// Package server implements `pixel serve`: a localhost HTTP API over the
// .pixel/sprites/ directory plus the embedded web/dist SPA (a React + dotting
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
	"github.com/adl32x/go-pixel-editor/internal/palette"
	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// Run starts the server, blocking until it exits (or an error occurs).
// args supports --port=NNNN (default 7788) and --no-open (skip launching
// the browser).
func Run(args []string) error {
	port := "7788"
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

	mux.HandleFunc("POST /api/sprites/{id}/layers", handleAddLayer)
	mux.HandleFunc("DELETE /api/sprites/{id}/layers/{layerId}", handleDeleteLayer)

	mux.HandleFunc("POST /api/sprites/{id}/animations", handleCreateAnimation)
	mux.HandleFunc("PATCH /api/sprites/{id}/animations/{name}", handlePatchAnimation)
	mux.HandleFunc("DELETE /api/sprites/{id}/animations/{name}", handleDeleteAnimation)

	mux.HandleFunc("GET /api/sprites/{id}/export.png", handleExportPNG)
	mux.HandleFunc("GET /api/sprites/{id}/export", handleExport)
	mux.HandleFunc("GET /api/export-formats", handleExportFormats)

	mux.HandleFunc("GET /api/palette", handleGetPalette)
	mux.HandleFunc("PUT /api/palette", handlePutPalette)
	mux.HandleFunc("GET /api/palette-presets", handleListPalettePresets)
	mux.HandleFunc("GET /api/palette-presets/{id}", handleGetPalettePreset)

	mux.Handle("/", staticHandler())

	url := fmt.Sprintf("http://localhost:%s", port)
	fmt.Println("pixel serve listening on", url)
	if open {
		openBrowser(url)
	}
	return http.ListenAndServe(":"+port, mux)
}

// spriteResponse adds a flat, grid-ordered frame id list to a Sprite for
// the HTTP layer. Every frame file belongs to exactly one animation row
// (normalized on load, see internal/sprite's normalizeAnimations), so this
// is just the rows flattened.
type spriteResponse struct {
	sprite.Sprite
	FrameIDs []string `json:"frameIds"`
}

func writeSprite(w http.ResponseWriter, s sprite.Sprite) {
	ids := s.OrderedFrameIDs()
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, spriteResponse{Sprite: s, FrameIDs: ids})
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

type addFrameRequest struct {
	// Animation is the row to append the new frame to ("" = last row).
	Animation string `json:"animation"`
}

func handleAddFrame(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	var req addFrameRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	f, err := sprite.AddFrame(s, req.Animation)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
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
	settings, err := sprite.LoadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	layers, err := f.ToLayerProps(*s, settings)
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
	settings, err := sprite.LoadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	frameID := r.PathValue("frameId")
	f, err := sprite.FrameFromLayerProps(*s, frameID, layers, settings)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
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
	if err := sprite.DeleteFrame(s, r.PathValue("frameId")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeSprite(w, *s)
}

type addLayerRequest struct {
	Name string `json:"name"`
}

// handleAddLayer adds a new (topmost) layer to the sprite, retrofitting a
// blank block onto every existing frame (see sprite.AddLayer) — the only
// safe way to add a layer once frames already exist.
func handleAddLayer(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	var req addLayerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := sprite.AddLayer(s, req.Name); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeSprite(w, *s)
}

// handleDeleteLayer removes a layer and strips its block from every
// existing frame (see sprite.DeleteLayer); refuses to remove a sprite's
// last layer.
func handleDeleteLayer(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	if err := sprite.DeleteLayer(s, r.PathValue("layerId")); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeSprite(w, *s)
}

type animationRequest struct {
	Name string `json:"name"`
}

// handleCreateAnimation appends a new, empty animation row.
func handleCreateAnimation(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	var req animationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := sprite.AddAnimation(s, req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeSprite(w, *s)
}

// handlePatchAnimation renames a row and/or reorders its frames / changes
// their duration overrides (see sprite.AnimationPatch).
func handlePatchAnimation(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	var patch sprite.AnimationPatch
	if err := decodeJSON(r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := sprite.UpdateAnimation(s, r.PathValue("name"), patch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeSprite(w, *s)
}

// handleDeleteAnimation removes a row along with every frame in it.
func handleDeleteAnimation(w http.ResponseWriter, r *http.Request) {
	s := findSprite(w, r.PathValue("id"))
	if s == nil {
		return
	}
	if err := sprite.DeleteAnimation(s, r.PathValue("name")); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeSprite(w, *s)
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
	settings, err := sprite.LoadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	png, err := export.FramePNG(*s, frame, settings)
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

	var anim *sprite.Animation
	if name := r.URL.Query().Get("animation"); name != "" {
		i := s.FindAnimation(name)
		if i < 0 {
			writeError(w, http.StatusNotFound, fmt.Errorf("no animation named %q", name))
			return
		}
		anim = &s.Animations[i]
	}

	frames, err := sprite.LoadFrames(*s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	settings, err := sprite.LoadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	bundle, err := format.Export(*s, frames, anim, settings)
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

// handleGetPalette returns the project's single active palette (see
// internal/sprite's Settings) — every sprite's frames index into this same
// list, there is no more per-sprite palette.
func handleGetPalette(w http.ResponseWriter, r *http.Request) {
	settings, err := sprite.LoadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, settings)
}

type putPaletteRequest struct {
	// PresetID, if set, switches to a built-in preset's colors wholesale.
	PresetID string `json:"presetId,omitempty"`
	// Colors, if set (and PresetID isn't), is a fully custom ordered color
	// list — a manual edit that doesn't match any named preset.
	Colors []string `json:"colors,omitempty"`
}

// handlePutPalette changes the project's active palette — the only
// supported way to do so, since it must remap every existing sprite's
// pixels to the new palette (see sprite.RemapPalette); there is
// deliberately no endpoint that just overwrites Settings.Palette directly.
func handlePutPalette(w http.ResponseWriter, r *http.Request) {
	var req putPaletteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	activePreset := req.PresetID
	colors := req.Colors
	if req.PresetID != "" {
		p, ok := palette.Get(req.PresetID)
		if !ok {
			writeError(w, http.StatusNotFound, fmt.Errorf("no palette preset %q", req.PresetID))
			return
		}
		colors = p.Colors
	} else if len(colors) > 0 {
		activePreset = "custom"
	} else {
		writeError(w, http.StatusBadRequest, fmt.Errorf("request must set presetId or colors"))
		return
	}

	settings, err := sprite.RemapPalette(activePreset, colors)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, settings)
}

type paletteMetaInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ColorCount int    `json:"colorCount"`
}

// handleListPalettePresets lists the built-in presets a project's palette
// can be set to (#0025's settings page picks from these) — just id/name/
// count, not the full color lists, to keep the listing light.
func handleListPalettePresets(w http.ResponseWriter, r *http.Request) {
	presets := palette.List()
	out := make([]paletteMetaInfo, len(presets))
	for i, p := range presets {
		out[i] = paletteMetaInfo{ID: p.ID, Name: p.Name, ColorCount: len(p.Colors)}
	}
	writeJSON(w, out)
}

// handleGetPalettePreset returns one preset's full color list.
func handleGetPalettePreset(w http.ResponseWriter, r *http.Request) {
	p, ok := palette.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("no palette preset %q", r.PathValue("id")))
		return
	}
	writeJSON(w, p)
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
