package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// instructionFenceRepo seeds a repository with an entity, CLAUDE.md and a
// code file, then commits an edit to CLAUDE.md beside an edit to the code
// file under a trailer naming the entity. It returns the root, the seed
// commit as the audit base, and a git runner.
func instructionFenceRepo(t *testing.T) (root, base string, git func(...string) string) {
	t.Helper()
	root = t.TempDir()
	write := repoFileWriter(t, root)
	git = func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	write("work/epics/E-0001-probe/epic.md", "---\nid: E-0001\ntitle: Probe\nstatus: active\n---\n\n## Goal\n\nProse.\n")
	write("CLAUDE.md", "# Rules\n\nKeep it simple.\n")
	write("app/app.go", "package app\n")
	git("add", "-A")
	git("commit", "-q", "-m", "seed")
	base = git("rev-parse", "HEAD")
	write("CLAUDE.md", "# Rules\n\nKeep it simple.\n\nA new rule.\n")
	write("app/app.go", "package app\n\nvar x = 1\n")
	git("add", "-A")
	git("commit", "-q", "-m", "feat: rule and code", "--trailer", "aiwf-entity: E-0001")
	return root, base, git
}

// TestInstructionFenceViolations_RunsTheKernelRule is M-0350 AC-6: this
// repository's gate runs aiwf check's instruction-file fence, compiled from
// the tree under test, over the commits since the base, so an installed
// aiwf that predates the rule cannot skip it. It honours the repository's
// guidance.fence_instruction_commits, as aiwf check does.
func TestInstructionFenceViolations_RunsTheKernelRule(t *testing.T) {
	t.Parallel()

	t.Run("a mixed commit is reported", func(t *testing.T) {
		t.Parallel()
		root, base, _ := instructionFenceRepo(t)
		vs, err := instructionFenceViolations(root, base)
		if err != nil {
			t.Fatal(err)
		}
		if len(vs) != 1 || vs[0].Policy != "instruction-fence" || vs[0].File != "app/app.go" || !strings.Contains(vs[0].Detail, check.CodeInstructionCommitUnrelatedFile.ID) {
			t.Errorf("violations = %+v, want one instruction-fence violation on app/app.go naming %s", vs, check.CodeInstructionCommitUnrelatedFile.ID)
		}
	})
	t.Run("the repository's setting turns it off", func(t *testing.T) {
		t.Parallel()
		root, base, _ := instructionFenceRepo(t)
		if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("guidance:\n  fence_instruction_commits: false\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if vs, err := instructionFenceViolations(root, base); err != nil || len(vs) != 0 {
			t.Errorf("violations = %+v, err %v; want none", vs, err)
		}
	})
	t.Run("no base no-ops", func(t *testing.T) {
		t.Parallel()
		root, _, _ := instructionFenceRepo(t)
		for _, base := range []string{"", zeroSHA} {
			if vs, err := instructionFenceViolations(root, base); err != nil || len(vs) != 0 {
				t.Errorf("base %q: violations = %+v, err %v; want none", base, vs, err)
			}
		}
	})
	t.Run("a base naming no commit is an error", func(t *testing.T) {
		t.Parallel()
		root, _, _ := instructionFenceRepo(t)
		if _, err := instructionFenceViolations(root, "no-such-ref"); err == nil {
			t.Error("want an error")
		}
	})
}

// TestInstructionFence_WiredIntoCoverageGate pins that the fence runs at the
// integration boundary: it is named in the coverage-gate run-pattern of both
// the CI workflow and the Makefile target.
func TestInstructionFence_WiredIntoCoverageGate(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, f := range []string{".github/workflows/go.yml", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if line := coverageGateRunLine(t, f, string(data)); !strings.Contains(line, "|InstructionFence)") && !strings.Contains(line, "|InstructionFence|") {
			t.Errorf("%s: coverage-gate run-pattern does not include InstructionFence:\n  %s", f, line)
		}
	}
}

// TestPolicy_InstructionFence runs the fence over this repository's
// commits since AIWF_COVERAGE_BASE.
func TestPolicy_InstructionFence(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		t.Skip("AIWF_COVERAGE_BASE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicyInstructionFence)
}

// TestPolicyInstructionFence_Env drives the env-fed entry point: an unset
// base no-ops, and a set one delegates. Serial (t.Setenv panics under
// t.Parallel) and documented in setup_test.go's skip-list.
func TestPolicyInstructionFence_Env(t *testing.T) {
	t.Setenv("AIWF_COVERAGE_BASE", "")
	if vs, err := PolicyInstructionFence(t.TempDir()); err != nil || vs != nil {
		t.Fatalf("unset base: violations %+v, err %v; want none", vs, err)
	}
	root, base, _ := instructionFenceRepo(t)
	t.Setenv("AIWF_COVERAGE_BASE", base)
	if vs, err := PolicyInstructionFence(root); err != nil || len(vs) != 1 {
		t.Fatalf("set base: violations %+v, err %v; want one", vs, err)
	}
}
