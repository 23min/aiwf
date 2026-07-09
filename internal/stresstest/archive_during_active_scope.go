package stresstest

import (
	"fmt"

	"github.com/23min/aiwf/internal/check"
)

// archive_during_active_scope.go — M-0243/AC-3: ArchiveDuringActiveScopeScenario
// reproduces G-0212 item 3 end-to-end: a parent epic is promoted
// straight to `done` (`aiwf promote <epic> done` carries no
// non-terminal-children guard, unlike `aiwf cancel`'s own
// epic-cancel-non-terminal-children refusal — confirmed empirically)
// while its child milestone remains non-terminal with a genuinely
// active, never-closed authorize scope. `aiwf archive --apply` sweeps
// the parent — and the milestone, which "rides with" its parent per
// ADR-0004 — into archive/ regardless. This is the only way to
// construct "a child's scope is still active" at archive time at all:
// a terminal child would have already auto-ended its own scope (per
// the scope FSM's documented auto-end-on-terminal-promote rule), so
// the child MUST still be non-terminal for its scope to still be
// active when the parent archives.
//
// The scenario's oracle, empirically confirmed (not assumed): scope
// resolution survives the sweep cleanly — `aiwf show`'s scopes array
// and `aiwf authorize --pause` both keep working correctly on the
// archived child. G-0212's literal fear (scope becomes unresolvable)
// does not hold. What the investigation surfaces instead is a
// different, real gap: `aiwf archive` allowed sweeping a non-terminal
// child into archive/ at all, producing a tree `aiwf check` then
// flags at error severity (archived-entity-not-terminal) — caught
// after the fact, not prevented up front the way `cancel` prevents
// the analogous case.

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

	if promEnv, startErr := runAiwfJSON(s.aiwfBin, dir, "promote", s.milestoneID, "in_progress"); startErr != nil { //coverage:ignore defensive: see the parent epic add above
		return fmt.Errorf("starting the child milestone: %w", startErr)
	} else if promEnv.Status != "ok" { //coverage:ignore defensive: a freshly-added milestone's draft->in_progress transition is always legal; see the activate-epic rationale above
		return fmt.Errorf("starting the child milestone: aiwf did not report ok (status=%s, error=%+v)", promEnv.Status, promEnv.Error)
	}

	epicBranch := "epic/" + s.epicID + "-parentep"
	if checkoutErr := runGit(dir, "checkout", "-q", "-b", epicBranch); checkoutErr != nil { //coverage:ignore defensive: creating a fresh branch off a repo this scenario itself just built has no realistic failure mode
		return fmt.Errorf("cutting the epic branch: %w", checkoutErr)
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

// Run promotes the parent epic straight to done, captures the child's
// pre-sweep scope state, archives, then captures the post-sweep scope
// state, attempts to pause the scope, and checks whether aiwf check
// flags the non-terminal child's presence under archive/.
func (s *ArchiveDuringActiveScopeScenario) Run(dir string) error {
	preEnv, err := runAiwfJSON(s.aiwfBin, dir, "show", s.milestoneID)
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("reading the pre-sweep scope state: %w", err)
	}
	preScopeState := scopeState(preEnv)

	if promEnv, doneErr := runAiwfJSON(s.aiwfBin, dir, "promote", s.epicID, "done"); doneErr != nil { //coverage:ignore defensive: see the pre-sweep show above
		return fmt.Errorf("promoting the parent epic to done: %w", doneErr)
	} else if promEnv.Status != "ok" { //coverage:ignore defensive: active->done is always legal for this scenario's own epic, which carries no non-terminal-children guard (confirmed empirically — the whole point of this scenario); see the activate-epic rationale in Setup above
		return fmt.Errorf("promoting the parent epic to done: aiwf did not report ok (status=%s, error=%+v)", promEnv.Status, promEnv.Error)
	}

	if archEnv, archiveErr := runAiwfJSON(s.aiwfBin, dir, "archive", "--apply"); archiveErr != nil { //coverage:ignore defensive: see the pre-sweep show above
		return fmt.Errorf("archiving the parent epic: %w", archiveErr)
	} else if archEnv.Status != "ok" { //coverage:ignore defensive: a just-done epic with nothing else pending is always sweepable; see the activate-epic rationale in Setup above
		return fmt.Errorf("archiving the parent epic: aiwf did not report ok (status=%s, error=%+v)", archEnv.Status, archEnv.Error)
	}

	postEnv, err := runAiwfJSON(s.aiwfBin, dir, "show", s.milestoneID)
	if err != nil { //coverage:ignore defensive: see the pre-sweep show above
		return fmt.Errorf("reading the post-sweep scope state: %w", err)
	}
	postScopeState := scopeState(postEnv)

	pauseEnv, err := runAiwfJSON(s.aiwfBin, dir, "authorize", s.milestoneID, "--pause", "post-archive pause probe")
	if err != nil { //coverage:ignore defensive: see the pre-sweep show above
		return fmt.Errorf("pausing the archived child's scope: %w", err)
	}

	checkEnv, err := runAiwfJSON(s.aiwfBin, dir, "check")
	if err != nil { //coverage:ignore defensive: see the pre-sweep show above
		return fmt.Errorf("running aiwf check after the sweep: %w", err)
	}
	archivedNotTerminalFound := hasFindingForEntity(checkEnv.Findings, check.CodeArchivedEntityNotTerminal, s.milestoneID)

	s.violations = classifyArchiveDuringActiveScope(preScopeState, postScopeState, pauseEnv.Status, archivedNotTerminalFound)
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

// hasFindingForEntity reports whether findings contains one with the
// given code targeting the given entity id.
func hasFindingForEntity(findings []verbEnvelopeFinding, code, entityID string) bool {
	for _, f := range findings {
		if f.Code == code && f.EntityID == entityID {
			return true
		}
	}
	return false
}

// classifyArchiveDuringActiveScope judges one archive-during-active-
// scope attempt: the premise (a genuinely active scope before the
// sweep) must hold, the scope's state must survive the sweep
// unchanged, the pause attempt must succeed, and aiwf check must flag
// the non-terminal child riding along into archive/ — anything else
// is either a broken premise or a silent, unflagged corruption.
func classifyArchiveDuringActiveScope(preScopeState, postScopeState, pauseStatus string, archivedNotTerminalFound bool) []Violation {
	var violations []Violation
	if preScopeState != "active" { //enums:ignore this compares a scope FSM state (internal/scope.StateActive) — a different closed set that happens to share entity.StatusActive's underlying string, not an entity status comparison
		violations = append(violations, Violation{Message: fmt.Sprintf("the child's scope was not active before the sweep (state=%q) — the scenario's premise did not hold", preScopeState)})
	}
	if postScopeState != "active" { //enums:ignore see the preScopeState comparison above — same scope-state rationale
		violations = append(violations, Violation{Message: fmt.Sprintf("the child's scope state changed after the sweep (state=%q, want active)", postScopeState)})
	}
	if pauseStatus != "ok" {
		violations = append(violations, Violation{Message: fmt.Sprintf("aiwf authorize --pause could not act on the still-open scope after the sweep (status=%s)", pauseStatus)})
	}
	if !archivedNotTerminalFound {
		violations = append(violations, Violation{Message: "aiwf check did not flag the non-terminal child that rode along into archive/ — a silent, unflagged structural anomaly"})
	}
	return violations
}
