package doctor

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHealthFileFrom_MapsAndEmpties pins the pure mapping: healthy →
// empty (non-nil) findings; warn/error carry through with source "aiwf".
func TestHealthFileFrom_MapsAndEmpties(t *testing.T) {
	t.Parallel()
	empty := healthFileFrom(nil, "2026-07-02T00:00:00Z")
	if empty.GeneratedAt != "2026-07-02T00:00:00Z" {
		t.Errorf("generated_at = %q", empty.GeneratedAt)
	}
	if len(empty.Findings) != 0 {
		t.Errorf("healthy → want 0 findings, got %d", len(empty.Findings))
	}

	hf := healthFileFrom([]Problem{
		{Severity: SeverityWarn, Message: "advisory"},
		{Severity: SeverityError, Message: "boom"},
	}, "ts")
	if len(hf.Findings) != 2 {
		t.Fatalf("want 2 findings, got %d", len(hf.Findings))
	}
	for _, f := range hf.Findings {
		if f.Source != "aiwf" {
			t.Errorf("source = %q, want aiwf", f.Source)
		}
	}
	if hf.Findings[0].Severity != "warn" || hf.Findings[1].Severity != "error" {
		t.Errorf("severities = %q/%q, want warn/error", hf.Findings[0].Severity, hf.Findings[1].Severity)
	}
}

// TestWriteHealth_WritesSchemaToMainCheckout drives the write end to end
// on a real git repo (no aiwf.yaml → a config error problem) and asserts
// a schema-valid file lands with the injected timestamp and the error.
func TestWriteHealth_WritesSchemaToMainCheckout(t *testing.T) {
	t.Parallel()
	root := healthTestRepo(t)
	const ts = "2026-07-02T12:00:00Z"
	if err := WriteHealth(context.Background(), root, ts, DoctorOptions{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".claude", "health.aiwf.json"))
	if err != nil {
		t.Fatal(err)
	}
	var hf healthFile
	if err := json.Unmarshal(raw, &hf); err != nil {
		t.Fatalf("health.aiwf.json is not valid JSON: %v", err)
	}
	if hf.GeneratedAt != ts {
		t.Errorf("generated_at = %q, want %q", hf.GeneratedAt, ts)
	}
	var cfg *healthFinding
	for i := range hf.Findings {
		if hf.Findings[i].Severity == "error" && strings.Contains(hf.Findings[i].Message, "aiwf.yaml") {
			cfg = &hf.Findings[i]
		}
	}
	if cfg == nil {
		t.Fatalf("no error finding naming aiwf.yaml; findings=%+v", hf.Findings)
	}
	if cfg.Source != "aiwf" {
		t.Errorf("source = %q, want aiwf", cfg.Source)
	}
}

func healthTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestWriteHealth_LinkedWorktreeResolvesToMainCheckout pins AC-2's "resolved
// even from a linked worktree" claim: invoked with a linked worktree as rootDir,
// the file lands in the MAIN checkout's .claude/, not the worktree's.
func TestWriteHealth_LinkedWorktreeResolvesToMainCheckout(t *testing.T) {
	t.Parallel()
	main := healthTestRepo(t)
	runGit(t, main, "commit", "--allow-empty", "-m", "init") // worktree add needs a commit
	linked := filepath.Join(t.TempDir(), "linked")
	runGit(t, main, "worktree", "add", "-q", linked, "-b", "wt")

	if err := WriteHealth(context.Background(), linked, "2026-07-02T00:00:00Z", DoctorOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(main, ".claude", "health.aiwf.json")); err != nil {
		t.Errorf("health file must land in the MAIN checkout's .claude/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(linked, ".claude", "health.aiwf.json")); !os.IsNotExist(err) {
		t.Errorf("health file must NOT land in the linked worktree (stat err = %v)", err)
	}
}
