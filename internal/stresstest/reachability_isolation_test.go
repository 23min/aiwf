package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// reachability_isolation_test.go — real-subprocess coverage for
// ReachabilityIsolationScenario (M-0241/AC-5). The pure decision
// logic (classifyReachabilityIsolation) is pinned exhaustively in
// reachability_isolation_classify_test.go against fabricated
// envelopes; this is the actual AC-5 scenario, deterministic (not a
// race like AC-2/AC-3) since it's a plain sequential commit-then-
// observe, never dependent on timing.

func TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewReachabilityIsolationScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), entity.KindGap, 1)
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "baseline check in worktree A") {
		t.Fatalf("expected the launch failure to name the baseline-check step, got: %v", err)
	}
}

// TestProbeShowFound_RealBinary pins probeShowFound's two direct
// outcomes against a real binary: found for an entity that exists,
// not-found for one that doesn't — the latter is exactly the
// G-0389 not-honoring-format=json path this helper works around.
func TestProbeShowFound_RealBinary(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	addEnv, err := runAiwfJSON(bin, dir, "add", "gap", "--title", "t", "--body", "b")
	if err != nil {
		t.Fatalf("add gap: %v", err)
	}
	id := addEnv.Metadata.EntityID

	found, err := probeShowFound(bin, dir, id)
	if err != nil {
		t.Fatalf("probeShowFound (existing id): %v", err)
	}
	if !found {
		t.Fatal("expected probeShowFound to report found for an existing entity")
	}

	notFound, err := probeShowFound(bin, dir, "G-9999")
	if err != nil {
		t.Fatalf("probeShowFound (missing id): %v", err)
	}
	if notFound {
		t.Fatal("expected probeShowFound to report not-found for a nonexistent entity")
	}
}

// TestReachabilityIsolationScenario_RealBinary_RunSurfacesACreationRefusal
// pre-seeds a colliding G-0001 entity file in worktree B (an id
// collision the `ids-unique` rule refuses at error severity) so
// Run's `aiwf add` call in worktree B reports something other than
// "ok", pinning that Run wraps and surfaces the refusal.
func TestReachabilityIsolationScenario_RealBinary_RunSurfacesACreationRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewReachabilityIsolationScenario(bin, entity.KindGap, 1)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	gapsDir := filepath.Join(dir, "wt-b", "work", "gaps")
	if err := os.MkdirAll(gapsDir, 0o755); err != nil {
		t.Fatalf("mkdir colliding gap dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gapsDir, "G-0001-collision.md"), []byte("not valid frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write colliding gap file: %v", err)
	}

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to surface the id-collision refusal from the `aiwf add` call in worktree B")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to be reported as a non-ok status, got: %v", err)
	}
}

// TestReachabilityIsolationScenario_RealBinaryConfirmsIsolationAndItsClose
// is the AC-5 scenario itself: a real commit in worktree B, observed
// (or rather not observed) from worktree A, then confirmed to close
// once merged.
func TestReachabilityIsolationScenario_RealBinaryConfirmsIsolationAndItsClose(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewReachabilityIsolationScenario(bin, entity.KindGap, 1)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("reachability-isolation scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
	}
}
