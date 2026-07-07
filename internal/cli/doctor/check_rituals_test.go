package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/initrepo"
)

// freshInitializedRootForRituals builds a real, fully-materialized
// aiwf repo (via initrepo.Init) — the "healthy worktree" fixture
// checkRitualsResult must report ok against.
func freshInitializedRootForRituals(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{SkipHook: true}); err != nil {
		t.Fatalf("initrepo.Init: %v", err)
	}
	return root
}

// TestCheckRitualsResult_FullyMaterializedReportsOK pins AC-1's silent
// exit-0 path: a repo with every ritual artifact present reports ok,
// with no message to surface.
func TestCheckRitualsResult_FullyMaterializedReportsOK(t *testing.T) {
	t.Parallel()
	root := freshInitializedRootForRituals(t)
	ok, message, err := checkRitualsResult(root)
	if err != nil {
		t.Fatalf("checkRitualsResult: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true; message = %q", message)
	}
	if message != "" {
		t.Errorf("message = %q, want empty for the ok case", message)
	}
}

// TestCheckRitualsResult_MissingRitualsReportsActionableMessage pins
// AC-1's actionable-stderr claim: a worktree missing ritual artifacts
// reports not-ok with a message naming the count and pointing at
// `aiwf update`.
func TestCheckRitualsResult_MissingRitualsReportsActionableMessage(t *testing.T) {
	t.Parallel()
	root := freshInitializedRootForRituals(t)
	// Remove one materialized ritual skill (an aiwfx-*/wf-* skill — the
	// set skills.MaterializedRituals actually walks, distinct from the
	// per-verb aiwf-* skills) to simulate a partially-materialized
	// worktree (e.g. an interrupted `aiwf worktree add`).
	ritualSkillDir := filepath.Join(root, ".claude", "skills", "wf-tdd-cycle")
	if _, statErr := os.Stat(ritualSkillDir); statErr != nil {
		t.Fatalf("fixture assumption broken — %s not present after init: %v", ritualSkillDir, statErr)
	}
	if rmErr := os.RemoveAll(ritualSkillDir); rmErr != nil {
		t.Fatalf("removing %s: %v", ritualSkillDir, rmErr)
	}

	ok, message, err := checkRitualsResult(root)
	if err != nil {
		t.Fatalf("checkRitualsResult: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false after removing a materialized skill")
	}
	if !strings.Contains(message, "aiwf update") {
		t.Errorf("message = %q, want it to mention `aiwf update`", message)
	}
}

// TestCheckRitualsResult_EmptyRootReportsAllMissing pins the primary
// target scenario: a bare worktree that never ran `aiwf init`/`update`
// at all (nothing under .claude/) reports not-ok.
func TestCheckRitualsResult_EmptyRootReportsAllMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ok, message, err := checkRitualsResult(root)
	if err != nil {
		t.Fatalf("checkRitualsResult: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false for a bare directory with no .claude/")
	}
	if !strings.Contains(message, "aiwf update") {
		t.Errorf("message = %q, want it to mention `aiwf update`", message)
	}
}

// TestRunCheckRituals_FullyMaterializedExitsOK pins the exit-code
// contract for the healthy case.
func TestRunCheckRituals_FullyMaterializedExitsOK(t *testing.T) {
	t.Parallel()
	root := freshInitializedRootForRituals(t)
	if got := RunCheckRituals(root); got != cliutil.ExitOK {
		t.Errorf("RunCheckRituals() = %d, want ExitOK", got)
	}
}

// TestRunCheckRituals_MissingRitualsExitsFindings pins the exit-code
// contract for the stale/missing case — ExitFindings (1), matching the
// repo's exit-code convention for "at least one error-severity finding".
func TestRunCheckRituals_MissingRitualsExitsFindings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if got := RunCheckRituals(root); got != cliutil.ExitFindings {
		t.Errorf("RunCheckRituals() = %d, want ExitFindings", got)
	}
}
