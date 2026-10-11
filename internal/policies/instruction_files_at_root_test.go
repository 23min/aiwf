package policies

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestPolicy_InstructionFilesAtRoot holds the live tree to M-0335 AC-5: no
// tracked CLAUDE.md or AGENTS.md exists below the repository root.
func TestPolicy_InstructionFilesAtRoot(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyInstructionFilesAtRoot)
}

// TestInstructionFilesBelowRoot pins the rule (M-0335 AC-5): a host
// instruction file is allowed at the root and reported, by its path,
// anywhere below it.
func TestInstructionFilesBelowRoot(t *testing.T) {
	t.Parallel()
	vs := instructionFilesBelowRoot([]string{
		"CLAUDE.md",
		"AGENTS.md",
		"internal/cli/CLAUDE.md",
		"docs/AGENTS.md",
		"docs/overview.md",
	})
	var got []string
	for _, v := range vs {
		if v.Policy != "instruction-files-at-root" {
			t.Errorf("violation Policy = %q, want instruction-files-at-root", v.Policy)
		}
		got = append(got, v.File)
	}
	if diff := cmp.Diff([]string{"internal/cli/CLAUDE.md", "docs/AGENTS.md"}, got); diff != "" {
		t.Errorf("reported files (-want +got):\n%s", diff)
	}
}

// TestPolicyInstructionFilesAtRoot_ReadsTrackedFiles pins the policy's
// reading of the tree (M-0335 AC-5): it judges the files git tracks, so a
// tracked nested instruction file is reported and an untracked one is
// not, and a directory git cannot list is an error rather than a pass.
func TestPolicyInstructionFilesAtRoot_ReadsTrackedFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "root\n")
	mustWrite(t, filepath.Join(root, "docs", "CLAUDE.md"), "tracked\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "CLAUDE.md", "docs/CLAUDE.md"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	mustWrite(t, filepath.Join(root, "notes", "AGENTS.md"), "untracked\n")
	vs, err := PolicyInstructionFilesAtRoot(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 || vs[0].File != "docs/CLAUDE.md" {
		t.Errorf("violations = %+v, want one on docs/CLAUDE.md", vs)
	}
	if _, err := PolicyInstructionFilesAtRoot(t.TempDir()); err == nil {
		t.Error("a directory outside any repository must be an error, not a pass")
	}
}
