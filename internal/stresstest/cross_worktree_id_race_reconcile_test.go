package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// cross_worktree_id_race_reconcile_test.go — reconcile's own refusal,
// decided from fabricated envelopes. The scenario that races two real
// worktrees stays in cross_worktree_id_race_test.go; this guard fires
// before any git work, so it belongs in the lane that runs on every
// push.

// TestCrossWorktreeIDRaceScenario_ReconcileErrorsWhenAnActorDidNotSucceed
// drives reconcile directly with a fabricated non-"ok" envelope,
// pinning the defensive guard against ever attempting to merge or
// classify a race whose add itself failed.
func TestCrossWorktreeIDRaceScenario_ReconcileErrorsWhenAnActorDidNotSucceed(t *testing.T) {
	t.Parallel()
	s := NewCrossWorktreeIDRaceScenario("unused", entity.KindGap, 1)
	err := s.reconcile(t.TempDir(), verbEnvelope{Status: "ok"}, verbEnvelope{Status: "error"})
	if err == nil {
		t.Fatal("expected reconcile to error when an actor's add did not report ok")
	}
}
