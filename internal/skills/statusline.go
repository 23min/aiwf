package skills

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/pathutil"
)

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
	// a stale copy on every `aiwf update` rather than preserving it
	// (G-0337, superseding the earlier scaffold-once lifecycle).
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
// The on-disk script is an aiwf-owned artifact: like the materialized
// skills and hooks it is byte-refreshed on every `aiwf update` (G-0337),
// so a local edit does not survive — customize by wiring your own
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

	// Always byte-refresh: write the embed when the destination is
	// absent or its content differs. A byte-equal copy is left untouched
	// (idempotent — no needless write, no mtime churn).
	existing, readErr := os.ReadFile(dest)
	switch {
	case readErr == nil:
		if !bytes.Equal(existing, statuslineEmbed) {
			if err := pathutil.AtomicWriteFile(dest, statuslineEmbed, 0o755); err != nil { //coverage:ignore AtomicWriteFile fails only on filesystem faults (disk-full/permission); tempdir-based tests can't reproduce
				return res, fmt.Errorf("refreshing %s: %w", dest, err)
			}
			res.Wrote = true
		}
	case os.IsNotExist(readErr):
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil { //coverage:ignore MkdirAll fails only on filesystem faults; tempdir-based tests can't reproduce
			return res, fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
		}
		if err := pathutil.AtomicWriteFile(dest, statuslineEmbed, 0o755); err != nil { //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce
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
