// Package skills renders and materializes aiwf's embedded workflow artifacts
// into the selected host's layout.
//
// The skill markdown lives under embedded/ and is compiled into the
// binary via go:embed. The on-disk skill files are a cache, not state:
// `aiwf init` and `aiwf update` rewrite every owned SKILL.md
// from the embed.
//
// Ownership is tracked by an on-disk manifest at
// `.aiwf-owned` in each generated family. Materialize retires only
// previously owned files, preserving supplemental skill content and
// foreign siblings. Unowned name collisions and linked output paths
// are refused before any family is written. A temporary hash receipt
// makes interrupted first writes recoverable without claiming foreign files.
//
// `aiwf doctor` consumes List() to byte-compare the on-disk files
// against the rendered content and report drift.
package skills

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
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

// ritualsFS holds canonical ritual sources (ADR-0016). Ritual skills
// materialize flattened alongside verb skills; agents and templates
// materialize into their own target directories.
//
//go:embed embedded-rituals
var ritualsFS embed.FS

// ritualsRoot is the embed path of the vendored rituals snapshot.
const ritualsRoot = "embedded-rituals"

// statuslineEmbed holds the aiwf-aware Claude Code statusline script
// (E-0039, M-0155). Unlike the skill artifacts above, the statusline is
// embedded as a single file (one shell script, not a tree) and carries a
// `__AIWF_VERSION__` marker sentinel that RenderStatusline substitutes at
// materialization time (G-0344) — so the embed is the single source of
// truth and no pre-rendered copy is tracked.
//
// Lifecycle (G-0337, G-0344): the explicit `--statusline` install path
// always refreshes the script to this binary's rendered version; a plain
// `aiwf update` upgrade-only auto-refreshes an already-installed marked
// copy but never below its stamped version and never creates one. Both
// paths write via RenderStatusline, never these raw bytes.
//
//go:embed embedded-statusline/statusline.sh
var statuslineEmbed []byte

// StatuslineBytes returns the raw embedded statusline script, with the
// `__AIWF_VERSION__` sentinel unsubstituted. The returned slice is the
// same shared backing array on every call — callers must treat it as
// read-only. Callers that materialize a copy want RenderStatusline (which
// stamps the version); this raw form is for content/shape assertions.
func StatuslineBytes() []byte {
	return statuslineEmbed
}

// Skill is one embedded document: its artifact name and content bytes.
// Skills use directory names; flat artifacts include their file extension.
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

// HooksDir is the host-relative directory consent-gated hook scripts
// (ADR-0032) materialize into. Sibling of SkillsDir/AgentsDir/TemplatesDir;
// pre-dates aiwf's own management as the home of the hand-authored
// validate-agent-isolation.sh (G-0099), a natural future registry entry.
const HooksDir = ".claude/hooks"

// SharedSettingsRelPath is the repo-relative path to Claude Code's shared
// settings file — the target ADR-0032 hooks wire into. Never the
// personal, gitignored settings.local.json SettingsPathForScope's project
// scope targets: a hook's consent decision is committed and shared, so
// every clone must see the same wiring.
const SharedSettingsRelPath = ".claude/settings.json"

// Target names an agent's on-disk layout: the host-relative dirs each
// materializable artifact kind writes into. It is the seam (ADR-0014 §4)
// that lets a non-Claude agent become a new value rather than a rewrite —
// Codex writes SKILL.md to `.agents/skills/`. An empty AgentsDir means
// aiwf does not install role cards for this target; the host may still support
// native subagents. An empty HooksDir likewise means no host-hook output.
type Target struct {
	Name         string // display name, e.g. "claude"
	SkillsDir    string // host-relative skills dir (dir-per-skill)
	AgentsDir    string // host-relative agents dir (flat); "" = no agents
	TemplatesDir string // host-relative templates dir (flat)
	HooksDir     string // host-relative hooks dir (flat); "" = no hooks
}

// ClaudeTarget pins the `.claude/{skills,agents,templates,hooks}` layout
// used by Materialize and lifecycle entry points when Claude is selected.
var ClaudeTarget = Target{
	Name:         "claude",
	SkillsDir:    SkillsDir,
	AgentsDir:    AgentsDir,
	TemplatesDir: TemplatesDir,
	HooksDir:     HooksDir,
}

// ManifestFile records owned SKILL.md parent names or flat artifact filenames,
// one safe basename per line. Each generated family has its own record.
const ManifestFile = ".aiwf-owned"

// ProvenanceReadme is the human-readable provenance note written at the
// root of the materialized skills dir. It surfaces, at-a-glance, that
// the listed adapters are aiwf-managed (not hand-authored) and refreshed
// by `aiwf update` — the discoverability surface that the `.aiwf-owned`
// manifest provides only mechanically. Provenance is a separate axis
// from a skill's name, so this note carries it without renaming any
// upstream-authored skill (which must stay byte-verbatim per the
// vendored-snapshot drift guard).
const ProvenanceReadme = "README.md"

// provenanceReadmeIntro and provenanceReadmeOutro surround the selected
// target's support-file references in ProvenanceReadme. Written for a
// consumer audience (the consumer cannot re-vendor — a newer ritual set
// arrives via `aiwf upgrade`). Version-agnostic on purpose so it needs
// no dependency on the version package; it points at `aiwf doctor` for
// the live version.
const provenanceReadmeIntro = `# aiwf-managed adapters

The directories listed in ` + "`.aiwf-owned`" + ` are **materialized by ` + "`aiwf`" + `** from a
pinned snapshot embedded in the ` + "`aiwf`" + ` binary — they are not hand-authored:

- ` + "`aiwf-*`" + `  — kernel verb skills
- ` + "`aiwfx-*`" + ` — planning / lifecycle rituals (coupled to the aiwf kernel)
- ` + "`wf-*`" + `    — generic engineering rituals (TDD, code review, doc-lint, patch)

`

const provenanceReadmeOutro = `

**Do not hand-edit these.** ` + "`aiwf update`" + ` overwrites them from the binary's
embedded snapshot; ` + "`aiwf upgrade`" + ` pulls a newer ritual version (it always
equals the binary version). Run ` + "`aiwf doctor`" + ` to see the version and confirm
they are in sync.

Anything you author yourself — a skill / agent / template outside those
prefixes, or one the ` + "`.aiwf-owned`" + ` manifest never claimed — is never touched.

` + provenanceOwnershipMarker + `
`

// List returns every verb skill rendered for Claude, in name-sorted order.
// Callers own the returned bytes. Materialization and drift checks use the
// same rendering contract; embedded sources may contain unresolved bindings.
func List() ([]Skill, error) {
	sources, err := listVerbSources()
	if err != nil {
		return nil, err
	}
	return RenderSkills(sources, ClaudeRenderBindings())
}

func listVerbSources() ([]Skill, error) {
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
// name-sorted order, rendered for Claude from the canonical
// `embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md` tree. The
// plugin wrapper is flattened away: Name is the skill directory name,
// which is what materializes under `.claude/skills/`. Agents and
// templates living in the same snapshot are intentionally not returned —
// only files literally named SKILL.md under a `skills/` parent qualify.
func ListRituals() ([]Skill, error) {
	sources, err := listRitualSources()
	if err != nil {
		return nil, err
	}
	return RenderSkills(sources, ClaudeRenderBindings())
}

func listRitualSources() ([]Skill, error) {
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

// ListRitualAgents returns ritual agents rendered for Claude, before optional
// model/effort tier injection. They materialize flat into `.claude/agents/`.
func ListRitualAgents() ([]Skill, error) {
	sources, err := listRitualFiles("agents")
	if err != nil {
		return nil, err
	}
	return RenderSkills(sources, ClaudeRenderBindings())
}

// ListRitualTemplates returns templates rendered for Claude that materialize
// flat into `.claude/templates/` (D-0015).
func ListRitualTemplates() ([]Skill, error) {
	sources, err := listRitualFiles("templates")
	if err != nil {
		return nil, err
	}
	return RenderSkills(sources, ClaudeRenderBindings())
}

// Materialize writes the embedded skills into `.claude/skills/<name>/`
// under root. Retires obsolete owned SKILL.md files and removes their
// directories only when empty. Foreign content is preserved, including
// supplemental files in owned skill directories. Unowned collisions,
// invalid ownership records, and linked output paths cause a refusal.
//
// This is the operation behind both `aiwf init` (first-time setup) and
// `aiwf update` (refresh after a binary upgrade).
func Materialize(root string) error {
	return materializeTo(root, ClaudeTarget, nil)
}

// MaterializeWithTiers is Materialize with per-agent model/effort tiers
// (G-0353) injected into the materialized agent-card frontmatter. initrepo
// derives tiers from aiwf.yaml's `agents:` block and passes them on every
// init/update; callers that don't tier use Materialize (nil tiers leaves the
// embedded card frontmatter unchanged).
func MaterializeWithTiers(root string, tiers map[string]AgentTier) error {
	return materializeTo(root, ClaudeTarget, tiers)
}

// AgentTier pins the model and reasoning effort a materialized agent card is
// written with (G-0353). An empty field is omitted from the card frontmatter,
// leaving that dimension inheriting the session default. It is the skills-layer
// mirror of config.Agent, kept here so skills stays free of a config import.
type AgentTier struct {
	Model  string
	Effort string
}

// AgentNames returns the base names (no ".md") of the shipped role agents, in
// sorted order. initrepo uses it to flag aiwf.yaml `agents:` keys that match no
// shipped agent, since config cannot enumerate them without importing skills.
func AgentNames() ([]string, error) {
	agents, err := ListRitualAgents()
	if err != nil {
		return nil, err //coverage:ignore ListRitualAgents walks a compiled-in embed; it cannot fail at runtime
	}
	names := make([]string, 0, len(agents))
	for _, a := range agents {
		names = append(names, strings.TrimSuffix(a.Name, ".md"))
	}
	sort.Strings(names)
	return names, nil
}

// applyAgentTiers returns cards with each card whose name (sans ".md") has a
// tier entry rewritten to carry the tier's model/effort in its frontmatter. A
// nil/empty tiers map returns cards unchanged; a card with no matching entry is
// left untouched. Pure: the input slice's card content is not mutated.
func applyAgentTiers(cards []Skill, tiers map[string]AgentTier) []Skill {
	if len(tiers) == 0 {
		return cards
	}
	out := make([]Skill, len(cards))
	for i, c := range cards {
		out[i] = c
		if tier, ok := tiers[strings.TrimSuffix(c.Name, ".md")]; ok {
			out[i].Content = injectAgentFrontmatter(c.Content, tier)
		}
	}
	return out
}

// injectAgentFrontmatter rewrites a card's YAML frontmatter to carry the tier's
// model/effort keys. It drops any existing top-level model:/effort: lines first,
// so the result is idempotent and independent of whether the embedded card
// already carried them. Empty tier fields are not written. The card is returned
// unchanged when it has no leading `---` frontmatter fence or the fence is
// unterminated. Assumes LF line endings (the embedded cards are LF).
func injectAgentFrontmatter(content []byte, tier AgentTier) []byte {
	if tier.Model == "" && tier.Effort == "" {
		return content
	}
	s := string(content)
	if !strings.HasPrefix(s, "---\n") {
		return content
	}
	lines := strings.Split(s, "\n")
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		return content
	}
	out := make([]string, 0, len(lines)+2)
	out = append(out, lines[0]) // opening "---"
	for i := 1; i < closeIdx; i++ {
		// Drop only column-0 model:/effort: keys — an indented occurrence is a
		// nested mapping value, not the top-level field this function manages.
		if strings.HasPrefix(lines[i], "model:") || strings.HasPrefix(lines[i], "effort:") {
			continue
		}
		out = append(out, lines[i])
	}
	if tier.Model != "" {
		out = append(out, "model: "+tier.Model)
	}
	if tier.Effort != "" {
		out = append(out, "effort: "+tier.Effort)
	}
	out = append(out, lines[closeIdx:]...) // closing "---" and the body
	return []byte(strings.Join(out, "\n"))
}

// MaterializeTo is Materialize parameterized by agent target (ADR-0014
// §4). It writes the skills (dir-per-skill) into target.SkillsDir, and
// the agents and templates (flat) into target.AgentsDir /
// target.TemplatesDir. A target with an empty AgentsDir materializes no
// agents. This is an artifact-layout operation; host selection and native
// operational instructions are separate responsibilities.
func MaterializeTo(root string, target Target) error {
	return materializeTo(root, target, nil)
}

// materializeTo is the shared implementation behind Materialize, MaterializeTo,
// and MaterializeWithTiers. tiers (may be nil) inject per-agent model/effort
// into the agent-card frontmatter before the flat agent files are written.
func materializeTo(root string, target Target, tiers map[string]AgentTier) error {
	sources, err := loadArtifactSources()
	if err != nil {
		return err
	}
	return materializeArtifacts(root, target, tiers, sources)
}

func loadArtifactSources() (artifactSources, error) {
	verbSkills, err := listVerbSources()
	if err != nil {
		return artifactSources{}, err
	}
	ritualSkills, err := listRitualSources()
	if err != nil {
		return artifactSources{}, err
	}
	agents, err := listRitualFiles("agents")
	if err != nil {
		return artifactSources{}, err
	}
	templates, err := listRitualFiles("templates")
	if err != nil {
		return artifactSources{}, err
	}
	// Verb skills (aiwf-*) and ritual skills (aiwfx-*, wf-*) share the
	// `.claude/skills/` namespace and the single ownership manifest. The
	// prefixes don't overlap, so the union has no name collisions.
	skills := make([]Skill, 0, len(verbSkills)+len(ritualSkills))
	skills = append(skills, verbSkills...)
	skills = append(skills, ritualSkills...)
	return artifactSources{skills: skills, agents: agents, templates: templates}, nil
}

type artifactSources struct {
	skills    []Skill
	agents    []Skill
	templates []Skill
}

func materializeArtifacts(root string, target Target, tiers map[string]AgentTier, sources artifactSources) error {
	families, err := renderArtifactFamilies(target, tiers, sources)
	if err != nil {
		return err
	}
	return materializeArtifactFamilies(context.Background(), root, families)
}

func renderArtifactFamilies(target Target, tiers map[string]AgentTier, sources artifactSources) ([]artifactFamily, error) {
	if target.AgentsDir == "" {
		sources.agents = nil
	}
	// Validate the complete selected set before creating directories, deleting
	// obsolete owned artifacts, or replacing any file. MaterializeTo also
	// accepts a caller-supplied layout.
	bindings := renderBindingsForTarget(target)
	all := make([]Skill, 0, len(sources.skills)+len(sources.agents)+len(sources.templates))
	all = append(all, sources.skills...)
	all = append(all, sources.agents...)
	all = append(all, sources.templates...)
	rendered, err := RenderSkills(all, bindings)
	if err != nil {
		return nil, err
	}
	skills := rendered[:len(sources.skills)]
	agentsEnd := len(sources.skills) + len(sources.agents)
	agents := applyAgentTiers(rendered[len(sources.skills):agentsEnd], tiers)
	templates := rendered[agentsEnd:]
	provenance, err := renderProvenance(target)
	if err != nil {
		return nil, err
	}

	families := []artifactFamily{{dir: target.SkillsDir, skills: true, files: skills, provenance: provenance}}
	if target.AgentsDir != "" {
		families = append(families, artifactFamily{dir: target.AgentsDir, files: agents})
	}
	families = append(families, artifactFamily{dir: target.TemplatesDir, files: templates})
	return families, nil
}

// GitignorePatterns returns the .gitignore lines that mask aiwf-
// materialized state and aiwf build artifacts in the consumer repo:
//   - directory wildcards that catch every materialized skill dir
//     (present and future): verb skills (`aiwf-*`) and the vendored
//     ritual skills (`aiwfx-*`, `wf-*`). The prefixes are distinct, so
//     three wildcards are needed — `aiwf-*` does not match `aiwfx-*`.
//   - the ownership manifest and temporary recovery receipt.
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
	return GitignorePatternsFor(ClaudeTarget)
}

// GitignorePatternsFor enumerates derivable files for one host layout.
// Root instruction files stay user-owned and eligible for version control.
func GitignorePatternsFor(target Target) ([]string, error) {
	pats := []string{
		target.SkillsDir + "/aiwf-*/",
		target.SkillsDir + "/aiwfx-*/",
		target.SkillsDir + "/wf-*/",
		target.SkillsDir + "/" + ManifestFile,
		target.SkillsDir + "/" + PendingManifestFile,
		target.SkillsDir + "/" + ProvenanceReadme,
		"/aiwf",
	}
	if target.Name == ClaudeTarget.Name {
		pats = append(pats, GuidanceFile)
	}
	if target.AgentsDir != "" {
		agents, err := ListRitualAgents()
		if err != nil {
			return nil, err
		}
		for _, a := range agents {
			pats = append(pats, target.AgentsDir+"/"+a.Name)
		}
		pats = append(pats, target.AgentsDir+"/"+ManifestFile, target.AgentsDir+"/"+PendingManifestFile)
	}
	templates, err := ListRitualTemplates()
	if err != nil {
		return nil, err
	}
	for _, tm := range templates {
		pats = append(pats, target.TemplatesDir+"/"+tm.Name)
	}
	pats = append(pats, target.TemplatesDir+"/"+ManifestFile, target.TemplatesDir+"/"+PendingManifestFile)
	return pats, nil
}
