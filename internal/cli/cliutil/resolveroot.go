package cliutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ResolveRoot picks the consumer repo root. If explicit is non-empty,
// it is used as-is (resolved to absolute). Otherwise, walks up from cwd
// looking for aiwf.yaml; if found, uses its parent. If not found, falls
// back to cwd (lenient pre-init behavior).
//
// This is the canonical exported home for root-dir resolution across
// the cmd/aiwf surface. Every verb that takes a `--root` flag funnels
// through ResolveRoot so the flag-empty + autodiscover behavior stays
// uniform.
func ResolveRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolving --root: %w", err)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting cwd: %w", err)
	}
	if found, ok := walkUpFor(cwd, "aiwf.yaml"); ok {
		return found, nil
	}
	return cwd, nil
}

// walkUpFor walks from start toward root looking for filename.
// Returns the directory containing filename (not the filename itself),
// and true if found.
func walkUpFor(start, filename string) (string, bool) {
	dir := start
	for {
		candidate := filepath.Join(dir, filename)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return dir, true
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
