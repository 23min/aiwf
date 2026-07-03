package skills

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/version"
)

// statuslineVersionSentinel is the placeholder in the embedded
// statusline script that RenderStatusline replaces with the binary's
// version string at materialization time (G-0344). Mirrors the
// guidance fragment's __AIWF_VERSION__ substitution.
const statuslineVersionSentinel = "__AIWF_VERSION__"

// statuslineVersionRE captures the version token from a materialized
// statusline's marker line: `# aiwf-statusline version: <token> …`.
// Presence of the line marks a copy as aiwf-managed (the analogue of
// the `# aiwf:<hook>` hook markers); the token is the installed version
// that the upgrade-only refresh (G-0344) compares against the binary.
var statuslineVersionRE = regexp.MustCompile(`(?m)^# aiwf-statusline version: (\S+).*$`)

// RenderStatusline returns the embedded statusline script with the
// version sentinel replaced by ver — the bytes aiwf materializes to a
// consumer's `.claude/statusline.sh`. Pure and deterministic; the
// scaffold and the upgrade-only refresh both write this.
func RenderStatusline(ver string) []byte {
	return bytes.ReplaceAll(statuslineEmbed, []byte(statuslineVersionSentinel), []byte(ver))
}

// InstalledStatuslineVersion parses the version token from a
// materialized statusline's marker line. ok=false means the content
// carries no aiwf marker — a legacy, hand-written, or foreign copy that
// the upgrade-only refresh must leave untouched (it can't prove the
// copy is aiwf-managed, nor order its version).
func InstalledStatuslineVersion(content []byte) (ver string, ok bool) {
	m := statuslineVersionRE.FindSubmatch(content)
	if m == nil {
		return "", false
	}
	return string(m[1]), true
}

// statuslineBody returns content with its aiwf version-marker line
// removed, so two copies that differ only in their stamped version
// compare equal. Used to isolate a genuine body edit (drift) from a
// mere version difference (which the version-relationship report
// covers separately).
func statuslineBody(content []byte) []byte {
	return statuslineVersionRE.ReplaceAll(content, nil)
}

// StatuslineBodyDrifted reports whether the on-disk script's body
// (ignoring the version-marker line) differs from the embedded copy —
// i.e. a local edit that `aiwf update --statusline` would overwrite.
func StatuslineBodyDrifted(onDisk []byte) bool {
	return !bytes.Equal(statuslineBody(onDisk), statuslineBody(statuslineEmbed))
}

// StatuslineRefreshAction classifies what plain `aiwf update` did (or
// declined to do) with one already-installed statusline copy under the
// upgrade-only auto-refresh (G-0344).
type StatuslineRefreshAction string

// StatuslineRefreshAction values.
const (
	// RefreshActionCurrent — installed version equals the binary's and
	// the body matches; nothing to do.
	RefreshActionCurrent StatuslineRefreshAction = "current"
	// RefreshActionUpgraded — the binary ships a newer version; the
	// script was rewritten to it.
	RefreshActionUpgraded StatuslineRefreshAction = "upgraded"
	// RefreshActionHealed — versions match but the body had drifted; the
	// aiwf-owned copy was restored.
	RefreshActionHealed StatuslineRefreshAction = "healed"
	// RefreshActionSkipped — left untouched: an unmarked (foreign/legacy)
	// copy, an installed version newer than the binary (never a blind
	// downgrade), or versions that can't be ordered (a dev/pre-release
	// build). Detail says which.
	RefreshActionSkipped StatuslineRefreshAction = "skipped"
)

// StatuslineRefreshOutcome reports the upgrade-only auto-refresh result
// for one installed copy. Emitted only for copies that exist on disk.
type StatuslineRefreshOutcome struct {
	Path   string
	Scope  StatuslineScope
	Action StatuslineRefreshAction
	// Detail is a short human explanation (version transition or the
	// reason for a skip). For RefreshActionCurrent it carries the current
	// version but is not displayed (LedgerLine reports show=false).
	Detail string
}

// LedgerLine renders the outcome as a one-line `aiwf update` ledger
// entry, returning show=false for an already-current copy so the common
// (unchanged) path stays quiet. Pure so the Current-skip is unit-testable
// without a tagged binary equal to the installed version (which an
// in-process (devel) test can never produce).
func (o StatuslineRefreshOutcome) LedgerLine() (line string, show bool) {
	if o.Action == RefreshActionCurrent {
		return "", false
	}
	return fmt.Sprintf("  %-9s  statusline (%s scope)  (%s)", o.Action, o.Scope, o.Detail), true
}

// decideStatuslineRefresh is the pure upgrade-only decision (G-0344):
// given the binary's version, the version parsed from the installed
// copy (marked=false when the marker is absent), and whether the body
// drifted, it returns the action and a human detail. It never
// downgrades and never acts when the two versions can't be ordered —
// version.Compare returns SkewUnknown for any dev/pseudo/pre-release
// value, which the safe default (skip) handles.
func decideStatuslineRefresh(binary, installed version.Info, marked, bodyDrifted bool) (action StatuslineRefreshAction, detail string) {
	if !marked {
		return RefreshActionSkipped, "unmarked copy (no aiwf version marker) — run `aiwf update --statusline` once to adopt versioned refresh"
	}
	switch version.Compare(binary, installed) {
	case version.SkewAhead:
		return RefreshActionUpgraded, fmt.Sprintf("%s → %s", installed.Version, binary.Version)
	case version.SkewBehind:
		return RefreshActionSkipped, fmt.Sprintf("installed %s is newer than this binary %s — not downgrading", installed.Version, binary.Version)
	case version.SkewUnknown:
		return RefreshActionSkipped, fmt.Sprintf("cannot order versions (installed %q, binary %q) — not auto-refreshing", installed.Version, binary.Version)
	default: // SkewEqual
		if bodyDrifted {
			return RefreshActionHealed, "restored aiwf-owned copy after a local edit"
		}
		return RefreshActionCurrent, binary.Version
	}
}

// AutoRefreshStatusline applies the upgrade-only auto-refresh (G-0344)
// to any already-installed statusline copy, at both the user
// (`$HOME/.claude/statusline.sh`) and project
// (`<root>/.claude/statusline.sh`) destinations. It refreshes only an
// aiwf-marked copy, only when the binary's version is newer-or-equal to
// the installed stamp, and never creates a copy or touches any settings
// file — that stays behind the explicit `--statusline` opt-in. The home
// directory is resolved via os.UserHomeDir(); tests pass it explicitly
// via AutoRefreshStatuslineForVersion.
func AutoRefreshStatusline(root string) ([]StatuslineRefreshOutcome, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving user home directory: %w", err) //coverage:ignore os.UserHomeDir fails only when $HOME is unset; not reproducible in the test env (mirrors the ScaffoldStatusline sibling)
	}
	return AutoRefreshStatuslineForVersion(root, home, version.Current())
}

// AutoRefreshStatuslineForVersion is AutoRefreshStatusline with the
// home directory and binary version injected — the testable core, so
// tests can drive the upgrade / downgrade / equal / unknown matrix
// without a real binary version or `$HOME`.
func AutoRefreshStatuslineForVersion(root, home string, binary version.Info) ([]StatuslineRefreshOutcome, error) {
	rendered := RenderStatusline(binary.Version)

	type candidate struct {
		path  string
		scope StatuslineScope
	}
	var candidates []candidate
	if home != "" {
		candidates = append(candidates, candidate{filepath.Join(home, statuslineRelPath), StatuslineScopeUser})
	}
	candidates = append(candidates, candidate{filepath.Join(root, statuslineRelPath), StatuslineScopeProject})

	var outcomes []StatuslineRefreshOutcome
	for _, c := range candidates {
		onDisk, err := os.ReadFile(c.path)
		if os.IsNotExist(err) {
			continue // not installed at this scope — nothing to refresh
		}
		if err != nil {
			return outcomes, fmt.Errorf("reading %s: %w", c.path, err)
		}
		installedRaw, marked := InstalledStatuslineVersion(onDisk)
		bodyDrifted := !bytes.Equal(onDisk, rendered)
		action, detail := decideStatuslineRefresh(binary, version.Parse(installedRaw), marked, bodyDrifted)
		if action == RefreshActionUpgraded || action == RefreshActionHealed {
			if err := pathutil.AtomicWriteFile(c.path, rendered, 0o755); err != nil { //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce
				return outcomes, fmt.Errorf("refreshing %s: %w", c.path, err)
			}
		}
		outcomes = append(outcomes, StatuslineRefreshOutcome{Path: c.path, Scope: c.scope, Action: action, Detail: detail})
	}
	return outcomes, nil
}

// StatuslineScope names which Claude Code settings tree a `--statusline`
// scaffold targets. Closed set; an unknown value is a usage error.
type StatuslineScope string

const (
	// StatuslineScopeProject writes to `<root>/.claude/statusline.sh`
	// (per-repo) and appends `.claude/statusline.sh` to `<root>/.gitignore`
	// idempotently. The activation snippet's command is
	// `${CLAUDE_PROJECT_DIR:-<root>}/.claude/statusline.sh` — anchored on the
	// repo root, not the cwd, so it resolves from a git worktree (aiwf's own
	// ritual working dir) where a cwd-relative path would not (G-0337). This
	// is the explicit opt-in scope; user scope is the default.
	StatuslineScopeProject StatuslineScope = "project"

	// StatuslineScopeUser writes to `<home>/.claude/statusline.sh` (per-user,
	// shared across every project) and does not touch any gitignore. The
	// activation snippet's command is `$HOME/.claude/statusline.sh` — anchored
	// on `$HOME`, which is POSIX-guaranteed and per-environment-correct, so a
	// single settings file resolves on both the host and inside a container
	// that shares `~/.claude` via a mount. Independent of repo location,
	// worktree, and mount layout — the default scope (G-0337).
	StatuslineScopeUser StatuslineScope = "user"
)

// statuslineRelPath is the on-disk relative path the scaffold writes
// to, identical for project and user scope (the difference is just
// what the rel-path is anchored against — repo root vs home).
const statuslineRelPath = ".claude/statusline.sh"

// StatuslineScaffoldResult reports what `ScaffoldStatusline` did.
type StatuslineScaffoldResult struct {
	// Path is the absolute destination path. Always set, even when
	// Wrote is false (so a caller can include it in a "script already
	// current at <Path>" message).
	Path string

	// Wrote is true when the script was written or refreshed on this
	// invocation. False when an already-current copy (byte-equal to the
	// embed) was left untouched. The scaffold is idempotent: it refreshes
	// a stale copy on every `aiwf update --statusline` rather than
	// preserving it (G-0337, superseding the earlier scaffold-once lifecycle).
	Wrote bool

	// GitignoreAppended is true when `.claude/statusline.sh` was added
	// to `<root>/.gitignore` on this invocation. False when the line
	// was already present (idempotent re-run) or the scope is User
	// (user scope lives outside any repo's tracked tree).
	GitignoreAppended bool

	// Command is the `statusLine.command` value for the wired settings —
	// `$HOME/.claude/statusline.sh` for user scope,
	// `${CLAUDE_PROJECT_DIR:-<root>}/.claude/statusline.sh` for project.
	// The single source of truth for the command string; callers consume
	// this rather than re-deriving it (G-0337).
	Command string

	// Snippet is the activation snippet the operator pastes into their
	// Claude Code settings file, wrapping Command in the JSON-ish block.
	Snippet string
}

// ScaffoldStatusline materializes the embedded statusline script to the
// scope-appropriate destination, refreshing a stale copy in place. The
// user's home directory is resolved via `os.UserHomeDir()`; tests that
// need a deterministic home use `ScaffoldStatuslineWithHome`.
//
// The on-disk script is an aiwf-owned artifact, byte-refreshed to the
// embedded copy on every `aiwf update --statusline` (G-0337) — so a
// local edit does not survive; customize by wiring your own
// `statusLine.command`, not by editing this copy.
func ScaffoldStatusline(root string, scope StatuslineScope) (StatuslineScaffoldResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return StatuslineScaffoldResult{}, fmt.Errorf("resolving user home directory: %w", err)
	}
	return ScaffoldStatuslineWithHome(root, home, scope)
}

// ScaffoldStatuslineWithHome is ScaffoldStatusline with the user-home
// directory passed in explicitly. Exposed for tests so they can pin a
// fresh `t.TempDir` as home without setting `$HOME` (which would
// require a serial test under `t.Parallel`).
func ScaffoldStatuslineWithHome(root, home string, scope StatuslineScope) (StatuslineScaffoldResult, error) {
	dest, snippetCmd, err := statuslineDest(root, home, scope)
	if err != nil {
		return StatuslineScaffoldResult{}, err
	}

	res := StatuslineScaffoldResult{
		Path:    dest,
		Command: snippetCmd,
		Snippet: FormatStatuslineSnippet(snippetCmd),
	}

	// The explicit `--statusline` path always refreshes to *this*
	// binary's version — the operator asked for this binary, so a
	// downgrade here is their deliberate act (unlike the passive
	// upgrade-only auto-refresh on a plain `aiwf update`; G-0344).
	rendered := RenderStatusline(version.Current().Version)

	// Byte-refresh: write when the destination is absent or its content
	// differs from the rendered copy. A byte-equal copy is left
	// untouched (idempotent — no needless write, no mtime churn).
	existing, readErr := os.ReadFile(dest)
	switch {
	case readErr == nil:
		if !bytes.Equal(existing, rendered) {
			if err := pathutil.AtomicWriteFile(dest, rendered, 0o755); err != nil { //coverage:ignore AtomicWriteFile fails only on filesystem faults (disk-full/permission); tempdir-based tests can't reproduce
				return res, fmt.Errorf("refreshing %s: %w", dest, err)
			}
			res.Wrote = true
		}
	case os.IsNotExist(readErr):
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil { //coverage:ignore MkdirAll fails only on filesystem faults; tempdir-based tests can't reproduce
			return res, fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
		}
		if err := pathutil.AtomicWriteFile(dest, rendered, 0o755); err != nil { //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce
			return res, fmt.Errorf("writing %s: %w", dest, err)
		}
		res.Wrote = true
	default:
		return res, fmt.Errorf("reading %s: %w", dest, readErr)
	}

	if scope == StatuslineScopeProject {
		appended, gErr := ensureStatuslineGitignoreEntry(root)
		if gErr != nil { //coverage:ignore ensureStatuslineGitignoreEntry fails only on filesystem faults; tempdir-based tests can't reproduce
			return res, gErr
		}
		res.GitignoreAppended = appended
	}
	return res, nil
}

// StatuslineDestForScope resolves the absolute on-disk script path and
// the `statusLine.command` value for scope, without writing anything —
// the read-only counterpart ScaffoldStatuslineWithHome uses internally,
// exposed for callers (like `aiwf update --remove`) that need to locate
// a scope's wiring without scaffolding it.
func StatuslineDestForScope(root, home string, scope StatuslineScope) (dest, cmdPath string, err error) {
	return statuslineDest(root, home, scope)
}

// StatuslineScriptStatus is the read-only inspection of the statusline
// script at dest — G-0354's precondition check for `aiwf update
// --remove`. It never deletes anything, so a caller can inspect both
// the script and the settings key *before* deciding whether either
// mutation is authorized (see RemoveStatuslineScriptFile).
//
//   - existed reports whether a file was present at dest.
//   - aiwfAuthored reports whether it carries the aiwf version marker
//     (`# aiwf-statusline version: …`).
func StatuslineScriptStatus(dest string) (existed, aiwfAuthored bool, err error) {
	content, readErr := os.ReadFile(dest)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("reading %s: %w", dest, readErr)
	}
	_, marked := InstalledStatuslineVersion(content)
	return true, marked, nil
}

// RemoveStatuslineScriptFile deletes dest unconditionally — the caller
// (RunStatuslineRemove) must have already authorized this via
// StatuslineScriptStatus (aiwf-authored, or an operator --force) before
// calling. No-op (removed=false) when dest doesn't exist, so it's safe
// to call even when the inspection already reported nothing to do.
func RemoveStatuslineScriptFile(dest string) (removed bool, err error) {
	if err := os.Remove(dest); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("removing %s: %w", dest, err)
	}
	return true, nil
}

// statuslineDest resolves the absolute on-disk destination path and the
// `statusLine.command` string the activation snippet should carry, based
// on the scope.
func statuslineDest(root, home string, scope StatuslineScope) (dest, snippetCmd string, err error) {
	switch scope {
	case StatuslineScopeProject:
		return filepath.Join(root, statuslineRelPath), ProjectStatuslineCommand(root), nil
	case StatuslineScopeUser:
		return filepath.Join(home, statuslineRelPath), UserStatuslineCommand(), nil
	default:
		return "", "", fmt.Errorf("unknown --scope %q (want %q or %q)", scope, StatuslineScopeProject, StatuslineScopeUser)
	}
}

// ProjectStatuslineCommand returns the project-scope `statusLine.command`
// value. It anchors on `$CLAUDE_PROJECT_DIR` (Claude Code sets it to the
// project root) and falls back to the absolute install-time root, so it
// resolves from a git worktree where a cwd-relative path would not
// (G-0337). The command runs in a shell, so the `${VAR:-default}`
// expansion is evaluated. Single source of truth for the string —
// doctor's remediation hint reuses it.
func ProjectStatuslineCommand(root string) string {
	return fmt.Sprintf("${CLAUDE_PROJECT_DIR:-%s}/%s", root, statuslineRelPath)
}

// UserStatuslineCommand returns the user-scope `statusLine.command`
// value: `$HOME/.claude/statusline.sh`. `$HOME` is POSIX-guaranteed and
// per-environment-correct, so one settings file (e.g. a `~/.claude`
// shared between host and container via a mount) resolves on both sides
// (G-0337).
func UserStatuslineCommand() string {
	return "$HOME/" + statuslineRelPath
}

// FormatStatuslineSnippet renders the JSON-ish activation snippet
// for the operator's Claude Code settings file.
func FormatStatuslineSnippet(cmdPath string) string {
	return fmt.Sprintf(`  "statusLine": {
    "type": "command",
    "command": "%s"
  }`, cmdPath)
}

// ensureStatuslineGitignoreEntry adds `.claude/statusline.sh` to
// `<root>/.gitignore` if it is not already present. Returns
// appended=true only when a new line was added (idempotent re-run is
// a no-op and reports false).
//
// Deliberately separate from `GitignorePatterns()` — that set is the
// *unconditional* reconciliation list that `ensureGitignore` rewrites
// on every bare `aiwf update`. The statusline ignore line belongs to
// the per-install `--statusline` path only, so a consumer who never
// opts in keeps a clean .gitignore. M-0155/AC-4 pins this.
func ensureStatuslineGitignoreEntry(root string) (appended bool, err error) {
	path := filepath.Join(root, ".gitignore")
	existing, rErr := os.ReadFile(path)
	if rErr != nil && !os.IsNotExist(rErr) {
		return false, fmt.Errorf("reading %s: %w", path, rErr)
	}
	want := statuslineRelPath
	// Already present (anywhere, on its own line)? No-op.
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == want {
			return false, nil
		}
	}
	var b strings.Builder
	b.Write(existing)
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		b.WriteByte('\n')
	}
	b.WriteString(want)
	b.WriteByte('\n')
	if err := pathutil.AtomicWriteFile(path, []byte(b.String()), 0o644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}
