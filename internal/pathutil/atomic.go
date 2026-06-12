package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to path so that a crash or hard kill
// mid-write never leaves a half-written file at path: the bytes go
// to a sibling temp file first, are fsynced to disk, and the temp
// file is renamed over path in a single atomic step. Readers see
// either the old content or the new content, never a truncated mix
// (G-0221).
//
// The temp file lives in path's parent directory so the final
// os.Rename never crosses a filesystem boundary. perm is applied to
// the temp file before the rename, so the final file carries exactly
// the requested mode (unlike os.WriteFile, umask does not apply).
// The temp file is removed on every error path.
//
// Two deliberate semantic edges: a symlink at path is replaced by a
// regular file rather than written through (os.Rename semantics),
// and the parent directory is not fsynced after the rename — the
// file's *content* is durable and the swap is atomic, but the rename
// itself may be lost on power failure, leaving the old file intact.
// Both match the canonical sequence G-0221 specifies.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	f, err := os.CreateTemp(dir, base+".aiwf-tmp-")
	if err != nil {
		return fmt.Errorf("creating temp file for %s: %w", path, err)
	}
	tmp := f.Name()
	if _, wErr := f.Write(data); wErr != nil { //coverage:ignore not portably triggerable: writing a fresh temp file fails only on disk-full / device errors
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("writing %s: %w", tmp, wErr)
	}
	if sErr := f.Sync(); sErr != nil { //coverage:ignore not portably triggerable: fsync on a healthy fd fails only on device errors
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("syncing %s: %w", tmp, sErr)
	}
	if cErr := f.Close(); cErr != nil { //coverage:ignore not portably triggerable: close after a successful fsync fails only on device errors
		_ = os.Remove(tmp)
		return fmt.Errorf("closing %s: %w", tmp, cErr)
	}
	if mErr := os.Chmod(tmp, perm); mErr != nil { //coverage:ignore not portably triggerable: chmod on an owned fresh temp file fails only if it is removed concurrently
		_ = os.Remove(tmp)
		return fmt.Errorf("chmod %s: %w", tmp, mErr)
	}
	if rErr := os.Rename(tmp, path); rErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s -> %s: %w", tmp, path, rErr)
	}
	return nil
}
