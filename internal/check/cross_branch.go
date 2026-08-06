package check

// M-0259/AC-2: the cross-branch tier shared by refsResolve
// (structured fields) and classifyBodyToken (prose tokens). Both
// consult the same second-tier resolver on a local-tree miss, before
// firing a hard `unresolved` (ADR-0030): an id known only on another
// local branch or remote-tracking ref is real, just not merged into
// this branch's working tree yet, so it classifies as a distinct
// cross-branch subcode instead.
//
// Which one is decided by the most visible ref carrying the id
// (ADR-0041, via trunk.RemoteVisible). A remote-tracking hit means the
// entity is published — anyone who fetches resolves the reference — and
// stays the non-blocking `cross-branch-pending`. Hits confined to local
// branch refs mean the entity exists on one working copy on earth, so
// the tree is not one that can be handed to anyone; that fires
// `cross-branch-local-only` at error severity, and the remedy it names
// is publishing the branch.
//
// Unlike the silent Trunk tier (G-0241, trunk is authoritative), the
// cross-branch tier is deliberately visible: a sibling branch is
// provisional (it can be rebased, renamed, or abandoned before it
// merges), so softening it silently would let a dangling reference
// masquerade as valid forever. Recomputed fresh from tree.CrossBranchHits
// on every `aiwf check` run (nothing here is cached), so a source
// branch's disappearance re-escalates the next run's classification
// back to `unresolved` on its own, and publishing or unpublishing that
// branch moves it between the two cross-branch subcodes the same way
// (M-0259/AC-4) — no separate escalation-tracking mechanism to drift.

import (
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
)

// IsCrossBranchClassification reports whether f classifies a reference
// by which git refs currently carry its target, rather than reporting a
// defect in the content being checked. The three cross-branch subcodes
// share that character: the reference is well-formed and the entity it
// names exists — what varies is where it is reachable from.
//
// The verb layer consults this to decide what it refuses a write for
// (ADR-0041). A cross-branch classification is not something an author
// can answer by changing the bytes they are writing: the fix is to push
// a branch, which may not even be theirs. So the boundary that enforces
// it is the push, via `aiwf check`, and authoring stays open.
func IsCrossBranchClassification(f Finding) bool {
	return strings.HasPrefix(f.Subcode, "cross-branch-")
}

// crossBranchIndex groups t.CrossBranchHits by canonicalized id. Nil
// t.CrossBranchHits (in-memory test trees, no-remote repos) yields an
// empty index, so every lookup misses and resolution degrades to
// today's two-tier (working tree, unresolved) behavior.
func crossBranchIndex(t *tree.Tree) map[string][]trunk.RefHit {
	idx := make(map[string][]trunk.RefHit, len(t.CrossBranchHits))
	for _, h := range t.CrossBranchHits {
		key := entity.Canonicalize(h.ID)
		idx[key] = append(idx[key], h)
	}
	return idx
}

// joinRefNames formats the distinct ref names in hits for a finding
// message, e.g. "refs/heads/sibling", or a comma-joined list when the
// id is visible on more than one ref. Delegates the dedup itself to
// trunk.DistinctRefs (M-0260) — aiwf show/list's read-side resolver
// needs the same distinct-ref-names list (there, to name the candidate
// refs of a cross-branch-collision it declines to arbitrate), so the
// dedup logic lives once on the package that owns RefHit rather than
// twice.
func joinRefNames(hits []trunk.RefHit) string {
	return strings.Join(trunk.DistinctRefs(hits), ", ")
}
