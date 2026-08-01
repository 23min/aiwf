package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/gitops"
)

// ClaimDivergenceError reports that a verb refused before deciding
// anything, because a file its decision rests on carries changes no verb
// recorded.
//
// It is the claim-side twin of UncommittedConflictError and reaches the
// operator the same way — as an ordinary verb error, which the CLI's
// shared handler already reports as a usage-level exit. The two are
// separate types because they refuse at different seams: this one before
// the verb has decided what to do, that one before the commit that would
// carry it (ADR-0038).
type ClaimDivergenceError struct {
	// Subject names what the verb was asked about — the entity or
	// composite id whose state it was about to read.
	Subject string
	// Diverged lists every path the decision rests on that no longer
	// matches HEAD.
	Diverged []gitops.Divergence
}

// Paths returns every diverging path, in the order reported.
func (e *ClaimDivergenceError) Paths() []string {
	paths := make([]string, 0, len(e.Diverged))
	for _, d := range e.Diverged {
		paths = append(paths, d.Path)
	}
	return paths
}

// pathsOfKind returns the diverging paths matching one kind.
func (e *ClaimDivergenceError) pathsOfKind(k gitops.DivergenceKind) []string {
	var out []string
	for _, d := range e.Diverged {
		if d.Kind == k {
			out = append(out, d.Path)
		}
	}
	return out
}

func (e *ClaimDivergenceError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: uncommitted changes at %s\n", e.Subject, strings.Join(e.Paths(), ", "))
	b.WriteString("  this verb reads that file to decide what to do and commits what it finds\n")
	b.WriteString("  there, so both the decision and the record would rest on bytes no verb wrote\n")
	if modified := e.pathsOfKind(gitops.DivergenceModified); len(modified) > 0 {
		fmt.Fprintf(&b, "  commit a body edit on its own with `aiwf edit-body <id>`, or set it aside with\n"+
			"  `git stash -u` (`git restore %s` discards it outright)\n",
			strings.Join(modified, " "))
	}
	if missing := e.pathsOfKind(gitops.DivergenceAbsentFromDisk); len(missing) > 0 {
		fmt.Fprintf(&b, "  recorded at HEAD but missing from the working tree: restore it with\n"+
			"  `git restore %s`\n", strings.Join(missing, " "))
	}
	b.WriteString("  then re-run the verb")
	return b.String()
}

// guardClaim refuses when a file the verb is about to read — to decide
// whether its request is already satisfied, and then to serialize around
// — differs from HEAD in the working copy.
//
// It runs in the verb's prelude, after the arguments have resolved and
// before any comparison. Placement is the property, not a preference.
// Observing early but refusing later was measured to leave two holes at
// `promote`, both in the window a deferred refusal opens: a
// `--superseded-by` whose reciprocal back-link was hand-edited onto disk
// committed a one-sided supersession at exit 0, because the verb read
// those bytes, concluded the back-link was already there, and emitted no
// op for that file — so the plan never named it and the commit-side
// guard never saw it. And the resolver re-point refusal, which fires
// before any plan exists, told an operator their gap was "already
// addressed and already carries a resolver" when HEAD said `open` with
// none, recommending `--force` on the strength of it. Refusing here
// forecloses both: nothing downstream runs against disputed bytes,
// whether or not it would have produced a plan (ADR-0038).
//
// paths is what the verb's decision rests on, which is its own to
// determine and is not always its target's file: three claims rest on
// aiwf.yaml rather than on any entity, and the sweeps have no target at
// all — archive compares per candidate move inside its own planner, and
// the rest carry a recorded reason for needing no comparison. The full
// inventory is internal/policies/noop_claim_scope.go.
//
// An entity's claim is refused whether or not HEAD records the path the
// entity currently occupies. The claim is about the entity, and an
// entity's file can move without any verb: after a plain `mv`, the
// working copy is the only thing that says where G-NNNN lives, so a
// path-absent-from-HEAD exemption would let the verb read a status HEAD
// contradicts and answer "already set; nothing to change" — the exact
// reproduction this guard exists to refuse — and let a write land beside
// HEAD's untouched copy, putting one id at two paths in the record.
//
// An unborn HEAD carries no record at all, so the guard stands down —
// the reading Apply's commit-side guard already takes, and one every
// verb meets, since a verb's own commit is routinely a repo's first.
func guardClaim(ctx context.Context, root, subject string, paths ...string) error {
	return guardClaimPaths(ctx, root, subject, false, paths)
}

// guardClaimConfig is the aiwf.yaml variant, and the exemption is the
// whole difference: `aiwf init` leaves that file uncommitted by design,
// so a config-scoped claim refusing a path absent from HEAD would make
// every verb that rewrites it unreachable until someone committed it.
// The file is not an entity and cannot move out from under an id, so the
// reasoning that forbids the exemption for a target entity does not
// reach it.
func guardClaimConfig(ctx context.Context, root, subject string, paths ...string) error {
	return guardClaimPaths(ctx, root, subject, true, paths)
}

func guardClaimPaths(ctx context.Context, root, subject string, exemptAbsentFromHEAD bool, paths []string) error {
	hasHEAD, headErr := gitops.HasHEAD(ctx, root)
	if headErr != nil {
		return fmt.Errorf("checking %s against HEAD: %w", subject, headErr)
	}
	if !hasHEAD {
		return nil
	}
	diverged, divErr := gitops.DivergentPaths(ctx, root, paths)
	if divErr != nil {
		return fmt.Errorf("checking %s against HEAD: %w", subject, divErr)
	}
	blocking := make([]gitops.Divergence, 0, len(diverged))
	for _, d := range diverged {
		if exemptAbsentFromHEAD && d.Kind == gitops.DivergenceAbsentFromHEAD {
			continue
		}
		blocking = append(blocking, d)
	}
	if len(blocking) == 0 {
		return nil
	}
	return &ClaimDivergenceError{Subject: subject, Diverged: blocking}
}
