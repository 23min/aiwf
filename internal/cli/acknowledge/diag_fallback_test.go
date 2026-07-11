package acknowledge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/gitops"
)

// diag_fallback_test.go — M-0249 follow-up: pins the `runID == ""`
// correlation-id fallback inside runIllegal / runMistag. cli.Execute
// always mints a real correlation id (NewRootCmd), so this branch is
// unreachable through the CLI surface — only a direct call bypassing
// NewCmd/Execute (a hand-built OutputFormat, zero value
// CorrelationID) exercises it. Mirrors the pattern established in
// internal/cli/contract/diag_fallback_internal_test.go.

// minimalRepo git-inits a bare aiwf.yaml-carrying repo — just enough
// for cliutil.ResolveRoot to succeed. runIllegal/runMistag's diagLog
// block runs before AcquireRepoLock, so the fallback fires regardless
// of whether the verb itself later succeeds.
func minimalRepo(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "aiwf.yaml"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := gitops.Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	return root
}

func fallbackRunID(t *testing.T, path string) string {
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

func TestRunIllegal_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := minimalRepo(t)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	// The SHA doesn't need to resolve — the diagLog fallback fires
	// before AcquireRepoLock, regardless of the verb's eventual
	// outcome.
	_ = runIllegal("deadbeef", "human/test", root, "testing the fallback", "", cliutil.OutputFormat{})

	if got := fallbackRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestRunMistag_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := minimalRepo(t)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	_ = runMistag("G-0001", "human/test", root, "testing the fallback", cliutil.OutputFormat{})

	if got := fallbackRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}
