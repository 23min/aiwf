package policies

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// statuslineRoot is the relative target path the project-scope
// scaffold writes to. Repeated as a literal here (rather than imported
// from skills) so the test asserts what M-0155 promised independently
// of any naming refactor in the helper.
const (
	statuslineProjectRelPath   = ".claude/statusline.sh"
	statuslineGitignorePattern = ".claude/statusline.sh"
)

// TestM0155_AC3_ScaffoldStatuslineRefreshesInPlace asserts M-0155/AC-3
// as revised by G-0337: the scaffold always byte-refreshes the embedded
// statusline script. A fresh destination is written; a stale (differing)
// copy is refreshed to the embed; an already-current (byte-equal) copy is
// left untouched (idempotent, Wrote=false); a directory at the
// destination surfaces a read error.
//
// This supersedes the earlier scaffold-once lifecycle (write-only-if-
// absent). The script is an aiwf-owned artifact, byte-refreshed on every
// `aiwf update` like the materialized skills and hooks — a local edit
// does not survive.
func TestM0155_AC3_ScaffoldStatuslineRefreshesInPlace(t *testing.T) {
	t.Parallel()
	t.Run("absent destination → write the embed", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir() // unused for project scope but provided for shape
		res, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
		if err != nil {
			t.Fatalf("ScaffoldStatuslineWithHome: %v", err)
		}
		if !res.Wrote {
			t.Errorf("AC-3: fresh destination must report Wrote=true, got false")
		}
		got, err := os.ReadFile(res.Path)
		if err != nil {
			t.Fatalf("reading scaffolded script at %s: %v", res.Path, err)
		}
		if !bytes.Equal(got, skills.StatuslineBytes()) {
			t.Errorf("AC-3: scaffolded script (%d bytes) must be byte-equal to the embed (%d bytes)", len(got), len(skills.StatuslineBytes()))
		}
	})
	t.Run("pre-existing stale content → refresh to the embed", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		dest := filepath.Join(root, statuslineProjectRelPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		// An older aiwf script (or any drift) must be refreshed, not
		// preserved — the scaffold-once lifecycle was retired in G-0337.
		if err := os.WriteFile(dest, []byte("# stale older statusline\n"), 0o755); err != nil {
			t.Fatalf("write stale: %v", err)
		}
		res, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
		if err != nil {
			t.Fatalf("ScaffoldStatuslineWithHome: %v", err)
		}
		if !res.Wrote {
			t.Errorf("AC-3: stale destination must be refreshed (Wrote=true), got false")
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("reading dest after scaffold: %v", err)
		}
		if !bytes.Equal(got, skills.StatuslineBytes()) {
			t.Errorf("AC-3: stale content must be refreshed to the embed, got %q", got)
		}
	})
	t.Run("already-current content → left untouched (idempotent)", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		dest := filepath.Join(root, statuslineProjectRelPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(dest, skills.StatuslineBytes(), 0o755); err != nil {
			t.Fatalf("write current: %v", err)
		}
		res, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
		if err != nil {
			t.Fatalf("ScaffoldStatuslineWithHome: %v", err)
		}
		if res.Wrote {
			t.Errorf("AC-3: byte-equal destination must report Wrote=false (idempotent), got true")
		}
	})
	t.Run("destination is a directory → error", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		dest := filepath.Join(root, statuslineProjectRelPath)
		// A directory at the destination makes ReadFile fail with a
		// non-not-exist error, exercising the default (fault) branch.
		if err := os.MkdirAll(dest, 0o755); err != nil {
			t.Fatalf("mkdir dest-as-dir: %v", err)
		}
		if _, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject); err == nil {
			t.Errorf("AC-3: a directory at the destination must surface a read error, got nil")
		}
	})
}

// TestScaffoldStatusline_UnknownScope asserts an unrecognized scope is a
// usage error (exercises the statuslineDest default branch) — the closed
// set is {project, user}.
func TestScaffoldStatusline_UnknownScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	if _, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScope("bogus")); err == nil {
		t.Errorf("unknown scope must return an error, got nil")
	}
}

// TestM0155_AC4_ProjectScopeWritesGitignoreAndAnchoredSnippet asserts
// M-0155/AC-4 as revised by G-0337: project-scope scaffold writes to
// `<root>/.claude/statusline.sh`, appends the script's path to
// `<root>/.gitignore` (idempotent — no double-append), and returns an
// activation snippet whose command is
// `${CLAUDE_PROJECT_DIR:-<root>}/.claude/statusline.sh` — anchored on the
// repo root, not the cwd, so it resolves from a git worktree where a bare
// relative path would not.
//
// The "only on the install path" gitignore behavior — not added to
// the unconditional `GitignorePatterns()` set — is asserted by
// `TestM0155_AC4_GitignorePatternsNotGlobal` below.
func TestM0155_AC4_ProjectScopeWritesGitignoreAndAnchoredSnippet(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	res, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatalf("ScaffoldStatuslineWithHome: %v", err)
	}

	// Destination path is repo-relative.
	wantDest := filepath.Join(root, statuslineProjectRelPath)
	if res.Path != wantDest {
		t.Errorf("AC-4: project-scope destination must be %q, got %q", wantDest, res.Path)
	}

	// .gitignore appended with the script's path.
	if !res.GitignoreAppended {
		t.Errorf("AC-4: project-scope must report GitignoreAppended=true")
	}
	giBytes, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !strings.Contains(string(giBytes), statuslineGitignorePattern) {
		t.Errorf("AC-4: project-scope must append %q to .gitignore; got %q", statuslineGitignorePattern, giBytes)
	}

	// Command is anchored on $CLAUDE_PROJECT_DIR with the repo root as a
	// fallback — cwd-independent (G-0337). Assert the exact contract, not
	// merely "the path is not absolute" (which the anchored form satisfies
	// vacuously).
	wantCmd := "${CLAUDE_PROJECT_DIR:-" + root + "}/.claude/statusline.sh"
	if res.Command != wantCmd {
		t.Errorf("AC-4: project-scope Command must be %q, got %q", wantCmd, res.Command)
	}
	if got := skills.ProjectStatuslineCommand(root); got != wantCmd {
		t.Errorf("AC-4: ProjectStatuslineCommand(root) must be %q, got %q", wantCmd, got)
	}
	if !strings.Contains(res.Snippet, wantCmd) {
		t.Errorf("AC-4: activation Snippet must embed the command %q, got %q", wantCmd, res.Snippet)
	}

	// Second invocation is idempotent on .gitignore (no double-append).
	res2, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatalf("second ScaffoldStatuslineWithHome: %v", err)
	}
	if res2.GitignoreAppended {
		t.Errorf("AC-4: second invocation must report GitignoreAppended=false (no double-append)")
	}
	giBytes2, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if got := strings.Count(string(giBytes2), statuslineGitignorePattern); got != 1 {
		t.Errorf("AC-4: .gitignore must contain `%s` exactly once after two scaffold calls, got %d", statuslineGitignorePattern, got)
	}
}

// TestM0155_AC4_GitignorePatternsNotGlobal asserts the second half of
// AC-4: the `.claude/statusline.sh` ignore line must NOT appear in the
// unconditional `GitignorePatterns()` set that `ensureGitignore`
// reconciles on every bare `aiwf update`. If it did, a consumer who
// never opted into `--statusline` would still see `.claude/statusline.sh`
// added to their .gitignore on the next `aiwf update` — the ignore line
// belongs to the per-install `--statusline` path only.
func TestM0155_AC4_GitignorePatternsNotGlobal(t *testing.T) {
	t.Parallel()
	patterns, err := skills.GitignorePatterns()
	if err != nil {
		t.Fatalf("GitignorePatterns: %v", err)
	}
	for _, p := range patterns {
		if p == statuslineGitignorePattern {
			t.Errorf("AC-4: `%s` must NOT appear in the global `GitignorePatterns()` set — it belongs to the per-install scaffold path only", statuslineGitignorePattern)
		}
	}
}

// TestM0155_AC5_UserScopeWritesHomeWithAnchoredSnippet asserts M-0155/AC-5
// as revised by G-0337: user-scope scaffold writes to
// `<home>/.claude/statusline.sh`, does NOT touch any gitignore (the
// user-scope target lives outside any repo's tracked tree), and returns
// an activation snippet whose command is `$HOME/.claude/statusline.sh` —
// anchored on `$HOME` (not a baked-absolute home path) so one settings
// file resolves on both the host and inside a container sharing
// `~/.claude` via a mount.
func TestM0155_AC5_UserScopeWritesHomeWithAnchoredSnippet(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	res, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeUser)
	if err != nil {
		t.Fatalf("ScaffoldStatuslineWithHome: %v", err)
	}

	wantDest := filepath.Join(home, ".claude", "statusline.sh")
	if res.Path != wantDest {
		t.Errorf("AC-5: user-scope destination must be %q, got %q", wantDest, res.Path)
	}
	if res.GitignoreAppended {
		t.Errorf("AC-5: user-scope must report GitignoreAppended=false (user scope lives outside any repo)")
	}
	if _, err := os.Stat(filepath.Join(root, ".gitignore")); err == nil {
		t.Errorf("AC-5: user-scope must NOT create a .gitignore under root")
	}

	// Command is $HOME-anchored — resolves per-environment, independent of
	// the baked home path (G-0337).
	wantCmd := "$HOME/.claude/statusline.sh"
	if res.Command != wantCmd {
		t.Errorf("AC-5: user-scope Command must be %q, got %q", wantCmd, res.Command)
	}
	if got := skills.UserStatuslineCommand(); got != wantCmd {
		t.Errorf("AC-5: UserStatuslineCommand() must be %q, got %q", wantCmd, got)
	}
	if !strings.Contains(res.Snippet, wantCmd) {
		t.Errorf("AC-5: activation Snippet must embed the command %q, got %q", wantCmd, res.Snippet)
	}
}
