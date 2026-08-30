package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/verb"
)

// endableScope creates an active epic with one open scope on it and
// returns that scope as the verb's Scopes input would carry it.
//
// The AuthSHA is read back from the applied commit rather than invented,
// because what the end mode emits has to be a SHA the replay can match,
// and a fabricated one would let a truncation bug pass unnoticed.
func endableScope(t *testing.T, r *runner) *scope.Scope {
	t.Helper()
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))
	r.must(verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		Reason:        "implement E-0001",
		CurrentBranch: "epic/E-0001-engine",
	}))
	return &scope.Scope{
		AuthSHA:   headSHA(t, r.root),
		Entity:    "E-0001",
		Agent:     "ai/claude",
		Principal: testActor,
		State:     scope.StateActive,
	}
}

// TestAuthorize_End_EmitsScopeEndsAndLeavesStatusAlone is M-0325/AC-1 at
// the verb layer. The seam-level proof — that the replay reports the
// scope as ended — lives in internal/cli/integration, but it drives the
// binary as a subprocess and so contributes nothing to this package's
// coverage; this is what exercises the emitting branch itself.
//
// The status assertion runs after Apply rather than over the plan,
// because the claim is about what the commit did to the tree, and a plan
// carrying no file ops would satisfy an over-the-plan check whatever the
// verb went on to write.
func TestAuthorize_End_EmitsScopeEndsAndLeavesStatusAlone(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	s := endableScope(t, r)
	statusBefore := r.tree().ByID("E-0001").Status

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeEnd,
		Reason: "taking this back in-loop",
		Scopes: []*scope.Scope{s},
	})
	if err != nil {
		t.Fatalf("Authorize --end: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("no plan; findings=%+v", res.Findings)
	}
	if !res.Plan.AllowEmpty {
		t.Error("Plan.AllowEmpty = false; an end commit has an empty diff like every other authorize commit")
	}
	if len(res.Plan.Ops) != 0 {
		t.Errorf("Plan.Ops len = %d, want 0 — an end writes no file, which is what leaves the "+
			"entity's status alone", len(res.Plan.Ops))
	}
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerVerb, "authorize")
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerScopeEnds, s.AuthSHA)
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerReason, "taking this back in-loop")
	// The aiwf-scope closed set is opened|paused|resumed. An end that
	// also stamped one of those would drive the replay's pause/resume
	// arm on top of the termination it already records.
	for _, tr := range res.Plan.Trailers {
		if tr.Key == gitops.TrailerScope {
			t.Errorf("plan carries %s=%q; termination is recorded by %s alone",
				gitops.TrailerScope, tr.Value, gitops.TrailerScopeEnds)
		}
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	if got := r.tree().ByID("E-0001").Status; got != statusBefore {
		t.Errorf("E-0001 status moved %q -> %q; an operator end must not close the entity", statusBefore, got)
	}
}

// TestAuthorize_End_WithoutReason_Refuses pins the requirement
// ADR-0047 argues for: an end changes no status, so its commit is the
// only artefact recording that the delegation was withdrawn, and
// without a reason on it nothing in the tree says why.
//
// It is stated here rather than left to the CLI's own check because the
// verb is reachable without going through that flag parsing, and a
// requirement enforced only at the outer layer is one an in-process
// caller does not meet.
func TestAuthorize_End_WithoutReason_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	s := endableScope(t, r)

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode: verb.AuthorizeEnd,
		// Whitespace, not empty: a check that tested only for "" would
		// accept this and commit a reason carrying no words.
		Reason: "   ",
		Scopes: []*scope.Scope{s},
	})
	if err == nil {
		t.Fatalf("Authorize --end with a blank reason succeeded (res=%+v); want refusal", res)
	}
	if !strings.Contains(err.Error(), "--reason") {
		t.Errorf("refusal does not name the flag that satisfies it; got %v", err)
	}
	if res != nil {
		t.Errorf("refusal returned a Result (%+v); the verb must produce no plan to apply", res)
	}
}

// TestAuthorize_End_ResolvesAPrefixToTheFullSHA is M-0325/AC-1's
// silent-failure guard at the verb layer.
//
// gitops.ValidateTrailer admits 7-40 hex for aiwf-scope-ends, while the
// replay matches the value by exact equality against a full SHA. An
// implementation that echoed the operator's abbreviation would therefore
// pass validation, land a commit, and leave the scope open — with no
// surface reporting anything wrong. Only the emitted value distinguishes
// the two, so that is what this asserts.
func TestAuthorize_End_ResolvesAPrefixToTheFullSHA(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	s := endableScope(t, r)

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:     verb.AuthorizeEnd,
		Reason:   "named by the prefix aiwf show prints",
		ScopeSHA: s.AuthSHA[:7],
		Scopes:   []*scope.Scope{s},
	})
	if err != nil {
		t.Fatalf("Authorize --end --scope <prefix>: %v", err)
	}
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerScopeEnds, s.AuthSHA)
	for _, tr := range res.Plan.Trailers {
		if tr.Key == gitops.TrailerScopeEnds && tr.Value != s.AuthSHA {
			t.Errorf("%s = %q, want the resolved full SHA %q", gitops.TrailerScopeEnds, tr.Value, s.AuthSHA)
		}
	}
}
