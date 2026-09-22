package projectguidance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/gitops"
)

// Match explains one pack's applicability without adopting it.
type Match struct {
	PackID      string
	Description string
	File        string
	Pattern     string
}

// Detect reports matching packs in catalogue order. Filename patterns come from
// a validated catalogue. Evidence uses the first pattern and lexical file match.
// Git ignore rules and dependency/cache directories exclude non-project material;
// directory names such as bin, build, and out do not imply generated content.
func Detect(ctx context.Context, root string, catalogue Catalogue) ([]Match, error) {
	files, err := gitops.ProjectFiles(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("detecting guidance; check project Git access: %w", err)
	}
	var eligible []string
	for _, name := range files {
		ok, err := detectionFile(root, name)
		if err != nil {
			return nil, fmt.Errorf("inspecting guidance candidate %s: %w", name, err)
		}
		if ok {
			eligible = append(eligible, name)
		}
	}
	var matches []Match
	for _, pack := range catalogue.Packs {
		if match, ok := matchPack(pack, eligible); ok {
			matches = append(matches, match)
		}
	}
	return matches, nil
}

func matchPack(pack Pack, files []string) (Match, bool) {
	for _, pattern := range pack.Detect {
		for _, name := range files {
			// Catalogue validation restricts patterns to valid basename globs.
			matched, _ := path.Match(pattern, path.Base(name))
			if matched {
				return Match{PackID: pack.ID, Description: pack.Description, File: name, Pattern: pattern}, true
			}
		}
	}
	return Match{}, false
}

func detectionFile(root, name string) (bool, error) {
	parts := strings.Split(name, "/")
	current := root
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !info.IsDir() {
			return false, nil
		}
		switch part {
		case ".git", "node_modules", "vendor", ".venv", "venv", "__pycache__", ".svelte-kit", ".next", ".nuxt", ".gradle", ".tox", ".mypy_cache", ".pytest_cache":
			return false, nil
		}
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			return false, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
	}
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(name)))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}
