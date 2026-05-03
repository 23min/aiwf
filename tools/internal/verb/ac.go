// Package verb's I2 acceptance-criteria operations.
//
// ACs are sub-elements of a milestone (composite id `M-NNN/AC-N`) and
// mutate the milestone file's frontmatter, not a separate file. Each
// AC verb returns a Plan whose Ops rewrite the parent milestone file
// in place, with trailers carrying the composite id as `aiwf-entity:`
// so `aiwf history` filters cleanly.

package verb

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// AddAC creates a new acceptance criterion under the named milestone.
// The new AC is appended to the milestone's acs[] (max+1 position-
// stable; cancelled entries count toward position) and a matching
// `### AC-<N> — <title>` heading is appended to the milestone body.
//
// When the parent milestone is `tdd: required`, the verb seeds
// `tdd_phase: red` — the only legal starting state under the FSM.
// Otherwise tdd_phase is left absent.
//
// Returns a Go error for "couldn't even start": milestone not found,
// not a milestone, empty title. Tree-level findings caused by the
// addition are returned in Result.Findings.
func AddAC(ctx context.Context, t *tree.Tree, parentID, title, actor string, tests *gitops.TestMetrics) (*Result, error) {
	_ = ctx
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("--title is required")
	}
	if entity.IsProseyTitle(title) {
		return nil, fmt.Errorf("title %q looks like prose, not a short label\n\nKeep the AC title short (≤80 chars, single sentence, no markdown formatting). It becomes the YAML `title:` field AND the `### AC-N — <title>` body heading; markdown or multi-sentence prose renders as one giant heading.\n\nUse a short label for --title, then hand-edit the body section under the heading to add detail prose, examples, references", title)
	}
	parent := t.ByID(parentID)
	if parent == nil {
		return nil, fmt.Errorf("milestone %q not found", parentID)
	}
	if parent.Kind != entity.KindMilestone {
		return nil, fmt.Errorf("%s is a %s, not a milestone — only milestones host ACs", parentID, parent.Kind)
	}

	nextID := fmt.Sprintf("AC-%d", len(parent.ACs)+1)
	newAC := entity.AcceptanceCriterion{
		ID:     nextID,
		Title:  title,
		Status: entity.StatusOpen,
	}
	if parent.TDD == "required" {
		newAC.TDDPhase = entity.TDDPhaseRed
	}

	modified := *parent
	modified.ACs = append([]entity.AcceptanceCriterion(nil), parent.ACs...)
	modified.ACs = append(modified.ACs, newAC)

	body, err := readBody(t.Root, parent.Path)
	if err != nil {
		return nil, err
	}
	body = appendACHeading(body, nextID, title)

	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", parent.ID, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(parent.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	compositeID := parent.ID + "/" + nextID
	subject := fmt.Sprintf("aiwf add ac %s %q", compositeID, title)
	trailers := standardTrailers("add", compositeID, actor)
	if newAC.TDDPhase == entity.TDDPhaseRed && tests != nil {
		trailers = appendTestsTrailer(trailers, tests)
	} else if tests != nil && newAC.TDDPhase != entity.TDDPhaseRed {
		// --tests is meaningful only when the AC enters a TDD phase. A
		// non-tdd-required milestone seeds the AC at no-phase, so test
		// metrics aren't applicable; surface the inconsistency rather
		// than silently dropping the trailer.
		return nil, fmt.Errorf("--tests is only valid when seeding red (parent milestone %s is not tdd: required)", parent.ID)
	}
	return plan(&Plan{
		Subject:  subject,
		Trailers: trailers,
		Ops:      []FileOp{{Type: OpWrite, Path: parent.Path, Content: content}},
	}), nil
}

// promoteAC handles `aiwf promote M-NNN/AC-N <newStatus>`. Mirrors
// Promote's shape: validate the FSM transition (unless force), rewrite
// the parent milestone file with the AC's new status, run projection
// findings, plan the commit. Trailers carry the composite id and
// aiwf-to: <newStatus>.
func promoteAC(t *tree.Tree, compositeID, newStatus, actor, reason string, force bool) (*Result, error) {
	parent, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if !force {
		if !entity.IsLegalACTransition(ac.Status, newStatus) {
			return nil, fmt.Errorf("AC status %q cannot transition to %q (allowed under FSM: see acTransitions)", ac.Status, newStatus)
		}
	}
	modified, err := withACMutation(parent, ac.ID, func(updated *entity.AcceptanceCriterion) {
		updated.Status = newStatus
	})
	if err != nil {
		return nil, err
	}
	return finalizeACPlan(t, parent, modified, "promote", compositeID, newStatus, actor, reason, force, nil,
		fmt.Sprintf("aiwf promote %s %s -> %s", compositeID, ac.Status, newStatus))
}

// PromoteACPhase handles `aiwf promote M-NNN/AC-N --phase <p>`.
// Advances the AC's tdd_phase along the linear FSM (red → green →
// (refactor →) done). Mutex with status changes — the dispatcher
// rejects passing both a positional state and --phase. force=true
// skips the FSM transition rule but coherence (closed-set membership
// of newPhase) still runs via projection findings.
//
// Trailers: aiwf-to: carries the new phase value (same trailer as
// for status changes; the verb name + composite id make it
// unambiguous which dimension moved). aiwf-force: when forced.
func PromoteACPhase(ctx context.Context, t *tree.Tree, compositeID, newPhase, actor, reason string, force bool, tests *gitops.TestMetrics) (*Result, error) {
	_ = ctx
	parent, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if !force {
		if !entity.IsLegalTDDPhaseTransition(ac.TDDPhase, newPhase) {
			return nil, fmt.Errorf("AC tdd_phase %q cannot transition to %q (allowed under FSM: see tddPhaseTransitions)", ac.TDDPhase, newPhase)
		}
	}
	modified, err := withACMutation(parent, ac.ID, func(updated *entity.AcceptanceCriterion) {
		updated.TDDPhase = newPhase
	})
	if err != nil {
		return nil, err
	}
	return finalizeACPlan(t, parent, modified, "promote", compositeID, newPhase, actor, reason, force, tests,
		fmt.Sprintf("aiwf promote %s --phase %s -> %s", compositeID, ac.TDDPhase, newPhase))
}

// cancelAC handles `aiwf cancel M-NNN/AC-N`. The AC's status flips to
// `cancelled`; the entry stays in acs[] at its original position. The
// "already cancelled" guard fires when the AC is already terminal —
// force does not relax that since there's no diff to write.
func cancelAC(t *tree.Tree, compositeID, actor, reason string, force bool) (*Result, error) {
	parent, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if ac.Status == entity.StatusCancelled {
		return nil, fmt.Errorf("%s is already cancelled", compositeID)
	}
	modified, err := withACMutation(parent, ac.ID, func(updated *entity.AcceptanceCriterion) {
		updated.Status = "cancelled"
	})
	if err != nil {
		return nil, err
	}
	// Cancel does not emit aiwf-to: per Step 5's design (target is
	// implicit). Pass empty `to` to suppress the trailer.
	return finalizeACPlan(t, parent, modified, "cancel", compositeID, "", actor, reason, force, nil,
		fmt.Sprintf("aiwf cancel %s -> cancelled", compositeID))
}

// renameAC handles `aiwf rename M-NNN/AC-N "<new-title>"`. Updates
// the AC's title in the milestone's frontmatter and rewrites the
// matching `### AC-<N>` body heading. One commit, no path change.
func renameAC(t *tree.Tree, compositeID, newTitle, actor string) (*Result, error) {
	if strings.TrimSpace(newTitle) == "" {
		return nil, fmt.Errorf("rename: new title is empty")
	}
	parent, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if ac.Title == newTitle {
		return nil, fmt.Errorf("%s title already %q", compositeID, newTitle)
	}
	modified, err := withACMutation(parent, ac.ID, func(updated *entity.AcceptanceCriterion) {
		updated.Title = newTitle
	})
	if err != nil {
		return nil, err
	}
	body, err := readBody(t.Root, parent.Path)
	if err != nil {
		return nil, err
	}
	body = rewriteACHeading(body, ac.ID, newTitle)
	content, err := entity.Serialize(modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", parent.ID, err)
	}
	proj := projectReplace(t, modified, filepath.ToSlash(parent.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}
	subject := fmt.Sprintf("aiwf rename %s title -> %q", compositeID, newTitle)
	return plan(&Plan{
		Subject:  subject,
		Trailers: standardTrailers("rename", compositeID, actor),
		Ops:      []FileOp{{Type: OpWrite, Path: parent.Path, Content: content}},
	}), nil
}

// lookupAC parses a composite id, finds the parent milestone, and
// returns it plus the matched AC. Returns a Go error when the
// composite is malformed, the parent is missing, or the AC id isn't
// in acs[].
func lookupAC(t *tree.Tree, compositeID string) (parent *entity.Entity, ac entity.AcceptanceCriterion, err error) {
	parentID, subID, ok := entity.ParseCompositeID(compositeID)
	if !ok {
		return nil, entity.AcceptanceCriterion{}, fmt.Errorf("%q is not a valid composite id (M-NNN/AC-N)", compositeID)
	}
	parent = t.ByID(parentID)
	if parent == nil {
		return nil, entity.AcceptanceCriterion{}, fmt.Errorf("milestone %q not found", parentID)
	}
	if parent.Kind != entity.KindMilestone {
		return nil, entity.AcceptanceCriterion{}, fmt.Errorf("%s is a %s, not a milestone", parentID, parent.Kind)
	}
	for _, candidate := range parent.ACs {
		if candidate.ID == subID {
			return parent, candidate, nil
		}
	}
	return nil, entity.AcceptanceCriterion{}, fmt.Errorf("%s has no %s in acs[]", parentID, subID)
}

// withACMutation returns a deep-copy of parent with the named AC
// passed through mutate. The original is not modified. Errors when
// the AC id isn't in acs[] (defensive — lookupAC normally catches
// this first).
func withACMutation(parent *entity.Entity, acID string, mutate func(*entity.AcceptanceCriterion)) (*entity.Entity, error) {
	modified := *parent
	modified.ACs = append([]entity.AcceptanceCriterion(nil), parent.ACs...)
	for i := range modified.ACs {
		if modified.ACs[i].ID == acID {
			mutate(&modified.ACs[i])
			return &modified, nil
		}
	}
	return nil, fmt.Errorf("%s not found in acs[]", acID)
}

// finalizeACPlan handles the post-mutation tail shared by promoteAC
// and cancelAC: serialize, run projection findings, build the plan
// with the right trailers. `to` is the aiwf-to value (empty for
// cancel); `force` toggles aiwf-force emission; `tests` (non-nil and
// non-zero) appends an aiwf-tests trailer.
func finalizeACPlan(t *tree.Tree, parent, modified *entity.Entity, verbName, compositeID, to, actor, reason string, force bool, tests *gitops.TestMetrics, subject string) (*Result, error) {
	body, err := readBody(t.Root, parent.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", parent.ID, err)
	}
	proj := projectReplace(t, modified, filepath.ToSlash(parent.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}
	trailers := transitionTrailers(verbName, compositeID, actor, reason, to, force)
	trailers = appendTestsTrailer(trailers, tests)
	return plan(&Plan{
		Subject:  subject,
		Body:     reason,
		Trailers: trailers,
		Ops:      []FileOp{{Type: OpWrite, Path: parent.Path, Content: content}},
	}), nil
}

// appendTestsTrailer appends an aiwf-tests trailer to trailers when
// tests is non-nil and non-zero. A zero-value TestMetrics is treated
// the same as nil — the verb path doesn't write meaningless
// `pass=0 fail=0 skip=0`.
func appendTestsTrailer(trailers []gitops.Trailer, tests *gitops.TestMetrics) []gitops.Trailer {
	if tests == nil {
		return trailers
	}
	formatted := gitops.FormatTestMetrics(*tests)
	if formatted == "" {
		return trailers
	}
	return append(trailers, gitops.Trailer{Key: gitops.TrailerTests, Value: formatted})
}

// standardTrailers builds the verb/entity/actor trailer triple for
// non-transition verbs (add, rename). Used by AC verbs that don't
// participate in the aiwf-to: / aiwf-force: schema.
func standardTrailers(verbName, id, actor string) []gitops.Trailer {
	return []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: verbName},
		{Key: gitops.TrailerEntity, Value: id},
		{Key: gitops.TrailerActor, Value: actor},
	}
}

// appendACHeading scaffolds a `### AC-<N> — <title>` heading at the
// end of the body so the new AC has prose anchor. The em-dash form
// is the canonical scaffold; the validator's coherence check accepts
// hyphen and colon variants when humans hand-edit.
//
// Adds a leading blank line if the body doesn't already end with one,
// keeping markdown rendering tidy.
func appendACHeading(body []byte, acID, title string) []byte {
	suffix := fmt.Sprintf("\n### %s — %s\n\n", acID, title)
	trimmed := strings.TrimRight(string(body), "\n")
	return []byte(trimmed + "\n" + suffix)
}

// acHeadingLinePattern matches a `### AC-N <separator> <title>` line
// for in-place rewriting on rename. The separator-and-title portion
// is optional so id-only headings get rewritten too. Anchored to line
// start with `(?m)` so a regex over multi-line input matches each
// candidate line.
var acHeadingLinePattern = regexp.MustCompile(`(?m)^### AC-(\d+)(?:\s*[—\-:]\s*[^\n]*)?$`)

// rewriteACHeading scans body for a `### AC-<N>` heading matching
// acID and rewrites it in place to the canonical em-dash form. When
// no matching heading is found, the body is returned unchanged — the
// `acs-body-coherence` warning will surface the missing heading on
// the next aiwf check, which is the user's signal to add one.
func rewriteACHeading(body []byte, acID, newTitle string) []byte {
	replacement := fmt.Sprintf("### %s — %s", acID, newTitle)
	return acHeadingLinePattern.ReplaceAllFunc(body, func(line []byte) []byte {
		// The regex matches any AC-N heading; only rewrite the one
		// whose AC id equals acID.
		s := string(line)
		idx := strings.Index(s, "AC-")
		if idx < 0 {
			return line
		}
		rest := s[idx:]
		end := len(rest)
		for j, r := range rest {
			if r != 'A' && r != 'C' && r != '-' && (r < '0' || r > '9') {
				end = j
				break
			}
		}
		if rest[:end] != acID {
			return line
		}
		return []byte(replacement)
	})
}
