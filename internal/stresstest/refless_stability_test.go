package stresstest

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/check"
)

// refless_stability_test.go — M-0300/AC-2 and AC-3. The comparison core
// is driven with constructed verdict pairs; the failing direction is
// unreachable against the real kernel, whose G-0556 repair landed on
// main before this property was written.

// blocking builds a verdict from the subjects a run classified as
// blocking, plus any it stated non-blocking.
func verdict(blocking, nonBlocking []readPathSubject) map[readPathSubject]bool {
	v := make(map[readPathSubject]bool, len(blocking)+len(nonBlocking))
	for _, s := range nonBlocking {
		v[s] = false
	}
	for _, s := range blocking {
		v[s] = true
	}
	return v
}

func subj(entityID, code string) readPathSubject {
	return readPathSubject{EntityID: entityID, Code: code}
}

func TestClassifyRefLessStability_ReportsASubjectThatStopsBlockingWhenARefDisappears(t *testing.T) {
	t.Parallel()

	// The shape G-0556 records: a reference resolving against a ref only
	// the author's machine holds passes locally and fails in every clone.
	// Whichever direction it flips, the verdict is reporting on the ref
	// graph rather than on the tree.
	withRefs := verdict(nil, []readPathSubject{subj("G-0001", check.CodeRefsResolve)})
	withoutRefs := verdict([]readPathSubject{subj("G-0001", check.CodeRefsResolve)}, nil)

	got := classifyRefLessStability("step 1", []string{"refs/heads/side"}, withRefs, withoutRefs)

	if len(got) != 1 {
		t.Fatalf("classifyRefLessStability() returned %d violations, want 1: %+v", len(got), got)
	}
	for _, want := range []string{"step 1", "G-0001", check.CodeRefsResolve, "refs/heads/side"} {
		if !strings.Contains(got[0].Message, want) {
			t.Errorf("violation message %q does not name %q", got[0].Message, want)
		}
	}
}

func TestClassifyRefLessStability_ReportsASubjectThatStartsBlockingWhenARefDisappears(t *testing.T) {
	t.Parallel()

	withRefs := verdict([]readPathSubject{subj("G-0001", check.CodeBodyProseID)}, nil)
	withoutRefs := verdict(nil, []readPathSubject{subj("G-0001", check.CodeBodyProseID)})

	if got := classifyRefLessStability("step 2", []string{"refs/heads/side"}, withRefs, withoutRefs); len(got) != 1 {
		t.Errorf("classifyRefLessStability() returned %d violations, want 1: %+v", len(got), got)
	}
}

func TestClassifyRefLessStability_AStableDispositionAgreesEvenWhenTheSubcodeRefines(t *testing.T) {
	t.Parallel()

	// With the branch present the full check says cross-branch-local-only;
	// without it, unresolved. Both block, so the two runs refine rather
	// than contradict — the correction that keeps this property from
	// firing on a correct tree.
	both := verdict([]readPathSubject{subj("G-0001", check.CodeRefsResolve)}, nil)

	if got := classifyRefLessStability("step 3", []string{"refs/heads/side"}, both, both); got != nil {
		t.Errorf("classifyRefLessStability() = %+v, want no violations", got)
	}
}

func TestClassifyRefLessStability_ASubjectPresentInOnlyOneRunIsNotAFlip(t *testing.T) {
	t.Parallel()

	// A collision finding genuinely depends on the other branch existing,
	// so it disappears with the ref rather than changing its mind.
	// Absence is not a claim here either.
	withRefs := verdict([]readPathSubject{subj("G-0001", check.CodeIDsUnique)}, nil)
	withoutRefs := map[readPathSubject]bool{}

	if got := classifyRefLessStability("step 4", []string{"refs/heads/side"}, withRefs, withoutRefs); got != nil {
		t.Errorf("classifyRefLessStability() = %+v, want no violations", got)
	}
}

func TestClassifyRefLessStability_OrdersViolationsDeterministically(t *testing.T) {
	t.Parallel()

	withRefs := verdict(nil, []readPathSubject{
		subj("M-0002", check.CodeACsShape),
		subj("G-0001", check.CodeRefsResolve),
	})
	withoutRefs := verdict([]readPathSubject{
		subj("M-0002", check.CodeACsShape),
		subj("G-0001", check.CodeRefsResolve),
	}, nil)

	first := classifyRefLessStability("step 5", []string{"refs/heads/side"}, withRefs, withoutRefs)
	if len(first) != 2 {
		t.Fatalf("classifyRefLessStability() returned %d violations, want 2: %+v", len(first), first)
	}
	if !strings.Contains(first[0].Message, "G-0001") {
		t.Errorf("violations are not subject-sorted; first is %q", first[0].Message)
	}
	for i := 0; i < 8; i++ {
		if diff := cmp.Diff(first, classifyRefLessStability("step 5", []string{"refs/heads/side"}, withRefs, withoutRefs)); diff != "" {
			t.Fatalf("classifyRefLessStability() is not deterministic (-first +repeat):\n%s", diff)
		}
	}
}

func TestBlockingSubjectsFrom_RecordsWhetherEachSubjectBlocks(t *testing.T) {
	t.Parallel()

	got := blockingSubjectsFrom([]verbEnvelopeFinding{
		{Code: check.CodeRefsResolve, Subcode: "unresolved", Severity: "error", EntityID: "G-001"},
		{Code: check.CodeArchiveSweepPending, Severity: "warning"},
	})

	want := map[readPathSubject]bool{
		{EntityID: "G-0001", Code: check.CodeRefsResolve}:   true,
		{EntityID: "", Code: check.CodeArchiveSweepPending}: false,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("blockingSubjectsFrom() mismatch (-want +got):\n%s", diff)
	}
}

func TestBlockingSubjectsFrom_OneBlockingFindingMakesTheSubjectBlockingRegardlessOfOrder(t *testing.T) {
	t.Parallel()

	// A rule firing twice on one entity — one blocking, one not — leaves
	// the subject blocking. Reading it as non-blocking because the warning
	// arrived last would make the disposition depend on sort order.
	for _, order := range [][]verbEnvelopeFinding{
		{
			{Code: check.CodeBodyProseID, Subcode: "malformed-shape", Severity: "error", EntityID: "M-0002"},
			{Code: check.CodeBodyProseID, Subcode: "cross-branch-pending", Severity: "warning", EntityID: "M-0002"},
		},
		{
			{Code: check.CodeBodyProseID, Subcode: "cross-branch-pending", Severity: "warning", EntityID: "M-0002"},
			{Code: check.CodeBodyProseID, Subcode: "malformed-shape", Severity: "error", EntityID: "M-0002"},
		},
	} {
		got := blockingSubjectsFrom(order)
		if !got[subj("M-0002", check.CodeBodyProseID)] {
			t.Errorf("blockingSubjectsFrom(%+v) reported the subject non-blocking", order)
		}
	}
}
