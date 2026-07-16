package check

// M-0259/AC-2: the cross-branch-pending tier shared by refsResolve
// (structured fields) and classifyBodyToken (prose tokens). Both
// consult the same second-tier resolver on a local-tree miss, before
// firing a hard `unresolved` (ADR-0030): an id known only on another
// local branch or remote-tracking ref is real, just not merged into
// this branch's working tree yet, so it classifies as a distinct,
// non-blocking `cross-branch-pending` subcode instead.
//
// Unlike the silent Trunk tier (G-0241, trunk is authoritative), the
// cross-branch tier is deliberately visible: a sibling branch is
// provisional (it can be rebased, renamed, or abandoned before it
// merges), so softening it silently would let a dangling reference
// masquerade as valid forever. Recomputed fresh from tree.CrossBranchHits
// on every `aiwf check` run (nothing here is cached), so a source
// branch's disappearance re-escalates the next run's classification
// back to `unresolved` on its own (M-0259/AC-4) — no separate
// escalation-tracking mechanism to drift.

import (
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
)

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

// crossBranchSubcode picks the refs-resolve/body-prose-id subcode and
// severity for a cross-branch hit (M-0259/AC-2/AC-3): genuine content
// divergence across refs (collision=true, from trunk.DetectCollisions)
// escalates to the blocking cross-branch-collision subcode; otherwise
// the hit is the ordinary non-blocking cross-branch-pending case.
// Takes the bool directly (not the tree) so both refsResolve
// (tree-backed lookup) and classifyBodyToken (BodyProseIndex-backed
// lookup, no tree access) can share it.
func crossBranchSubcode(collision bool) (subcode string, severity Severity) {
	if collision {
		return "cross-branch-collision", SeverityError
	}
	return "cross-branch-pending", SeverityWarning
}

// joinRefNames formats the distinct ref names in hits for a finding
// message, e.g. "refs/heads/sibling", or a comma-joined list when the
// id is visible on more than one ref.
func joinRefNames(hits []trunk.RefHit) string {
	seen := make(map[string]bool, len(hits))
	refs := make([]string, 0, len(hits))
	for _, h := range hits {
		if seen[h.Ref] {
			continue
		}
		seen[h.Ref] = true
		refs = append(refs, h.Ref)
	}
	return strings.Join(refs, ", ")
}
