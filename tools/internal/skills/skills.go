// Package skills owns the materialization of aiwf's host adapters
// (Claude Code skills) into a consumer repo's `.claude/skills/aiwf-*/`
// tree.
//
// The skill markdown lives under embedded/ and is compiled into the
// binary via go:embed. The on-disk skill files are a cache, not state:
// `aiwf init` and `aiwf update` wipe every `aiwf-*/` dir and rewrite from
// the embed. Non-`aiwf-*` directories under `.claude/skills/` are
// untouched — the `aiwf-` prefix is the namespace boundary.
//
// `aiwf doctor` consumes List() to byte-compare the on-disk files
// against the embedded content and report drift.
package skills

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// embedFS holds the canonical SKILL.md content for every aiwf-* skill.
// The directory layout under embedded/ mirrors what materializes
// on disk under `.claude/skills/`.
//
//go:embed embedded
var embedFS embed.FS

// Skill is one embedded skill: its directory name (e.g. "aiwf-add") and
// the bytes that should be written to `.claude/skills/<name>/SKILL.md`.
type Skill struct {
	Name    string // directory name, e.g. "aiwf-add"
	Content []byte // SKILL.md contents
}

// SkillsDir is the host-relative directory the materializer writes
// into and `aiwf update` rewrites from. Claude Code's convention.
const SkillsDir = ".claude/skills"

// List returns every embedded skill in name-sorted order. The byte
// content is freshly read from the embed each call (cheap, since the
// embed is in-memory) so callers may mutate the returned slice without
// affecting future calls.
func List() ([]Skill, error) {
	entries, err := fs.ReadDir(embedFS, "embedded")
	if err != nil {
		return nil, fmt.Errorf("reading embedded skills: %w", err)
	}
	out := make([]Skill, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "aiwf-") {
			continue
		}
		content, err := fs.ReadFile(embedFS, filepath.ToSlash(filepath.Join("embedded", name, "SKILL.md")))
		if err != nil {
			return nil, fmt.Errorf("reading embedded skill %s: %w", name, err)
		}
		out = append(out, Skill{Name: name, Content: content})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Materialize wipes every `.claude/skills/aiwf-*/` directory under root
// and writes the embedded skills back. Non-`aiwf-*` directories are
// untouched. Creates `.claude/skills/` if missing.
//
// This is the operation behind both `aiwf init` (first-time setup) and
// `aiwf update` (refresh after a binary upgrade).
func Materialize(root string) error {
	skillsRoot := filepath.Join(root, SkillsDir)
	if err := os.MkdirAll(skillsRoot, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", SkillsDir, err)
	}

	// Wipe existing aiwf-* dirs so we never leave stale skills behind.
	entries, readErr := os.ReadDir(skillsRoot)
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		return fmt.Errorf("reading %s: %w", SkillsDir, readErr)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !strings.HasPrefix(e.Name(), "aiwf-") {
			continue
		}
		if rmErr := os.RemoveAll(filepath.Join(skillsRoot, e.Name())); rmErr != nil {
			return fmt.Errorf("removing stale skill %s: %w", e.Name(), rmErr)
		}
	}

	skills, err := List()
	if err != nil {
		return err
	}
	for _, s := range skills {
		dir := filepath.Join(skillsRoot, s.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), s.Content, 0o644); err != nil {
			return fmt.Errorf("writing %s/SKILL.md: %w", s.Name, err)
		}
	}
	return nil
}

// MaterializedPaths returns the repo-relative (forward-slash) paths
// that Materialize will produce, in name-sorted order. Used by
// `aiwf init` to populate `.gitignore`.
func MaterializedPaths() ([]string, error) {
	skills, err := List()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(skills))
	for _, s := range skills {
		out = append(out, SkillsDir+"/"+s.Name+"/")
	}
	return out, nil
}
