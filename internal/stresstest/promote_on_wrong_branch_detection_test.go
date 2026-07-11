package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// promote_on_wrong_branch_detection_test.go — real-subprocess coverage
// for PromoteOnWrongBranchDetectionScenario (G-0270): drives a real
// `aiwf promote` + `aiwf check` subprocess pair against a repo whose
// branch was switched out from under the activation promote, then
// checked back out before check runs — the same real-binary shape
// head_drift_test.go uses for the sibling G-0269 scenario.
//
// Unlike HeadDriftScenario's own AC-5 (deliberately expected-red),
// this scenario is expected to PASS: G-0270's fix (the ancestor-check
// redesign of promote-on-wrong-branch plus the local-branch-scoped
// candidate gather) means `aiwf check`, even run from a different
// branch than the one the commit landed on and even though that
// branch's name doesn't match any ritual shape, now reports the
// misplacement.
func TestPromoteOnWrongBranchDetectionScenario_RealBinary_DetectsTheMisplacedActivation(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewPromoteOnWrongBranchDetectionScenario(bin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("expected aiwf check to detect the misplaced activation commit; got violations: %+v", result.Violations)
	}
}

func TestPromoteOnWrongBranchDetectionScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewPromoteOnWrongBranchDetectionScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"))
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "seeding the epic") {
		t.Fatalf("expected the failure to name the seeding step, got: %v", err)
	}
}

// TestPromoteOnWrongBranchDetectionScenario_RealBinary_SetupSurfacesASeedingRefusal
// pre-seeds a colliding E-0001 entity file before Setup's own `aiwf
// add` call, mirroring the same pre-seed technique
// head_drift_test.go's sibling test uses, pinning that Setup wraps
// and surfaces the refusal.
func TestPromoteOnWrongBranchDetectionScenario_RealBinary_SetupSurfacesASeedingRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	epicsDir := filepath.Join(dir, "work", "epics", "E-0001-collision")
	if err := os.MkdirAll(epicsDir, 0o755); err != nil {
		t.Fatalf("mkdir colliding epic dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(epicsDir, "epic.md"), []byte("not valid frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write colliding epic file: %v", err)
	}

	s := NewPromoteOnWrongBranchDetectionScenario(bin)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to surface the seeding refusal")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to name the seeding step, got: %v", err)
	}
}
