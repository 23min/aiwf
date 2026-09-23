package gitops

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// ProjectFiles lists tracked and untracked paths relative to root, excluding
// ignored paths even when tracked. It does not recurse into submodules.
// Callers must inspect the working tree: index entries can be missing or symlinks.
func ProjectFiles(ctx context.Context, root string) ([]string, error) {
	raw, err := output(ctx, root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("listing project files in %s: %w", root, err)
	}
	ignored, err := output(ctx, root, "ls-files", "--cached", "--ignored", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("listing ignored tracked files in %s: %w", root, err)
	}
	excluded := make(map[string]bool)
	for _, name := range strings.Split(ignored, "\x00") {
		excluded[name] = true
	}
	var files []string
	for _, name := range strings.Split(raw, "\x00") {
		if !excluded[name] {
			files = append(files, name)
		}
	}
	slices.Sort(files)
	return slices.Compact(files), nil
}
