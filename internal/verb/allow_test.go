package verb_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/scope"
	"github.com/23min/ai-workflow-v2/internal/tree"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// TestAllow_HumanActorBypassesScopeCheck: a human/... actor with no
// principal is allowed unconditionally. Scope state and reachability
// are not consulted.
func TestAllow_HumanActorBypassesScopeCheck(t *testing.T) {
	res := verb.Allow(verb.AllowInput{
		Kind:     verb.VerbAct,
		TargetID: "M-007",
		Actor:    "human/peter",
	})
	if !res.Allowed {
		t.Errorf("Allowed = false, want true: %s", res.Reason)
	}
	if res.Scope != nil {
		t.Errorf("Scope = %+v, want nil for direct human act", res.Scope)
	}
}

// TestAllow_HumanActorWithPrincipalRefused: a human acting "on
// behalf of" themselves is incoherent — principal is forbidden when
// the actor is human/.
func TestAllow_HumanActorWithPrincipalRefused(t *testing.T) {
	res := verb.Allow(verb.AllowInput{
		Kind:      verb.VerbAct,
		TargetID:  "M-007",
		Actor:     "human/peter",
		Principal: "human/peter",
	})
	if res.Allowed {
		t.Error("Allowed = true; want false (principal forbidden for human)")
	}
	if !strings.Contains(res.Reason, "principal forbidden") {
		t.Errorf("Reason = %q, want principal-forbidden message", res.Reason)
	}
}

// TestAllow_NonHumanActorNeedsPrincipal: an ai/... actor without a
// principal is refused before scope lookup runs.
func TestAllow_NonHumanActorNeedsPrincipal(t *testing.T) {
	res := verb.Allow(verb.AllowInput{
		Kind:     verb.VerbAct,
		TargetID: "M-007",
		Actor:    "ai/claude",
	})
	if res.Allowed {
		t.Error("Allowed = true; want false (principal required for non-human)")
	}
	if !strings.Contains(res.Reason, "principal required") {
		t.Errorf("Reason = %q, want principal-required message", res.Reason)
	}
}

// TestAllow_NonHumanActorNoActiveScope: an ai/... actor with a
// principal but no active scope on a reachable entity is refused
// with the no-active-scope reason.
func TestAllow_NonHumanActorNoActiveScope(t *testing.T) {
	tr := buildAllowTree(t)
	res := verb.Allow(verb.AllowInput{
		Kind:      verb.VerbAct,
		TargetID:  "M-001",
		Actor:     "ai/claude",
		Principal: "human/peter",
		Tree:      tr,
		Scopes:    nil,
	})
	if res.Allowed {
		t.Error("Allowed = true; want false (no scopes)")
	}
	if !strings.Contains(res.Reason, "no active scope") {
		t.Errorf("Reason = %q, want no-active-scope message", res.Reason)
	}
}

// TestAllow_NonHumanActorAllowedViaActiveScope: an active scope on
// E-01 lets the agent operate on M-001 (which reaches E-01 via
// parent). The matching scope is returned for trailer decoration.
func TestAllow_NonHumanActorAllowedViaActiveScope(t *testing.T) {
	tr := buildAllowTree(t)
	scopes := []*scope.Scope{
		{AuthSHA: "deadbee", Entity: "E-01", Agent: "ai/claude", Principal: "human/peter", State: scope.StateActive},
	}
	res := verb.Allow(verb.AllowInput{
		Kind:      verb.VerbAct,
		TargetID:  "M-001",
		Actor:     "ai/claude",
		Principal: "human/peter",
		Tree:      tr,
		Scopes:    scopes,
	})
	if !res.Allowed {
		t.Fatalf("Allowed = false; want true. Reason: %s", res.Reason)
	}
	if res.Scope == nil || res.Scope.AuthSHA != "deadbee" {
		t.Errorf("Scope = %+v, want the deadbee scope", res.Scope)
	}
}

// TestAllow_NonHumanActorRefusedOutOfScope: an active scope on E-09
// does NOT authorize work on M-001 (which doesn't reach E-09).
func TestAllow_NonHumanActorRefusedOutOfScope(t *testing.T) {
	tr := buildAllowTree(t)
	scopes := []*scope.Scope{
		{AuthSHA: "outscope", Entity: "E-09", Agent: "ai/claude", Principal: "human/peter", State: scope.StateActive},
	}
	res := verb.Allow(verb.AllowInput{
		Kind:      verb.VerbAct,
		TargetID:  "M-001",
		Actor:     "ai/claude",
		Principal: "human/peter",
		Tree:      tr,
		Scopes:    scopes,
	})
	if res.Allowed {
		t.Error("Allowed = true; want false (M-001 does not reach E-09)")
	}
}

// TestAllow_PausedScopeDoesNotAuthorize: a paused scope is ignored.
// The agent must wait for --resume before operating again.
func TestAllow_PausedScopeDoesNotAuthorize(t *testing.T) {
	tr := buildAllowTree(t)
	scopes := []*scope.Scope{
		{AuthSHA: "paused1", Entity: "E-01", State: scope.StatePaused},
	}
	res := verb.Allow(verb.AllowInput{
		Kind:      verb.VerbAct,
		TargetID:  "M-001",
		Actor:     "ai/claude",
		Principal: "human/peter",
		Tree:      tr,
		Scopes:    scopes,
	})
	if res.Allowed {
		t.Error("Allowed = true; want false (paused scope must not authorize)")
	}
}

// TestAllow_PicksMostRecentlyOpenedActiveScope: when multiple active
// scopes match, the most-recently-opened wins (deterministic
// selection). Verified by giving each scope a different AuthSHA and
// checking which one comes back.
func TestAllow_PicksMostRecentlyOpenedActiveScope(t *testing.T) {
	tr := buildAllowTree(t)
	scopes := []*scope.Scope{
		{AuthSHA: "older11", Entity: "E-01", State: scope.StateActive},
		{AuthSHA: "newer22", Entity: "E-01", State: scope.StateActive},
	}
	res := verb.Allow(verb.AllowInput{
		Kind:      verb.VerbAct,
		TargetID:  "M-001",
		Actor:     "ai/claude",
		Principal: "human/peter",
		Tree:      tr,
		Scopes:    scopes,
	})
	if !res.Allowed {
		t.Fatalf("Allowed = false; want true: %s", res.Reason)
	}
	if res.Scope == nil || res.Scope.AuthSHA != "newer22" {
		t.Errorf("Scope = %+v, want newer22 (most-recently-opened)", res.Scope)
	}
}

// TestAllow_VerbCreateUsesCreationRefs: a creation act doesn't have
// a target id in the tree yet; reachability runs against the new
// entity's outbound references.
func TestAllow_VerbCreateUsesCreationRefs(t *testing.T) {
	tr := buildAllowTree(t)
	scopes := []*scope.Scope{
		{AuthSHA: "scope11", Entity: "E-01", State: scope.StateActive},
	}
	// Adding a milestone with parent: E-01 — its creation refs
	// include E-01 itself. Allowed.
	res := verb.Allow(verb.AllowInput{
		Kind:         verb.VerbCreate,
		CreationRefs: []string{"E-01"},
		Actor:        "ai/claude",
		Principal:    "human/peter",
		Tree:         tr,
		Scopes:       scopes,
	})
	if !res.Allowed {
		t.Errorf("Allowed = false; want true: %s", res.Reason)
	}
	// Adding an unrelated entity (parent: E-09) — does not reach E-01.
	res = verb.Allow(verb.AllowInput{
		Kind:         verb.VerbCreate,
		CreationRefs: []string{"E-09"},
		Actor:        "ai/claude",
		Principal:    "human/peter",
		Tree:         tr,
		Scopes:       scopes,
	})
	if res.Allowed {
		t.Error("Allowed = true; want false (E-09 does not reach E-01)")
	}
}

// TestAllow_VerbMoveBothEndpoints: a move act requires BOTH source
// and destination to reach the scope-entity. Either alone is
// insufficient.
func TestAllow_VerbMoveBothEndpoints(t *testing.T) {
	tr := buildAllowTree(t)
	scopes := []*scope.Scope{
		{AuthSHA: "scope11", Entity: "E-01", State: scope.StateActive},
	}
	// Source M-001 reaches E-01; destination M-002 also reaches E-01.
	// Allowed.
	res := verb.Allow(verb.AllowInput{
		Kind:       verb.VerbMove,
		TargetID:   "M-002",
		MoveSource: "M-001",
		Actor:      "ai/claude",
		Principal:  "human/peter",
		Tree:       tr,
		Scopes:     scopes,
	})
	if !res.Allowed {
		t.Fatalf("Allowed = false; want true: %s", res.Reason)
	}
	// Destination is OUT of scope (M-009 not in tree). Refused.
	res = verb.Allow(verb.AllowInput{
		Kind:       verb.VerbMove,
		TargetID:   "M-009",
		MoveSource: "M-001",
		Actor:      "ai/claude",
		Principal:  "human/peter",
		Tree:       tr,
		Scopes:     scopes,
	})
	if res.Allowed {
		t.Error("Allowed = true; want false (destination out of scope)")
	}
}

// TestAllow_EmptyActorRefused: defensive — the cmd dispatcher should
// have caught this earlier, but Allow refuses on its own to keep the
// invariant "no commit lands without an identified operator."
func TestAllow_EmptyActorRefused(t *testing.T) {
	res := verb.Allow(verb.AllowInput{
		Kind:     verb.VerbAct,
		TargetID: "M-001",
	})
	if res.Allowed {
		t.Error("Allowed = true; want false (empty actor)")
	}
	if !strings.Contains(res.Reason, "actor is required") {
		t.Errorf("Reason = %q, want actor-required message", res.Reason)
	}
}

// buildAllowTree constructs a small in-memory tree for the Allow
// tests: epic E-01 with two milestones (M-001, M-002, both with
// parent E-01), plus an unrelated epic E-09 with no children.
func buildAllowTree(t *testing.T) *tree.Tree {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"work/epics/E-01-platform/epic.md": "---\nid: E-01\ntitle: Platform\nstatus: active\n---\n",
		"work/epics/E-01-platform/M-001-cache.md": "---\nid: M-001\ntitle: Cache warmup\n" +
			"status: in_progress\nparent: E-01\n---\n",
		"work/epics/E-01-platform/M-002-evict.md": "---\nid: M-002\ntitle: Eviction policy\n" +
			"status: draft\nparent: E-01\n---\n",
		"work/epics/E-09-unrelated/epic.md": "---\nid: E-09\ntitle: Unrelated\nstatus: proposed\n---\n",
	}
	for relPath, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	if len(tr.Entities) == 0 {
		t.Fatalf("tree empty; expected fixtures to load")
	}
	return tr
}
