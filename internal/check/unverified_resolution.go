package check

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/tree"
)

// unverifiedSubcode is the non-blocking classification a read-only
// surface substitutes for `unresolved` when it never built the
// cross-branch view (G-0558).
const unverifiedSubcode = "unresolved-unverified"

// MarkUnverifiedResolution downgrades the `unresolved` verdicts in
// findings when t was loaded without the cross-branch ref scan.
//
// `unresolved` asserts an id exists at no tier — not the working tree,
// not trunk, not any local or remote-tracking ref. A load that skipped
// the scan has evidence for the working tree alone, and an empty hit
// set cannot distinguish "looked everywhere and found nothing" from
// "never looked". Without this pass a ref-less surface reports the
// strong verdict off the weak evidence, contradicting `aiwf check` on
// the same bytes and raising blocking errors the authoritative gate
// does not — on the surfaces an operator runs between commits.
//
// This is a presentation-layer pass, applied by a reporting surface to
// its own findings, rather than a change to the resolution rules. The
// rules are shared with the verb layer, which projects findings to
// decide whether to refuse a mutation: there, an error-severity
// `unresolved` is what suppresses the plan, and softening it would let
// a verb commit a reference that resolves nowhere. A surface that only
// prints may decline to make a claim it cannot support; a surface that
// acts on the claim has to go build the evidence instead.
//
// A tree that WAS scanned passes through untouched, so a caller that
// later switches to a loader building the full view stops downgrading
// without a matching edit here.
//
// findings is modified in place and returned for call-site convenience;
// the caller owns the slice and must not keep the original expecting it
// to be unchanged.
func MarkUnverifiedResolution(findings []Finding, t *tree.Tree) []Finding {
	if t == nil || t.CrossBranchScanned {
		return findings
	}
	for i := range findings {
		if !dependsOnTheTierStack(findings[i].Code, findings[i].Subcode) {
			continue
		}
		findings[i].Subcode = unverifiedSubcode
		findings[i].Severity = SeverityWarning
		findings[i].Message = unverifiedMessage(findings[i].Message)
		findings[i].Hint = HintFor(findings[i].Code, unverifiedSubcode)
	}
	return findings
}

// dependsOnTheTierStack reports whether a finding's verdict rests on
// having consulted every resolution tier, and so cannot be reached from
// the working tree alone. Membership is per (code, subcode), because
// two rules can share a subcode name and consult different tiers.
//
// `unresolved` qualifies under both codes: it asserts the referenced id
// is allocated nowhere.
//
// `unresolved-milestone` qualifies under body-prose-id only.
// classifyBodyToken consults the trunk tier before emitting it, so a
// richer load can change its verdict. refsResolve's composite branch
// resolves the parent against the working-tree index alone
// (resolveCompositeRef takes that map and nothing else), so its verdict
// is identical under either loader — downgrading it would manufacture
// the very fast-vs-full disagreement this pass exists to remove.
//
// `unresolved-ac` qualifies under neither. It fires only once the
// parent entity is in hand, and asserts something about the AC list in
// that file, which the caller is holding.
//
// `narrow-width` qualifies under neither either, for the opposite
// reason to `unresolved`: it asserts the id resolves SOMEWHERE, which a
// working-tree hit alone establishes. A ref-less load reaching it has
// already found the entity; one that has not stops short and reports
// `unresolved`, which this pass then downgrades.
func dependsOnTheTierStack(code, subcode string) bool {
	switch subcode {
	case "unresolved":
		return code == CodeRefsResolve || code == CodeBodyProseID
	case "unresolved-milestone":
		return code == CodeBodyProseID
	default:
		return false
	}
}

// unverifiedNeutralClause is the phrase every rewritten message carries
// in place of the rule's stronger one. Tests assert its presence to
// detect a rule whose wording drifted past the cases below, which would
// otherwise degrade silently through the fallback.
const unverifiedNeutralClause = "resolves to no entity in this working tree"

// unverifiedMessage rewrites a message so it states what the surface
// actually established. Each downgraded rule phrases the strong verdict
// its own way and asserts more than a ref-less load can support, so the
// asserting clause is replaced rather than appended to.
//
// This couples to the rules' wording. A reword lands in the fallback,
// where the message keeps its overclaim and merely gains a hedge — so
// the contract test drives the real rules rather than fixtures, and
// fails on a message missing unverifiedNeutralClause.
func unverifiedMessage(msg string) string {
	const tail = "; the cross-branch view was not built, so it may exist on an unmerged branch"

	// refs-resolve/unresolved:   …references unknown id "X"
	// body-prose-id/unresolved:  …unknown id "X" (no entity allocated at this id)
	if before, id, ok := strings.Cut(msg, "unknown id "); ok {
		id = strings.TrimSuffix(id, " (no entity allocated at this id)")
		return fmt.Sprintf("%s%s, which %s%s", before, id, unverifiedNeutralClause, tail)
	}
	// body-prose-id/unresolved-milestone: …whose parent "X" is not allocated
	if before, ok := strings.CutSuffix(msg, " is not allocated"); ok {
		return fmt.Sprintf("%s %s%s", before, unverifiedNeutralClause, tail)
	}
	return msg + tail
}
