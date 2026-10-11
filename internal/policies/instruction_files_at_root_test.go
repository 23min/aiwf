package policies

import (
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
