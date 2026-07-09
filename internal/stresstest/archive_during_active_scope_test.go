package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// archive_during_active_scope_test.go — real-subprocess coverage for
// ArchiveDuringActiveScopeScenario (M-0243/AC-3). The pure decision
// logic (classifyArchiveDuringActiveScope) is pinned exhaustively in
// archive_during_active_scope_classify_test.go against fabricated
// outcomes; this is the actual scenario, driving a real epic/milestone
// pair through a real authorize-then-archive sequence.

func TestArchiveDuringActiveScopeScenario_RealBinary_ConfirmsScopeSurvivesTheSweep(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewArchiveDuringActiveScopeScenario(bin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("archive-during-active-scope scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
	}
}

func TestArchiveDuringActiveScopeScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewArchiveDuringActiveScopeScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"))
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "seeding the parent epic") {
		t.Fatalf("expected the failure to name the seeding step, got: %v", err)
	}
}

// TestArchiveDuringActiveScopeScenario_RealBinary_SetupSurfacesASeedingRefusal
// pre-seeds a colliding E-0001 entity file before Setup's own `aiwf
// add` call, mirroring M-0241/AC-5's same pre-seed technique, pinning
// that Setup wraps and surfaces the refusal.
func TestArchiveDuringActiveScopeScenario_RealBinary_SetupSurfacesASeedingRefusal(t *testing.T) {
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

	s := NewArchiveDuringActiveScopeScenario(bin)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to surface the seeding refusal")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to name the seeding step, got: %v", err)
	}
}

// TestScopeState_ReturnsEmptyStringWhenNoScopesPresent pins scopeState's
// empty-scopes branch directly — a real "show" call before any
// authorize was ever opened returns no scopes array at all.
func TestScopeState_ReturnsEmptyStringWhenNoScopesPresent(t *testing.T) {
	t.Parallel()
	if got := scopeState(verbEnvelope{}); got != "" {
		t.Fatalf("scopeState on an empty envelope = %q, want empty string", got)
	}
}

// TestHasFindingForEntity_MatchesOnBothCodeAndEntityID pins
// hasFindingForEntity's discriminating branches: a finding with the
// right code but wrong entity, or the right entity but wrong code,
// must not count as a match.
func TestHasFindingForEntity_MatchesOnBothCodeAndEntityID(t *testing.T) {
	t.Parallel()
	findings := []verbEnvelopeFinding{
		{Code: check.CodeArchivedEntityNotTerminal, EntityID: "M-0002"},
		{Code: "some-other-code", EntityID: "M-0001"},
	}
	if hasFindingForEntity(findings, check.CodeArchivedEntityNotTerminal, "M-0001") {
		t.Fatal("expected no match: code matches a different entity")
	}
	if hasFindingForEntity(findings, "some-other-code", "M-0002") {
		t.Fatal("expected no match: entity matches a different code")
	}
	if !hasFindingForEntity(findings, check.CodeArchivedEntityNotTerminal, "M-0002") {
		t.Fatal("expected a match on the exact code+entity pair")
	}
}
