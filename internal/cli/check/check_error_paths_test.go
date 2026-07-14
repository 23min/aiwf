package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// M-0256/AC-1 backfill: Run's ResolveRoot guard, the RunProvenanceCheck
// and RunTestsMetricsCheck error propagators, and every os.Stdout write
// guard are `//coverage:ignore`d in check.go itself (see the file for
// per-line rationale). The branches below are genuinely triggerable.
//
// Serial: TestRun_PrettyWithoutJSONWarns uses testutil.CaptureStderr,
// which swaps the process-global os.Stderr.

func writeAiwfYAML(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
}

// TestRun_PrettyWithoutJSONWarns covers Run's --pretty-without-json
// advisory branch: --pretty has no effect without --format=json, so
// Run prints a warning and continues. --shape-only keeps the rest of
// the run trivial against an empty tree.
func TestRun_PrettyWithoutJSONWarns(t *testing.T) {
	root := t.TempDir()
	stderr := testutil.CaptureStderr(t, func() {
		Run(root, "text", true, "", true, false, false, nil, "")
	})
	if !strings.Contains(string(stderr), "--pretty has no effect without --format=json") {
		t.Errorf("stderr = %q, want the --pretty warning", stderr)
	}
}

// TestRun_LoadTreeWithTrunkFailure covers Run's bare
// cliutil.LoadTreeWithTrunk guard: a syntactically broken aiwf.yaml
// makes config.Load (called inside LoadTreeWithTrunk) fail with a
// non-ErrNotFound error, which LoadTreeWithTrunk wraps and returns.
func TestRun_LoadTreeWithTrunkFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, "tdd: [unterminated\n")
	code := Run(root, "text", false, "", false, false, false, nil, "")
	if code != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", code)
	}
}

// TestRun_LoadContractsBlockFailure covers Run's
// cliutil.LoadContractsBlock guard, reusing the malformed-contracts-
// block trigger already proven at
// internal/cli/add/add_error_paths_test.go: lenient enough for
// config.Load's non-strict top-level parse (so LoadTreeWithTrunk
// succeeds) but invalid under aiwfyaml.Read's stricter contracts-block
// parsing.
func TestRun_LoadContractsBlockFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, "contracts:\n  bindings:\n    - not a valid binding\n")
	code := Run(root, "text", false, "", false, false, false, nil, "")
	if code != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", code)
	}
}

// TestRunShapeOnly_TreeLoadCanceledContext and
// TestRunFast_TreeLoadCanceledContext cover tree.Load's fatal
// ctx.Err() guard (internal/tree/tree.go's walkRoots loop checks
// ctx.Err() before touching the filesystem on every iteration) via
// each --shape-only/--fast sub-mode's own bare tree.Load call —
// unexported so only reachable via a direct in-package call, not
// through the public Run entry point (which always passes a fresh
// context.Background()).
func TestRunShapeOnly_TreeLoadCanceledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	code := runShapeOnly(ctx, t.TempDir(), "text", false)
	if code != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", code)
	}
}

func TestRunFast_TreeLoadCanceledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	code := runFast(ctx, t.TempDir(), "text", false)
	if code != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", code)
	}
}
