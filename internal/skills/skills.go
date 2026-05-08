// Package skills owns the materialization of aiwf's host adapters
// (Claude Code skills) into a consumer repo's `.claude/skills/aiwf-*/`
// tree.
//
// The skill markdown lives under embedded/ and is compiled into the
// binary via go:embed. The on-disk skill files are a cache, not state:
// `aiwf init` and `aiwf update` rewrite every owned skill directory
// from the embed.
//
// Ownership is tracked by an on-disk manifest at
// `.claude/skills/.aiwf-owned`, written after every successful
// Materialize. Materialize wipes only directories named in the
// previous manifest that are no longer in the current embed (the
// "skill removed in this release" cleanup case). Foreign directories
// — third-party plugins under the `aiwf-*` prefix, or anything
// without the prefix — are never touched. This keeps the namespace
// safe to share with companion plugins (e.g., `aiwf-rituals-*`)
// without aiwf clobbering their content.
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

// ManifestFile is the on-disk record of which skill directories aiwf
// claims ownership of. One name per line, no trailing whitespace.
// Lives next to the skill dirs so a single stat tells aiwf whether
// any prior materialization happened.
const ManifestFile = ".aiwf-owned"

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

// Materialize writes the embedded skills into `.claude/skills/<name>/`
// under root. Wipes any directory listed in the prior ownership
// manifest that is no longer in the current embed (clean up after a
// release that removed a skill). Foreign directories — anything not
// in the prior manifest — are left alone, even if they share the
// `aiwf-` prefix.
//
// This is the operation behind both `aiwf init` (first-time setup) and
// `aiwf update` (refresh after a binary upgrade).
func Materialize(root string) error {
	skillsRoot := filepath.Join(root, SkillsDir)
	if err := os.MkdirAll(skillsRoot, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", SkillsDir, err)
	}

	prior, err := readManifest(skillsRoot)
	if err != nil {
		return err
	}

	skills, err := List()
	if err != nil {
		return err
	}

	currentSet := make(map[string]bool, len(skills))
	for _, s := range skills {
		currentSet[s.Name] = true
	}

	// Wipe directories the prior manifest claimed we owned but the
	// current embed no longer ships. Anything else (foreign dirs,
	// third-party plugins) is left alone.
	for _, name := range prior {
		if currentSet[name] {
			continue
		}
		if rmErr := os.RemoveAll(filepath.Join(skillsRoot, name)); rmErr != nil {
			return fmt.Errorf("removing previously-owned skill %s: %w", name, rmErr)
		}
	}

	// Write each currently-embedded skill. Existing dirs with the
	// same name (whether previously owned or pre-existing on first
	// run against an old aiwf install) get their SKILL.md overwritten.
	for _, s := range skills {
		dir := filepath.Join(skillsRoot, s.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), s.Content, 0o644); err != nil {
			return fmt.Errorf("writing %s/SKILL.md: %w", s.Name, err)
		}
	}

	if err := writeManifest(skillsRoot, skills); err != nil {
		return err
	}
	return nil
}

// readManifest returns the list of skill names the prior Materialize
// claimed ownership of. A missing manifest returns an empty slice and
// no error — first-run case.
func readManifest(skillsRoot string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(skillsRoot, ManifestFile))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// writeManifest records the names of the currently-embedded skills as
// the new ownership set. Atomic via temp-file + rename.
func writeManifest(skillsRoot string, skills []Skill) error {
	var b strings.Builder
	for _, s := range skills {
		b.WriteString(s.Name)
		b.WriteByte('\n')
	}
	path := filepath.Join(skillsRoot, ManifestFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming manifest: %w", err)
	}
	return nil
}

// GitignorePatterns returns the .gitignore lines that mask aiwf-
// materialized state in the consumer repo. Two entries: a directory
// wildcard that catches every aiwf-* skill dir (present and future),
// and the ownership manifest. The wildcard is what makes the .gitignore
// future-proof — adding a new embedded skill no longer requires every
// consumer to re-run `aiwf init` to refresh their .gitignore (G19).
//
// The trailing slash on the wildcard restricts the match to
// directories, so a non-aiwf file accidentally named like `aiwf-x.md`
// at that level would not be silently ignored.
func GitignorePatterns() []string {
	return []string{
		SkillsDir + "/aiwf-*/",
		SkillsDir + "/" + ManifestFile,
	}
}
