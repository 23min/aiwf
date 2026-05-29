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

// ritualsFS holds the vendored ai-workflow-rituals snapshot (E-0038),
// pinned via rituals.lock and refreshed by `make sync-rituals`. The
// ritual *skills* under it materialize flattened into
// `.claude/skills/<skill-name>/` alongside the verb skills; agents and
// templates in the same snapshot are materialized by a later milestone.
//
//go:embed embedded-rituals
var ritualsFS embed.FS

// ritualsRoot is the embed path of the vendored rituals snapshot.
const ritualsRoot = "embedded-rituals"

// Skill is one embedded skill: its directory name (e.g. "aiwf-add") and
// the bytes that should be written to `.claude/skills/<name>/SKILL.md`.
type Skill struct {
	Name    string // directory name, e.g. "aiwf-add"
	Content []byte // SKILL.md contents
}

// SkillsDir is the host-relative directory the materializer writes
// into and `aiwf update` rewrites from. Claude Code's convention.
const SkillsDir = ".claude/skills"

// AgentsDir is the host-relative directory the ritual agents
// (planner/builder/reviewer/deployer) materialize into. Claude Code's
// convention, sibling of SkillsDir.
const AgentsDir = ".claude/agents"

// TemplatesDir is the host-relative directory the ritual templates
// (adr/decision/epic-spec/milestone-spec) materialize into. Sibling of
// SkillsDir and AgentsDir per D-0015; ADR-0014 §3 left the location open
// ("→ their referenced locations") and §4 makes it a per-target value.
const TemplatesDir = ".claude/templates"

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

// ListRituals returns every embedded ritual *skill* (aiwfx-*, wf-*) in
// name-sorted order, walking the vendored
// `embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md` tree. The
// plugin wrapper is flattened away: Name is the skill directory name,
// which is what materializes under `.claude/skills/`. Agents and
// templates living in the same snapshot are intentionally not returned —
// only files literally named SKILL.md under a `skills/` parent qualify.
func ListRituals() ([]Skill, error) {
	var out []Skill
	err := fs.WalkDir(ritualsFS, ritualsRoot, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		// Expect .../skills/<skill>/SKILL.md; anything else (agents,
		// templates) is skipped.
		parts := strings.Split(p, "/")
		if len(parts) < 3 || parts[len(parts)-3] != "skills" {
			return nil
		}
		name := parts[len(parts)-2]
		content, readErr := fs.ReadFile(ritualsFS, p)
		if readErr != nil {
			return fmt.Errorf("reading embedded ritual skill %s: %w", name, readErr)
		}
		out = append(out, Skill{Name: name, Content: content})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking embedded rituals: %w", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// listRitualFiles returns every embedded ritual file living directly
// under a directory named parentDir (e.g. "agents", "templates"), in
// name-sorted order. Unlike skills, these artifacts materialize flat —
// the file itself is the unit, so Name carries the `.md` suffix. The
// `.gitkeep` placeholder under an empty templates/ dir is a dotfile and
// is excluded from the embed by go:embed's default dot-skip.
func listRitualFiles(parentDir string) ([]Skill, error) {
	var out []Skill
	err := fs.WalkDir(ritualsFS, ritualsRoot, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		// Expect .../<parentDir>/<file>; the immediate parent must match.
		parts := strings.Split(p, "/")
		if len(parts) < 2 || parts[len(parts)-2] != parentDir {
			return nil
		}
		name := parts[len(parts)-1]
		content, readErr := fs.ReadFile(ritualsFS, p)
		if readErr != nil {
			return fmt.Errorf("reading embedded ritual %s/%s: %w", parentDir, name, readErr)
		}
		out = append(out, Skill{Name: name, Content: content})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking embedded rituals for %s: %w", parentDir, err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ListRitualAgents returns the vendored ritual agents (planner, builder,
// reviewer, deployer) that materialize flat into `.claude/agents/`.
func ListRitualAgents() ([]Skill, error) {
	return listRitualFiles("agents")
}

// ListRitualTemplates returns the vendored ritual templates (adr,
// decision, epic-spec, milestone-spec) that materialize flat into
// `.claude/templates/` (D-0015).
func ListRitualTemplates() ([]Skill, error) {
	return listRitualFiles("templates")
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

	verbSkills, err := List()
	if err != nil {
		return err
	}
	ritualSkills, err := ListRituals()
	if err != nil {
		return err
	}
	// Verb skills (aiwf-*) and ritual skills (aiwfx-*, wf-*) share the
	// `.claude/skills/` namespace and the single ownership manifest. The
	// prefixes don't overlap, so the union has no name collisions.
	skills := make([]Skill, 0, len(verbSkills)+len(ritualSkills))
	skills = append(skills, verbSkills...)
	skills = append(skills, ritualSkills...)

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

	// Agents and templates are flat single-file artifacts living in
	// their own Claude-target dirs, each with its own ownership manifest
	// (same wipe-and-rewrite contract as the skills above). Agents and
	// templates have no namespacing prefix, so ownership is tracked
	// entirely through the manifest — a user-authored file the manifest
	// never claimed is left untouched.
	agents, err := ListRitualAgents()
	if err != nil {
		return err
	}
	if err := materializeFlatFiles(root, AgentsDir, agents); err != nil {
		return err
	}
	templates, err := ListRitualTemplates()
	if err != nil {
		return err
	}
	if err := materializeFlatFiles(root, TemplatesDir, templates); err != nil {
		return err
	}
	return nil
}

// materializeFlatFiles writes flat single-file artifacts (agents,
// templates) into destDir under root, owning them via a per-dir
// `.aiwf-owned` manifest. It mirrors Materialize's contract for the
// flat case: wipe any file the prior manifest claimed that the current
// embed no longer ships, overwrite the currently-embedded files, and
// leave foreign (non-manifest) files alone. The file basename — carried
// in each Skill.Name with its `.md` suffix — is the on-disk name.
func materializeFlatFiles(root, destDir string, files []Skill) error {
	dir := filepath.Join(root, destDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", destDir, err)
	}

	prior, err := readManifest(dir)
	if err != nil {
		return err
	}
	currentSet := make(map[string]bool, len(files))
	for _, f := range files {
		currentSet[f.Name] = true
	}
	for _, name := range prior {
		if currentSet[name] {
			continue
		}
		if rmErr := os.RemoveAll(filepath.Join(dir, name)); rmErr != nil {
			return fmt.Errorf("removing previously-owned file %s: %w", name, rmErr)
		}
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Name), f.Content, 0o644); err != nil {
			return fmt.Errorf("writing %s/%s: %w", destDir, f.Name, err)
		}
	}
	return writeManifest(dir, files)
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
// materialized state and aiwf build artifacts in the consumer repo:
//   - directory wildcards that catch every materialized skill dir
//     (present and future): verb skills (`aiwf-*`) and the vendored
//     ritual skills (`aiwfx-*`, `wf-*`). The prefixes are distinct, so
//     three wildcards are needed — `aiwf-*` does not match `aiwfx-*`.
//   - the ownership manifest.
//   - `/aiwf` — a stray binary `go build ./cmd/aiwf` drops at the
//     consumer's repo root (G-0057). The leading slash anchors to
//     repo root so `cmd/aiwf/` and any future package named `aiwf`
//     stay trackable.
//
// The wildcard is what makes the .gitignore future-proof — adding a
// new embedded skill no longer requires every consumer to re-run
// `aiwf init` to refresh their .gitignore (G19).
//
// The trailing slash on the wildcard restricts the match to
// directories, so a non-aiwf file accidentally named like `aiwf-x.md`
// at that level would not be silently ignored.
//
// Agents and templates have no namespacing prefix (their basenames are
// `builder.md`, `adr.md`, …), so a directory wildcard would also mask
// user-authored files. They are therefore enumerated by exact path,
// derived from the embed (not hardcoded) so an upstream rename can't
// silently desync the gitignore from what materializes. ensureGitignore
// reconciles missing lines on every `aiwf init`/`update`, so a new
// ritual agent arriving with a binary upgrade has its line appended by
// the same `update` that materializes it.
func GitignorePatterns() ([]string, error) {
	pats := []string{
		SkillsDir + "/aiwf-*/",
		SkillsDir + "/aiwfx-*/",
		SkillsDir + "/wf-*/",
		SkillsDir + "/" + ManifestFile,
		"/aiwf",
	}
	agents, err := ListRitualAgents()
	if err != nil {
		return nil, err
	}
	for _, a := range agents {
		pats = append(pats, AgentsDir+"/"+a.Name)
	}
	pats = append(pats, AgentsDir+"/"+ManifestFile)
	templates, err := ListRitualTemplates()
	if err != nil {
		return nil, err
	}
	for _, tm := range templates {
		pats = append(pats, TemplatesDir+"/"+tm.Name)
	}
	pats = append(pats, TemplatesDir+"/"+ManifestFile)
	return pats, nil
}
