package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// writeSettingsWithCommand writes <dir>/.claude/<name> carrying the given
// statusLine.command value. Used to build the G-0337 doctor fixtures.
func writeSettingsWithCommand(t *testing.T, dir, name, command string) {
	t.Helper()
	d := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"statusLine":{"type":"command","command":"` + command + `"}}` + "\n"
	if err := os.WriteFile(filepath.Join(d, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// containsSub reports whether any line contains sub.
func containsSub(lines []string, sub string) bool {
	for _, l := range lines {
		if strings.Contains(l, sub) {
			return true
		}
	}
	return false
}

// TestStatuslineReport_G0337_PrecedenceConflict asserts the precedence
// warning fires when a statusLine is wired in BOTH project and user
// settings (project shadows user), and stays silent otherwise.
func TestStatuslineReport_G0337_PrecedenceConflict(t *testing.T) {
	t.Parallel()
	t.Run("both wired → warn", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		writeSettingsWithCommand(t, root, "settings.local.json", "${CLAUDE_PROJECT_DIR:-"+root+"}/.claude/statusline.sh")
		writeSettingsWithCommand(t, home, "settings.json", "$HOME/.claude/statusline.sh")
		out := appendPrecedenceCheck(nil, root, home)
		if !containsSub(out, "precedence:") {
			t.Errorf("expected a precedence warning when both project and user are wired; got %v", out)
		}
	})
	t.Run("only user wired → silent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		writeSettingsWithCommand(t, home, "settings.json", "$HOME/.claude/statusline.sh")
		out := appendPrecedenceCheck(nil, root, home)
		if containsSub(out, "precedence:") {
			t.Errorf("expected no precedence warning when only user is wired; got %v", out)
		}
	})
	t.Run("only project wired → silent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		writeSettingsWithCommand(t, root, "settings.local.json", "${CLAUDE_PROJECT_DIR:-"+root+"}/.claude/statusline.sh")
		out := appendPrecedenceCheck(nil, root, home)
		if containsSub(out, "precedence:") {
			t.Errorf("expected no precedence warning when only project is wired; got %v", out)
		}
	})
}

// TestStatuslineReport_G0337_ProjectCommandHealth asserts the project
// command check fires for cwd-relative and stale-fallback commands and
// stays silent for a resolvable command or absent project settings.
func TestStatuslineReport_G0337_ProjectCommandHealth(t *testing.T) {
	t.Parallel()
	t.Run("bare-relative → cwd-relative warning", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeSettingsWithCommand(t, root, "settings.local.json", ".claude/statusline.sh")
		out := appendProjectCommandCheck(nil, root)
		if !containsSub(out, "cwd-relative") {
			t.Errorf("expected a cwd-relative warning; got %v", out)
		}
	})
	t.Run("stale fallback → does-not-resolve warning", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeSettingsWithCommand(t, root, "settings.local.json", "${CLAUDE_PROJECT_DIR:-/no/such/mount}/.claude/statusline.sh")
		out := appendProjectCommandCheck(nil, root)
		if !containsSub(out, "does not resolve") {
			t.Errorf("expected a stale-fallback warning; got %v", out)
		}
	})
	t.Run("resolvable fallback → silent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		installStatusline(t, root) // makes <root>/.claude/statusline.sh exist
		writeSettingsWithCommand(t, root, "settings.local.json", "${CLAUDE_PROJECT_DIR:-"+root+"}/.claude/statusline.sh")
		out := appendProjectCommandCheck(nil, root)
		if containsSub(out, "command:") {
			t.Errorf("expected no command warning for a resolvable fallback; got %v", out)
		}
	})
	t.Run("absent project settings → silent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		out := appendProjectCommandCheck(nil, root)
		if len(out) != 0 {
			t.Errorf("expected no output when no project settings exist; got %v", out)
		}
	})
}

// TestStatuslineCmdPathForScope asserts the remediation-hint command is
// the skills single-source value for each scope (G-0337).
func TestStatuslineCmdPathForScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if got, want := statuslineCmdPathForScope("project", root), skills.ProjectStatuslineCommand(root); got != want {
		t.Errorf("project: got %q, want %q", got, want)
	}
	if got, want := statuslineCmdPathForScope("user", root), skills.UserStatuslineCommand(); got != want {
		t.Errorf("user: got %q, want %q", got, want)
	}
}

// TestStatusLineCommand covers every branch of the settings-command
// reader: absent file, malformed JSON, no statusLine key, a statusLine
// that is not an object, and a well-formed command value.
func TestStatusLineCommand(t *testing.T) {
	t.Parallel()
	t.Run("absent file → empty", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if got := statusLineCommand(filepath.Join(root, ".claude", "settings.json")); got != "" {
			t.Errorf("absent file must yield empty, got %q", got)
		}
	})
	t.Run("malformed json → empty", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		d := filepath.Join(root, ".claude")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(d, "settings.json")
		if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := statusLineCommand(p); got != "" {
			t.Errorf("malformed json must yield empty, got %q", got)
		}
	})
	t.Run("no statusLine key → empty", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		d := filepath.Join(root, ".claude")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(d, "settings.json")
		if err := os.WriteFile(p, []byte(`{"other":true}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := statusLineCommand(p); got != "" {
			t.Errorf("missing statusLine key must yield empty, got %q", got)
		}
	})
	t.Run("statusLine not an object → empty", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		d := filepath.Join(root, ".claude")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(d, "settings.json")
		if err := os.WriteFile(p, []byte(`{"statusLine":"oops"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := statusLineCommand(p); got != "" {
			t.Errorf("non-object statusLine must yield empty, got %q", got)
		}
	})
	t.Run("present → value", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeSettingsWithCommand(t, root, "settings.json", "$HOME/.claude/statusline.sh")
		if got := statusLineCommand(filepath.Join(root, ".claude", "settings.json")); got != "$HOME/.claude/statusline.sh" {
			t.Errorf("present command must be returned, got %q", got)
		}
	})
}

// TestResolvedFallbackPath covers each branch of the fallback extractor.
func TestResolvedFallbackPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in, want string
	}{
		{"anchored form", "${CLAUDE_PROJECT_DIR:-/x}/.claude/statusline.sh", "/x/.claude/statusline.sh"},
		{"no CLAUDE_PROJECT_DIR prefix", "$HOME/.claude/statusline.sh", ""},
		{"prefix but no closing brace", "${CLAUDE_PROJECT_DIR:-/x/.claude/statusline.sh", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolvedFallbackPath(tc.in); got != tc.want {
				t.Errorf("resolvedFallbackPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
