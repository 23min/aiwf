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
// Per this milestone's own Constraints, AC-5 is allowed to fail
// (expected-red): the confirmed defect IS the expected outcome this
// scenario exists to demonstrate.
func TestHeadDriftScenario_RealBinary_ConfirmsTheIncidentStillReproduces(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewHeadDriftScenario(bin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if result.Passed {
		t.Fatal("expected the scenario to report the confirmed G-0269 head-drift defect as a violation, not pass cleanly")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected exactly 1 violation (the confirmed head-drift landing), got %d: %+v", len(result.Violations), result.Violations)
	}
	if !strings.Contains(result.Violations[0].Message, "G-0269") {
		t.Fatalf("expected the violation to name G-0269, got: %+v", result.Violations[0])
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
