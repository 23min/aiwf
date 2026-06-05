package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAppendCommitMsgHookReport_Missing: G-0218 chokepoint not installed
// → problem + missing-line + remediation-hint.
func TestAppendCommitMsgHookReport_Missing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := appendCommitMsgHookReport(nil, 0, root)
	if problems != 1 {
		t.Errorf("problems = %d, want 1", problems)
	}
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "missing") || !strings.Contains(out, "aiwf update") {
		t.Errorf("missing-hook report should name `missing` + remediation; got:\n%s", out)
	}
}

// TestAppendCommitMsgHookReport_AlienHook: present but no marker —
// not aiwf-managed, no problem bump (consumer's own hook is left
// alone elsewhere; doctor just reports what's there).
func TestAppendCommitMsgHookReport_AlienHook(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "commit-msg"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := appendCommitMsgHookReport(nil, 0, root)
	if problems != 0 {
		t.Errorf("problems = %d, want 0 (alien hook is reported but not a problem-count bump)", problems)
	}
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "not aiwf-managed") {
		t.Errorf("alien-hook report should name `not aiwf-managed`; got:\n%s", out)
	}
}

// TestAppendCommitMsgHookReport_OurHookAiwfNotOnPATH: marker present
// but no aiwf on PATH → problem bumped + named diagnostic. Mirrors
// the pre-push/pre-commit/post-commit symmetric branches.
func TestAppendCommitMsgHookReport_OurHookAiwfNotOnPATH(t *testing.T) {
	root := t.TempDir()
	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "#!/bin/sh\n# aiwf:commit-msg\ncommand -v aiwf\nexec aiwf check --commit-msg \"$1\"\n"
	if err := os.WriteFile(filepath.Join(hooks, "commit-msg"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	// Point PATH at an empty dir so the exec.LookPath("aiwf") call
	// fails. t.Setenv blocks t.Parallel; this test is intentionally
	// serial.
	t.Setenv("PATH", filepath.Join(t.TempDir(), "no-aiwf-here"))
	lines, problems := appendCommitMsgHookReport(nil, 0, root)
	if problems != 1 {
		t.Errorf("problems = %d, want 1", problems)
	}
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "aiwf binary not found on PATH") {
		t.Errorf("PATH-missing report should name the not-found diagnostic; got:\n%s", out)
	}
}

// TestAppendCommitMsgHookReport_OurHook: marker present + aiwf on
// PATH → ok line, no problem bump.
func TestAppendCommitMsgHookReport_OurHook(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "#!/bin/sh\n# aiwf:commit-msg\ncommand -v aiwf\nexec aiwf check --commit-msg \"$1\"\n"
	if err := os.WriteFile(filepath.Join(hooks, "commit-msg"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := appendCommitMsgHookReport(nil, 0, root)
	if problems != 0 {
		t.Errorf("problems = %d, want 0", problems)
	}
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "ok") {
		t.Errorf("aiwf-managed hook should report ok; got:\n%s", out)
	}
}
