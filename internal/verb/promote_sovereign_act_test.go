package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_EpicActive_RefusesNonHumanActor pins M-0095/AC-1: the
// `epic / proposed → active` edge is a sovereign act. Any actor that
// does not begin with `human/` is refused with a typed error naming the
// rule and a remedy that works.
//
// The remedy is the human-run path only. `--force` was a real override
// for a non-human actor until the coherence guard at verb.Apply began
// refusing a force trailer from one, and this message is reachable only
// for a non-human actor — so offering it would be wrong every time the
// message is shown.
func TestPromote_EpicActive_RefusesNonHumanActor(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Sovereign", testActor, verb.AddOptions{}))

	_, err := verb.Promote(r.ctx, r.tree(), "E-0001", "active", "ai/claude", "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected error promoting epic to active with non-human actor; got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "human/") {
		t.Errorf("error should reference the human/ requirement; got %v", err)
	}
	if strings.Contains(msg, "--force") {
		t.Errorf("error offers --force, which this actor cannot use: the coherence guard "+
			"refuses a force trailer from a non-human actor, so following the advice fails; got %v", err)
	}
	if !strings.Contains(msg, "have a human run the verb") {
		t.Errorf("error should name a remedy that works; got %v", err)
	}
	if !strings.Contains(msg, "sovereign") {
		t.Errorf("error should name the act as sovereign so the reader understands why; got %v", err)
	}
	// The gate takes the verb name as an argument, so each call site
	// supplies its own. The cancel site is pinned in cancel_guards_test.go;
	// this is the promote site's half. Nothing derives one from the other,
	// so a check at one says nothing about the other.
	if !strings.Contains(msg, "aiwf promote") {
		t.Errorf("error must name the verb the operator ran; got %v", err)
	}
}

// TestPromote_EpicActive_HumanActorSucceeds pins M-0095/AC-2: the
// happy default path — a `human/...` actor promoting a `proposed` epic
// to `active` succeeds without `--force` or `--reason`. The rule
// targets only non-human actors; humans are unaffected.
func TestPromote_EpicActive_HumanActorSucceeds(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Sovereign", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	e := r.tree().ByID("E-0001")
	if e == nil || e.Status != entity.StatusActive {
		t.Fatalf("human-actor promote should have landed active; got %+v", e)
	}
}

// Transition-scoping needs no epic test: all four legal epic
// transitions are sovereign, so the negative space is empty. Only
// kind-scoping remains a live claim, and it is pinned below.

// TestPromote_SovereignEdge_HumanIsAPrefixNotASubstring pins the
// boundary of the actor predicate: human-ness is the `human/` prefix,
// not the presence of the word anywhere in the actor string. Without
// it, a predicate testing for containment accepts `ai/human-helper` at
// every sovereign edge.
//
// The verb layer is the cheaper place to state it, not the only one —
// the gate returns before a plan exists, so the binary reaches it with
// an arbitrary actor too.
func TestPromote_SovereignEdge_HumanIsAPrefixNotASubstring(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Prefix boundary", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	_, err := verb.Promote(r.ctx, r.tree(), "E-0001", "done", "ai/human-helper", "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("an actor merely containing \"human\" closed the epic; want refusal")
	}
	if !strings.Contains(err.Error(), "sovereign act requires a human/ actor") {
		t.Errorf("refusal did not come from the sovereign gate; got %v", err)
	}
}

// TestPromote_EpicActive_OtherKindsUnaffected pins M-0095/AC-4: the
// rule is scoped to `entity.KindEpic`. Non-human actors invoking
// promote on other kinds — milestone, contract, gap, ADR — are not
// blocked by this rule. (Other rules may apply; this test asserts the
// absence of the sovereign-act-rule's message specifically.)
func TestPromote_EpicActive_OtherKindsUnaffected(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		setup     func(r *runner) // returns with the entity created (and parent epic if needed)
		id        string
		newStatus entity.Status
	}{
		{
			name: "milestone draft -> in_progress",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Parent", testActor, verb.AddOptions{}))
				r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
			},
			id:        "M-0001",
			newStatus: entity.StatusInProgress,
		},
		{
			name: "contract proposed -> active",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindContract, "Schema", testActor, verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindContract)}))
			},
			id:        "C-0001",
			newStatus: entity.StatusActive,
		},
		{
			name: "gap open -> addressed",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Missing", testActor, verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
			},
			id:        "G-0001",
			newStatus: entity.StatusAddressed,
		},
		{
			name: "adr proposed -> accepted",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Choice", testActor, verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
			},
			id:        "ADR-0001",
			newStatus: "accepted",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner(t)
			tc.setup(r)
			_, err := verb.Promote(r.ctx, r.tree(), tc.id, tc.newStatus, "ai/claude", "", false, verb.PromoteOptions{})
			// The rule under test must not fire. Other refusals (e.g.,
			// resolver requirements on gap addressed) may legitimately
			// produce an error; we only assert the absence of the
			// sovereign-act message.
			if err != nil && strings.Contains(err.Error(), "sovereign") {
				t.Errorf("rule should not fire on %s; got %v", tc.name, err)
			}
		})
	}
}
