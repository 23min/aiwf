package check

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/tree"
)

// fenceEpic is a live epic the fixtures' aiwf-entity trailers name.
const fenceEpic = "---\nid: E-0001\ntitle: Probe epic\nstatus: active\n---\n\n## Goal\n\nReal prose.\n\n## Scope\n\nReal prose.\n\n## Out of scope\n\nReal prose.\n"

// writeRepoFile writes content at rel under root, creating directories.
func writeRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// loadTree loads the entity tree at root.
func loadTree(t *testing.T, root string) *tree.Tree {
	t.Helper()
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	return tr
}

// mixedInstructionCommit seeds a repository with CLAUDE.md and a code file,
// then commits an edit to each together, and returns the root and the seed
// commit as the audit base.
func mixedInstructionCommit(t *testing.T) (root, base string) {
	t.Helper()
	root = t.TempDir()
	write := func(rel, content string) { writeRepoFile(t, root, rel, content) }
	gitRun(t, root, "init", "-q", "-b", "main")
	write("work/epics/E-0001-probe-epic/epic.md", fenceEpic)
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
	findings, err := RunInstructionFenceCheck(context.Background(), root, base, loadTree(t, root))
	if err != nil {
		t.Fatalf("RunInstructionFenceCheck: %v", err)
	}
	if len(findings) != 1 || findings[0].Code != check.CodeInstructionCommitUnrelatedFile.ID || findings[0].Path != "app/app.go" {
		t.Fatalf("findings = %+v, want one %s on app/app.go", findings, check.CodeInstructionCommitUnrelatedFile.ID)
	}
}

// TestRunInstructionFenceCheck_NoRangeReportsNothing pins the fence's two
// silent cases: a branch with an upstream but no commits of its own yet,
// where the range names no HEAD to walk, and a branch with neither an
// upstream nor --since, where the provenance audit reports the scope
// advisory once and the fence adds nothing.
func TestRunInstructionFenceCheck_NoRangeReportsNothing(t *testing.T) {
	t.Parallel()
	remote, _ := mixedInstructionCommit(t)
	empty := t.TempDir()
	gitRun(t, empty, "init", "-q", "-b", "main")
	gitRun(t, empty, "remote", "add", "origin", remote)
	gitRun(t, empty, "fetch", "-q", "origin")
	gitRun(t, empty, "config", "branch.main.remote", "origin")
	gitRun(t, empty, "config", "branch.main.merge", "refs/heads/main")
	root, _ := mixedInstructionCommit(t)
	for name, dir := range map[string]string{"an upstream but no commits": empty, "no upstream and no --since": root} {
		findings, err := RunInstructionFenceCheck(context.Background(), dir, "", loadTree(t, dir))
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

// TestRunInstructionFenceCheck_ResolvesTheTrailerThroughTheTree is M-0350
// AC-2's seam: the trailer resolves against the loaded tree, so an entity
// in an archive directory counts as one that exists, and a value naming no
// entity is reported.
func TestRunInstructionFenceCheck_ResolvesTheTrailerThroughTheTree(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gitRun(t, root, "init", "-q", "-b", "main")
	writeRepoFile(t, root, "work/gaps/archive/G-0001-archived-gap.md", "---\nid: G-0001\ntitle: Archived gap\nstatus: addressed\n---\n\n## What's missing\n\nProse.\n\n## Why it matters\n\nProse.\n")
	writeRepoFile(t, root, "CLAUDE.md", "# Rules\n")
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "seed")
	base := headSHA(t, root)
	for i, entity := range []string{"G-0001", "G-0002"} {
		writeRepoFile(t, root, "CLAUDE.md", "# Rules\n"+strings.Repeat("\nA rule.\n", i+1))
		gitRun(t, root, "commit", "-q", "-am", "docs: rule", "--trailer", "aiwf-entity: "+entity)
	}
	findings, err := RunInstructionFenceCheck(context.Background(), root, base, loadTree(t, root))
	if err != nil {
		t.Fatalf("RunInstructionFenceCheck: %v", err)
	}
	if len(findings) != 1 || findings[0].Code != check.CodeInstructionCommitEntityMissing.ID || !strings.Contains(findings[0].Message, "G-0002") {
		t.Errorf("findings = %+v, want one %s naming G-0002", findings, check.CodeInstructionCommitEntityMissing.ID)
	}
}

// TestRun_InstructionFenceSetting is M-0350 AC-4 through the verb: with no
// setting the fence's findings are errors and aiwf check exits non-zero;
// with guidance.fence_instruction_commits off the rule reports nothing;
// and with no audited range the provenance scope advisory stands and the
// rule reports nothing.
//
// Serial: captures the process-global os.Stdout.
func TestRun_InstructionFenceSetting(t *testing.T) {
	for _, tc := range []struct {
		name, yaml string
		noRange    bool
		wantFence  bool
		wantExit   int
	}{
		{name: "no setting", wantFence: true, wantExit: 1},
		{name: "turned off", yaml: "guidance:\n  fence_instruction_commits: false\n"},
		{name: "no audited range", noRange: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, base := mixedInstructionCommit(t)
			if tc.yaml != "" {
				writeAiwfYAML(t, root, tc.yaml)
			}
			since := base
			if tc.noRange {
				since = ""
			}
			var rc int
			out := testutil.CaptureStdout(t, func() {
				rc = Run(root, "json", false, since, false, false, false, nil, "")
			})
			var env struct {
				Findings []check.Finding `json:"findings"`
			}
			if err := json.Unmarshal(out, &env); err != nil {
				t.Fatalf("decoding check output: %v\n%s", err, out)
			}
			fence, advisory := false, false
			for _, f := range env.Findings {
				if strings.HasPrefix(f.Code, "instruction-commit-") {
					fence = true
					if f.Severity != check.SeverityError {
						t.Errorf("%s at %s, want error", f.Code, f.Severity)
					}
				}
				advisory = advisory || (f.Code == check.CodeProvenanceUntrailedScopeUndefined && strings.Contains(f.Message, "instruction-file fence"))
			}
			if fence != tc.wantFence || rc != tc.wantExit {
				t.Errorf("fence findings %v, exit %d; want %v, exit %d\n%+v", fence, rc, tc.wantFence, tc.wantExit, env.Findings)
			}
			if tc.noRange && !advisory {
				t.Errorf("no audited range: the scope advisory must stand and name the fence; got %+v", env.Findings)
			}
		})
	}
}
