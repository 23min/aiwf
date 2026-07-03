package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestRefreshStatuslineInPlace_PrintsLedgerForUnmarkedCopy covers the
// ledger-print loop in refreshStatuslineInPlace (update.go:176-179) — the
// diff-scoped-coverage gap the G-0344 statusline work left on main.
//
// An *unmarked* installed statusline is the deterministic show=true path:
// decideStatuslineRefresh returns "skipped (unmarked)" without any version
// comparison, so LedgerLine reports show=true regardless of the test binary's
// (devel) version — which an in-process build can never make equal to an
// installed stamp. $HOME points at an empty dir so only the project-scope
// copy contributes an outcome, keeping the printed ledger deterministic.
//
// Serial (no t.Parallel): mutates $HOME via t.Setenv and captures os.Stdout
// (via testutil.CaptureStdout) — both process-globals; see setup_test.go.
func TestRefreshStatuslineInPlace_PrintsLedgerForUnmarkedCopy(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // empty home → no user-scope statusline candidate

	root := t.TempDir()
	claudeDir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// An aiwf-unmarked script: no `# aiwf-statusline version:` marker line,
	// so the refresh decision is "skipped (unmarked)" — a show=true outcome.
	script := filepath.Join(claudeDir, "statusline.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho hello\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := string(testutil.CaptureStdout(t, func() { refreshStatuslineInPlace(root) }))

	if !strings.Contains(out, "statusline") || !strings.Contains(strings.ToLower(out), "skipped") {
		t.Errorf("expected a `skipped` ledger line for the unmarked project statusline, got %q", out)
	}
}
