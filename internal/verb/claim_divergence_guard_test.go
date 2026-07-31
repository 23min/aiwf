package verb_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// assertClaimRefused pins what a claim-side refusal must be: a typed
// error naming the diverging path, and no NoOp result standing in for a
// decision the verb was not in a position to make.
//
// Identity, not a substring of the message — a refusal a caller can only
// recognize by reading English is one that breaks whenever the wording
// improves.
func assertClaimRefused(t *testing.T, res *verb.Result, err error, wantPath string) {
	t.Helper()
	if err == nil {
		t.Fatalf("verb returned no error; got result %+v", res)
	}
	var claimErr *verb.ClaimDivergenceError
	if !errors.As(err, &claimErr) {
		t.Fatalf("error is not a *verb.ClaimDivergenceError: %v", err)
	}
	if res != nil && res.NoOp {
		t.Error("verb returned a NoOp alongside the refusal; the claim must not be made at all")
	}
	var named bool
	for _, p := range claimErr.Paths() {
		if p == wantPath {
			named = true
		}
	}
	if !named {
		t.Errorf("refusal does not name %q; names %v", wantPath, claimErr.Paths())
	}
}

// TestSetPriority_SameValueOverDivergentFrontmatter_Refuses is the
// false-negative half of the measured pair, and the reason the guard
// runs before the comparison rather than inside the NoOp return.
//
// HEAD carries `high`. The operator hand-edits the working copy to `low`
// and then asks for `low`. The loaded tree parses that working copy, so
// the verb sees the request already satisfied and reports success — while
// the record says otherwise and the operator's edit is dropped.
func TestSetPriority_SameValueOverDivergentFrontmatter_Refuses(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor))
	path := dirtyEntity(t, r, "G-0001", "priority: high", "priority: low")

	res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "low", false, testActor)
	assertClaimRefused(t, res, err, path)
}

// TestSetPriority_SameValueOverUncommittedBodyEdit_Refuses is the
// whole-file half of AC-1. The frontmatter agrees with HEAD and the
// requested value is genuinely already stored, so a frontmatter-scoped
// comparison would converge — reporting "nothing to change" over a file
// that has plenty to change, just not through this verb.
func TestSetPriority_SameValueOverUncommittedBodyEdit_Refuses(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor))
	path := dirtyEntity(t, r, "G-0001", "## Why it matters", "## Why it matters\n\nUNBLESSED BODY EDIT.\n")

	res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor)
	assertClaimRefused(t, res, err, path)
}

// TestSetPriority_SameValueOnACleanTree_StillConverges is the negative
// control the rest of AC-1 rests on. A guard that refused here would not
// be a precondition, it would be the end of same-state convergence
// (ADR-0036) — every idempotent re-run turning into an error.
func TestSetPriority_SameValueOnACleanTree_StillConverges(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor))

	res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor)
	if err != nil {
		t.Fatalf("SetPriority on a clean tree: %v", err)
	}
	if !res.NoOp {
		t.Errorf("same-state request on a clean tree did not converge: %+v", res)
	}
}

// TestSetPriority_UnrelatedDirtyEntity_StillConverges pins the scope.
// The refusal is about the entity the claim asserts over, so an operator
// mid-edit on some other file is not blocked from running a verb that has
// nothing to do with it.
func TestSetPriority_UnrelatedDirtyEntity_StillConverges(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Another gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	r.must(verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor))
	dirtyEntity(t, r, "G-0002", "## Why it matters", "## Why it matters\n\nEDIT ELSEWHERE.\n")

	res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor)
	if err != nil {
		t.Fatalf("SetPriority with an unrelated dirty entity: %v", err)
	}
	if !res.NoOp {
		t.Errorf("an unrelated dirty entity blocked convergence: %+v", res)
	}
}

// TestPromote_SameStatusOverDivergentFrontmatter_Refuses shows the
// property is the seam's, not one verb's. promote reaches its NoOp
// through the FSM rather than a field comparison, so covering it and
// set-priority together covers both shapes of same-state claim.
func TestPromote_SameStatusOverDivergentFrontmatter_Refuses(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	path := dirtyEntity(t, r, "G-0001", "status: open", "status: addressed")

	res, err := verb.Promote(r.ctx, r.tree(), "G-0001", entity.Status("addressed"),
		testActor, "", false, verb.PromoteOptions{})
	assertClaimRefused(t, res, err, path)
}

// TestPromote_SupersededByDivergentReciprocal_Refuses is the regression
// pin for the vector that decides where this guard runs.
//
// --superseded-by writes both sides of the link, so promote reads the
// superseding ADR to decide whether the reciprocal is already stored.
// With that back-link hand-edited onto disk, the read says "already
// there", no op is emitted for that file, and the plan therefore never
// names it — so the commit-side guard, which is keyed on the plan's
// paths, cannot see it either. Measured before this guard: a one-sided
// supersession committed at exit 0, with `aiwf check` silent, because the
// same working copy that caused it also satisfies adr-supersession-mutual.
//
// The status genuinely differs here, so no same-state comparison is
// reached. That is the point: a guard that only refused at the converge
// point would observe this divergence and discard it.
func TestPromote_SupersededByDivergentReciprocal_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old choice", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "New choice", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0002", "accepted", testActor, "", false, verb.PromoteOptions{}))

	// Hand-add the reciprocal to the *superseding* ADR only.
	reciprocal := dirtyEntity(t, r, "ADR-0002", "status: accepted", "status: accepted\nsupersedes:\n    - ADR-0001")
	before := headSHA(t, r.root)

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0002"})
	assertClaimRefused(t, res, err, reciprocal)
	if after := headSHA(t, r.root); after != before {
		t.Errorf("HEAD advanced to %s; a one-sided supersession landed", after)
	}
}

// TestSetPriority_UntrackedEntityFile_StillConverges pins the carve-out
// this guard shares with Apply's: a path absent from HEAD carries no
// record for the verb's reading to contradict.
//
// Without it the two seams would answer one condition differently — Apply
// exempts an untracked path a plan names as its write destination, so the
// real-work half would proceed while the same-state half refused. It is
// also what keeps a freshly-initialised repo usable, where `aiwf init`
// leaves aiwf.yaml uncommitted by design.
func TestSetPriority_UntrackedEntityFile_StillConverges(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor))

	// Remove the gap from the record while leaving it on disk, so its
	// path is untracked rather than modified.
	e := r.tree().ByID("G-0001")
	if e == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	if out, rmErr := exec.Command("git", "-C", r.root, "rm", "--cached", "-q", e.Path).CombinedOutput(); rmErr != nil {
		t.Fatalf("git rm --cached: %v\n%s", rmErr, out)
	}
	// Commit the staged removal without re-adding: the file must end up
	// present on disk and absent from HEAD, which is the state under test.
	if out, cErr := exec.Command("git", "-C", r.root, "commit", "-q",
		"-m", "fixture: drop the gap from the record").CombinedOutput(); cErr != nil {
		t.Fatalf("git commit: %v\n%s", cErr, out)
	}

	res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor)
	if err != nil {
		t.Fatalf("SetPriority over an untracked entity file: %v", err)
	}
	if !res.NoOp {
		t.Errorf("an untracked entity file blocked convergence: %+v", res)
	}
}

// TestSetPriority_EntityFileMissingFromDisk_Refuses covers the third
// divergence kind. A path recorded at HEAD and gone from the working tree
// is a disagreement the verb must not read past — its own remedy differs
// from the other two, so the message arm is distinct.
func TestSetPriority_EntityFileMissingFromDisk_Refuses(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor))

	e := r.tree().ByID("G-0001")
	if e == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	path := filepath.ToSlash(e.Path)
	tr := r.tree() // load before the file disappears
	if rmErr := os.Remove(filepath.Join(r.root, e.Path)); rmErr != nil {
		t.Fatalf("removing %s: %v", path, rmErr)
	}

	res, err := verb.SetPriority(r.ctx, tr, "G-0001", "high", false, testActor)
	assertClaimRefused(t, res, err, path)
	if !strings.Contains(err.Error(), "missing from the working tree") {
		t.Errorf("refusal does not name the missing-from-disk remedy:\n%v", err)
	}
}

// TestACVerbs_DivergentParentMilestone_Refuse covers the four composite
// branches together. Each resolves through lookupAC and then reads the
// parent milestone's file — for the AC's stored status or title, and for
// the `### AC-N — <title>` heading two of them rewrite — so the parent's
// file is what their decisions rest on.
//
// They need their own coverage rather than riding on the exported scan:
// promoteAC, cancelAC, renameAC and retitleAC are unexported, and
// verb_result_noop_invariant.go walks exported entry points only.
func TestACVerbs_DivergentParentMilestone_Refuse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		call func(r *runner) (*verb.Result, error)
	}{
		{
			name: "promote",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", entity.StatusMet,
					testActor, "", false, verb.PromoteOptions{})
			},
		},
		{
			name: "cancel",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "", false)
			},
		},
		{
			name: "rename",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Rename(r.ctx, r.tree(), "M-0001/AC-1", "A different criterion", testActor, 0)
			},
		},
		{
			name: "retitle",
			call: func(r *runner) (*verb.Result, error) {
				return verb.Retitle(r.ctx, r.tree(), "M-0001/AC-1", "A different criterion", testActor, "", 0)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := acFixture(t, 1)
			path := dirtyEntity(t, r, "M-0001", "## Goal", "## Goal\n\nUNBLESSED EDIT TO THE PARENT.\n")

			res, err := tc.call(r)
			assertClaimRefused(t, res, err, path)
		})
	}
}

// appendToEntity adds a body line to an entity's file without committing,
// the shape of an operator's in-progress edit. Returns the repo-relative
// path. Appending rather than substituting keeps it kind-agnostic — every
// entity has an end of file, not every kind shares a body heading.
func appendToEntity(t *testing.T, r *runner, id string) string {
	t.Helper()
	e := r.tree().ByID(id)
	if e == nil {
		t.Fatalf("%s missing from the fixture tree", id)
	}
	abs := filepath.Join(r.root, e.Path)
	raw, err := os.ReadFile(abs) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", id, err)
	}
	if err := os.WriteFile(abs, append(raw, []byte("\nUNBLESSED EDIT.\n")...), 0o600); err != nil {
		t.Fatalf("writing %s: %v", id, err)
	}
	return filepath.ToSlash(e.Path)
}

// TestEveryGuardedVerb_DivergentTarget_Refuses walks the entity-level
// verbs one by one. Each computes its own claim scope — `e.Path` for the
// field writers, the resolved source path for the move-shaped ones — so
// covering them individually is what shows no site was wired with a
// scope that happens to be inert.
func TestEveryGuardedVerb_DivergentTarget_Refuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		fixture func(t *testing.T) (*runner, string) // runner, id whose file gets dirtied
		call    func(r *runner) (*verb.Result, error)
	}{
		{
			name: "cancel",
			fixture: func(t *testing.T) (*runner, string) {
				t.Helper()
				return newGapRunner(t), "G-0001"
			},
			call: func(r *runner) (*verb.Result, error) {
				return verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false)
			},
		},
		{
			name: "rename",
			fixture: func(t *testing.T) (*runner, string) {
				t.Helper()
				return newGapRunner(t), "G-0001"
			},
			call: func(r *runner) (*verb.Result, error) {
				return verb.Rename(r.ctx, r.tree(), "G-0001", "a-new-slug", testActor, 0)
			},
		},
		{
			name: "retitle",
			fixture: func(t *testing.T) (*runner, string) {
				t.Helper()
				return newGapRunner(t), "G-0001"
			},
			call: func(r *runner) (*verb.Result, error) {
				return verb.Retitle(r.ctx, r.tree(), "G-0001", "A new title", testActor, "", 0)
			},
		},
		{
			name: "set-area",
			fixture: func(t *testing.T) (*runner, string) {
				t.Helper()
				r := newRunner(t)
				r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
				return r, "E-0001"
			},
			call: func(r *runner) (*verb.Result, error) {
				return verb.SetArea(r.ctx, r.tree(), []string{"core"}, "E-0001", "core", false, testActor)
			},
		},
		{
			name: "milestone tdd",
			fixture: func(t *testing.T) (*runner, string) {
				t.Helper()
				return acFixture(t, 0), "M-0001"
			},
			call: func(r *runner) (*verb.Result, error) {
				return verb.MilestoneTDD(r.ctx, r.tree(), "M-0001", "required", testActor, "")
			},
		},
		{
			name: "milestone depends-on",
			fixture: func(t *testing.T) (*runner, string) {
				t.Helper()
				r := acFixture(t, 0)
				r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Second", testActor,
					verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
				return r, "M-0001"
			},
			call: func(r *runner) (*verb.Result, error) {
				return verb.MilestoneDependsOn(r.ctx, r.tree(), "M-0001", []string{"M-0002"}, false, testActor, "")
			},
		},
		{
			name: "move",
			fixture: func(t *testing.T) (*runner, string) {
				t.Helper()
				r := acFixture(t, 0)
				r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Second", testActor, verb.AddOptions{}))
				return r, "M-0001"
			},
			call: func(r *runner) (*verb.Result, error) {
				return verb.Move(r.ctx, r.tree(), "M-0001", "E-0002", testActor)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, id := tc.fixture(t)
			path := appendToEntity(t, r, id)
			before := headSHA(t, r.root)

			res, err := tc.call(r)
			assertClaimRefused(t, res, err, path)
			if after := headSHA(t, r.root); after != before {
				t.Errorf("HEAD advanced to %s on a refusal", after)
			}
		})
	}
}

// TestSetPriority_RootIsNotARepo_FailsLoud pins that a comparison which
// cannot be made is an error, never a silent pass. A guard that read
// "cannot ask git" as "nothing diverges" would be inert in exactly the
// situations where the record is least trustworthy.
func TestSetPriority_RootIsNotARepo_FailsLoud(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	loaded := r.tree()
	e := loaded.ByID("G-0001")
	if e == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	detached := &tree.Tree{Root: t.TempDir(), Entities: loaded.Entities}

	_, err := verb.SetPriority(r.ctx, detached, "G-0001", "high", false, testActor)
	if err == nil {
		t.Fatal("SetPriority against a non-repo root returned no error")
	}
	if !strings.Contains(err.Error(), "against HEAD") {
		t.Errorf("error does not name the failed comparison:\n%v", err)
	}
}

// TestSetPriority_UnreadableEntityPath_FailsLoud covers the other
// fail-loud arm: the comparison reached git, and reading the path itself
// failed. Same rule — an unanswerable question is not a clean answer.
func TestSetPriority_UnreadableEntityPath_FailsLoud(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	loaded := r.tree()
	e := loaded.ByID("G-0001")
	if e == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	// Point the entity at its own directory, which exists in the repo but
	// cannot be read as a file.
	misdirected := *e
	misdirected.Path = filepath.ToSlash(filepath.Dir(e.Path))
	detached := &tree.Tree{Root: r.root, Entities: []*entity.Entity{&misdirected}}

	_, err := verb.SetPriority(r.ctx, detached, "G-0001", "high", false, testActor)
	if err == nil {
		t.Fatal("SetPriority over an unreadable entity path returned no error")
	}
	if !strings.Contains(err.Error(), "against HEAD") {
		t.Errorf("error does not name the failed comparison:\n%v", err)
	}
}

// TestPromote_UnresolvableSupersededBy_LeavesResolutionToItsOwnCheck
// covers the second arm of promote's claim scope. An id naming nothing
// contributes no path: this guard answers whether a file agrees with the
// record, not whether an argument resolves, and the resolver validation
// downstream owns that message. Reporting it here would give the operator
// a divergence complaint about a typo.
func TestPromote_UnresolvableSupersededBy_LeavesResolutionToItsOwnCheck(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old choice", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-9999"})
	var claimErr *verb.ClaimDivergenceError
	if errors.As(err, &claimErr) {
		t.Fatalf("refused as a divergence rather than an unresolvable reference:\n%v", err)
	}
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	// The unresolved id surfaces as a projection finding, which is where
	// reference resolution belongs — the guard neither reports it nor
	// masks it.
	var sawUnresolved bool
	for _, f := range res.Findings {
		if f.Code == check.CodeRefsResolve && f.Severity == check.SeverityError {
			sawUnresolved = true
		}
	}
	if !sawUnresolved {
		t.Errorf("no refs-resolve error finding for the unresolvable target: %+v", res.Findings)
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil — the reference does not resolve", res.Plan)
	}
}
