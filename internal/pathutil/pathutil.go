// Package pathutil contains small, dependency-free path helpers shared
// across aiwf packages. The helpers are designed to fail closed:
// when input is unusable (relative paths, empty strings, broken
// symlinks), they return false / an error rather than guessing.
package pathutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotAbsolute is returned by Resolve when its input is a relative
// path. Callers are expected to make paths absolute before calling
// Resolve (typically via filepath.Join with an absolute repo root).
var ErrNotAbsolute = errors.New("path must be absolute")

// Inside reports whether candidate, after cleaning, lies at or under
// root. Both paths must be absolute; relative inputs return false.
//
// The check appends a path separator to root before prefix-matching so
// that "/repo" does not match "/repository". Paths equal to root are
// considered inside.
func Inside(root, candidate string) bool {
	if root == "" || candidate == "" {
		return false
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(candidate) {
		return false
	}
	r := filepath.Clean(root)
	c := filepath.Clean(candidate)
	if r == c {
		return true
	}
	return strings.HasPrefix(c, r+string(filepath.Separator))
}

// Resolve returns candidate with symlinks evaluated and the result
// cleaned. candidate must be absolute; relative inputs return
// ErrNotAbsolute so callers fail closed instead of silently joining
// against an unrelated working directory.
//
// If the path does not exist, Resolve returns the cleaned absolute
// form (lexical fallback) and a nil error so callers can still report
// a clean "missing" finding rather than crashing.
//
// Broken symlinks and symlink loops produce errors — they are treated
// as "unresolvable" so callers fail closed.
func Resolve(candidate string) (string, error) {
	if !filepath.IsAbs(candidate) {
		return "", fmt.Errorf("resolving %s: %w", candidate, ErrNotAbsolute)
	}
	cleaned := filepath.Clean(candidate)
	if _, err := os.Lstat(cleaned); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cleaned, nil
		}
		return "", fmt.Errorf("resolving %s: %w", candidate, err)
	}
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", candidate, err)
	}
	return resolved, nil
}
