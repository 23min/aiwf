package entityview

import (
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/entity"
)

// ReadEntityBody reads the entity file at root/relPath and returns the
// body bytes (the prose after the closing `---`). Errors are
// swallowed — `aiwf show` already emits findings for unreadable /
// malformed entities via the load-error finding; surfacing the same
// problem on the body field would double-count. Empty body or missing
// file produces nil.
//
// Entity.Path is repo-relative (the loader normalizes it that way) so
// callers must join with root before hitting the filesystem; doing
// the join in this helper keeps each caller from re-deriving it.
func ReadEntityBody(root, relPath string) []byte {
	if relPath == "" {
		return nil
	}
	abs := relPath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, relPath)
	}
	content, err := os.ReadFile(abs)
	if err != nil {
		return nil
	}
	_, body, ok := entity.Split(content)
	if !ok {
		return nil
	}
	return body
}
