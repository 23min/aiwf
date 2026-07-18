// Package trunk reads entity ids visible in git refs for the
// allocator's cross-branch view. Its primary surface, Read, scans the
// configured trunk ref's tree, applying the policy from
// id-allocation.md:
//
//   - Not a git repository at all → skip. Tooling that legitimately
//     runs outside a repo (test fixtures, exploratory invocations on
//     plain directories) has no cross-branch surface to police.
//
//   - Configured trunk ref resolves → return the (kind, id, path)
//     triples found under work/ and docs/adr/ in that ref's tree.
//
//   - Trunk ref missing AND was explicitly set in aiwf.yaml → hard
//     error. The user named a specific ref; if it doesn't exist
//     they should fix the typo or fetch it. Silent degradation
//     would defeat their explicit intent.
//
//   - Trunk ref missing AND was the default (no allocate.trunk in
//     aiwf.yaml) AND no refs/remotes/* tracking refs exist → skip.
//     Covers "no remote configured" (sandbox repos), "remote
//     configured but never fetched" (transient setup), and "freshly
//     cloned an empty bare" (canonical first-push setup, where the
//     bare has no branches so the clone has none either). None has
//     anything to collide with.
//
//   - Trunk ref missing AND was the default AND tracking refs DO
//     exist → hard error. The remote has been populated; an
//     unresolvable default trunk is a real misconfiguration (the
//     team's trunk is named something other than main, or the
//     consumer hasn't fetched it). Setting allocate.trunk in
//     aiwf.yaml fixes it.
//
// LocalRefIDs is the second surface: it widens that view to every
// local branch ref (refs/heads/*) for the allocator only — never the
// ids-unique check — so a sibling worktree's freshly-committed id is
// seen at allocation time (M-0212).
//
// LocalRefHits/RemoteRefHits and detectCollisions (M-0259) widen this
// further into a second consumer beyond the allocator: the per-id
// path/ref view they carry, plus blob-content comparison across refs,
// feed the check layer's cross-branch-pending/cross-branch-collision
// classification (ADR-0030) — so the package's scope is no longer
// "for the allocator" alone, but every read-only consumer of "what ids
// are visible where, across every locally-knowable ref."
//
// The package is read-only and has no per-process cache; callers
// invoke once per verb run.
package trunk

import (
	"context"
	"errors"
	"fmt"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// ID names one entity that exists in the trunk ref's tree, by kind,
// id string, and the repo-relative path the trunk has it at.
type ID struct {
	Kind entity.Kind
	ID   string
	Path string
}

// Result carries the trunk ids visible to the allocator and the
// cross-tree check. Skipped is true when the repo has no remotes and
// the trunk read was deliberately bypassed.
type Result struct {
	IDs     []ID
	Skipped bool
}

// Read returns the entity ids in cfg's configured trunk ref,
// following the policy in this package's doc comment.
//
// cfg may be nil — in which case the default trunk ref is used.
// Callers pass workdir as the consumer repo root.
func Read(ctx context.Context, workdir string, cfg *config.Config) (Result, error) {
	if !gitops.IsRepo(ctx, workdir) {
		return Result{Skipped: true}, nil
	}
	ref, explicit := cfg.AllocateTrunkRef()
	paths, err := gitops.LsTreePaths(ctx, workdir, ref, "work/", "docs/adr/")
	if err != nil {
		if !errors.Is(err, gitops.ErrRefNotFound) {
			return Result{}, fmt.Errorf("reading trunk ref %q: %w", ref, err)
		}
		if explicit {
			return Result{}, fmt.Errorf("trunk ref %q does not resolve in this repo; fetch it or fix allocate.trunk in aiwf.yaml", ref)
		}
		// Default trunk missing. Skip when no tracking refs exist
		// (empty remote, never-fetched, or no remote at all); error
		// when tracking refs are present (the team's trunk is named
		// something other than main and allocate.trunk needs to be
		// set).
		hasTracking, hErr := gitops.HasAnyRemoteTrackingRefs(ctx, workdir)
		if hErr != nil {
			return Result{}, fmt.Errorf("checking remote-tracking refs: %w", hErr)
		}
		if !hasTracking {
			return Result{Skipped: true}, nil
		}
		return Result{}, fmt.Errorf("trunk ref %q does not resolve in this repo; fetch it or set allocate.trunk in aiwf.yaml", ref)
	}

	return Result{IDs: idsFromPaths(paths)}, nil
}

// idsFromPaths converts repo-relative entity paths (as returned by
// gitops.LsTreePaths) into typed ID triples, dropping any path that
// does not classify as a recognized entity file. Shared by Read (the
// trunk ref) and LocalRefIDs (every local branch ref).
func idsFromPaths(paths []string) []ID {
	ids := make([]ID, 0, len(paths))
	for _, p := range paths {
		k, ok := entity.PathKind(p)
		if !ok {
			continue
		}
		id, ok := entity.IDFromPath(p, k)
		if !ok {
			continue
		}
		ids = append(ids, ID{Kind: k, ID: id, Path: p})
	}
	return ids
}

// RefHit names one entity id visible on a specific local or
// remote-tracking git ref (M-0259/AC-1): the same (kind, id, path)
// triple as trunk.ID, tagged with the originating ref name. Widens
// LocalRefIDs/RemoteRefIDs' bare-id view so a consumer beyond the
// allocator (the check-side cross-branch-pending tier, the read-side
// show/list resolver) can tell WHICH ref and path a hit came from,
// not just that the id exists somewhere.
type RefHit struct {
	Kind entity.Kind
	ID   string
	Path string
	Ref  string
}

// LocalRefIDs returns the entity id strings reachable from every
// local branch ref (refs/heads/*) in workdir's repository. It is the
// allocator's broadened cross-branch view (M-0212): a sibling git
// worktree's freshly-committed entity already lives in the shared
// local refs, so unioning these ids into the allocator's max keeps
// the next allocation from colliding with it (G-0272 class 1).
//
// Best-effort and read-only: it never returns an error. On any odd
// repo state — not a git repository, no local branches, or a ref that
// lists but fails to read — it degrades to the ids it could collect
// (down to none), never blocking or failing the caller's allocation
// (M-0212/AC-2). The scan narrows the collision window; it is not a
// correctness gate, and the irreducible cross-machine race stays
// `aiwf reallocate`'s to cure.
//
// Unlike Read, the result feeds the allocator ONLY — never the
// ids-unique trunk-collision check, which keeps its working-tree-vs-
// trunk basis. Folding every sibling branch into the uniqueness
// comparison would false-flag the same entity present on two branches
// (e.g. a feature branch forked from main); E-0052's decision is to
// take only the prevention half here.
//
// Cost is one `git ls-tree` per local branch — O(local branches)
// subprocesses per call. Trivial at the solo / handful-of-branches
// scale this targets; it grows linearly, so a repo carrying hundreds
// of stale local branches would pay for them on every allocation.
//
// Derived from LocalRefHits (M-0259/AC-1) by dropping path/ref —
// additive widening only, this function's shape and behavior are
// unchanged from before AC-1.
func LocalRefIDs(ctx context.Context, workdir string) []string {
	return HitIDStrings(LocalRefHits(ctx, workdir))
}

// RemoteRefIDs returns the entity id strings reachable from every
// remote-tracking ref (refs/remotes/*) in workdir's repository — the
// remote-side mirror of LocalRefIDs (M-0214). An entity pushed to any
// remote branch (a teammate's not-yet-merged work, a CI checkout) is
// visible in the local remote-tracking refs, so unioning these ids into
// the allocator's max keeps the next allocation from colliding with it
// (G-0316).
//
// Same best-effort, allocation-only contract as LocalRefIDs: it never
// returns an error, degrades to nil on odd repo states, and feeds the
// allocator ONLY — never the ids-unique check, which keeps its
// working-tree-vs-trunk basis.
//
// Derived from RemoteRefHits (M-0259/AC-1) by dropping path/ref —
// additive widening only.
func RemoteRefIDs(ctx context.Context, workdir string) []string {
	return HitIDStrings(RemoteRefHits(ctx, workdir))
}

// LocalRefHits is LocalRefIDs' widened form (M-0259/AC-1): every hit
// on every local branch ref, carrying kind/path/ref alongside the id.
// Same best-effort, read-only, allocation-plus-check-consumer contract
// as LocalRefIDs — never errors, degrades to nil on odd repo states.
func LocalRefHits(ctx context.Context, workdir string) []RefHit {
	return refHits(ctx, workdir, gitops.LocalBranchRefs)
}

// RemoteRefHits is RemoteRefIDs' widened form (M-0259/AC-1): every hit
// on every remote-tracking ref, carrying kind/path/ref alongside the
// id. Same best-effort, read-only contract as RemoteRefIDs.
func RemoteRefHits(ctx context.Context, workdir string) []RefHit {
	return refHits(ctx, workdir, gitops.RemoteTrackingRefs)
}

// refHits scans every ref returned by listRefs — ls-treeing each and
// collecting the entity ids tagged with the ref they came from — for
// the broadened cross-branch view. Best-effort and read-only: it
// never errors, degrading to the hits it could collect (down to none)
// on any odd repo state. Shared by LocalRefHits (local branches,
// M-0212/M-0259) and RemoteRefHits (remote-tracking refs, M-0214/
// M-0259); the two differ only in which refs they enumerate.
func refHits(ctx context.Context, workdir string, listRefs func(context.Context, string) ([]string, error)) []RefHit {
	if !gitops.IsRepo(ctx, workdir) {
		return nil
	}
	refs, err := listRefs(ctx, workdir)
	if err != nil { //coverage:ignore not portably triggerable: once IsRepo passed, the `git for-each-ref` that backs both listers returns 0 even for a repo with broken refs (it warns and skips them); a non-zero exit needs a git-level failure (missing binary) that the unit harness cannot stage. Degrade to the rest of the allocator's view.
		return nil
	}
	var hits []RefHit
	for _, ref := range refs {
		paths, err := gitops.LsTreePaths(ctx, workdir, ref, "work/", "docs/adr/")
		if err != nil {
			// An individual ref that lists but won't read (corrupt or
			// raced away mid-scan) is skipped, not fatal — degrade to
			// the rest.
			continue
		}
		for _, id := range idsFromPaths(paths) {
			hits = append(hits, RefHit{Kind: id.Kind, ID: id.ID, Path: id.Path, Ref: ref})
		}
	}
	return hits
}

// DistinctRefs returns the distinct ref names carried by hits, in
// first-seen order — the candidate-ref list a caller surfaces when
// hits disagree (M-0259/AC-3's cross-branch-collision finding,
// M-0260/AC-3's aiwf show/list refusal to arbitrate between them).
func DistinctRefs(hits []RefHit) []string {
	seen := make(map[string]bool, len(hits))
	var refs []string
	for _, h := range hits {
		if seen[h.Ref] {
			continue
		}
		seen[h.Ref] = true
		refs = append(refs, h.Ref)
	}
	return refs
}

// HitIDStrings returns just the id strings from hits, in order.
// Convenience for LocalRefIDs/RemoteRefIDs, which only need id values.
func HitIDStrings(hits []RefHit) []string {
	if len(hits) == 0 {
		return nil
	}
	ids := make([]string, len(hits))
	for i, h := range hits {
		ids[i] = h.ID
	}
	return ids
}

// needsBlobStats reports whether detectCollisions must open a BlobReader
// and issue blob-stat round-trips for hits: true iff some canonicalized
// id carries two or more hits (a single-hit id has nothing to compare
// against). It is the stat-gate detectCollisions consults before any git
// work, and the scale property E-0067/M-0265/AC-4 pins — zero blob-stats
// when it returns false, which is exactly the all-ids-present-locally
// state ScanCrossBranch's lazy filter produces.
func needsBlobStats(hits []RefHit) bool {
	seen := make(map[string]bool, len(hits))
	for _, h := range hits {
		id := entity.Canonicalize(h.ID)
		if seen[id] {
			return true
		}
		seen[id] = true
	}
	return false
}

// detectCollisions groups hits (as returned by LocalRefHits/
// RemoteRefHits, concatenated by the caller) by canonicalized id, and
// — for any id appearing on more than one distinct ref — compares
// blob content at each hit's path via gitops.BlobReader.Stat
// (M-0259/AC-3, G-0415). Returns the set of canonicalized ids whose
// content diverges across refs — the caller (refsResolve/
// classifyBodyToken) surfaces this as the distinct, non-blocking
// cross-branch-collision subcode rather than the ordinary
// cross-branch-pending one (D-0036): divergence alone can't
// distinguish a genuine duplicate-mint collision from an ordinary
// same-entity edit still unmerged on a sibling branch, so this
// function reports divergence and leaves severity/interpretation to
// the caller.
//
// Best-effort and read-only, mirroring LocalRefHits/RemoteRefHits'
// contract: a blob-read failure for a given hit excludes it from the
// comparison rather than erroring the whole call (G-0415's accepted
// transient-failure limitation) — degrading toward "no collision
// detected" is the safe direction, since a spurious collision finding
// is noise the caller has to triage for nothing.
//
// Opens one BlobReader subprocess only when at least one id has more
// than one hit. In a repo with several local branches or a configured
// remote, multi-hit ids are the common case (every entity shared with
// a sibling branch or the trunk ref), not the exception — so the
// BlobReader typically does spawn on most verb runs; it just never
// spawns for the id-free or fully-solo-branch case, where there is
// nothing to compare.
//
// Composed only by ScanCrossBranch, which applies the lazy locally-absent
// filter before calling this — so the union+collision composition lives in
// exactly one place (G-0418). Kept unexported deliberately: calling it from
// another package (or re-exporting it) reopens the O(entities×refs)
// duplication this consolidation removed. Route cross-branch collision
// detection through ScanCrossBranch instead.
func detectCollisions(ctx context.Context, workdir string, hits []RefHit) map[string]bool {
	collisions := make(map[string]bool)
	// Gate on the stat-work check before any grouping or git work: with
	// no multi-hit id there is nothing to compare, so return without
	// opening a BlobReader (zero blob-stats).
	if !needsBlobStats(hits) {
		return collisions
	}

	byID := make(map[string][]RefHit, len(hits))
	for _, h := range hits {
		key := entity.Canonicalize(h.ID)
		byID[key] = append(byID[key], h)
	}

	br, err := gitops.NewBlobReader(ctx, workdir)
	if err != nil {
		// Can't compare without the reader; degrade to "no collision
		// detected" for every multi-hit id rather than erroring the
		// whole call.
		return collisions
	}
	defer func() { _ = br.Close() }()

	for id, group := range byID {
		if len(group) < 2 {
			continue
		}
		var first string
		diverges := false
		for _, h := range group {
			sha, err := br.Stat(h.Ref, h.Path)
			if err != nil {
				// Unreadable hit (raced away, corrupt) — excluded from
				// the comparison, not fatal.
				continue
			}
			if first == "" {
				first = sha
				continue
			}
			if sha != first {
				diverges = true
			}
		}
		if diverges {
			collisions[id] = true
		}
	}
	return collisions
}

// CrossBranchScan is the outcome of one cross-branch ref scan: the
// per-ref-class hit slices the allocator's id view needs (LocalHits /
// RemoteHits), their union (Hits — the cross-branch read-side index),
// and the divergent-content id set among them (Collisions). Composed
// once by ScanCrossBranch so no consumer re-derives the union or
// re-runs detectCollisions (E-0067, G-0418).
type CrossBranchScan struct {
	// LocalHits is every entity-id hit on a local branch ref
	// (refs/heads/*) — the allocator's LocalRefIDs view (M-0212).
	LocalHits []RefHit
	// RemoteHits is every hit on a remote-tracking ref (refs/remotes/*)
	// — the allocator's RemoteRefIDs view (M-0214).
	RemoteHits []RefHit
	// Hits is LocalHits followed by RemoteHits: the cross-branch union
	// the read-side resolver (crossBranchIndex / show / list) groups by
	// id on a local-tree miss (ADR-0030).
	Hits []RefHit
	// Collisions is the canonicalized-id set whose hits carry divergent
	// blob content across refs (detectCollisions, G-0415), consulted on
	// a local-tree miss to escalate cross-branch-pending to
	// cross-branch-collision (D-0036). Domain bound: only ids ABSENT from
	// the local tree (per ScanCrossBranch's presentLocally) can appear
	// here — a locally-present diverging id is deliberately omitted by the
	// lazy filter, so a bare Collisions[id] lookup returns false even for
	// a genuinely diverging present id. Read it only after a local-tree
	// miss (as every consumer does).
	Collisions map[string]bool
}

// ScanCrossBranch composes the local + remote ref-hit union once and
// detects content collisions among them, returning both the union (for
// the cross-branch read-side index) and its per-ref-class halves (for
// the allocator's id view). It is the single composition point for the
// cross-branch scan E-0060 shipped copied across three call sites
// (cliutil.LoadTreeWithTrunk, list.crossBranchListRows,
// show.buildCrossBranchShowView); consolidating it here keeps the
// "hits scanned equal the hits handed to detectCollisions" coupling in
// one place (G-0418).
//
// Collision detection is lazy: only hits whose id is absent from the
// local tree (presentLocally reports false) are handed to
// detectCollisions, so the O(entities×refs) blob-stat pass shrinks to
// the locally-absent id set — nothing in the common all-merged state,
// where every id resolves locally. This is behavior-preserving because
// every consumer reads a collision result only after a local-tree miss
// (ADR-0030): a locally-present id's collision entry is never observed,
// so declining to compute it changes no output. A nil presentLocally
// disables the filter (every id treated as absent, collisions computed
// over the full union) — nil-safety for absentHits, not a mode any
// production caller uses; all three callers pass a real predicate.
//
// Best-effort and read-only, inheriting LocalRefHits/RemoteRefHits/
// detectCollisions' contract: it never errors, degrading to empty
// slices and an empty collision map on odd repo state.
func ScanCrossBranch(ctx context.Context, workdir string, presentLocally func(canonicalID string) bool) CrossBranchScan {
	local := LocalRefHits(ctx, workdir)
	remote := RemoteRefHits(ctx, workdir)
	hits := append(append([]RefHit(nil), local...), remote...)
	return CrossBranchScan{
		LocalHits:  local,
		RemoteHits: remote,
		Hits:       hits,
		Collisions: detectCollisions(ctx, workdir, absentHits(hits, presentLocally)),
	}
}

// absentHits returns the subset of hits whose canonical id is absent
// from the local tree per presentLocally — the exact hit set
// ScanCrossBranch hands to detectCollisions (E-0067/M-0265/AC-2).
// presentLocally is consulted once per distinct id (not once per hit),
// so the filter's own cost stays O(distinct ids), matching the
// read-side loop's existing per-id tr.ByID cost rather than multiplying
// it by the ref count. A nil predicate returns hits unchanged (no local
// tree to filter against).
func absentHits(hits []RefHit, presentLocally func(canonicalID string) bool) []RefHit {
	if presentLocally == nil {
		return hits
	}
	// Group by canonical id so presentLocally is consulted once per
	// distinct id, preserving first-seen id order for a deterministic
	// result.
	byID := make(map[string][]RefHit, len(hits))
	order := make([]string, 0, len(hits))
	for _, h := range hits {
		id := entity.Canonicalize(h.ID)
		if _, seen := byID[id]; !seen {
			order = append(order, id)
		}
		byID[id] = append(byID[id], h)
	}
	var out []RefHit
	for _, id := range order {
		if presentLocally(id) {
			continue
		}
		out = append(out, byID[id]...)
	}
	return out
}

// IDStrings returns just the id strings from r, in the order they
// appear in r.IDs. Convenience for AllocateID, which only needs the
// id values.
func (r Result) IDStrings() []string {
	if len(r.IDs) == 0 {
		return nil
	}
	out := make([]string, len(r.IDs))
	for i, id := range r.IDs {
		out[i] = id.ID
	}
	return out
}
