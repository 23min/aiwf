package verb

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// MilestoneTDD sets a milestone's `tdd:` policy field after creation —
// the post-allocation mutator for the create-time `--tdd` flag on
// `aiwf add milestone`. Closes the `tdd:` slice of G-0168's
// verb-chokepoint hole so changing the policy is a first-class,
// trailered act rather than a hand-edit.
//
// Gating is uniform-ordinary (D-0048): the verb carries no directional
// or sovereign carve-out — weakening (`required -> none`) takes the
// identical path as strengthening. The provenance/allow-rule gating
// that governs every ordinary mutation applies through the caller's
// ProvenanceContext; this verb adds no status-edge entry to the
// sovereign-act tier.
//
// Emits one OpWrite with `aiwf-verb: milestone-tdd` trailers, producing
// the kernel's per-mutation atomicity guarantee. reason is optional
// free-form prose; when non-empty it lands in the commit body so the
// rationale surfaces in `aiwf history`.
func MilestoneTDD(ctx context.Context, t *tree.Tree, id, policy, actor, reason string) (*Result, error) {
	_ = ctx
	if entity.IsCompositeID(id) {
		return nil, fmt.Errorf("milestone tdd does not accept composite ids; pass a milestone id (M-NNNN)")
	}
	if !entity.IsAllowedTDDPolicy(policy) {
		return nil, fmt.Errorf("--policy %q is not a recognized TDD policy; allowed: %s", policy, strings.Join(entity.AllowedTDDPolicies(), ", "))
	}

	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("milestone %q not found", id)
	}
	if e.Kind != entity.KindMilestone {
		return nil, fmt.Errorf("%q is of kind %s, not milestone", id, e.Kind)
	}

	// Refuse-with-hint: a flip to `required` must not strand an
	// already-`met` AC that has no `tdd_phase: done`. Requiring TDD
	// after the fact cannot honestly manufacture a `red`/`done` phase
	// on work that already passed, so the verb names the offending ACs
	// and aborts rather than either seeding a false phase or letting
	// the projection check reject the write as an untargeted finding.
	// Detection mirrors the acs-tdd-audit rule (internal/check/acs.go).
	if policy == "required" {
		var stranded []string
		for _, ac := range e.ACs {
			if ac.Status == entity.StatusMet && ac.TDDPhase != entity.TDDPhaseDone {
				stranded = append(stranded, ac.ID)
			}
		}
		if len(stranded) > 0 {
			return nil, fmt.Errorf(
				"cannot set tdd: required — the following met AC(s) have no tdd_phase: done: %s; requiring TDD now would strand them. Keep the policy at advisory or none, since the phase ladder cannot be re-run on already-met work",
				strings.Join(stranded, ", "))
		}
	}

	modified := *e
	modified.TDD = policy

	body, err := readBody(t.Root, e.Path)
	if err != nil { //coverage:ignore filesystem IO failure reading a file the loaded tree already resolved; no realistic unit-test trigger
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil { //coverage:ignore serialization of a struct that round-tripped through the loader; no realistic unit-test trigger
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	canonID := entity.Canonicalize(id)
	subject := fmt.Sprintf("aiwf milestone tdd %s -> %s", canonID, policy)
	result := plan(&Plan{
		Subject: subject,
		Body:    reason,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "milestone-tdd"},
			{Key: gitops.TrailerEntity, Value: canonID},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	})
	result.Metadata = map[string]any{"entity_id": canonID, "tdd": policy}
	return result, nil
}
