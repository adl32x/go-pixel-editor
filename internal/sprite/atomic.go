package sprite

import (
	"os"
	"path/filepath"
)

// writeFileAtomic writes data to path without ever exposing a reader to a
// partially-written or truncated file. os.WriteFile alone opens with
// O_TRUNC then writes in a separate step, so a concurrent request reading
// path (e.g. a different HTTP handler's Sprite.Find, racing with a Save)
// can observe a momentarily-empty file. Writing to a temp file in the same
// directory and renaming over path avoids that: rename is atomic on a
// single filesystem, so readers always see either the old or the new
// content in full, never a torn intermediate state.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
