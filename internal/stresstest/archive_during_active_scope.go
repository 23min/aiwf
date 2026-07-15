package stresstest

import (
	"fmt"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/verb"
)

// archive_during_active_scope.go — M-0243/AC-3, updated by M-0244/AC-2's
// triage sweep: ArchiveDuringActiveScopeScenario originally reproduced
// G-0212 item 3 by promoting a parent epic straight to `done` while its
// child milestone remained non-terminal with a genuinely active,
// never-closed authorize scope, then sweeping both into archive/ —
// surfacing G-0393 (`aiwf archive` could sweep a non-terminal milestone
// alongside its terminal parent) as a real, distinct gap. G-0393 is now
// fixed: `aiwf promote <epic> done` (and `cancelled`) refuses with
// EpicPromoteNonTerminalChildrenError whenever a child milestone is
// still non-terminal, mirroring `aiwf cancel`'s own
// epic-cancel-non-terminal-children guard. This scenario now proves
// that fix holds under a real authorize scope: the promote is refused,
// the epic never reaches done, and the child's active scope is
// unaffected — so the invalid archived-non-terminal-child state this
// scenario used to be able to construct is no longer reachable through
// the normal verb surface at all.
//
// G-0212 item 3's own literal fear — scope resolution becomes
// unresolvable once its holder crosses the archive boundary — was
// already confirmed unfounded before G-0393 closed (see M-0243's
// milestone spec); that finding stands independent of this update.

// archiveDuringActiveScopeExpectedWarnings is the baseline of finding
// codes this scenario's post-attempt check is expected to carry
// (M-0257/AC-1), beyond the refused-promote assertion
// classifyArchiveDuringActiveScope already pins directly:
//
//   - epic-active-no-drafted-milestones: the parent epic reaches
//     "active" and stays there for the rest of the scenario, while its
//     one child milestone is walked straight to "in_progress" and
//     never replaced by a fresh draft one.
//   - provenance-untrailered-scope-undefined: this scenario's
//     disposable repo never configures an upstream remote.
//
// Any OTHER finding — any error-severity finding, or a warning with a
// code not in this set — is a real violation.
var archiveDuringActiveScopeExpectedWarnings = map[string]bool{
	check.CodeEpicActiveNoDraftedMilestones:     true,
	check.CodeProvenanceUntrailedScopeUndefined: true,
}

// ArchiveDuringActiveScopeScenario implements Scenario.
type ArchiveDuringActiveScopeScenario struct {
	aiwfBin     string
	epicID      string
	milestoneID string
	violations  []Violation
}

// NewArchiveDuringActiveScopeScenario builds a scenario driving one
// epic/milestone pair through an authorize-then-archive sequence.
func NewArchiveDuringActiveScopeScenario(aiwfBin string) *ArchiveDuringActiveScopeScenario {
	return &ArchiveDuringActiveScopeScenario{aiwfBin: aiwfBin}
}

// Setup creates an active epic with an in_progress milestone, opens
// an authorize scope on the milestone from an epic-shaped branch (the
// rung-pair preflight requires it), then merges that branch back into
// the base branch so the authorize commit — which touches no file,
// carrying only trailers — stays reachable from HEAD for every
// subsequent verb in this scenario. Skipping this merge would leave
// the authorize commit stranded on an unmerged sibling branch, making
// every trailer-grep-based lookup (aiwf show/history/authorize
// --pause) blind to it for a reason that has nothing to do with
// archiving — a confound ruled out empirically before writing this
// scenario.
func (s *ArchiveDuringActiveScopeScenario) Setup(dir string) error {
	if err := gitInitAndConfig(dir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}
	baseBranch, err := currentBranch(dir)
	if err != nil { //coverage:ignore defensive: reading the branch this scenario itself just initialized has no realistic failure mode
		return fmt.Errorf("reading the base branch: %w", err)
	}

	epicEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic", "--title", "parentep", "--body", "parent epic for the archive-during-active-scope scenario")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("seeding the parent epic: %w", err)
	}
	if epicEnv.Status != "ok" {
		return fmt.Errorf("seeding the parent epic: aiwf did not report ok (status=%s, error=%+v)", epicEnv.Status, epicEnv.Error)
	}
	s.epicID = epicEnv.Metadata.EntityID

	if promEnv, activateErr := runAiwfJSON(s.aiwfBin, dir, "promote", s.epicID, "active"); activateErr != nil { //coverage:ignore defensive: see the parent epic add above
		return fmt.Errorf("activating the parent epic: %w", activateErr)
	} else if promEnv.Status != "ok" { //coverage:ignore defensive: a freshly-added epic's proposed->active transition is always legal; the generic "verb refuses with a non-ok status" mechanism is already exercised by Setup's own epic-add collision test (TestArchiveDuringActiveScopeScenario_RealBinary_SetupSurfacesASeedingRefusal) — this scenario doesn't need to re-prove it at every subsequent step
		return fmt.Errorf("activating the parent epic: aiwf did not report ok (status=%s, error=%+v)", promEnv.Status, promEnv.Error)
	}

	msEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "milestone", "--epic", s.epicID, "--tdd", "none", "--title", "childms", "--body", "child milestone for the archive-during-active-scope scenario")
	if err != nil { //coverage:ignore defensive: see the parent epic add above
		return fmt.Errorf("seeding the child milestone: %w", err)
	}
	if msEnv.Status != "ok" { //coverage:ignore defensive: a fresh milestone add under a just-activated epic has no realistic refusal mode in this scenario's own sequence; the generic collision-refusal mechanism is already exercised by the epic-add test referenced above
		return fmt.Errorf("seeding the child milestone: aiwf did not report ok (status=%s, error=%+v)", msEnv.Status, msEnv.Error)
	}
	s.milestoneID = msEnv.Metadata.EntityID

	// G-0269's activating-promote branch guard requires the epic's
	// ritual branch checked out before the milestone in_progress
	// promote below — cut it first, ahead of the authorize call that
	// also needs it for the rung-pair preflight.
	epicBranch := "epic/" + s.epicID + "-parentep"
	if checkoutErr := runGit(dir, "checkout", "-q", "-b", epicBranch); checkoutErr != nil { //coverage:ignore defensive: creating a fresh branch off a repo this scenario itself just built has no realistic failure mode
		return fmt.Errorf("cutting the epic branch: %w", checkoutErr)
	}

	if promEnv, startErr := runAiwfJSON(s.aiwfBin, dir, "promote", s.milestoneID, "in_progress"); startErr != nil { //coverage:ignore defensive: see the parent epic add above
		return fmt.Errorf("starting the child milestone: %w", startErr)
	} else if promEnv.Status != "ok" { //coverage:ignore defensive: a freshly-added milestone's draft->in_progress transition is always legal; see the activate-epic rationale above
		return fmt.Errorf("starting the child milestone: aiwf did not report ok (status=%s, error=%+v)", promEnv.Status, promEnv.Error)
	}

	authEnv, err := runAiwfJSON(s.aiwfBin, dir, "authorize", s.milestoneID, "--to", "ai/claude", "--branch", "milestone/"+s.milestoneID+"-childms", "--reason", "archive-during-active-scope stress scope")
	if err != nil { //coverage:ignore defensive: see the parent epic add above
		return fmt.Errorf("authorizing the child milestone: %w", err)
	}
	if authEnv.Status != "ok" { //coverage:ignore defensive: authorizing a freshly-started, never-before-scoped milestone from a just-cut ritual-shape branch is always legal in this scenario's own sequence; see the activate-epic rationale above
		return fmt.Errorf("authorizing the child milestone: aiwf did not report ok (status=%s, error=%+v)", authEnv.Status, authEnv.Error)
	}

	if err := runGit(dir, "checkout", "-q", baseBranch); err != nil { //coverage:ignore defensive: switching back to a branch this scenario itself just left has no realistic failure mode
		return fmt.Errorf("returning to the base branch: %w", err)
	}
	if err := runGit(dir, "merge", "-q", "--no-edit", epicBranch); err != nil { //coverage:ignore defensive: the epic branch's only commit is the trailer-only authorize commit, which touches no file — this merge is always a clean fast-forward with no realistic conflict
		return fmt.Errorf("merging the epic branch back: %w", err)
	}
	return nil
}

// Run captures the child's pre-attempt scope state, attempts to
// promote the parent epic straight to done, then confirms G-0393's
// guard refused it: the epic's own status is unaffected and the
// child's scope is unaffected — there is no sweep to run anymore,
// because the invalid state this scenario used to construct at this
// exact step can no longer be reached.
func (s *ArchiveDuringActiveScopeScenario) Run(dir string) error {
	preEnv, err := runAiwfJSON(s.aiwfBin, dir, "show", s.milestoneID)
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("reading the pre-attempt scope state: %w", err)
	}
	preScopeState := scopeState(preEnv)

	promEnv, err := runAiwfJSON(s.aiwfBin, dir, "promote", s.epicID, "done")
	if err != nil { //coverage:ignore defensive: see the pre-attempt show above
		return fmt.Errorf("promoting the parent epic to done: %w", err)
	}

	epicEnv, err := runAiwfJSON(s.aiwfBin, dir, "show", s.epicID)
	if err != nil { //coverage:ignore defensive: see the pre-attempt show above
		return fmt.Errorf("reading the epic's post-attempt status: %w", err)
	}

	postEnv, err := runAiwfJSON(s.aiwfBin, dir, "show", s.milestoneID)
	if err != nil { //coverage:ignore defensive: see the pre-attempt show above
		return fmt.Errorf("reading the post-attempt scope state: %w", err)
	}
	postScopeState := scopeState(postEnv)

	s.violations = classifyArchiveDuringActiveScope(preScopeState, promEnv.Status, errorCode(promEnv), epicEnv.Result.Status, postScopeState)

	// M-0257/AC-1: alongside the refused-promote assertion above,
	// confirm the resulting tree stays check-clean beyond baseline
	// noise — this scenario never ran `aiwf check` at all before.
	checkEnv, err := runAiwfJSON(s.aiwfBin, dir, "check")
	if err != nil { //coverage:ignore defensive: see the pre-attempt show above
		return fmt.Errorf("running aiwf check after the refused promote: %w", err)
	}
	s.violations = append(s.violations, classifyAgainstBaseline(checkEnv.Findings, archiveDuringActiveScopeExpectedWarnings)...)
	return nil
}

// Verify returns every violation Run collected.
func (s *ArchiveDuringActiveScopeScenario) Verify(_ string) []Violation {
	return s.violations
}

// scopeState returns the first scope's state from env's result, or ""
// if env carries no scopes at all.
func scopeState(env verbEnvelope) string {
	if len(env.Result.Scopes) == 0 {
		return ""
	}
	return env.Result.Scopes[0].State
}

// errorCode returns env's error code, or "" if env carries no error at
// all (env.Error is nil on a status:"ok" envelope).
func errorCode(env verbEnvelope) string {
	if env.Error == nil {
		return ""
	}
	return env.Error.Code
}

// classifyArchiveDuringActiveScope judges one promote-while-child-active
// attempt: the premise (a genuinely active scope beforehand) must
// hold, the promote must be refused (not silently allowed through),
// the refusal must carry G-0393's own structured code (not some other,
// coincidental refusal), and the epic's status plus the child's scope
// must both be left exactly as they were — anything else is either a
// broken premise or a regression of the fix this scenario pins.
func classifyArchiveDuringActiveScope(preScopeState, promoteStatus, promoteErrorCode, epicStatusAfter, postScopeState string) []Violation {
	var violations []Violation
	if preScopeState != "active" { //enums:ignore this compares a scope FSM state (internal/scope.StateActive) — a different closed set that happens to share entity.StatusActive's underlying string, not an entity status comparison
		violations = append(violations, Violation{Message: fmt.Sprintf("the child's scope was not active before the attempt (state=%q) — the scenario's premise did not hold", preScopeState)})
	}
	if promoteStatus == "ok" { //enums:ignore this compares a JSON envelope's status field, a different closed set from entity.Status
		violations = append(violations, Violation{Message: "promoting the epic to done while a child milestone was non-terminal unexpectedly succeeded — G-0393's guard did not fire"})
	}
	if promoteErrorCode != verb.CodeEpicPromoteNonTerminalChildren.ID {
		violations = append(violations, Violation{Message: fmt.Sprintf("the promote refusal carried error code %q, want %q — refused for the wrong reason", promoteErrorCode, verb.CodeEpicPromoteNonTerminalChildren.ID)})
	}
	if epicStatusAfter != "active" { //enums:ignore this compares against the epic's own entity.Status; kept string-typed to match this file's other envelope-derived comparisons
		violations = append(violations, Violation{Message: fmt.Sprintf("the epic's status changed to %q despite the promote being refused", epicStatusAfter)})
	}
	if postScopeState != "active" { //enums:ignore see the preScopeState comparison above — same scope-state rationale
		violations = append(violations, Violation{Message: fmt.Sprintf("the child's scope state changed after the refused attempt (state=%q, want active)", postScopeState)})
	}
	return violations
}
