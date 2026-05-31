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

// TestM0155_AC3_ScaffoldStatuslineWritesIfAbsent asserts M-0155/AC-3:
// the scaffold-if-absent write path materializes the embedded
// statusline script only when no copy already exists at the
// destination. Idempotent: a second call against the same destination
// reports Wrote=false and leaves the on-disk content unchanged.
//
// The "if absent" guard is what makes M-0155's "embedded but excluded
// from the unconditional refresh set" lifecycle real: any consumer
// edits to `.claude/statusline.sh` survive a subsequent `aiwf update
// --statusline`. The test drives both legs — fresh-target (write
// happens, bytes match the embed) and pre-existing-target (no write,
// pre-existing content preserved verbatim).
func TestM0155_AC3_ScaffoldStatuslineWritesIfAbsent(t *testing.T) {
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
	t.Run("pre-existing destination → skip + preserve content", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		dest := filepath.Join(root, statuslineProjectRelPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		sentinel := []byte("# operator-edited custom script\n")
		if err := os.WriteFile(dest, sentinel, 0o755); err != nil {
			t.Fatalf("write sentinel: %v", err)
		}
		res, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
		if err != nil {
			t.Fatalf("ScaffoldStatuslineWithHome: %v", err)
		}
		if res.Wrote {
			t.Errorf("AC-3: pre-existing destination must report Wrote=false, got true")
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("reading dest after scaffold: %v", err)
		}
		if !bytes.Equal(got, sentinel) {
			t.Errorf("AC-3: pre-existing content must be preserved verbatim; got %q", got)
		}
	})
}

// TestM0155_AC4_ProjectScopeWritesGitignoreAndRelativeSnippet asserts
// M-0155/AC-4: project-scope scaffold writes to
// `<root>/.claude/statusline.sh`, appends the script's path to
// `<root>/.gitignore` (idempotent — no double-append), and returns an
// activation snippet whose command path is repo-relative (so the
// printed snippet works regardless of where the consumer launches
// Claude Code from inside the repo).
//
// The "only on the install path" gitignore behavior — not added to
// the unconditional `GitignorePatterns()` set — is asserted by
// `TestM0155_AC4_GitignorePatternsNotGlobal` below.
func TestM0155_AC4_ProjectScopeWritesGitignoreAndRelativeSnippet(t *testing.T) {
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

	// Snippet contains a repo-relative path (not absolute, not under home).
	if res.Snippet == "" {
		t.Errorf("AC-4: project-scope must return a non-empty activation Snippet")
	}
	if strings.Contains(res.Snippet, home) || filepath.IsAbs(extractCommandPath(res.Snippet)) {
		t.Errorf("AC-4: project-scope activation Snippet must use a repo-relative command path, got %q", res.Snippet)
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
// added to their .gitignore on the next `aiwf update`, contradicting
// the scaffold-once lifecycle.
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

// TestM0155_AC5_UserScopeWritesHomeWithAbsoluteSnippet asserts
// M-0155/AC-5: user-scope scaffold writes to
// `<home>/.claude/statusline.sh`, does NOT touch any gitignore (the
// user-scope target lives outside any repo's tracked tree), and
// returns an activation snippet whose command path is absolute (so the
// snippet works from any worktree of any repo in the same
// devcontainer).
func TestM0155_AC5_UserScopeWritesHomeWithAbsoluteSnippet(t *testing.T) {
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

	// Snippet path is absolute (starts with the home dir).
	cmdPath := extractCommandPath(res.Snippet)
	if !filepath.IsAbs(cmdPath) {
		t.Errorf("AC-5: user-scope activation Snippet must use an absolute command path, got %q", res.Snippet)
	}
	if !strings.HasPrefix(cmdPath, home) {
		t.Errorf("AC-5: user-scope activation Snippet command path must live under home (%s), got %q", home, cmdPath)
	}
}

// extractCommandPath pulls the script-path token out of an activation
// snippet. The snippet's shape is operator-readable JSON-ish — the
// `command` value is the relevant field. This helper is best-effort:
// it finds the substring after `"command"` (or `command:`) up to the
// next quote or whitespace. The two scope-specific tests above only
// need to know whether the path is absolute and where it points; this
// helper extracts enough for that.
func extractCommandPath(snippet string) string {
	// Prefer JSON-ish: `"command": "<path>"`.
	if i := strings.Index(snippet, `"command"`); i >= 0 {
		rest := snippet[i+len(`"command"`):]
		// skip whitespace and colon
		rest = strings.TrimLeft(rest, " \t:\n")
		if strings.HasPrefix(rest, `"`) {
			rest = rest[1:]
			if j := strings.Index(rest, `"`); j >= 0 {
				return rest[:j]
			}
		}
	}
	// Fall back to bare line containing the script name.
	for _, line := range strings.Split(snippet, "\n") {
		if strings.Contains(line, "statusline.sh") {
			fields := strings.Fields(line)
			for _, f := range fields {
				if strings.Contains(f, "statusline.sh") {
					return strings.Trim(f, `"',`)
				}
			}
		}
	}
	return ""
}
