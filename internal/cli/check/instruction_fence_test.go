package check

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// mixedInstructionCommit seeds a repository with CLAUDE.md and a code file,
// then commits an edit to each together, and returns the root and the seed
// commit as the audit base.
func mixedInstructionCommit(t *testing.T) (root, base string) {
	t.Helper()
	root = t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitRun(t, root, "init", "-q", "-b", "main")
	write("CLAUDE.md", "# Rules\n\nKeep it simple.\n")
	write("app/app.go", "package app\n")
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "seed")
	base = headSHA(t, root)
	write("CLAUDE.md", "# Rules\n\nKeep it simple.\n\nA new rule.\n")
	write("app/app.go", "package app\n\nvar x = 1\n")
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "feat: rule and code", "--trailer", "aiwf-entity: E-0001")
	return root, base
}

// TestRunInstructionFenceCheck_JudgesTheAuditedRange is M-0350 AC-1's seam:
// the fence judges the range the provenance audit resolves, so --since
// names it.
func TestRunInstructionFenceCheck_JudgesTheAuditedRange(t *testing.T) {
	t.Parallel()
	root, base := mixedInstructionCommit(t)
	findings, err := RunInstructionFenceCheck(context.Background(), root, base, "")
	if err != nil {
		t.Fatalf("RunInstructionFenceCheck: %v", err)
	}
	if len(findings) != 1 || findings[0].Code != check.CodeInstructionCommitUnrelatedFile.ID || findings[0].Path != "app/app.go" {
		t.Fatalf("findings = %+v, want one %s on app/app.go", findings, check.CodeInstructionCommitUnrelatedFile.ID)
	}
}

// TestRun_ReportsInstructionFence is M-0350 AC-1 through the verb: aiwf
// check reports the mixed commit and exits with findings.
//
// Serial: captures the process-global os.Stdout.
func TestRun_ReportsInstructionFence(t *testing.T) {
	root, base := mixedInstructionCommit(t)
	var rc int
	out := testutil.CaptureStdout(t, func() {
		rc = Run(root, "json", false, base, false, false, false, nil, "")
	})
	var env struct {
		Findings []check.Finding `json:"findings"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("decoding check output: %v\n%s", err, out)
	}
	found := false
	for _, f := range env.Findings {
		found = found || (f.Code == check.CodeInstructionCommitUnrelatedFile.ID && f.Path == "app/app.go")
	}
	if !found || rc != 1 {
		t.Errorf("exit %d, findings %+v; want exit 1 with %s on app/app.go", rc, env.Findings, check.CodeInstructionCommitUnrelatedFile.ID)
	}
}

// TestRunInstructionFenceCheck_NoRangeReportsNothing pins the fence's two
// silent cases: a repository with no commits, and a branch with neither an
// upstream nor --since, where the provenance audit reports the scope
// advisory once and the fence adds nothing.
func TestRunInstructionFenceCheck_NoRangeReportsNothing(t *testing.T) {
	t.Parallel()
	empty := t.TempDir()
	gitRun(t, empty, "init", "-q", "-b", "main")
	root, _ := mixedInstructionCommit(t)
	for name, dir := range map[string]string{"no commits": empty, "no upstream and no --since": root} {
		findings, err := RunInstructionFenceCheck(context.Background(), dir, "", "")
		if err != nil || len(findings) != 0 {
			t.Errorf("%s: findings %+v, err %v; want none", name, findings, err)
		}
	}
}

// TestRun_InstructionFenceSkipsTrunkCommits is M-0350 AC-1's range through
// the verb: a mixed commit already on the configured trunk has been
// pushed, so `aiwf check --since` over it reports nothing.
//
// Serial: captures the process-global os.Stdout.
func TestRun_InstructionFenceSkipsTrunkCommits(t *testing.T) {
	root, base := mixedInstructionCommit(t)
	remote := t.TempDir()
	gitRun(t, remote, "init", "-q", "--bare")
	gitRun(t, root, "remote", "add", "origin", remote)
	gitRun(t, root, "push", "-q", "origin", "main")
	gitRun(t, root, "fetch", "-q", "origin")
	var rc int
	out := testutil.CaptureStdout(t, func() {
		rc = Run(root, "json", false, base, false, false, false, nil, "")
	})
	var env struct {
		Findings []check.Finding `json:"findings"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("decoding check output: %v\n%s", err, out)
	}
	for _, f := range env.Findings {
		if f.Code == check.CodeInstructionCommitUnrelatedFile.ID {
			t.Errorf("a commit on the trunk was judged: %+v (exit %d)", f, rc)
		}
	}
}
