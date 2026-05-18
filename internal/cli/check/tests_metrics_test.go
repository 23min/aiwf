package check

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestRunTestsMetricsCheck_RequireFalseIsNoop pins the fast-path:
// when require=false (the default), the function returns nil findings
// without touching git, regardless of tree contents.
func TestRunTestsMetricsCheck_RequireFalseIsNoop(t *testing.T) {
	t.Parallel()
	findings, err := RunTestsMetricsCheck(context.Background(), t.TempDir(), &tree.Tree{}, false)
	if err != nil {
		t.Fatalf("RunTestsMetricsCheck: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected nil findings with require=false; got %+v", findings)
	}
}

// TestRunTestsMetricsCheck_EmptyRepoIsNoop pins the second guard:
// when the root isn't a git repo (no HEAD), the function returns nil
// rather than erroring on history walks.
func TestRunTestsMetricsCheck_EmptyRepoIsNoop(t *testing.T) {
	t.Parallel()
	findings, err := RunTestsMetricsCheck(context.Background(), t.TempDir(), &tree.Tree{}, true)
	if err != nil {
		t.Fatalf("RunTestsMetricsCheck: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected nil findings on non-git tempdir; got %+v", findings)
	}
}
