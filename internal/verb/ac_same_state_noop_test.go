package verb_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// ac_same_state_noop_test.go covers M-0281/AC-9: composite ids follow the same
// convergence rules as entity ids, and `cancel` stops writing AC states the FSM
// forbids.
//
// The two halves share a root cause. `cancelAC` decided for itself which
// statuses meant "nothing left to cancel" instead of asking the FSM, so it was
// wrong about `deferred` (terminal, but not the one it hardcoded) and silent
// about statuses the FSM does not know at all.

// acFixture builds a milestone with n acceptance criteria, all `open`.
func acFixture(t *testing.T, n int) *runner {
	t.Helper()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	for i := 0; i < n; i++ {
		r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "criterion", testActor))
	}
	return r
}

// TestCancelAC_TerminalStatus_ReturnsNoOp covers every terminal AC status, not
// just the one the old guard hardcoded. Both AC terminals are removal-class —
// `deferred` and `cancelled` each mean "off the milestone's contract", neither
// claims the criterion succeeded — so cancel has nothing left to do from either.
// `deferred` is the case that was wrong: the verb transitioned it to `cancelled`
// along an edge the FSM does not contain.
func TestCancelAC_TerminalStatus_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	for _, terminal := range []entity.Status{entity.StatusDeferred, entity.StatusCancelled} {
		t.Run(string(terminal), func(t *testing.T) {
			t.Parallel()
			if !entity.IsTerminalACStatus(terminal) {
				t.Fatalf("fixture assumes %q is terminal in the AC FSM; it is not", terminal)
			}
			r := acFixture(t, 1)
			r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", terminal, testActor, "", false, verb.PromoteOptions{}))
			before := countCommits(t, r.root)

			res, err := verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "probe", false)
			if err != nil {
				t.Fatalf("cancel of a %q AC returned a Go error, want a NoOp: %v", terminal, err)
			}
			if !res.NoOp {
				t.Errorf("res.NoOp = false, want true — %q is terminal, so there is nothing to cancel", terminal)
			}
			if !strings.Contains(res.NoOpMessage, string(terminal)) {
				t.Errorf("NoOpMessage = %q, want it to name the actual status %q", res.NoOpMessage, terminal)
			}
			if got := countCommits(t, r.root); got != before {
				t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, before)
			}
		})
	}
}

// TestCancelAC_TerminalStatus_NoOpEvenUnderForce pins that the convergence fires
// above `--force`, matching entity-level cancel: a sovereign override exists to
// relax the FSM, and there is no transition here for it to relax.
func TestCancelAC_TerminalStatus_NoOpEvenUnderForce(t *testing.T) {
	t.Parallel()
	r := acFixture(t, 1)
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", entity.StatusDeferred, testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "forced probe", true)
	if err != nil {
		t.Fatalf("forced cancel of a deferred AC: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true — nothing for a sovereign override to re-apply")
	}
}

// TestCancelAC_NonTerminalStatus_StillTransitions guards against the guard
// over-reaching. `met` is deliberately not terminal: an AC is a claim inside a
// contract that can still be rescoped, so a met criterion may legitimately be
// descoped while its milestone runs. Cancel must keep doing real work there.
func TestCancelAC_NonTerminalStatus_StillTransitions(t *testing.T) {
	t.Parallel()
	r := acFixture(t, 1)
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", entity.StatusMet, testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "descoped", false)
	if err != nil {
		t.Fatalf("cancel of a met AC: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — met is not terminal, so cancel has work to do")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan performing met -> cancelled")
	}
}

// TestCancelAC_UnrecognizedStatus_RefusedNotWritten is the junk-status case, and
// the reason the verb consults the FSM rather than relying on an assertion about
// the FSM's shape. An unrecognized status is not terminal — IsTerminal-style
// predicates answer false for unknown input by design, so downstream checks keep
// firing on junk rather than silently exempting it — so it falls past the
// convergence guard. Without an FSM consult the verb laundered it into
// `cancelled`; `aiwf check` flags the status but nothing flagged the write.
func TestCancelAC_UnrecognizedStatus_RefusedNotWritten(t *testing.T) {
	t.Parallel()
	r := acFixture(t, 1)
	writeACStatus(t, r, "blocked")

	res, err := verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "probe", false)
	if err == nil {
		t.Fatalf("cancel of an AC at an unrecognized status returned res=%+v, want a refusal", res)
	}
	if !strings.Contains(err.Error(), "cannot transition") {
		t.Errorf("err = %q, want the FSM refusal", err)
	}
}

// TestCancelAC_UnrecognizedStatus_ForceStillOverrides keeps the sovereign path
// open: --force is what relaxes the FSM, and refusing the unforced call must not
// remove the operator's escape hatch for a repair.
func TestCancelAC_UnrecognizedStatus_ForceStillOverrides(t *testing.T) {
	t.Parallel()
	r := acFixture(t, 1)
	writeACStatus(t, r, "blocked")

	res, err := verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "repairing a hand-edit", true)
	if err != nil {
		t.Fatalf("forced cancel of an unrecognized-status AC: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — the AC is not terminal, so force performs the write")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan (force relaxes the FSM)")
	}
}

// TestPromoteAC_SameStatus_ReturnsNoOp covers the convergence half across every
// status an AC can hold: re-promoting to the status already recorded has nothing
// to change, so it converges rather than returning the FSM's self-transition
// refusal — the same short-circuit-above-ValidateTransition shape entity-level
// promote uses.
func TestPromoteAC_SameStatus_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	for _, status := range entity.AllowedACStatuses() {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()
			r := acFixture(t, 1)
			if status != entity.StatusOpen {
				r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", status, testActor, "", false, verb.PromoteOptions{}))
			}
			before := countCommits(t, r.root)

			res, err := verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", status, testActor, "", false, verb.PromoteOptions{})
			if err != nil {
				t.Fatalf("re-promoting an AC to %q returned a Go error, want a NoOp: %v", status, err)
			}
			if !res.NoOp {
				t.Errorf("res.NoOp = false, want true — the AC is already %q", status)
			}
			if got := countCommits(t, r.root); got != before {
				t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, before)
			}
		})
	}
}

// TestPromoteAC_SameStatus_NoOpEvenUnderForce mirrors entity-level promote:
// force relaxes the FSM, and there is no transition to relax.
func TestPromoteAC_SameStatus_NoOpEvenUnderForce(t *testing.T) {
	t.Parallel()
	r := acFixture(t, 1)
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", entity.StatusMet, testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", entity.StatusMet, testActor, "forced", true, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("forced same-status AC promote: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (nothing to change even under --force)")
	}
}

// TestPromoteAC_DifferentStatus_StillTransitions is the control: the same-status
// guard must not swallow a real transition.
func TestPromoteAC_DifferentStatus_StillTransitions(t *testing.T) {
	t.Parallel()
	r := acFixture(t, 1)

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", entity.StatusMet, testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("open -> met: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — open -> met is a real transition")
	}
}

// writeACStatus rewrites AC-1's status in the milestone file to an arbitrary
// value, bypassing the verbs — the on-disk shape a hand-edit produces, and the
// only way to reach a status the FSM does not recognize.
func writeACStatus(t *testing.T, r *runner, status string) {
	t.Helper()
	path := filepath.Join(r.root, r.tree().ByID("M-0001").Path)
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the milestone: %v", err)
	}
	patched := strings.Replace(string(raw), "      status: open\n", "      status: "+status+"\n", 1)
	if patched == string(raw) {
		t.Fatalf("fixture did not contain an open AC status to rewrite:\n%s", raw)
	}
	if writeErr := os.WriteFile(path, []byte(patched), 0o600); writeErr != nil {
		t.Fatalf("writing the patched milestone: %v", writeErr)
	}
}

// TestPromoteAC_UnrecognizedStatus_RefusedNotConverged is the AC analogue of the
// entity-level guard: cancel already refused a junk status, and promote must
// too, or the two verbs disagree about whether an unrecognized status is real.
func TestPromoteAC_UnrecognizedStatus_RefusedNotConverged(t *testing.T) {
	t.Parallel()
	r := acFixture(t, 1)
	writeACStatus(t, r, "blocked")

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "blocked", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("promote to an unrecognized AC status returned res=%+v, want a refusal", res)
	}
	if !strings.Contains(err.Error(), "cannot transition") {
		t.Errorf("err = %q, want the FSM refusal", err)
	}
}

// TestACVerbs_RejectAnUnresolvableCompositeID pins R1 at the AC verbs' entry:
// every one resolves its composite id before doing anything else, so a
// composite naming a milestone or an AC that does not exist is refused rather
// than converged or written. lookupAC is the shared resolver; these drive it
// through each verb so a future path that skips it is caught.
func TestACVerbs_RejectAnUnresolvableCompositeID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		call func(r *runner) (*verb.Result, error)
	}{
		{
			name: "cancel a composite whose AC does not exist",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Cancel(r.ctx, r.tree(), "M-0001/AC-9", testActor, "probe", false)
			},
		},
		{
			name: "promote a composite whose AC does not exist",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Promote(r.ctx, r.tree(), "M-0001/AC-9", entity.StatusMet, testActor, "", false, verb.PromoteOptions{})
			},
		},
		{
			name: "rename a composite whose AC does not exist",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Rename(r.ctx, r.tree(), "M-0001/AC-9", "a new title", testActor, 0)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := acFixture(t, 1)
			res, err := tc.call(r)
			if err == nil {
				t.Fatalf("%s returned res=%+v, want a refusal", tc.name, res)
			}
		})
	}
}

// TestRenameAC_EmptyTitle_Refused keeps the empty-title guard ahead of the
// same-title convergence: an empty new title is not a request that is already
// satisfied, it is a request that cannot be satisfied.
func TestRenameAC_EmptyTitle_Refused(t *testing.T) {
	t.Parallel()
	r := acFixture(t, 1)

	res, err := verb.Rename(r.ctx, r.tree(), "M-0001/AC-1", "   ", testActor, 0)
	if err == nil {
		t.Fatalf("rename to an empty title returned res=%+v, want a refusal", res)
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %q, want it to name the empty title", err)
	}
}

// TestRetitleAndRenameAC_HeadingDrifted_StillRewrite pins the body-heading half
// of the AC guards. Both verbs write the frontmatter title AND the
// `### AC-N — <title>` heading, so an AC whose title is right while its heading
// has drifted still has work to do. Comparing the title alone reported "nothing
// to rename" over exactly that state, leaving the stale prose in place.
func TestRetitleAndRenameAC_HeadingDrifted_StillRewrite(t *testing.T) {
	t.Parallel()
	const title = "criterion"
	cases := []struct {
		name string
		call func(r *runner) (*verb.Result, error)
	}{
		{
			name: "retitle",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Retitle(r.ctx, r.tree(), "M-0001/AC-1", title, testActor, "", 0)
			},
		},
		{
			name: "rename",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Rename(r.ctx, r.tree(), "M-0001/AC-1", title, testActor, 0)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := acFixture(t, 1)
			writeACHeading(t, r, "### AC-1 — STALE HEADING")

			res, err := tc.call(r)
			if err != nil {
				t.Fatalf("%s over a drifted heading: %v", tc.name, err)
			}
			if res.NoOp {
				t.Errorf("res.NoOp = true, want false — the heading differs, so there is prose to rewrite")
			}
			if res.Plan == nil {
				t.Fatal("res.Plan = nil, want a plan restoring the heading")
			}
		})
	}
}

// writeACHeading replaces AC-1's body heading with an arbitrary line, producing
// the frontmatter-vs-body drift only a hand edit can create.
func writeACHeading(t *testing.T, r *runner, heading string) {
	t.Helper()
	path := filepath.Join(r.root, r.tree().ByID("M-0001").Path)
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the milestone: %v", err)
	}
	patched := regexp.MustCompile(`(?m)^### AC-1 .*$`).ReplaceAllString(string(raw), heading)
	if patched == string(raw) {
		t.Fatalf("fixture had no AC-1 heading to rewrite:\n%s", raw)
	}
	if writeErr := os.WriteFile(path, []byte(patched), 0o600); writeErr != nil {
		t.Fatalf("writing the patched milestone: %v", writeErr)
	}
}
