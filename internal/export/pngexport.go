package export

import (
	"bytes"
	"image/png"

	"github.com/adl32x/go-pixel-editor/internal/sprite"
)

// FramePNG flattens a single frame's visible layers into a PNG — used by
// the quick-preview /export.png endpoint, distinct from the pluggable
// Format registry above (which always operates on a whole clip/sprite).
func FramePNG(s sprite.Sprite, f sprite.Frame, settings sprite.Settings) ([]byte, error) {
	img, err := compositeFrame(s, f, settings)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
