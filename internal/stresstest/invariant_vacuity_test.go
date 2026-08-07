package stresstest

import (
	"strings"
	"testing"
)

// invariant_vacuity_test.go — M-0300/AC-3: every property the walker
// judges with has been observed reporting a violation, not merely
// passing.
//
// The distinction is the milestone's named risk. Both defects that
// motivated these properties are repaired on main, so all of them pass
// against every repository the harness can build — and a property that
// has only ever been seen to pass is indistinguishable from one that
// cannot fail. The table below is the chokepoint: a property registered
// without a state that makes it report fails this test by name.

// violationFixture builds a repository, and an `aiwf` to read it with,
// that together violate one property.
//
// Where the fixture stands in for `aiwf` it is because the kernel is
// correct: no repository makes the real surfaces contradict each other,
// or makes a verdict flip when a ref the tree does not need disappears.
// The repository, the ref-stripped copy, and the subprocesses are real;
// what is constructed is the answer a surface gives.
type violationFixture func(t *testing.T) (aiwfBin, dir string)

var violationFixtures = map[string]violationFixture{
	// A real repository carrying an entity, read by an `aiwf` that
	// reports an empty list: ground truth and the surface disagree about
	// what exists.
	"list-vs-ground-truth": func(t *testing.T) (string, string) {
		t.Helper()
		dir := newVerbSequenceTestRepo(t)
		if _, err := runAiwfJSON(sharedTestBinary(t), dir, "add", "epic", "--title", "epic a", "--body", "b"); err != nil {
			t.Fatalf("add epic: %v", err)
		}
		return writeFakeAiwfList(t, `{"status":"ok","findings":[],"result":[]}`), dir
	},

	// Two surfaces classifying one subject incompatibly: the gate calls
	// it pending at warning severity, the cheaper surface calls it
	// unresolved and blocks.
	"read-path agreement": func(t *testing.T) (string, string) {
		t.Helper()
		bin := writeFakeAiwfSurfaces(t, map[string]string{
			"check":              findingsEnvelope(`{"code":"refs-resolve","subcode":"cross-branch-pending","severity":"warning","entity_id":"G-0001"}`),
			"check --fast":       findingsEnvelope(`{"code":"refs-resolve","subcode":"unresolved","severity":"error","entity_id":"G-0001"}`),
			"check --shape-only": findingsEnvelope(),
			"status":             `{"status":"ok","result":{"health":{"errors":0}}}`,
		})
		return bin, t.TempDir()
	},

	// A real repository with a real removable ref, really stripped in a
	// real copy — read by an `aiwf` whose verdict blocks only once the
	// ref is gone.
	"ref-less verdict stability": func(t *testing.T) (string, string) {
		t.Helper()
		dir := newCrossBranchReferenceRepo(t, sharedTestBinary(t))
		bin := writeFakeAiwfByWorkingCopy(t,
			findingsEnvelope(`{"code":"refs-resolve","subcode":"cross-branch-pending","severity":"warning","entity_id":"G-0001"}`),
			findingsEnvelope(`{"code":"refs-resolve","subcode":"unresolved","severity":"error","entity_id":"G-0001"}`))
		return bin, dir
	},
}

func TestWalkInvariants_EveryPropertyHasBeenObservedFailing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)

	for _, inv := range walkInvariants() {
		t.Run(inv.Name(), func(t *testing.T) {
			t.Parallel()

			build, ok := violationFixtures[inv.Name()]
			if !ok {
				t.Fatalf("the %q property is registered with no state that makes it report; "+
					"a property observed only passing cannot be told apart from one that cannot fail", inv.Name())
			}

			bin, dir := build(t)
			violations, err := inv.Evaluate(bin, dir, "constructed")
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if len(violations) == 0 {
				t.Fatalf("the %q property reported nothing against a state that violates it", inv.Name())
			}
			if !strings.Contains(violations[0].Message, "constructed") {
				t.Errorf("violation message %q does not name the step that produced the state", violations[0].Message)
			}
		})
	}
}
