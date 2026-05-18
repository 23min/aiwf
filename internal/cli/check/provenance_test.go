package check

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestRunProvenanceCheck_EmptyRepoIsNoop pins the fast-path: when the
// root isn't a git repo (no HEAD), RunProvenanceCheck returns nil
// without erroring on the absent git log.
func TestRunProvenanceCheck_EmptyRepoIsNoop(t *testing.T) {
	t.Parallel()
	findings, err := RunProvenanceCheck(context.Background(), t.TempDir(), &tree.Tree{}, "")
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected nil findings on non-git tempdir; got %+v", findings)
	}
}

// TestParseUntrailedCommits_EmptyInput pins the parser's empty-input
// branch. Other parser shapes are exercised via the cmd/aiwf-side
// integration tests (TestParseUntrailedCommits_Malformed) that
// migrate with the rest of the integration test set in AC-6.
func TestParseUntrailedCommits_EmptyInput(t *testing.T) {
	t.Parallel()
	got := ParseUntrailedCommits("")
	if len(got) != 0 {
		t.Errorf("ParseUntrailedCommits(\"\") = %+v, want empty", got)
	}
}
