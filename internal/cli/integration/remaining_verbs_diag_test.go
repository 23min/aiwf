package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// remaining_verbs_diag_test.go — M-0249: pins that the last nine
// mutating verbs (archive, import, rename, rename-area, retitle,
// rewidth, set-area, worktree add, and contract's five sub-verbs) now
// emit a "verb.completed" diagnostic-log record when AIWF_LOG is set,
// completing E-0061's own instrumentation coverage across every
// mutating verb (see wired_verbs_diag_test.go for the first seven).

func TestArchiveDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "diag probe", "--actor", "human/test", "--root", root)
	mustRun(t, "cancel", "G-0001", "--reason", "no longer needed", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"archive", "--apply", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf archive: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "archive" {
		t.Errorf("verb = %q, want %q", rec.Verb, "archive")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestImportDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	manifestPath := filepath.Join(t.TempDir(), "manifest.yaml")
	manifest := "version: 1\nactor: human/test\nentities:\n  - kind: gap\n    id: auto\n    frontmatter:\n      title: Imported probe\n      status: open\n    body: \"## What's missing\\n\\nFixture.\\n\\n## Why it matters\\n\\nFixture.\\n\"\n"
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"import", manifestPath, "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf import: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "import" {
		t.Errorf("verb = %q, want %q", rec.Verb, "import")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestRenameDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "diag probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"rename", "G-0001", "renamed-slug", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf rename: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "rename" {
		t.Errorf("verb = %q, want %q", rec.Verb, "rename")
	}
	if rec.Entity != "G-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "G-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestRenameAreaDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupAreaRepo(t)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"rename-area", "platform", "infra", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf rename-area: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "rename-area" {
		t.Errorf("verb = %q, want %q", rec.Verb, "rename-area")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestRetitleDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "diag probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"retitle", "G-0001", "New title", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf retitle: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "retitle" {
		t.Errorf("verb = %q, want %q", rec.Verb, "retitle")
	}
	if rec.Entity != "G-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "G-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

// TestRewidthDiag_EmitsVerbCompletedEvent uses a freshly-inited repo
// (already canonical width — nothing to rewidth), so the verb takes
// its NoOp path. The diagLog block runs before that outcome is known,
// so a NoOp still emits verb.completed.
func TestRewidthDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"rewidth", "--apply", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf rewidth: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "rewidth" {
		t.Errorf("verb = %q, want %q", rec.Verb, "rewidth")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestSetAreaDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Untagged", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"set-area", "E-0001", "platform", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf set-area: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "set-area" {
		t.Errorf("verb = %q, want %q", rec.Verb, "set-area")
	}
	if rec.Entity != "E-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "E-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

// TestWorktreeAddDiag_EmitsVerbCompletedEvent drives the real binary
// (worktree add shells out to real `git worktree add` and needs a
// real committed base), mirroring TestWorktreeAddMetadata_ReportsBranchAndPath's
// own setupInitedRepo + runSplit pattern. AIWF_LOG* is inherited by
// the subprocess via the same os.Environ() passthrough every other
// scenario/test in this repo relies on.
func TestWorktreeAddDiag_EmitsVerbCompletedEvent(t *testing.T) {
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupInitedRepo(t)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	stdout, stderr, code := runSplit(t, root, bin, "worktree", "add", "feature/diag-check", filepath.Join(t.TempDir(), "wt"))
	if code != 0 {
		t.Fatalf("aiwf worktree add: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "worktree-add" {
		t.Errorf("verb = %q, want %q", rec.Verb, "worktree-add")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestContractBindDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test")
	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")
	writeFixtureFile(t, root, "fixtures/v1/valid/good.json", "PASS")
	mustRun(t, "add", "contract", "--body", "## Purpose\n\nFixture.\n\n## Stability\n\nFixture.\n", "--title", "Public API", "--root", root, "--actor", "human/test")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"contract", "bind", "C-0001", "--validator", "fake", "--schema", "schema.cue", "--fixtures", "fixtures", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf contract bind: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "contract-bind" {
		t.Errorf("verb = %q, want %q", rec.Verb, "contract-bind")
	}
	if rec.Entity != "C-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "C-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestContractUnbindDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test")
	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")
	writeFixtureFile(t, root, "fixtures/v1/valid/good.json", "PASS")
	mustRun(t, "add", "contract", "--body", "## Purpose\n\nFixture.\n\n## Stability\n\nFixture.\n", "--title", "Public API", "--root", root, "--actor", "human/test", "--validator", "fake", "--schema", "schema.cue", "--fixtures", "fixtures")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"contract", "unbind", "C-0001", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf contract unbind: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "contract-unbind" {
		t.Errorf("verb = %q, want %q", rec.Verb, "contract-unbind")
	}
	if rec.Entity != "C-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "C-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestContractRecipeInstallDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"contract", "recipe", "install", "--from", customPath, "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf contract recipe install: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "contract-recipe-install" {
		t.Errorf("verb = %q, want %q", rec.Verb, "contract-recipe-install")
	}
	if rec.Entity != "fake" {
		t.Errorf("entity = %q, want %q", rec.Entity, "fake")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestContractRecipeRemoveDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"contract", "recipe", "remove", "fake", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf contract recipe remove: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "contract-recipe-remove" {
		t.Errorf("verb = %q, want %q", rec.Verb, "contract-recipe-remove")
	}
	if rec.Entity != "fake" {
		t.Errorf("entity = %q, want %q", rec.Entity, "fake")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

// TestContractVerifyDiag_EmitsVerbCompletedEvent pins contract
// verify's own best-effort actor rationale (no --actor flag), mirroring
// check.Run's/show.Run's identical pattern.
func TestContractVerifyDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"contract", "verify", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf contract verify: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "contract-verify" {
		t.Errorf("verb = %q, want %q", rec.Verb, "contract-verify")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}
