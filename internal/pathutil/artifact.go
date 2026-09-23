package pathutil

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnsafeArtifactPath identifies a nonlocal path or symlink in an artifact path.
var ErrUnsafeArtifactPath = errors.New("unsafe artifact path")

// InspectArtifactPath inspects every component below root without following links.
// It does not lock paths against concurrent filesystem replacement.
func InspectArtifactPath(ctx context.Context, root, relative string) (fs.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("inspecting artifact %s: %w", relative, err)
	}
	if !filepath.IsLocal(relative) || relative == "." || filepath.Clean(relative) != relative || strings.ContainsAny(relative, "\\:\x00") {
		return nil, fmt.Errorf("%w: %q must be a clean repository-relative path", ErrUnsafeArtifactPath, relative)
	}
	parts := strings.Split(relative, string(filepath.Separator))
	path := root
	var info fs.FileInfo
	for i, part := range parts {
		path = filepath.Join(path, part)
		var err error
		info, err = os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("inspecting artifact %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: %s is a symlink; use a regular artifact path", ErrUnsafeArtifactPath, path)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, fmt.Errorf("%w: %s is not a directory", ErrUnsafeArtifactPath, path)
		}
	}
	return info, nil
}
