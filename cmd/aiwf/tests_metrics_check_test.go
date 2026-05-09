package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// TestRunTestsMetricsCheck_DefaultIsSilent: with require_test_metrics
// off (the default), the check produces no findings even when an AC
// is at tdd_phase: done with no aiwf-tests trailer in history.
func TestRunTestsMetricsCheck_DefaultIsSilent(t *testing.T) {
	root := setupTDDDoneAC(t)

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	findings, err := runTestsMetricsCheck(context.Background(), root, tr, false)
	if err != nil {
		t.Fatalf("runTestsMetricsCheck: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings when require=false; got %+v", findings)
	}
}

// TestRunTestsMetricsCheck_WarnsWhenRequireOnAndTrailerMissing:
// require=true, milestone tdd: required, AC at done, no commit in
// history carries aiwf-tests → exactly one warning fires.
func TestRunTestsMetricsCheck_WarnsWhenRequireOnAndTrailerMissing(t *testing.T) {
	root := setupTDDDoneAC(t)

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	findings, err := runTestsMetricsCheck(context.Background(), root, tr, true)
	if err != nil {
		t.Fatalf("runTestsMetricsCheck: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding; got %d (%+v)", len(findings), findings)
	}
	f := findings[0]
	if f.Code != "acs-tdd-tests-missing" {
		t.Errorf("Code = %q, want acs-tdd-tests-missing", f.Code)
	}
	if f.Severity != check.SeverityWarning {
		t.Errorf("Severity = %q, want warning", f.Severity)
	}
	if f.EntityID != "M-0001/AC-1" {
		t.Errorf("EntityID = %q, want M-001/AC-1", f.EntityID)
	}
}

// TestRunTestsMetricsCheck_SilentWhenTrailerOnHistory: when at least
// one commit in the AC's history carries an aiwf-tests trailer, the
// warning does not fire even with require=true.
func TestRunTestsMetricsCheck_SilentWhenTrailerOnHistory(t *testing.T) {
	root := setupTDDDoneAC(t)

	// Hand-author an empty commit on M-001/AC-1 carrying aiwf-tests
	// (mimics what step-2a's --tests flag will produce on real
	// promotion). The history walker reads via the tolerant trailer
	// parser.
	const subject = "promote(M-001/AC-1) green metrics"
	const body = "aiwf-verb: promote\naiwf-entity: M-001/AC-1\naiwf-actor: human/test\naiwf-to: green\naiwf-tests: pass=12 fail=0 skip=0\n"
	if err := osExec(t, root, "git", "commit", "--allow-empty", "-m", subject, "-m", body); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	findings, err := runTestsMetricsCheck(context.Background(), root, tr, true)
	if err != nil {
		t.Fatalf("runTestsMetricsCheck: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings (trailer is on a commit in AC history); got %+v", findings)
	}
}

// TestRunTestsMetricsCheck_SilentForNonRequiredMilestone: a milestone
// without `tdd: required` does not get the warning even when the AC
// is at tdd_phase: done. Load-bearing — the warning is a property of
// the consumer's TDD policy, not a universal AC requirement.
func TestRunTestsMetricsCheck_SilentForNonRequiredMilestone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Optional", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "Engine")
	// tdd is not set on the milestone (default: not required); promote
	// the AC's status to met. tdd_phase remains absent.
	mustRun(t, "promote", "--actor", "human/test", "--root", root, "M-0001/AC-1", "met")

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	findings, err := runTestsMetricsCheck(context.Background(), root, tr, true)
	if err != nil {
		t.Fatalf("runTestsMetricsCheck: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings on non-tdd-required milestone; got %+v", findings)
	}
}

// TestRunCheck_TestsMetricsWarningSurfacesViaDispatcher: the
// dispatcher seam test — `aiwf check` reads aiwf.yaml's
// tdd.require_test_metrics flag and surfaces the warning code in its
// output. Catches the bug class where the check function exists but
// runCheck forgets to call it.
func TestRunCheck_TestsMetricsWarningSurfacesViaDispatcher(t *testing.T) {
	root := setupTDDDoneAC(t)

	// Flip require_test_metrics on in aiwf.yaml.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "tdd:\n  require_test_metrics: true\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	captured := captureStdout(t, func() {
		// rc=1 expected because the warning fires; check exits 1 on
		// any findings.
		_ = run([]string{"check", "--root", root})
	})
	if !strings.Contains(string(captured), "acs-tdd-tests-missing") {
		t.Errorf("expected acs-tdd-tests-missing in check output; got:\n%s", captured)
	}
}

// setupTDDDoneAC scaffolds a repo with one milestone marked tdd:
// required and one AC walked through red→green→done — without any
// aiwf-tests trailer along the way. Used as the fixture for the
// require=true + missing-trailer assertion path.
func setupTDDDoneAC(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "required", "--epic", "E-0001", "--title", "Required", "--actor", "human/test", "--root", root)

	mustRun(t, "add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "Engine")
	// AC is auto-seeded at red because the milestone is tdd: required;
	// walk it to done with no metrics flagged.
	mustRun(t, "promote", "--actor", "human/test", "--root", root, "M-0001/AC-1", "--phase", "green")
	mustRun(t, "promote", "--actor", "human/test", "--root", root, "M-0001/AC-1", "--phase", "done")
	return root
}
