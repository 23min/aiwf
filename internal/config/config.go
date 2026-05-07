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
// to error so the pre-push hook blocks the push. Today the bumper
// covers `entity-body-empty` (M-066/AC-2); M-065's
// `milestone-tdd-undeclared` will join the same bumper when its rule
// lands. Single source of truth for the project's TDD strictness
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
func (c *Config) Validate() error {
	return nil
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
	if writeErr := os.WriteFile(path, []byte(strings.Join(out, "")), 0o644); writeErr != nil {
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
	if writeErr := os.WriteFile(path, []byte(strings.Join(out, "")), 0o644); writeErr != nil {
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
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", FileName, err)
	}
	return nil
}
