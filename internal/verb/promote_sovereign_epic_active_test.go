package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_EpicActive_RefusesNonHumanActor pins M-0095/AC-1: the
// `epic / proposed → active` edge is a sovereign act. Any actor that
// does not begin with `human/` is refused with a typed error pointing
// at the rule and the `--force --reason` override path.
func TestPromote_EpicActive_RefusesNonHumanActor(t *testing.T) {
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
	if !strings.Contains(msg, "--force") {
		t.Errorf("error should mention --force override path; got %v", err)
	}
	if !strings.Contains(msg, "sovereign") {
		t.Errorf("error should name the act as sovereign so the reader understands why; got %v", err)
	}
}

// TestPromote_EpicActive_HumanActorSucceeds pins M-0095/AC-2: the
// happy default path — a `human/...` actor promoting a `proposed` epic
// to `active` succeeds without `--force` or `--reason`. The rule
// targets only non-human actors; humans are unaffected.
func TestPromote_EpicActive_HumanActorSucceeds(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Sovereign", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	e := r.tree().ByID("E-0001")
	if e == nil || e.Status != entity.StatusActive {
		t.Fatalf("human-actor promote should have landed active; got %+v", e)
	}
}

// TestPromote_EpicActive_OtherTransitionsUnaffected pins M-0095/AC-3:
// the rule is scoped exactly to `proposed → active`. Other epic
// transitions performed by a non-human actor are not refused *by this
// rule*. Each subtest stages an epic in the appropriate starting state
// and asserts the rule's error message (with its tell-tale "sovereign"
// substring) does not appear when the non-human actor moves it.
func TestPromote_EpicActive_OtherTransitionsUnaffected(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(r *runner) // leaves E-0001 in the appropriate starting state
		newStatus string
	}{
		{
			name: "proposed -> cancelled",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Cancelled path", testActor, verb.AddOptions{}))
			},
			newStatus: entity.StatusCancelled,
		},
		{
			name: "active -> done",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Done path", testActor, verb.AddOptions{}))
				r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
			},
			newStatus: entity.StatusDone,
		},
		{
			name: "active -> cancelled",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Active cancelled", testActor, verb.AddOptions{}))
				r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
			},
			newStatus: entity.StatusCancelled,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner(t)
			tc.setup(r)
			_, err := verb.Promote(r.ctx, r.tree(), "E-0001", tc.newStatus, "ai/claude", "", false, verb.PromoteOptions{})
			// The rule under test must not fire. Other refusals (e.g.,
			// resolver requirements) may legitimately produce an error;
			// we only assert the absence of the sovereign-act message.
			if err != nil && strings.Contains(err.Error(), "sovereign") {
				t.Errorf("rule should not fire on %s; got %v", tc.name, err)
			}
		})
	}
}

// TestPromote_EpicActive_OtherKindsUnaffected pins M-0095/AC-4: the
// rule is scoped to `entity.KindEpic`. Non-human actors invoking
// promote on other kinds — milestone, contract, gap, ADR — are not
// blocked by this rule. (Other rules may apply; this test asserts the
// absence of the sovereign-act-rule's message specifically.)
func TestPromote_EpicActive_OtherKindsUnaffected(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(r *runner) // returns with the entity created (and parent epic if needed)
		id        string
		newStatus string
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
				r.must(verb.Add(r.ctx, r.tree(), entity.KindContract, "Schema", testActor, verb.AddOptions{}))
			},
			id:        "C-0001",
			newStatus: entity.StatusActive,
		},
		{
			name: "gap open -> addressed",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Missing", testActor, verb.AddOptions{}))
			},
			id:        "G-0001",
			newStatus: entity.StatusAddressed,
		},
		{
			name: "adr proposed -> accepted",
			setup: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Choice", testActor, verb.AddOptions{}))
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
