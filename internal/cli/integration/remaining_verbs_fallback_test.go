package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/archive"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/contract"
	"github.com/23min/aiwf/internal/cli/importcmd"
	"github.com/23min/aiwf/internal/cli/rename"
	"github.com/23min/aiwf/internal/cli/renamearea"
	"github.com/23min/aiwf/internal/cli/retitle"
	"github.com/23min/aiwf/internal/cli/rewidth"
	"github.com/23min/aiwf/internal/cli/setarea"
	"github.com/23min/aiwf/internal/cli/worktree"
)

// remaining_verbs_fallback_test.go — M-0249 follow-up: pins the
// `runID == ""` correlation-id fallback (and, for worktree add and
// contract verify, the best-effort actor-resolution fallback) inside
// the nine verbs wired in remaining_verbs_diag_test.go. cli.Execute
// always mints a real correlation id (NewRootCmd), so these branches
// are unreachable through the CLI surface — only a direct call
// bypassing NewCmd/Execute (a hand-built OutputFormat / bare empty
// correlationID) exercises them. Mirrors correlation_id_test.go's
// established *FallsBackWhenOutputFormatCarriesNone pattern for the
// first seven wired verbs.

// commitAll stages and commits everything in root — `aiwf worktree
// add` cuts the new branch off HEAD, which needs a real commit to
// carry aiwf.yaml into the new worktree (setupCLITestRepo + `aiwf
// init` alone leave aiwf.yaml uncommitted).
func commitAll(t *testing.T, root string) {
	t.Helper()
	if out, err := testutil.RunGit(root, "add", "-A"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := testutil.RunGit(root, "commit", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func readRunID(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	return rec.RunID
}

func TestArchiveDiag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := archive.Run("human/test", "", root, "", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("archive.Run: rc=%d", rc)
	}
	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestImportDiag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	manifestPath := filepath.Join(t.TempDir(), "manifest.yaml")
	manifest := "version: 1\nactor: human/test\nentities:\n  - kind: gap\n    id: auto\n    frontmatter:\n      title: Fallback probe\n      status: open\n    body: \"## What's missing\\n\\nFixture.\\n\\n## Why it matters\\n\\nFixture.\\n\"\n"
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := importcmd.Run(manifestPath, root, "human/test", "", "", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("importcmd.Run: rc=%d", rc)
	}
	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestRenameDiag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "Fallback probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := rename.Run("G-0001", "renamed-slug", "human/test", "", root, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("rename.Run: rc=%d", rc)
	}
	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestRenameAreaDiag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupAreaRepo(t)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := renamearea.Run("platform", "infra", "human/test", "", root, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("renamearea.Run: rc=%d", rc)
	}
	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestRetitleDiag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "Fallback probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := retitle.Run("G-0001", "New title", "human/test", "", root, "", cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("retitle.Run: rc=%d", rc)
	}
	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

// TestRewidthDiag_FallsBackWhenOutputFormatCarriesNone uses a
// freshly-inited repo (already canonical width — nothing to
// rewidth), so the verb takes its NoOp path; the diagLog block runs
// before that outcome is known, so a NoOp still exercises the
// fallback. Mirrors TestRewidthDiag_EmitsVerbCompletedEvent's own
// rationale.
func TestRewidthDiag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := rewidth.Run("human/test", "", root, true, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("rewidth.Run: rc=%d", rc)
	}
	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestSetAreaDiag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Untagged", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := setarea.Run([]string{"E-0001", "platform"}, "human/test", "", root, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("setarea.Run: rc=%d", rc)
	}
	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

// TestWorktreeAddDiag_FallsBackWhenOutputFormatCarriesNone is a
// direct, in-process worktree.Run call (unlike
// TestWorktreeAddDiag_EmitsVerbCompletedEvent, which drives a real
// binary subprocess via runSplit — a separately-built, uninstrumented
// process invisible to this test binary's coverage profile). Only a
// direct call exercises the runID=="" fallback statement.
func TestWorktreeAddDiag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	commitAll(t, root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := worktree.Run("feature/fallback-probe", filepath.Join(t.TempDir(), "wt"), "", root, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("worktree.Run: rc=%d", rc)
	}
	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

// TestWorktreeAddDiag_ActorResolutionFailureStillEmitsEvent pins
// worktree.Run's own best-effort actor rationale (no --actor flag,
// mirroring check.Run's/show.Run's identical pattern): when git
// config user.email is also unavailable, the verb still completes and
// still logs, with an empty actor field. Cannot use t.Parallel():
// t.Setenv panics under parallel.
func TestWorktreeAddDiag_ActorResolutionFailureStillEmitsEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	commitAll(t, root)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// Intentionally no .gitconfig at home — git config user.email fails.

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := worktree.Run("feature/actor-fail-probe", filepath.Join(t.TempDir(), "wt"), "", root, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("worktree.Run: rc=%d", rc)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Actor != "" {
		t.Errorf("actor = %q, want empty (git config user.email was deliberately unavailable)", rec.Actor)
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

// TestContractVerifyDiag_FallsBackWithNoActorAndNoCorrelationID pins
// contract.Run's own best-effort-actor + runID fallback in one call:
// broken git config forces actorErr != nil (contract/verify.go:77),
// and a bare empty correlationID forces the runID == "" fallback
// (contract/verify.go:81) — both branches live in the same function,
// so one direct call with both conditions covers both statements.
// Cannot use t.Parallel(): t.Setenv panics under parallel.
func TestContractVerifyDiag_FallsBackWithNoActorAndNoCorrelationID(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// Intentionally no .gitconfig at home — git config user.email fails.

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := contract.Run(root, "text", false, "")
	if rc != cliutil.ExitOK {
		t.Fatalf("contract.Run: rc=%d", rc)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Actor != "" {
		t.Errorf("actor = %q, want empty (git config user.email was deliberately unavailable)", rec.Actor)
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty even though correlationID was passed as \"\"; the fallback mint did not fire")
	}
}
