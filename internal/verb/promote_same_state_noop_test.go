package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_SameStatus_NoResolverFlag_ReturnsNoOp pins M-0281/AC-1: a
// promote whose target status already equals the entity's current status,
// with no resolver flag, converges to a NoOp Result instead of returning a
// Go error. The canonical operator case is `aiwf promote M-NNNN done` run a
// second time (interactively, or from a forgotten script). The guard is
// kind-agnostic; an ADR is the cleanest fixture — `accepted` carries no
// resolver requirement, needs no ACs, and is not a sovereign act.
func TestPromote_SameStatus_NoResolverFlag_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Render envelope", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("same-status promote returned a Go error, want a clean NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (re-promoting to the current status is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}

// TestPromote_SameStatus_ResolverBackfill_StillMutates guards the AC-1
// wrinkle: the same-status NoOp must NOT swallow the G-0096 resolver-backfill
// carve-out. A gap forced to `addressed` with an empty resolver is the stray
// state backfill repairs; re-promoting `addressed` with a resolver flag must
// still produce a Plan (write the resolver), never a NoOp.
func TestPromote_SameStatus_ResolverBackfill_StillMutates(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	// Force `addressed` with an empty resolver — the pre-G-0096 stray state.
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "stray fixture", true, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"M-0001"}})
	if err != nil {
		t.Fatalf("resolver backfill returned a Go error: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false (a resolver-backfill same-status promote mutates)")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan (backfill writes the resolver)")
	}
}

// TestPromote_SameStatus_IdenticalResolver_ReturnsNoOp closes AC-1's stated
// claim: a promote converges when the target equals the current status AND no
// other field is changing. The guard originally keyed on "no resolver flag was
// supplied", which is a narrower condition — so re-running the tracker-closure
// command this repo's own gate discipline treats as routine,
// `promote <gap> addressed --by-commit <sha>`, still refused with the FSM's
// "cannot transition to itself" at exit 1. The resolver it carries is already
// stored, so applying it would write the same bytes: nothing is changing, and
// the outcome belongs on the converging side.
func TestPromote_SameStatus_IdenticalResolver_ReturnsNoOp(t *testing.T) {
	t.Parallel()

	// Both resolver flavors a gap accepts. --by-commit is the one the routine
	// tracker-closure command uses, so it carries the case; --by shares the
	// guard and is covered alongside it rather than assumed equivalent.
	cases := []struct {
		name    string
		resolve func(r *runner) verb.PromoteOptions
	}{
		{
			name: "--by entity id",
			resolve: func(_ *runner) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedBy: []string{"M-0001"}}
			},
		},
		{
			name: "--by-commit sha",
			resolve: func(r *runner) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedByCommit: []string{resolveHeadSHA(r.t, r.root)}}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := gapResolverFixture(t)
			opts := tc.resolve(r)
			r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, opts))
			before := countCommits(t, r.root)

			res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, opts)
			if err != nil {
				t.Fatalf("re-running an identical resolver promote returned a Go error, want a NoOp: %v", err)
			}
			if !res.NoOp {
				t.Errorf("res.NoOp = false, want true — status and resolver both already read as requested")
			}
			if res.Plan != nil {
				t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
			}
			if got := countCommits(t, r.root); got != before {
				t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, before)
			}
		})
	}
}

// gapResolverFixture builds an epic, two milestones and an open gap — the
// referents the gap-addressed resolver flags point at.
func gapResolverFixture(t *testing.T) *runner {
	t.Helper()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Index", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	return r
}

// TestPromote_SameStatus_DifferentResolver_StillRefused is the other side of
// that guard: convergence is keyed on the resolver being *identical*, not on
// one merely being present. Re-pointing a resolver that is already set is
// deliberately not this verb's job — the G-0096 back-fill carve-out requires
// the current resolver to be empty, so that this path cannot become a generic
// "rewrite the resolver" surface. The same-state NoOp must not quietly become
// one either: a same-status promote naming a *different* resolver keeps
// refusing, and the operator keeps needing a deliberate verb or --force.
func TestPromote_SameStatus_DifferentResolver_StillRefused(t *testing.T) {
	t.Parallel()

	// Each case records one resolver, then re-promotes naming a different
	// value of the same flavor. Covering both flavors matters: they are
	// separate comparisons, and only one of them is the flag the routine
	// tracker-closure command passes.
	cases := []struct {
		name             string
		first, different func(r *runner) verb.PromoteOptions
	}{
		{
			name: "--by entity id",
			first: func(_ *runner) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedBy: []string{"M-0001"}}
			},
			different: func(_ *runner) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedBy: []string{"M-0002"}}
			},
		},
		{
			name: "--by-commit sha",
			first: func(r *runner) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedByCommit: []string{resolveHeadSHA(r.t, r.root)}}
			},
			// Resolved after the first promote has landed, so it names a real
			// but different commit — --by-commit refuses a SHA that resolves
			// to nothing, which would mask the comparison under test.
			different: func(r *runner) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedByCommit: []string{resolveHeadSHA(r.t, r.root)}}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := gapResolverFixture(t)
			r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, tc.first(r)))

			res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, tc.different(r))
			if err == nil {
				t.Fatalf("re-pointing a set resolver returned res=%+v, want the refusal to stand", res)
			}
			// The message must name the RESOLVER, not the status: the status
			// is not what the operator got wrong, and "cannot transition to"
			// gives them nothing to act on.
			if !strings.Contains(err.Error(), "already carries a resolver") {
				t.Errorf("err = %q, want a resolver-specific refusal naming --force as the override", err)
			}
			if strings.Contains(err.Error(), "cannot transition to") {
				t.Errorf("err = %q, must not report a status-transition problem for a resolver request", err)
			}
		})
	}
}

// supersededADRPair builds two accepted ADRs and supersedes the first by the
// second, leaving both sides of the link recorded: `superseded_by` on ADR-0001
// and the `supersedes` back-link on ADR-0002.
func supersededADRPair(t *testing.T) *runner {
	t.Helper()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old choice", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "New choice", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0002", "accepted", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0002"}))
	return r
}

// TestPromote_SameStatus_IdenticalSupersededBy_ReturnsNoOp is the supersession
// arm of the convergence guard: with both sides of the link already recorded,
// a re-run of the identical command writes nothing and converges.
func TestPromote_SameStatus_IdenticalSupersededBy_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := supersededADRPair(t)
	before := countCommits(t, r.root)

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0002"})
	if err != nil {
		t.Fatalf("re-running an identical supersede returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true — status and both sides of the link already read as requested")
	}
	if got := countCommits(t, r.root); got != before {
		t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, before)
	}
}

// TestPromote_SameStatus_MissingBackLink_DoesNotConverge guards the subtle half
// of the supersession arm. `--superseded-by` records a pointer on the superseded
// ADR *and* a `supersedes` back-link on the superseding one, because
// adr-supersession-mutual is a two-sided invariant. A tree carrying only the
// first half — written before the reciprocal write existed, or hand-edited — is
// not in the state the command describes, however much the superseded side
// alone may look like it.
//
// Judging convergence on that side alone would report "already superseded" and
// exit 0 over a broken invariant. Consulting the reciprocal keeps the verb
// honest: it declines to converge, and the operator gets the standing FSM
// refusal instead of a false success. Repairing the back-link is not something
// this path can do — the back-fill carve-out needs an *empty* resolver, and
// this one is set — so the refusal is the pre-existing behavior, preserved
// rather than papered over. `aiwf check`'s adr-supersession-mutual rule is what
// names the real problem.
func TestPromote_SameStatus_MissingBackLink_DoesNotConverge(t *testing.T) {
	t.Parallel()
	r := supersededADRPair(t)

	// Strip the back-link, leaving ADR-0001.superseded_by pointing at an
	// ADR-0002 that no longer claims it.
	superseding := r.tree().ByID("ADR-0002")
	if superseding == nil {
		t.Fatal("ADR-0002 missing from the fixture tree")
	}
	path := filepath.Join(r.root, superseding.Path)
	raw, err := os.ReadFile(path) //nolint:gosec // test fixture path built from the loaded tree
	if err != nil {
		t.Fatalf("reading the superseding ADR: %v", err)
	}
	stripped := strings.Replace(string(raw), "supersedes:\n    - ADR-0001\n", "", 1)
	if stripped == string(raw) {
		t.Fatalf("fixture did not contain the expected back-link block:\n%s", raw)
	}
	if writeErr := os.WriteFile(path, []byte(stripped), 0o600); writeErr != nil {
		t.Fatalf("writing the stripped ADR: %v", writeErr)
	}

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0002"})
	if err == nil {
		t.Fatalf("re-running the supersede over a missing back-link returned res=%+v, want no convergence — "+
			"the two-sided link is incomplete, so 'already superseded' would be a false success", res)
	}
	// The message must name the back-link, which is what is actually
	// missing — not a status transition the operator never asked for.
	if !strings.Contains(err.Error(), "supersedes") {
		t.Errorf("err = %q, want it to name the missing back-link", err)
	}
}

// TestPromote_SameStatus_Force_StillNoOp pins that --force does not turn a
// no-change same-status promote into a no-diff commit attempt: force relaxes
// the FSM transition rule, but there is still nothing to change, so the guard
// returns a NoOp regardless of force.
func TestPromote_SameStatus_Force_StillNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Render envelope", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "forced rerun", true, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("forced same-status promote errored: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (nothing to change even under --force)")
	}
}

// writeEntityStatus rewrites an entity's frontmatter status to an arbitrary
// value, bypassing the verbs — the on-disk shape a hand-edit produces, and the
// only way to reach a status the kind's closed set does not contain.
func writeEntityStatus(t *testing.T, r *runner, id, status string) {
	t.Helper()
	e := r.tree().ByID(id)
	if e == nil {
		t.Fatalf("%s missing from the fixture tree", id)
	}
	path := filepath.Join(r.root, e.Path)
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", id, err)
	}
	patched := strings.Replace(string(raw), "status: "+string(e.Status)+"\n", "status: "+status+"\n", 1)
	if patched == string(raw) {
		t.Fatalf("fixture did not contain %q to rewrite:\n%s", "status: "+string(e.Status), raw)
	}
	if writeErr := os.WriteFile(path, []byte(patched), 0o600); writeErr != nil {
		t.Fatalf("writing %s: %v", id, writeErr)
	}
}

// TestPromote_UnrecognizedStatus_RefusedNotConverged pins R1 ahead of R2 at the
// entity level. A status the kind's closed set does not contain is not a state
// to converge on — it is a tree that needs repairing — so a promote naming that
// same junk value must reach the FSM and be refused, not report "already
// <invalid>" at exit 0. The convergence guard sits above the FSM consult, so
// without an explicit recognized-status condition it answered first.
func TestPromote_UnrecognizedStatus_RefusedNotConverged(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Render envelope", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	writeEntityStatus(t, r, "ADR-0001", "bogus")

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "bogus", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("promote to an unrecognized status returned res=%+v, want a refusal", res)
	}
	if !strings.Contains(err.Error(), "not a recognized") {
		t.Errorf("err = %q, want a refusal naming the unrecognized status", err)
	}
}

// TestCancel_UnrecognizedStatus_RefusedNotWritten is the entity-level analogue
// of the AC fix: an unrecognized status is not terminal, so it fell past the
// convergence guard and reached the write, laundering junk into the kind's
// terminal-cancel status under an ordinary cancel trailer with `aiwf check`
// reporting nothing.
func TestCancel_UnrecognizedStatus_RefusedNotWritten(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	writeEntityStatus(t, r, "G-0001", "bogus")

	res, err := verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "probe", false)
	if err == nil {
		t.Fatalf("cancel from an unrecognized status returned res=%+v, want a refusal", res)
	}
	if !strings.Contains(err.Error(), "not a recognized") {
		t.Errorf("err = %q, want a refusal naming the unrecognized status", err)
	}
}

// TestCancel_UnrecognizedStatus_ForceStillOverrides keeps the repair path open:
// --force is what relaxes the FSM, and refusing the unforced call must not strip
// the operator's way to dispose of a hand-damaged entity.
func TestCancel_UnrecognizedStatus_ForceStillOverrides(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	writeEntityStatus(t, r, "G-0001", "bogus")

	res, err := verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "repairing a hand-edit", true)
	if err != nil {
		t.Fatalf("forced cancel from an unrecognized status: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — the entity is not terminal, so force performs the write")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan (force relaxes the FSM)")
	}
}

// TestPromote_SameStatus_ResolverAtAnotherSpelling_ReturnsNoOp pins that the
// resolver comparison compares referents, not spellings — the corollary this
// milestone wrote into CLAUDE.md and then honored for `move` and `milestone
// depends-on` while leaving `promote`'s resolver arm behind.
//
// Both spellings below are ones an operator actually reaches for: `M-001` is a
// legal narrow id, and the 7-char SHA is the form `aiwf history` prints, so the
// short form is what a copy-paste produces. Comparing raw strings refused both
// with a message claiming the operator was re-pointing a resolver they had in
// fact matched, and pointed them at `--force`, which then wrote the narrower
// spelling over a canonical one.
func TestPromote_SameStatus_ResolverAtAnotherSpelling_ReturnsNoOp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// Both spellings are derived from one pre-promote SHA, so the
		// respelling names the same commit rather than a later one.
		spellings func(sha string) (stored, requested verb.PromoteOptions)
	}{
		{
			name: "--by at a narrower legacy width",
			spellings: func(string) (verb.PromoteOptions, verb.PromoteOptions) {
				return verb.PromoteOptions{AddressedBy: []string{"M-0001"}},
					verb.PromoteOptions{AddressedBy: []string{"M-001"}}
			},
		},
		{
			name: "--by-commit abbreviated to the width aiwf history prints",
			spellings: func(sha string) (verb.PromoteOptions, verb.PromoteOptions) {
				return verb.PromoteOptions{AddressedByCommit: []string{sha}},
					verb.PromoteOptions{AddressedByCommit: []string{sha[:7]}}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := gapResolverFixture(t)
			stored, requested := tc.spellings(resolveHeadSHA(t, r.root))
			r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, stored))
			before := countCommits(t, r.root)

			res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, requested)
			if err != nil {
				t.Fatalf("re-promoting with an equivalent respelling returned a Go error, want a NoOp: %v", err)
			}
			if !res.NoOp {
				t.Errorf("res.NoOp = false, want true — the respelled resolver names the value already stored")
			}
			if got := countCommits(t, r.root); got != before {
				t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, before)
			}
		})
	}
}

// TestPromote_SameStatus_ResolverDiffers_StillRefuses covers the arms of the
// resolver comparison that convergence does NOT reach. Each case is a shape a
// referent-based comparison must still call a difference, and each exercises a
// distinct branch: a shorter list, a list whose SHAs resolve to different
// commits, and one naming a SHA that resolves to nothing at all.
func TestPromote_SameStatus_ResolverDiffers_StillRefuses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		requested func(sha string) verb.PromoteOptions
	}{
		{
			name: "a shorter list is not the stored list",
			requested: func(string) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedByCommit: []string{}}
			},
		},
		{
			name: "a SHA that resolves to a different commit",
			requested: func(string) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedByCommit: []string{"0000000000000000000000000000000000000000"}}
			},
		},
		{
			name: "a SHA that resolves to nothing",
			requested: func(string) verb.PromoteOptions {
				return verb.PromoteOptions{AddressedByCommit: []string{"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := gapResolverFixture(t)
			sha := resolveHeadSHA(t, r.root)
			r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
				verb.PromoteOptions{AddressedByCommit: []string{sha}}))

			opts := tc.requested(sha)
			if len(opts.AddressedByCommit) == 0 {
				// An empty list supplies no resolver at all, so the guard sees
				// nothing to compare and converges — assert that, not a refusal.
				res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, opts)
				if err != nil {
					t.Fatalf("promote with no resolver flag: %v", err)
				}
				if !res.NoOp {
					t.Errorf("res.NoOp = false, want true — no resolver supplied means nothing to compare")
				}
				return
			}
			res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, opts)
			if err == nil {
				t.Fatalf("promote naming a different resolver returned res=%+v, want a refusal", res)
			}
		})
	}
}

// TestPromote_SameStatus_ResolverListLengthDiffers_StillRefuses drives the
// length-mismatch arm directly: two stored SHAs against one requested.
func TestPromote_SameStatus_ResolverListLengthDiffers_StillRefuses(t *testing.T) {
	t.Parallel()
	r := gapResolverFixture(t)
	first := resolveHeadSHA(t, r.root)
	second := commitOne(t, r.root, "probe.md", "probe\n", "second commit")
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedByCommit: []string{first, second}}))

	res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedByCommit: []string{first}})
	if err == nil {
		t.Fatalf("promote with a shorter resolver list returned res=%+v, want a refusal", res)
	}
}

// TestPromote_SameStatus_MixedResolverSpellings_ReturnsNoOp covers a list where
// some entries match byte-for-byte and others only by referent — the shape an
// operator produces by pasting one SHA from `aiwf history` and copying another
// in full. The exact matches short-circuit; the rest resolve.
func TestPromote_SameStatus_MixedResolverSpellings_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := gapResolverFixture(t)
	first := resolveHeadSHA(t, r.root)
	second := commitOne(t, r.root, "probe.md", "probe\n", "second commit")
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedByCommit: []string{first, second}}))
	before := countCommits(t, r.root)

	// first verbatim, second abbreviated.
	res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedByCommit: []string{first, second[:7]}})
	if err != nil {
		t.Fatalf("mixed spellings returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true — both entries name the stored commits")
	}
	if got := countCommits(t, r.root); got != before {
		t.Errorf("commit count = %s, want %s", got, before)
	}
}

// TestPromote_SameStatus_StoredResolverUnresolvable_StillRefuses covers the arm
// where the STORED SHA is the one that cannot be resolved. --force bypasses the
// write-time SHA validation, so a tree can carry a resolver pointing at nothing;
// the comparison must then report a difference rather than treating two
// unresolvable values as equal.
func TestPromote_SameStatus_StoredResolverUnresolvable_StillRefuses(t *testing.T) {
	t.Parallel()
	r := gapResolverFixture(t)
	const bogus = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "forced stray", true,
		verb.PromoteOptions{AddressedByCommit: []string{bogus}}))

	res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedByCommit: []string{resolveHeadSHA(t, r.root)}})
	if err == nil {
		t.Fatalf("promote against an unresolvable stored resolver returned res=%+v, want a refusal", res)
	}
}
