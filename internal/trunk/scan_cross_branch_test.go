package trunk

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// --- E-0067/M-0265/AC-2: ScanCrossBranch runs DetectCollisions only
// for ids absent from the local working tree. absentHits is the exact
// hit set handed to DetectCollisions; the helper never blob-stats a
// locally-present id. ---

// TestAbsentHits_OnlyLocallyAbsentIDsReachDetectCollisions pins the
// exact hit set DetectCollisions receives: given hits for ids both
// present and absent locally, only the absent-id hits pass through —
// every hit of a locally-present id is dropped, regardless of how many
// refs carry it.
func TestAbsentHits_OnlyLocallyAbsentIDsReachDetectCollisions(t *testing.T) {
	t.Parallel()
	hits := []RefHit{
		{Kind: "gap", ID: "G-0001", Path: "work/gaps/G-0001-a.md", Ref: "refs/heads/main"},
		{Kind: "gap", ID: "G-0002", Path: "work/gaps/G-0002-b.md", Ref: "refs/heads/main"},
		{Kind: "gap", ID: "G-0001", Path: "work/gaps/G-0001-a.md", Ref: "refs/heads/sib"},
		{Kind: "gap", ID: "G-0002", Path: "work/gaps/G-0002-b.md", Ref: "refs/heads/sib"},
		{Kind: "gap", ID: "G-0003", Path: "work/gaps/G-0003-c.md", Ref: "refs/heads/sib"},
	}
	// G-0001 and G-0003 are present in the local tree; G-0002 is not.
	present := map[string]bool{"G-0001": true, "G-0003": true}
	got := absentHits(hits, func(id string) bool { return present[id] })
	want := []RefHit{
		{Kind: "gap", ID: "G-0002", Path: "work/gaps/G-0002-b.md", Ref: "refs/heads/main"},
		{Kind: "gap", ID: "G-0002", Path: "work/gaps/G-0002-b.md", Ref: "refs/heads/sib"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("absentHits mismatch (-want +got):\n%s", diff)
	}
}

// TestAbsentHits_NilPredicate_ReturnsAllHits covers the nil-predicate
// path: with no local tree to consult (e.g. an in-memory caller), every
// hit is treated as absent and reaches DetectCollisions — the pre-lazy
// eager behavior.
func TestAbsentHits_NilPredicate_ReturnsAllHits(t *testing.T) {
	t.Parallel()
	hits := []RefHit{
		{Kind: "gap", ID: "G-0001", Path: "work/gaps/G-0001-a.md", Ref: "refs/heads/main"},
		{Kind: "gap", ID: "G-0001", Path: "work/gaps/G-0001-a.md", Ref: "refs/heads/sib"},
	}
	if diff := cmp.Diff(hits, absentHits(hits, nil)); diff != "" {
		t.Errorf("absentHits(hits, nil) mismatch (-want +got):\n%s", diff)
	}
}

// TestAbsentHits_AllPresent_ReturnsNothing covers the all-present case:
// when every id is present locally, no hit reaches DetectCollisions —
// the zero-work state the lazy scan targets (see AC-4 for the stat
// count).
func TestAbsentHits_AllPresent_ReturnsNothing(t *testing.T) {
	t.Parallel()
	hits := []RefHit{
		{Kind: "gap", ID: "G-0001", Path: "work/gaps/G-0001-a.md", Ref: "refs/heads/main"},
		{Kind: "gap", ID: "G-0001", Path: "work/gaps/G-0001-a.md", Ref: "refs/heads/sib"},
	}
	if got := absentHits(hits, func(string) bool { return true }); len(got) != 0 {
		t.Errorf("absentHits(all-present) = %v, want empty", got)
	}
}

// TestScanCrossBranch_SkipsLocallyPresentColliders pins the wiring: the
// helper detects a collision for a locally-absent id but never for a
// locally-present one, even when the present id genuinely diverges
// across refs. With an eager (unfiltered) scan the present id's
// divergence would surface in Collisions — the miss-guarded consumers
// would never read it, but computing it is the O(entities×refs) waste
// this milestone removes.
func TestScanCrossBranch_SkipsLocallyPresentColliders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	// main holds G-0001.
	commitFile(t, ctx, dir, "work/gaps/G-0001-a.md", "G-0001 main\n")
	// b1 diverges G-0001 and introduces G-0002.
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "b1")
	writeFile(t, dir, "work/gaps/G-0001-a.md", "G-0001 b1 diverged\n")
	mustRun(t, ctx, dir, "add", "work/gaps/G-0001-a.md")
	commitFile(t, ctx, dir, "work/gaps/G-0002-b.md", "G-0002 b1\n")
	// b2 forks from main (G-0001 unchanged) and diverges G-0002.
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "b2", "main")
	commitFile(t, ctx, dir, "work/gaps/G-0002-b.md", "G-0002 b2 diverged\n")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	// G-0001 is present in the local tree; G-0002 is not.
	present := func(id string) bool { return id == "G-0001" }
	scan := ScanCrossBranch(ctx, dir, present)

	if scan.Collisions["G-0001"] {
		t.Error("Collisions[G-0001] = true; a locally-present id must be skipped, not blob-stat'd")
	}
	if !scan.Collisions["G-0002"] {
		t.Errorf("Collisions[G-0002] = false; a locally-absent divergent id must be detected. got %v", scan.Collisions)
	}
}

// --- E-0067/M-0265/AC-4: when every id is present locally, the lazy
// scan hands DetectCollisions a hit set requiring zero blob-stat
// round-trips — the scale property that cost tracks the locally-absent
// set, not entities × refs. ---

// TestScanCrossBranch_AllPresent_ZeroBlobStats pins the scale property
// on a many-entity, multi-ref fixture: every id appears on three refs
// (so the eager union genuinely requires blob-stats), yet every id is
// present in the local tree, so the filtered hit set requires zero.
func TestScanCrossBranch_AllPresent_ZeroBlobStats(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	// Commit several entities on main, then fork two sibling branches
	// carrying the same ids — every id lands on main + sib1 + sib2.
	present := map[string]bool{}
	for i := 1; i <= 8; i++ {
		id := fmt.Sprintf("G-%04d", i)
		commitFile(t, ctx, dir, fmt.Sprintf("work/gaps/%s-x.md", id), fmt.Sprintf("content %d\n", i))
		present[id] = true
	}
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sib1")
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sib2", "main")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	union := append(LocalRefHits(ctx, dir), RemoteRefHits(ctx, dir)...)
	// Non-vacuity: the eager scan genuinely had blob-stat work — every id
	// is multi-hit (on all three refs).
	if !needsBlobStats(union) {
		t.Fatalf("vacuous fixture: union has no multi-hit id, no stat work to skip (union=%d hits)", len(union))
	}
	// Every id present locally → the filter drops all → zero blob-stats.
	absent := absentHits(union, func(id string) bool { return present[id] })
	if needsBlobStats(absent) {
		t.Errorf("DetectCollisions would blob-stat the filtered set (%d hits); want zero when every id is present locally", len(absent))
	}
	if len(absent) != 0 {
		t.Errorf("absentHits handed %d hits to DetectCollisions; want 0", len(absent))
	}
}
