package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StatuslineScope names which Claude Code settings tree a `--statusline`
// scaffold targets. Closed set; an unknown value is a usage error.
type StatuslineScope string

const (
	// StatuslineScopeProject writes to `<root>/.claude/statusline.sh`
	// (per-repo) and appends `.claude/statusline.sh` to `<root>/.gitignore`
	// idempotently. The activation snippet's command path is repo-relative,
	// so the snippet works regardless of where Claude Code launches inside
	// the repo.
	StatuslineScopeProject StatuslineScope = "project"

	// StatuslineScopeUser writes to `<home>/.claude/statusline.sh` (per-user,
	// shared across every project) and does not touch any gitignore. The
	// activation snippet's command path is absolute, so the snippet works
	// from any worktree of any repo in the same (dev)container.
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
	// present at <Path>; left untouched" message).
	Path string

	// Wrote is true when the script was actually written. False when a
	// pre-existing file was preserved verbatim (the scaffold-once
	// lifecycle's load-bearing guard).
	Wrote bool

	// GitignoreAppended is true when `.claude/statusline.sh` was added
	// to `<root>/.gitignore` on this invocation. False when the line
	// was already present (idempotent re-run) or the scope is User
	// (user scope lives outside any repo's tracked tree).
	GitignoreAppended bool

	// Snippet is the activation snippet the operator pastes into their
	// Claude Code settings file. Repo-relative path for project scope,
	// absolute path for user scope.
	Snippet string
}

// ScaffoldStatusline materializes the embedded statusline script to
// the scope-appropriate destination if no copy already exists there.
// The user's home directory is resolved via `os.UserHomeDir()`; tests
// that need a deterministic home use `ScaffoldStatuslineWithHome`.
//
// Per M-0155: this is a *separate* write path from `Materialize` —
// not routed through the wipe-and-rewrite refresh set — so any
// consumer edits to the on-disk script survive a subsequent
// `aiwf update --statusline`.
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
		Snippet: formatStatuslineSnippet(snippetCmd),
	}

	// Skip if a copy already exists — the scaffold-once invariant.
	if _, statErr := os.Stat(dest); statErr == nil {
		// Even when the file exists, project scope still ensures the
		// gitignore entry is present (a consumer who hand-placed the
		// script wouldn't have it). Idempotent: no-op when present.
		if scope == StatuslineScopeProject {
			appended, gErr := ensureStatuslineGitignoreEntry(root)
			if gErr != nil {
				return res, gErr
			}
			res.GitignoreAppended = appended
		}
		return res, nil
	} else if !os.IsNotExist(statErr) {
		return res, fmt.Errorf("stat %s: %w", dest, statErr)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return res, fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
	}
	if err := os.WriteFile(dest, statuslineEmbed, 0o755); err != nil {
		return res, fmt.Errorf("writing %s: %w", dest, err)
	}
	res.Wrote = true

	if scope == StatuslineScopeProject {
		appended, gErr := ensureStatuslineGitignoreEntry(root)
		if gErr != nil {
			return res, gErr
		}
		res.GitignoreAppended = appended
	}
	return res, nil
}

// statuslineDest resolves the absolute destination path and the
// command-path string the activation snippet should display, based on
// the scope.
func statuslineDest(root, home string, scope StatuslineScope) (dest, snippetCmd string, err error) {
	switch scope {
	case StatuslineScopeProject:
		// Absolute on-disk destination; relative snippet command (cwd-relative).
		return filepath.Join(root, statuslineRelPath), statuslineRelPath, nil
	case StatuslineScopeUser:
		// Absolute on-disk destination and absolute snippet command.
		abs := filepath.Join(home, statuslineRelPath)
		return abs, abs, nil
	default:
		return "", "", fmt.Errorf("unknown --scope %q (want %q or %q)", scope, StatuslineScopeProject, StatuslineScopeUser)
	}
}

// formatStatuslineSnippet renders the JSON-ish activation snippet
// for the operator's Claude Code settings file.
func formatStatuslineSnippet(cmdPath string) string {
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
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}
