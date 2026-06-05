package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// branch_of_sha_test.go — M-0161/AC-6 (G-0206) unit-level
// coverage of gitBranchOracle.BranchOfSHA per CLAUDE.md
// §"Test the seam, not just the layer" + §"Test untested
// code paths before declaring code paths 'done'". The E2E
// coverage at internal/cli/integration/isolation_escape_rename_scenarios_test.go
// exercises the rule's resolution flow; these unit tests
// pin the BranchOfSHA helper's branch-selection contract.
//
// AC-6's tie-breaking heuristic:
//
//  1. Filter ritual candidates (exclude "main"); fall back
//     to full set if no ritual candidate exists.
//  2. Among candidates, prefer the one where the recorded
//     SHA is closest to the tip (smallest distance).
//  3. On distance-ties, prefer the alphabetically earlier
//     branch (deterministic — matches the oracle's iteration
//     order).

// TestBranchOfSHA_AC6_UnknownSHA_ReturnsEmpty pins the
// fail-shut-on-correctness contract: a SHA not on any
// ritual ref returns empty (rule then stays silent).
func TestBranchOfSHA_AC6_UnknownSHA_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	root := setupAC3RepoAllHealthy(t)
	oracle, err := newGitBranchOracle(context.Background(), root)
	if err != nil {
		t.Fatalf("newGitBranchOracle: %v", err)
	}
	got := oracle.BranchOfSHA("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if got != "" {
		t.Errorf("BranchOfSHA(unknown SHA) = %q; want empty (fail-shut on correctness)", got)
	}
}

// TestBranchOfSHA_AC6_SingleOwner_ReturnsRitualBranch pins
// the simple case: a SHA on exactly one ritual branch
// (epic/E-0001-engine) returns that branch.
func TestBranchOfSHA_AC6_SingleOwner_ReturnsRitualBranch(t *testing.T) {
	t.Parallel()
	root := setupAC3RepoAllHealthy(t)
	headSHA := gitOutput(t, root, "rev-parse", "refs/heads/epic/E-0001-engine")

	oracle, err := newGitBranchOracle(context.Background(), root)
	if err != nil {
		t.Fatalf("newGitBranchOracle: %v", err)
	}
	got := oracle.BranchOfSHA(headSHA)
	if got != "epic/E-0001-engine" {
		t.Errorf("BranchOfSHA(epic tip) = %q; want %q", got, "epic/E-0001-engine")
	}
}

// TestBranchOfSHA_AC6_SharedSHA_PrefersRitualOverTrunk pins
// the trunk-exclusion heuristic: a SHA reachable from both
// main and a ritual branch (e.g., the original tip when a
// branch is cut from main) resolves to the ritual branch.
// The bound branch is by definition ritual; trunk is excluded.
func TestBranchOfSHA_AC6_SharedSHA_PrefersRitualOverTrunk(t *testing.T) {
	t.Parallel()
	root := setupAC6SharedSHARepo(t)
	// The recorded SHA is main's pre-branch-cut tip; both main
	// and epic/E-0001-engine reach it via first-parent.
	sharedSHA := gitOutput(t, root, "rev-parse", "main")

	oracle, err := newGitBranchOracle(context.Background(), root)
	if err != nil {
		t.Fatalf("newGitBranchOracle: %v", err)
	}
	got := oracle.BranchOfSHA(sharedSHA)
	if got != "epic/E-0001-engine" {
		t.Errorf("BranchOfSHA(shared SHA) = %q; want %q (ritual preferred over main)", got, "epic/E-0001-engine")
	}
}

// TestBranchOfSHA_AC6_RenameSurvives pins the load-bearing
// claim: after `git branch -m foo bar`, the recorded SHA
// still resolves to whichever ritual branch reaches it —
// the renamed-to branch (`bar`).
func TestBranchOfSHA_AC6_RenameSurvives(t *testing.T) {
	t.Parallel()
	root := setupAC6RenamedRepo(t)
	// SHA recorded at scope-open = the tip BEFORE rename. The
	// rename preserves the tip on the renamed-to branch.
	renamedTip := gitOutput(t, root, "rev-parse", "refs/heads/epic/E-0001-renamed")

	oracle, err := newGitBranchOracle(context.Background(), root)
	if err != nil {
		t.Fatalf("newGitBranchOracle: %v", err)
	}
	got := oracle.BranchOfSHA(renamedTip)
	if got != "epic/E-0001-renamed" {
		t.Errorf("BranchOfSHA(renamed tip) = %q; want %q (rename survival)", got, "epic/E-0001-renamed")
	}
}

// TestBranchOfSHA_AC6_OnlyTrunkOwner_ReturnsTrunk pins the
// fallback path: when the SHA is on main only (no ritual
// candidate), main wins. Edge case for completeness; not a
// case the rule would normally consult (the bound branch is
// always ritual).
func TestBranchOfSHA_AC6_OnlyTrunkOwner_ReturnsTrunk(t *testing.T) {
	t.Parallel()
	root := setupAC3RepoAllHealthy(t)
	// Use a SHA UNIQUE to main: cut a new commit on main that's
	// NOT in epic. Then BranchOfSHA(mainOnlySHA) has only main
	// as a candidate → fallback returns main.
	if err := os.WriteFile(filepath.Join(root, "main-only.md"), []byte("main-only\n"), 0o644); err != nil {
		t.Fatalf("write main-only.md: %v", err)
	}
	gitRun(t, root, "add", "main-only.md")
	gitRun(t, root, "commit", "-m", "main-only commit")
	mainOnlySHA := strings.TrimSpace(gitOutput(t, root, "rev-parse", "main"))

	oracle, err := newGitBranchOracle(context.Background(), root)
	if err != nil {
		t.Fatalf("newGitBranchOracle: %v", err)
	}
	got := oracle.BranchOfSHA(mainOnlySHA)
	if got != "main" {
		t.Errorf("BranchOfSHA(main-only SHA) = %q; want %q (fallback when no ritual owner)", got, "main")
	}
}

// setupAC6SharedSHARepo builds a repo where main and a ritual
// branch share a SHA in their first-parent indexes. Used by
// the trunk-exclusion test.
func setupAC6SharedSHARepo(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	gitRun(t, root, "branch", "-M", "main")
	writeFile(t, root, "seed.md", "seed\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "baseline")
	// Cut the ritual branch from main's current tip and stay
	// at that point (don't advance main; the shared SHA is
	// main's current tip = ritual.tip).
	gitRun(t, root, "checkout", "-b", "epic/E-0001-engine")
	gitRun(t, root, "checkout", "main")
	return root
}

// setupAC6RenamedRepo builds a repo with a renamed ritual
// branch. The renamed branch's tip is the same SHA it had
// before the rename — the rename preserves the tip.
func setupAC6RenamedRepo(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	gitRun(t, root, "branch", "-M", "main")
	writeFile(t, root, "seed.md", "seed\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "baseline")

	gitRun(t, root, "checkout", "-b", "epic/E-0001-engine")
	writeFile(t, root, "epic.md", "epic\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "epic work")
	// Rename.
	gitRun(t, root, "branch", "-m", "epic/E-0001-engine", "epic/E-0001-renamed")
	gitRun(t, root, "checkout", "main")
	return root
}
