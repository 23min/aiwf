package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// head_drift_test.go — real-subprocess coverage for HeadDriftScenario
// (M-0243/AC-5). The pure decision logic (classifyHeadDrift) is
// pinned exhaustively in head_drift_classify_test.go against
// fabricated outcomes; this is the actual scenario, driving a real
// `aiwf promote` subprocess against a repo whose branch was switched
// out from under it between the preflight read and the promote call.
//
// G-0269's pre-commit branch guard closes this gap: the activation
// promote refuses outright once the current branch no longer matches
// its own preflight's expectation, so a real run reports 0 violations.
func TestHeadDriftScenario_RealBinary_GuardPreventsTheIncident(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewHeadDriftScenario(bin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("expected the G-0269 branch guard to refuse the drifted-branch promote and report 0 violations, got: %+v", result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d: %+v", len(result.Violations), result.Violations)
	}
}

func TestHeadDriftScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewHeadDriftScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"))
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "seeding the epic") {
		t.Fatalf("expected the failure to name the seeding step, got: %v", err)
	}
}

// TestHeadDriftScenario_RealBinary_SetupSurfacesASeedingRefusal
// pre-seeds a colliding E-0001 entity file before Setup's own `aiwf
// add` call, mirroring the same pre-seed technique used elsewhere in
// this package, pinning that Setup wraps and surfaces the refusal.
func TestHeadDriftScenario_RealBinary_SetupSurfacesASeedingRefusal(t *testing.T) {
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

	s := NewHeadDriftScenario(bin)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to surface the seeding refusal")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to name the seeding step, got: %v", err)
	}
}
