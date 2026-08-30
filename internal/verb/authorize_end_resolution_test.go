package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/verb"
)

// endTarget builds a scope on E-0001 in the given state with a
// fabricated auth SHA.
//
// Fabricated is right here, unlike in endableScope: these cases assert
// which candidate the resolver picks and what it says when it cannot,
// and a hand-written SHA makes the prefix relationships between
// candidates exact rather than incidental. Nothing downstream replays
// these, so no real commit needs to carry them.
func endTarget(sha string, state scope.State, agent string) *scope.Scope {
	return &scope.Scope{
		AuthSHA: sha + strings.Repeat("0", 40-len(sha)),
		Entity:  "E-0001",
		Agent:   agent,
		State:   state,
	}
}

// TestAuthorize_End_AlreadyEnded_ConvergesToNoOp is M-0325/AC-2's
// convergence arm at the verb layer, where the integration test's
// subprocess contributes no coverage.
//
// The NoOp carries no Plan, which is what the one-mutation-one-commit
// invariant rests on: a Result with a plan would be applied, and an
// applied plan is a commit whether or not it changed anything.
func TestAuthorize_End_AlreadyEnded_ConvergesToNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	s := endableScope(t, r)
	s.State = scope.StateEnded

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:     verb.AuthorizeEnd,
		Reason:   "ending it again",
		ScopeSHA: s.AuthSHA,
		Scopes:   []*scope.Scope{s},
	})
	if err != nil {
		t.Fatalf("re-ending an ended scope errored instead of converging: %v", err)
	}
	if !res.NoOp {
		t.Errorf("Result.NoOp = false; a request whose effect already holds converges rather than committing")
	}
	if res.Plan != nil {
		t.Errorf("NoOp Result carries a Plan (%+v); it would be applied, and an applied plan is a commit", res.Plan)
	}
	if !strings.Contains(res.NoOpMessage, "already ended") {
		t.Errorf("NoOpMessage = %q; it must report the state as already holding", res.NoOpMessage)
	}
}

// TestAuthorize_End_MalformedActor_FailsTrailerValidation covers the
// end mode's write-time shape check.
//
// Authorize's own gate admits any actor beginning `human/`, which is a
// weaker rule than the `<role>/<id>` shape aiwf-actor must satisfy — so
// a two-slash identity passes the gate and reaches trailer validation.
// That is the one input that reaches this arm: the mode's other
// trailers are verb-constructed and cannot be malformed.
func TestAuthorize_End_MalformedActor_FailsTrailerValidation(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	s := endableScope(t, r)

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", "human/peter/extra", verb.AuthorizeOptions{
		Mode:   verb.AuthorizeEnd,
		Reason: "the actor, not the request, is what is malformed here",
		Scopes: []*scope.Scope{s},
	})
	if err == nil {
		t.Fatalf("a malformed actor produced a plan (%+v); the trailer would fail its own shape rule at write time", res)
	}
	if !strings.Contains(err.Error(), "aiwf-actor") {
		t.Errorf("refusal does not name the offending trailer; got %v", err)
	}
}

// TestAuthorize_End_TargetResolution_Refusals is M-0325/AC-2's refusal
// arm, and it is written as one table because the cases share a single
// rule: the resolver either names exactly one scope or refuses.
//
// The first two cases are the R1-before-R2 pair. They differ only in
// whether the --scope value matches a real scope, and the population
// they resolve against is what decides them — an implementation
// searching only non-ended scopes would refuse both, and one converging
// whenever anything was ended would accept both. Neither collapse shows
// up unless the pair is present.
func TestAuthorize_End_TargetResolution_Refusals(t *testing.T) {
	t.Parallel()

	ended := endTarget("aaaaaaa", scope.StateEnded, "ai/claude")
	active := endTarget("bbbbbbb", scope.StateActive, "ai/claude")
	// Two candidates sharing a four-character prefix, so an ambiguous
	// --scope is one the resolver will actually consider: anything
	// shorter is refused for being short before ambiguity is reached.
	twinA := endTarget("dddd1111", scope.StateActive, "ai/claude")
	twinB := endTarget("dddd2222", scope.StatePaused, "ai/other")

	cases := []struct {
		name     string
		scopeSHA string
		scopes   []*scope.Scope
		want     []string
	}{
		{
			name:     "--scope names nothing, even though a scope exists",
			scopeSHA: "9999999",
			scopes:   []*scope.Scope{ended},
			want:     []string{"no scope on E-0001 matches", "9999999"},
		},
		{
			name:     "--scope prefix matches more than one scope",
			scopeSHA: "dddd",
			scopes:   []*scope.Scope{twinA, twinB},
			want:     []string{"matches 2 scopes", "ai/claude", "ai/other"},
		},
		{
			// Distinct from the row above: too short to resolve at all,
			// refused before any candidate is considered. A unique short
			// prefix is not the same as a name the operator meant, and
			// this act cannot be undone.
			name:     "--scope shorter than the minimum prefix",
			scopeSHA: "dd",
			scopes:   []*scope.Scope{twinA, twinB},
			want:     []string{"too short to name a scope", "at least 4"},
		},
		{
			name:   "bare --end when every scope is ended",
			scopes: []*scope.Scope{ended},
			want:   []string{"no non-ended scope on E-0001 to end"},
		},
		{
			// A paused scope is a candidate: ADR-0047 scopes ending to
			// every non-ended state, so pausing one does not narrow an
			// ambiguous entity down to a single answer.
			name:   "bare --end with one active and one paused candidate",
			scopes: []*scope.Scope{active, twinB},
			want:   []string{"--scope", "ai/claude", "ai/other"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			_ = endableScope(t, r)

			res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
				Mode:     verb.AuthorizeEnd,
				Reason:   "a reason, so the refusal under test is the one that fires",
				ScopeSHA: tc.scopeSHA,
				Scopes:   tc.scopes,
			})
			if err == nil {
				t.Fatalf("resolution succeeded (res=%+v); want refusal", res)
			}
			if res != nil {
				t.Errorf("refusal returned a Result (%+v); the verb must produce no plan to apply", res)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusal does not contain %q; got %v", want, err)
				}
			}
		})
	}
}
