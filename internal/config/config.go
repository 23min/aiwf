// Package config loads and writes the consumer repo's `aiwf.yaml`.
//
// The file is small and deliberately so — see
// docs/pocv3/design/design-decisions.md §"aiwf.yaml config". The fields are:
//
//	hosts: [claude-code]      # optional; PoC default and only supported value
//	status_md:                # optional; opt-out for the STATUS.md auto-update
//	  auto_update: false      # default true — see StatusMdAutoUpdate
//
// Identity is runtime-derived (per `provenance-model.md`):
//   - `--actor <role>/<id>` flag on the verb wins.
//   - else `git config user.email` → `human/<localpart>`.
//   - else verb refuses with a usage error.
//
// Two legacy fields are tolerated on read for the migration window:
//   - `actor:` (pre-I2.5) — captured into LegacyActor; ignored for identity.
//     Stripped on `aiwf update`. See StripLegacyActor.
//   - `aiwf_version:` (pre-G47) — captured into LegacyAiwfVersion; was a
//     set-once pin that never auto-maintained itself, producing chronic
//     doctor noise. Stripped on `aiwf update`. See StripLegacyAiwfVersion.
//
// Validation rules:
//   - ActorPattern is the published regex for `<role>/<id>`; callers
//     that resolve identity at runtime (cmd/aiwf, initrepo) consult it.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/23min/aiwf/internal/areamatch"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/pathutil"
)

// FileName is the canonical filename at the consumer repo root.
const FileName = "aiwf.yaml"

// ErrNotFound reports that aiwf.yaml does not exist in the queried
// directory. Callers (notably resolveActor) handle this gracefully,
// since the file is optional pre-`aiwf init`.
var ErrNotFound = errors.New("aiwf.yaml not found")

// ActorPattern enforces the actor format documented in
// docs/pocv3/design/design-decisions.md: `<role>/<identifier>`, exactly one '/',
// no whitespace, neither side empty.
var ActorPattern = regexp.MustCompile(`^[^\s/]+/[^\s/]+$`)

// Config is the in-memory shape of aiwf.yaml. Hosts is omitted when
// the on-disk file leaves it absent (which is the typical case).
//
// StatusMd is the opt-out surface for the pre-commit hook that keeps
// `STATUS.md` in sync with the entity tree. Default behavior (block
// absent, or block present with `auto_update` absent) is on; an
// explicit `auto_update: false` opts out. See StatusMdAutoUpdate.
//
// LegacyActor captures any pre-I2.5 `actor:` key still present in the
// on-disk file. The value is ignored for identity resolution (which is
// runtime-derived); the field exists so `aiwf doctor` can surface a
// deprecation note pointing the user at `git config user.email`.
type Config struct {
	LegacyAiwfVersion string   `yaml:"aiwf_version,omitempty"`
	LegacyActor       string   `yaml:"actor,omitempty"`
	Hosts             []string `yaml:"hosts,omitempty"`
	StatusMd          StatusMd `yaml:"status_md,omitempty"`
	TDD               TDD      `yaml:"tdd,omitempty"`
	HTML              HTML     `yaml:"html,omitempty"`
	Allocate          Allocate `yaml:"allocate,omitempty"`
	Tree              Tree     `yaml:"tree,omitempty"`
	Archive           Archive  `yaml:"archive,omitempty"`
	Entities          Entities `yaml:"entities,omitempty"`
	Guidance          Guidance `yaml:"guidance,omitempty"`
	Areas             Areas    `yaml:"areas,omitempty"`
}

// Member is a single declared workstream area (E-0044, M-0179): a Name (the
// tag entities carry in their `area:` frontmatter) and an optional Paths list
// locating the area's source in a monorepo. Paths are validated as well-formed
// strings only at this layer — glob matching against them is deferred to
// M-0180, where the first match call site lives. A member declared in the
// legacy string form (`members: [app-a]`) decodes with Name set and Paths nil.
//
// LOCKSTEP: aiwfyaml.AreaMember mirrors this struct field-for-field (the
// comment-preserving writer is deliberately zero-dependency on config), and
// verb.RenameArea copies Member → AreaMember by hand. The two are not
// compile-linked: adding a field here means also adding it to
// aiwfyaml.AreaMember and its copy site in renamearea.go, or the new field is
// silently dropped on rename.
type Member struct {
	Name  string   `yaml:"name"`
	Paths []string `yaml:"paths,omitempty"`
}

// Areas declares the closed set of workstream area tags (E-0043, E-0044).
// Members is the closed member set the optional `area` frontmatter field
// validates against (label + optional location). Default is a DISPLAY LABEL
// ONLY for the untagged complement in grouped views — never a member of the
// tag set, never written to an entity. An empty block (no members) leaves the
// `area` field inert.
//
// Members accepts a backward-compatible dual form (E-0044, M-0179): a bare
// string (`- app-a`, the legacy E-0043 shape) or a `name`/`paths` mapping
// (`- {name: app-a, paths: [projects/app-a/**]}`). Name-consuming readers go
// through MemberNames(), the derived single source of truth for the member
// label set.
type Areas struct {
	Members []Member `yaml:"members,omitempty"`
	Default string   `yaml:"default,omitempty"`
	// Required (M-0178) opts the 1:1 monorepo into strictness: when true,
	// an untagged entity of a self-tagging root kind is a blocking
	// `area-required` (error) finding. Default false (absent) leaves the
	// pre-knob (E-0043) behavior byte-for-byte unchanged. validate()
	// rejects required:true with zero members (an unsatisfiable "every
	// entity must be a member of the empty set").
	Required bool `yaml:"required,omitempty"`
}

// MemberNames returns the declared member names in declaration order — the
// derived single source of truth every name-consuming reader (`add --area`
// validation, the `area-unknown` check, the grouping resolver, `--area`
// completion) reads. Returns nil for an empty member set, matching the prior
// `[]string` field's nil-when-absent semantics so consumers that compare
// against nil are unaffected by the label+location migration.
func (a Areas) MemberNames() []string {
	if len(a.Members) == 0 {
		return nil
	}
	names := make([]string, len(a.Members))
	for i, m := range a.Members {
		names[i] = m.Name
	}
	return names
}

// UnmarshalYAML decodes the areas block, accepting each member in either the
// legacy string form or the `name`/`paths` mapping form (E-0044, M-0179).
//
// A bare scalar member must be a YAML string (`!!str`): the explicit tag check
// keeps an unquoted `42`/`true`/`~` from silently becoming a string member,
// routing it to the malformed-member guard instead. A quoted numeric (`"42"`)
// is a string and is accepted. A mapping member decodes `name`/`paths`; an
// explicit empty `paths: []` is normalized to nil so it equals an absent
// `paths` (both express "no paths"). Decode-time errors (a non-list `paths`, a
// member node that is neither scalar nor mapping) are wrapped to name the
// offending member rather than shipping the bare yaml.v3 text. Semantic rules
// (emptiness, whitespace, uniqueness, path hygiene, default) live in validate().
func (a *Areas) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Members  []yaml.Node `yaml:"members"`
		Default  string      `yaml:"default"`
		Required bool        `yaml:"required"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	a.Default = raw.Default
	a.Required = raw.Required
	a.Members = make([]Member, 0, len(raw.Members))
	for i := range raw.Members {
		n := &raw.Members[i]
		switch n.Kind {
		case yaml.ScalarNode:
			if n.Tag != "!!str" {
				return memberKindError(i, n)
			}
			a.Members = append(a.Members, Member{Name: n.Value})
		case yaml.MappingNode:
			var m Member
			if err := n.Decode(&m); err != nil {
				return fmt.Errorf("areas.members[%d]: member %q: %w", i, memberNodeName(n), err)
			}
			if len(m.Paths) == 0 {
				// yaml.v3 decodes `paths: []` to a non-nil empty slice;
				// normalize to nil so explicit-empty equals absent.
				m.Paths = nil
			}
			a.Members = append(a.Members, m)
		default:
			return memberKindError(i, n)
		}
	}
	return nil
}

// memberKindError reports a member node that is neither a string member nor a
// name/paths mapping — a bare sequence, or a non-`!!str` scalar like `42` /
// `true` / `~`. Names the offending member by index (and value when the node
// carries one) so the operator can locate it.
func memberKindError(i int, n *yaml.Node) error {
	return fmt.Errorf("areas.members[%d]: %q is neither a string member nor a name/paths mapping", i, n.Value)
}

// memberNodeName extracts the `name` scalar from a mapping member node for
// error context, since a failed full decode leaves the decoded struct's Name
// empty. Returns "" when the mapping has no `name` key.
func memberNodeName(n *yaml.Node) string {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == "name" {
			return n.Content[i+1].Value
		}
	}
	return ""
}

// validate enforces the areas-block schema. Member names must be non-empty,
// free of leading/trailing whitespace, and unique across both declaration
// forms; each path entry must be non-empty, whitespace-clean, and a
// syntactically well-formed glob (the Tier-1 gate, M-0180 — a malformed glob
// is a hard error at load naming the bad glob, rather than being silently
// skipped by the dead-glob/overlap checks at runtime); default (if set) must
// be a non-empty, whitespace-clean label that names a non-empty member set and
// is not itself a member — it labels the untagged complement, which is
// disjoint from every declared area.
func (a Areas) validate() error {
	seen := make(map[string]bool, len(a.Members))
	for _, m := range a.Members {
		name := m.Name
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("areas.members contains an empty member")
		}
		if name != strings.TrimSpace(name) {
			return fmt.Errorf("areas.members member %q has leading or trailing whitespace", name)
		}
		// `global` is the reserved cross-cutting sentinel (ADR-0021,
		// M-0184), not a declarable member: it is one value of the area
		// dimension, never a second axis. Reject it here so the only way
		// an entity carries `area: global` is the affirmative escape valve,
		// not a member set that shadows the sentinel.
		if name == entity.AreaGlobal {
			return fmt.Errorf("areas.members may not declare the reserved %q area; it is the cross-cutting sentinel (ADR-0021)", entity.AreaGlobal)
		}
		if seen[name] {
			return fmt.Errorf("areas.members contains duplicate member %q", name)
		}
		seen[name] = true
		for _, p := range m.Paths {
			if strings.TrimSpace(p) == "" {
				return fmt.Errorf("areas.members member %q contains an empty path entry", name)
			}
			if p != strings.TrimSpace(p) {
				return fmt.Errorf("areas.members member %q path %q has leading or trailing whitespace", name, p)
			}
			// Tier-1 glob-syntax gate (M-0180): route the check through the
			// areamatch SSOT so config-load owns malformed globs and never
			// imports doublestar directly. A bad glob is a hard load error,
			// not a silently-skipped runtime no-op.
			if err := areamatch.Validate(p); err != nil {
				return fmt.Errorf("areas.members member %q path %q is not a valid glob: %w", name, p, err)
			}
		}
	}
	if a.Default != "" {
		if strings.TrimSpace(a.Default) == "" {
			return fmt.Errorf("areas.default is whitespace-only")
		}
		if a.Default != strings.TrimSpace(a.Default) {
			return fmt.Errorf("areas.default %q has leading or trailing whitespace", a.Default)
		}
		if len(a.Members) == 0 {
			return fmt.Errorf("areas.default %q is set but no members are declared", a.Default)
		}
		if seen[a.Default] {
			return fmt.Errorf("areas.default %q must not also be a member; it labels the untagged complement", a.Default)
		}
	}
	// M-0178: `required: true` asserts "every entity belongs to a declared
	// area", which is unsatisfiable with an empty member set. Mirrors the
	// `default`-needs-members rejection above.
	if a.Required && len(a.Members) == 0 {
		return fmt.Errorf("areas.required is set but no members are declared")
	}
	return nil
}

// Tree is the consumer's policy for what may live under `work/`.
// AllowPaths is a list of repo-relative glob patterns (filepath.Match
// semantics) that exempt files from the tree-discipline check —
// useful for project-specific scratch dirs or templates the consumer
// genuinely wants alongside the entity tree. Strict promotes the
// `unexpected-tree-file` finding from a warning to an error so the
// pre-push hook blocks the push.
//
// Default behavior (empty Tree block): contract artifact dirs are
// auto-exempt; everything else under work/ is reported as a warning.
// See docs/pocv3/design/tree-discipline.md.
type Tree struct {
	AllowPaths []string `yaml:"allow_paths,omitempty"`
	Strict     bool     `yaml:"strict,omitempty"`
}

// Allocate carries the consumer's id-allocator configuration. Trunk
// names the git ref the trunk-aware allocator unions into its view of
// existing ids; an empty value means "use the default trunk ref" and
// AllocateTrunkRef returns DefaultAllocateTrunk in that case.
//
// See docs/pocv3/design/id-allocation.md for the full model.
type Allocate struct {
	Trunk string `yaml:"trunk,omitempty"`
}

// DefaultAllocateTrunk is the trunk ref the allocator falls back to
// when aiwf.yaml.allocate.trunk is unset. Mirrors what `git clone`
// produces for a standard upstream project.
const DefaultAllocateTrunk = "refs/remotes/origin/main"

// AllocateTrunkRef returns the configured trunk ref (or the default)
// and whether the value was explicitly set in aiwf.yaml. The
// "explicit" bit drives the missing-ref policy: an explicitly-named
// ref that doesn't resolve is a hard error; an unconfigured default
// that doesn't resolve falls back to working-tree-only when the repo
// also has no remotes.
func (c *Config) AllocateTrunkRef() (ref string, explicit bool) {
	if c == nil || c.Allocate.Trunk == "" {
		return DefaultAllocateTrunk, false
	}
	return c.Allocate.Trunk, true
}

// TrunkBranchShortName returns the short branch name derived from the
// configured trunk ref (`AllocateTrunkRef`). It is the pure-derivation
// last-path-segment of the ref:
//
//	refs/remotes/<remote>/<name>  →  <name>
//	refs/heads/<name>             →  <name>
//
// Used by the M-0161/AC-1 verb-layer authorize carve-out so the "main +
// ritual --branch" predicate honors the operator's configured trunk
// name rather than hardcoding the literal `"main"`. Pure: no git access,
// no I/O — the config's value is the single source of truth.
//
// Returns the empty string when the configured ref does not have a
// parseable last segment (`"garbage"` with no slash, `"refs/heads/"`
// with trailing slash, etc.) — callers should treat empty as
// "no resolvable trunk name; do not match" rather than `==""` against
// an empty CurrentBranch (which would silently coincide on detached
// HEAD).
//
// Empty `allocate.trunk` (or nil receiver) falls through to
// `DefaultAllocateTrunk` (`refs/remotes/origin/main`) → returns
// `"main"`, preserving backwards-compatibility for repos that never
// configured the value.
func (c *Config) TrunkBranchShortName() string {
	ref, _ := c.AllocateTrunkRef()
	// Pure last-path-segment derivation. Works for both
	// refs/remotes/<remote>/<name> and refs/heads/<name> shapes
	// without forking on the prefix — the segment after the last
	// "/" IS the short name in either shape.
	idx := strings.LastIndex(ref, "/")
	if idx < 0 || idx == len(ref)-1 {
		// No slash, or trailing slash leaves an empty segment.
		// Return empty so the caller knows there's no resolvable
		// short-name to match.
		return ""
	}
	return ref[idx+1:]
}

// HTML holds the consumer's settings for the static-site render
// produced by `aiwf render --format=html`. OutDir is the directory
// the renderer writes into (relative to the repo root unless given
// as an absolute path); CommitOutput records the consumer's intent
// to commit the rendered files. The gitignore block managed by
// `aiwf init` / `aiwf update` is *derived* from CommitOutput — the
// consumer expresses intent here, and the framework reconciles the
// gitignore on the next admin verb run.
//
// Default OutDir: "site" — the standard SSG convention.
// Default CommitOutput: false — gitignore the output and publish
// via CI.
type HTML struct {
	OutDir       string `yaml:"out_dir,omitempty"`
	CommitOutput bool   `yaml:"commit_output,omitempty"`
}

// DefaultHTMLOutDir is the path the renderer falls back to when
// aiwf.yaml.html.out_dir is unset.
const DefaultHTMLOutDir = "site"

// HTMLOutDir returns the configured output directory or the default
// when unset. Callers should resolve to an absolute path against the
// repo root before passing to the renderer.
func (c *Config) HTMLOutDir() string {
	if c == nil || c.HTML.OutDir == "" {
		return DefaultHTMLOutDir
	}
	return c.HTML.OutDir
}

// TDD carries opt-in governance for the TDD model. RequireTestMetrics
// gates the `acs-tdd-tests-missing` warning emitted by `aiwf check`:
// when true, every AC at `tdd_phase: done` under a `tdd: required`
// milestone must have at least one commit in its history carrying an
// `aiwf-tests:` trailer or the check warns. Default false — the
// trailer is informational metadata; consumers who want stricter
// governance opt in at the project level.
//
// Strict promotes a defined set of TDD-related findings from warning
// to error so the pre-push hook blocks the push. The bumper covers
// `entity-body-empty` (M-066/AC-2) and `milestone-tdd-undeclared`
// (G-0268). Single source of truth for the project's TDD strictness
// posture — no parallel field, no second config knob. Default false.
// See check.ApplyTDDStrict for the precise set of codes covered.
type TDD struct {
	RequireTestMetrics bool `yaml:"require_test_metrics,omitempty"`
	Strict             bool `yaml:"strict,omitempty"`
}

// StatusMd carries the opt-out for the pre-commit hook that
// regenerates `STATUS.md`. AutoUpdate is a tristate via *bool:
// nil means "not specified, take the default (true)", &false is an
// explicit opt-out, &true is an explicit opt-in. Use the getter
// Config.StatusMdAutoUpdate rather than reading the pointer directly
// so callers don't have to repeat the default.
type StatusMd struct {
	AutoUpdate *bool `yaml:"auto_update,omitempty"`
}

// Archive carries the consumer's drift-control configuration for the
// per-kind archive convention (ADR-0004). SweepThreshold is a tristate
// via *int: nil means "not specified, take the default (no threshold —
// `archive-sweep-pending` stays advisory)", &N is an explicit hard
// threshold past which the aggregate finding escalates from warning
// to error. Mirrors the StatusMd.AutoUpdate tristate so "unset" is
// always distinguishable from a meaningful zero. Use the getter
// Config.ArchiveSweepThreshold rather than reading the pointer
// directly so callers don't have to repeat the default.
//
// Default behavior (empty Archive block, or absent archive.sweep_threshold):
// `archive-sweep-pending` stays advisory regardless of count. Teams
// choose their own discipline; the kernel does not nag.
type Archive struct {
	SweepThreshold *int `yaml:"sweep_threshold,omitempty"`
}

// ArchiveSweepThreshold returns the configured threshold and a bool
// indicating whether the consumer explicitly set one. When set=false,
// `aiwf check` does not escalate `archive-sweep-pending` regardless
// of count (the default-permissive behavior per ADR-0004 §"Drift
// control" layer 2). Tolerant of a nil receiver so callers in
// `cmd/aiwf/main.go` can invoke before / without a loaded Config.
func (c *Config) ArchiveSweepThreshold() (n int, set bool) {
	if c == nil || c.Archive.SweepThreshold == nil {
		return 0, false
	}
	return *c.Archive.SweepThreshold, true
}

// Entities carries the consumer's policy for entity-shape constraints
// the kernel applies when writing new entity files. TitleMaxLength
// caps the length of `--title` accepted by mutating verbs that write
// titles (`aiwf add`, `aiwf retitle`, `aiwf import`) and the length
// of `<new-slug>` accepted by `aiwf rename`. Title and slug share the
// same budget so on-disk filenames and frontmatter titles stay in
// sync — every kernel render surface (CLI tables, HTML render,
// git-log subjects, `aiwf history`, filesystem) degrades uniformly
// rather than diverging.
//
// Default behavior (empty Entities block, or absent
// entities.title_max_length): the cap is DefaultEntityTitleMaxLength
// (80 chars — the Conventional Commits subject-line convention).
// Consumers who want longer or shorter caps override here per
// G-0102.
type Entities struct {
	TitleMaxLength *int `yaml:"title_max_length,omitempty"`
}

// DefaultEntityTitleMaxLength is the kernel-default cap on entity
// title (and slug) length. 80 chars matches the Conventional Commits
// subject-line convention so an entity-touching commit subject that
// quotes the title verbatim still fits within typical commit-subject
// guidelines.
const DefaultEntityTitleMaxLength = 80

// EntityTitleMaxLength returns the configured title-length cap or
// the kernel default when unset. A non-positive configured value is
// treated as the default — the cap exists to prevent filesystem and
// table-layout pathologies, not to enable disabling them.
func (c *Config) EntityTitleMaxLength() int {
	if c == nil || c.Entities.TitleMaxLength == nil || *c.Entities.TitleMaxLength <= 0 {
		return DefaultEntityTitleMaxLength
	}
	return *c.Entities.TitleMaxLength
}

// StatusMdAutoUpdate returns whether the consumer wants the
// pre-commit hook installed and `STATUS.md` regenerated on every
// commit. Default true: the framework's opt-out, not opt-in. The
// committed `STATUS.md` is the user's content once tracked; flipping
// the flag controls whether the *hook* is installed, not whether
// the file is deleted.
func (c *Config) StatusMdAutoUpdate() bool {
	if c.StatusMd.AutoUpdate == nil {
		return true
	}
	return *c.StatusMd.AutoUpdate
}

// Guidance carries the consumer's opt-out for aiwf maintaining its
// per-turn LLM guidance import in the repo-root `CLAUDE.md` (ADR-0018).
// WireClaudeMd is a tristate via *bool mirroring StatusMd.AutoUpdate:
// nil → default (true), &false → explicit opt-out, &true → explicit
// opt-in. Use the getter Config.WireClaudeMd, not the pointer.
//
// Default behavior (empty Guidance block, or absent
// guidance.wire_claudemd): aiwf wires and self-heals the marker-wrapped
// `@.claude/aiwf-guidance.md` import on every `aiwf init` / `aiwf
// update` — the framework's opt-out, not opt-in. There is deliberately
// no CLI flag; the wiring is automatic, like skill/hook materialization.
type Guidance struct {
	WireClaudeMd *bool `yaml:"wire_claudemd,omitempty"`
}

// WireClaudeMd returns whether aiwf should maintain its guidance import
// in the consumer's `CLAUDE.md`. Default true (opt-out, not opt-in).
// Tolerant of a nil receiver so callers can invoke before / without a
// loaded Config.
func (c *Config) WireClaudeMd() bool {
	if c == nil || c.Guidance.WireClaudeMd == nil {
		return true
	}
	return *c.Guidance.WireClaudeMd
}

// Load reads aiwf.yaml from root. Returns ErrNotFound when the file is
// absent so callers can distinguish "missing config" (acceptable
// pre-init) from "malformed config" (always an error).
func Load(root string) (*Config, error) {
	path := filepath.Join(root, FileName)
	bytes, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", FileName, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	return &cfg, nil
}

// Validate enforces the documented constraints. Called by Load and
// expected to be called by Write before serialization.
//
// Identity (the actor field) is no longer stored — it's runtime-
// derived per `provenance-model.md`. Any incoming `actor:` key is
// captured by LegacyActor for the deprecation note in `aiwf doctor`,
// but is not validated here (a malformed legacy value is harmless
// since runtime resolution doesn't consult it).
//
// `aiwf_version:` is no longer required (G47). Pre-G47 yamls still
// load fine; the legacy value is captured into LegacyAiwfVersion and
// stripped on `aiwf update` via StripLegacyAiwfVersion.
//
// The areas block (E-0043) is the first cross-field constraint validated
// here; the method remains the entry point for future rules.
func (c *Config) Validate() error {
	return c.Areas.validate()
}

// StripLegacyActor removes any top-level `actor:` line from
// root/aiwf.yaml and rewrites the file in place. The strip is
// textual (line-based) rather than a YAML round-trip so user
// comments and key ordering survive — the legacy `actor:` key is
// the only field we know to be dead, and a re-marshal would
// regenerate the file in the marshaler's preferred shape.
//
// Returns (false, nil) when no `actor:` line is present (file
// stays byte-identical), (true, nil) when one was removed, or an
// error when the file is unreadable / unwritable. Idempotent:
// callers may invoke on every `aiwf update` without churn.
//
// `actor:` only matches at column 0 (i.e. a top-level YAML key).
// A nested key with an actor field name in some hypothetical
// future block would not be touched.
func StripLegacyActor(root string) (changed bool, err error) {
	path := filepath.Join(root, FileName)
	bytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", FileName, err)
	}
	content := string(bytes)
	lines := splitKeepEOL(content)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if isTopLevelActorLine(line) {
			changed = true
			continue
		}
		out = append(out, line)
	}
	if !changed {
		return false, nil
	}
	if writeErr := pathutil.AtomicWriteFile(path, []byte(strings.Join(out, "")), 0o644); writeErr != nil {
		return false, fmt.Errorf("writing %s: %w", FileName, writeErr)
	}
	return true, nil
}

// isTopLevelActorLine reports whether a single line (with or
// without trailing newline) is a top-level `actor:` key. Indented
// lines and lines where `actor` is a key inside another mapping
// are left alone — the strip targets only the legacy top-level
// field documented in pre-I2.5 aiwf.yaml.
func isTopLevelActorLine(line string) bool {
	trimmed := strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(trimmed, "actor:") {
		return false
	}
	// Reject "actorxxx:" — only a colon-or-whitespace boundary counts.
	rest := trimmed[len("actor"):]
	return strings.HasPrefix(rest, ":")
}

// StripLegacyAiwfVersion removes any top-level `aiwf_version:` line
// from root/aiwf.yaml and rewrites the file in place. Same shape as
// StripLegacyActor: textual line-based strip so user comments and
// key ordering survive.
//
// Returns (false, nil) when no line is present, (true, nil) when one
// was removed. Idempotent — callers may invoke on every `aiwf update`
// without churn.
//
// Filed under G47: the field was historically required and stamped
// at init time, but never auto-maintained, producing chronic doctor
// noise. The information is now reachable via `aiwf version` (the
// running binary) and `aiwf doctor --check-latest` (newer release
// available); the stored pin is dead weight.
func StripLegacyAiwfVersion(root string) (changed bool, err error) {
	path := filepath.Join(root, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", FileName, err)
	}
	content := string(data)
	lines := splitKeepEOL(content)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if isTopLevelAiwfVersionLine(line) {
			changed = true
			continue
		}
		out = append(out, line)
	}
	if !changed {
		return false, nil
	}
	if writeErr := pathutil.AtomicWriteFile(path, []byte(strings.Join(out, "")), 0o644); writeErr != nil {
		return false, fmt.Errorf("writing %s: %w", FileName, writeErr)
	}
	return true, nil
}

// isTopLevelAiwfVersionLine: same shape as isTopLevelActorLine, for
// the `aiwf_version:` key.
func isTopLevelAiwfVersionLine(line string) bool {
	trimmed := strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(trimmed, "aiwf_version:") {
		return false
	}
	rest := trimmed[len("aiwf_version"):]
	return strings.HasPrefix(rest, ":")
}

// splitKeepEOL splits content into lines while preserving each
// line's trailing newline, so re-joining produces byte-identical
// output for unchanged content.
func splitKeepEOL(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// Write marshals cfg to root/aiwf.yaml. Refuses to overwrite an
// existing file — callers (notably `aiwf init`) decide what to do
// when one is already there.
//
// Empty-config-only by contract. The sole caller is `aiwf init`, which
// writes an empty &Config{} (areas omitted). Write must NOT be used to
// serialize a populated areas block: yaml.Marshal would emit every Member
// in mapping form (`- name: app-a`), churning a legacy bare-string member
// and breaking the M-0179 zero-migration parity. Post-init edits to the
// areas block route through the comment-preserving aiwfyaml writer
// (aiwfyaml.SetAreas), which emits bare strings for paths-less members.
func Write(root string, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	path := filepath.Join(root, FileName)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", FileName)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("statting %s: %w", FileName, err)
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", FileName, err)
	}
	// Post-G47 a default Config has no required fields; yaml.Marshal
	// renders that as the literal "{}" document. Two consequences are
	// bad: (1) the file looks like noise to a reader, and (2) any
	// later hand-edit that *appends* a yaml block (e.g., `html: ...`)
	// produces a two-document stream where only the first ("{}") is
	// loaded, silently dropping the user's edit. Write a friendly
	// comment header instead so the file reads as intentional and
	// appended blocks are parsed by `config.Load`.
	if strings.TrimSpace(string(out)) == "{}" {
		out = []byte("# aiwf consumer-repo config. Append top-level keys (e.g. html: { commit_output: true })\n# to opt into framework features. See `aiwf doctor` and the README for the full list.\n")
	}
	if err := pathutil.AtomicWriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", FileName, err)
	}
	return nil
}
