package stresstest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/gitops"
)

// force_override_durability.go — M-0243/AC-4: ForceOverrideDurabilityScenario
// covers G-0212 items 5 and 6 in one scenario, since both probe the
// same underlying fact: `aiwf check`'s force/ack machinery trusts
// whatever's reachable from HEAD, with no mechanism binding an
// override to the specific context it was meant for.
//
// Item 5 (force-push unreaches an acknowledge-illegal target):
// reproduced via a rebase that drops JUST the acknowledgment commit
// while keeping the originally-flagged commit reachable — the same
// reachability effect a force-push produces, without the extra
// plumbing of an actual remote. Empirically confirmed: the
// illegal-transition finding reappears, exactly as if never
// acknowledged — a real audit-trail regression, since nothing else
// signals that an acknowledgment ever existed for it. Treated as a
// violation.
//
// Item 6 (cherry-pick of a force-amend commit onto a different
// branch): a force-promote's trailers (`aiwf-force:`, `aiwf-actor:
// human/...`) are preserved verbatim by `git cherry-pick`, so the
// kernel's trust-the-trailer model accepts the cherry-picked commit
// on the new branch exactly as it accepted the original — no audit
// trail ties the two together. Empirically confirmed. This is the
// CURRENT, by-design trust model (a force override is validated by
// its trailers alone, wherever they appear) rather than a narrower
// bug a mechanical check could catch without also breaking legitimate
// cherry-picks (e.g. this repo's own milestone-to-epic merges rely on
// the same trailer-preservation). So this half of the scenario reports
// only premise breaks, never the carryover fact itself.

// ForceOverrideDurabilityScenario implements Scenario.
type ForceOverrideDurabilityScenario struct {
	aiwfBin     string
	ackEpicID   string
	ackEpicPath string
	milestoneID string
	violations  []Violation
}

// NewForceOverrideDurabilityScenario builds a scenario driving both
// the ack-revocation-by-rebase and cherry-picked-force-carryover
// sequences against one disposable repo.
func NewForceOverrideDurabilityScenario(aiwfBin string) *ForceOverrideDurabilityScenario {
	return &ForceOverrideDurabilityScenario{aiwfBin: aiwfBin}
}

// Setup creates two independent epics: one promoted to done (the
// item-5 target for a manual illegal-transition edit), and one
// carrying a draft milestone (the item-6 target for a force-promote),
// with two sibling branches cut off the point where that milestone is
// still draft.
func (s *ForceOverrideDurabilityScenario) Setup(dir string) error {
	if err := gitInitAndConfig(dir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}

	ackEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic", "--title", "acktarget", "--body", "epic for the ack-revocation-by-rebase scenario")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("seeding the ack-target epic: %w", err)
	}
	if ackEnv.Status != "ok" {
		return fmt.Errorf("seeding the ack-target epic: aiwf did not report ok (status=%s, error=%+v)", ackEnv.Status, ackEnv.Error)
	}
	ackEpicID := ackEnv.Metadata.EntityID

	if promEnv, activateErr := runAiwfJSON(s.aiwfBin, dir, "promote", ackEpicID, "active"); activateErr != nil { //coverage:ignore defensive: see the ack-target epic add above
		return fmt.Errorf("activating the ack-target epic: %w", activateErr)
	} else if promEnv.Status != "ok" { //coverage:ignore defensive: a freshly-added epic's proposed->active transition is always legal
		return fmt.Errorf("activating the ack-target epic: aiwf did not report ok (status=%s, error=%+v)", promEnv.Status, promEnv.Error)
	}
	if promEnv, doneErr := runAiwfJSON(s.aiwfBin, dir, "promote", ackEpicID, "done"); doneErr != nil { //coverage:ignore defensive: see the ack-target epic add above
		return fmt.Errorf("finishing the ack-target epic: %w", doneErr)
	} else if promEnv.Status != "ok" { //coverage:ignore defensive: an active epic's own active->done transition is always legal
		return fmt.Errorf("finishing the ack-target epic: aiwf did not report ok (status=%s, error=%+v)", promEnv.Status, promEnv.Error)
	}
	s.ackEpicPath = filepath.Join(dir, "work", "epics", ackEpicID+"-acktarget", "epic.md")
	s.ackEpicID = ackEpicID

	forceEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic", "--title", "forceparent", "--body", "epic for the force-override cherry-pick scenario")
	if err != nil { //coverage:ignore defensive: see the ack-target epic add above
		return fmt.Errorf("seeding the force-parent epic: %w", err)
	}
	if forceEnv.Status != "ok" { //coverage:ignore defensive: a fresh epic add under a disposable repo has no realistic collision here; the general collision-refusal mechanism is exercised elsewhere in this package (e.g. TestArchiveDuringActiveScopeScenario_RealBinary_SetupSurfacesASeedingRefusal)
		return fmt.Errorf("seeding the force-parent epic: aiwf did not report ok (status=%s, error=%+v)", forceEnv.Status, forceEnv.Error)
	}
	forceEpicID := forceEnv.Metadata.EntityID

	if promEnv, activateErr := runAiwfJSON(s.aiwfBin, dir, "promote", forceEpicID, "active"); activateErr != nil { //coverage:ignore defensive: see the ack-target epic add above
		return fmt.Errorf("activating the force-parent epic: %w", activateErr)
	} else if promEnv.Status != "ok" { //coverage:ignore defensive: a freshly-added epic's proposed->active transition is always legal
		return fmt.Errorf("activating the force-parent epic: aiwf did not report ok (status=%s, error=%+v)", promEnv.Status, promEnv.Error)
	}

	msEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "milestone", "--epic", forceEpicID, "--tdd", "none", "--title", "forcetarget", "--body", "milestone for the force-override cherry-pick scenario")
	if err != nil { //coverage:ignore defensive: see the ack-target epic add above
		return fmt.Errorf("seeding the force-target milestone: %w", err)
	}
	if msEnv.Status != "ok" { //coverage:ignore defensive: a fresh milestone add under a just-activated epic has no realistic refusal mode here
		return fmt.Errorf("seeding the force-target milestone: aiwf did not report ok (status=%s, error=%+v)", msEnv.Status, msEnv.Error)
	}
	s.milestoneID = msEnv.Metadata.EntityID

	if err := runGit(dir, "branch", "branch-a"); err != nil { //coverage:ignore defensive: creating a branch ref in a repo this scenario itself just built has no realistic failure mode
		return fmt.Errorf("cutting branch-a: %w", err)
	}
	if err := runGit(dir, "branch", "branch-b"); err != nil { //coverage:ignore defensive: see branch-a above
		return fmt.Errorf("cutting branch-b: %w", err)
	}
	return nil
}

// Run drives item 5 (ack-revocation-by-rebase) then item 6
// (cherry-picked-force-carryover) against dir, and classifies the
// combined outcome.
func (s *ForceOverrideDurabilityScenario) Run(dir string) error {
	preAckFlagged, postAckFlagged, postRebaseFlagged, err := s.runAckRevocationByRebase(dir)
	if err != nil { //coverage:ignore defensive: runAckRevocationByRebase's own internal error branches each carry their own reachability rationale; this is a trivial propagation wrapper with no logic of its own
		return err
	}
	forceAccepted, cherryPickClean, trailersPreserved, err := s.runCherryPickCarryover(dir)
	if err != nil { //coverage:ignore defensive: see the runAckRevocationByRebase propagation above
		return err
	}
	s.violations = classifyForceOverrideDurability(preAckFlagged, postAckFlagged, postRebaseFlagged, forceAccepted, cherryPickClean, trailersPreserved)
	return nil
}

// runAckRevocationByRebase manually edits the ack-target epic's status
// backwards (done -> active — illegal, since done is terminal),
// confirms aiwf check flags it, acknowledges it, confirms the
// acknowledgment suppresses the finding, then rebases to drop JUST the
// acknowledgment commit (keeping the originally-flagged commit and a
// trailing innocuous commit reachable) — the same reachability effect
// a force-push produces, without an actual remote — and confirms
// whether the finding is revived.
func (s *ForceOverrideDurabilityScenario) runAckRevocationByRebase(dir string) (preAckFlagged, postAckFlagged, postRebaseFlagged bool, err error) {
	raw, readErr := os.ReadFile(s.ackEpicPath)
	if readErr != nil { //coverage:ignore defensive: reading the epic file this scenario's own Setup just wrote has no realistic failure mode
		return false, false, false, fmt.Errorf("reading the ack-target epic file: %w", readErr)
	}
	edited := strings.Replace(string(raw), "status: done", "status: active", 1)
	if edited == string(raw) { //coverage:ignore defensive: Setup always leaves the ack-target epic at status done immediately before this call
		return false, false, false, fmt.Errorf("the ack-target epic file did not contain the expected %q line to edit", "status: done")
	}
	if writeErr := os.WriteFile(s.ackEpicPath, []byte(edited), 0o644); writeErr != nil { //coverage:ignore defensive: overwriting a file this scenario's own Setup just wrote has no realistic failure mode
		return false, false, false, fmt.Errorf("writing the manual illegal edit: %w", writeErr)
	}
	if commitErr := runGit(dir, "commit", "-q", "-am", "manual illegal status revert"); commitErr != nil { //coverage:ignore defensive: committing a real diff to a file this scenario itself just edited has no realistic failure mode
		return false, false, false, fmt.Errorf("committing the manual illegal edit: %w", commitErr)
	}
	xSHA, shaErr := headSHA(dir)
	if shaErr != nil { //coverage:ignore defensive: see headSHA's own rationale
		return false, false, false, fmt.Errorf("reading the illegal-edit commit SHA: %w", shaErr)
	}

	preCheckEnv, checkErr := runAiwfJSON(s.aiwfBin, dir, "check")
	if checkErr != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return false, false, false, fmt.Errorf("running aiwf check before the acknowledgment: %w", checkErr)
	}
	preAckFlagged = hasFindingSubcodeForEntity(preCheckEnv.Findings, check.CodeFSMHistoryConsistent, "illegal-transition", s.ackEpicID)

	ackEnv, ackErr := runAiwfJSON(s.aiwfBin, dir, "acknowledge", "illegal", xSHA, "--for-entity", s.ackEpicID, "--reason", "testing ack durability against a rebase that drops just the ack")
	if ackErr != nil { //coverage:ignore defensive: see the check call above
		return false, false, false, fmt.Errorf("acknowledging the illegal edit: %w", ackErr)
	}
	if ackEnv.Status != "ok" { //coverage:ignore defensive: the target SHA and --for-entity id are always freshly derived from this scenario's own just-created commit and epic, with no realistic refusal mode here; the general collision-refusal mechanism is exercised at its source (TestForceOverrideDurabilityScenario_RealBinary_SetupSurfacesASeedingRefusal)
		return false, false, false, fmt.Errorf("acknowledging the illegal edit: aiwf did not report ok (status=%s, error=%+v)", ackEnv.Status, ackEnv.Error)
	}
	ySHA, shaErr := headSHA(dir)
	if shaErr != nil { //coverage:ignore defensive: see headSHA's own rationale
		return false, false, false, fmt.Errorf("reading the acknowledgment commit SHA: %w", shaErr)
	}

	postAckCheckEnv, checkErr := runAiwfJSON(s.aiwfBin, dir, "check")
	if checkErr != nil { //coverage:ignore defensive: see the check call above
		return false, false, false, fmt.Errorf("running aiwf check after the acknowledgment: %w", checkErr)
	}
	postAckFlagged = hasFindingSubcodeForEntity(postAckCheckEnv.Findings, check.CodeFSMHistoryConsistent, "illegal-transition", s.ackEpicID)

	scratchPath := filepath.Join(dir, "innocuous.txt")
	if writeErr := os.WriteFile(scratchPath, []byte("innocuous\n"), 0o644); writeErr != nil { //coverage:ignore defensive: writing a fresh scratch file under this scenario's own disposable repo has no realistic failure mode
		return false, false, false, fmt.Errorf("writing the trailing innocuous file: %w", writeErr)
	}
	if addErr := runGit(dir, "add", "-A"); addErr != nil { //coverage:ignore defensive: staging a fresh scratch file has no realistic failure mode
		return false, false, false, fmt.Errorf("staging the trailing innocuous file: %w", addErr)
	}
	if commitErr := runGit(dir, "commit", "-q", "-m", "innocuous follow-up commit"); commitErr != nil { //coverage:ignore defensive: committing a staged scratch file has no realistic failure mode
		return false, false, false, fmt.Errorf("committing the trailing innocuous file: %w", commitErr)
	}

	if rebaseErr := runGit(dir, "rebase", "-q", "--onto", xSHA, ySHA, "HEAD"); rebaseErr != nil { //coverage:ignore defensive: the range being dropped (the ack commit alone) and the range being replayed (the trailing scratch-file commit) touch disjoint paths, so this rebase is always a clean fast path with no realistic conflict
		return false, false, false, fmt.Errorf("rebasing to drop the acknowledgment commit: %w", rebaseErr)
	}

	postRebaseCheckEnv, checkErr := runAiwfJSON(s.aiwfBin, dir, "check")
	if checkErr != nil { //coverage:ignore defensive: see the check call above
		return false, false, false, fmt.Errorf("running aiwf check after the rebase: %w", checkErr)
	}
	postRebaseFlagged = hasFindingSubcodeForEntity(postRebaseCheckEnv.Findings, check.CodeFSMHistoryConsistent, "illegal-transition", s.ackEpicID)
	return preAckFlagged, postAckFlagged, postRebaseFlagged, nil
}

// runCherryPickCarryover force-promotes the force-target milestone on
// branch-a, confirms the force is accepted cleanly, cherry-picks that
// commit onto the sibling branch-b (forked from the same pre-force
// point), and confirms the cherry-pick applies cleanly with its
// aiwf-force/aiwf-actor trailers preserved verbatim.
func (s *ForceOverrideDurabilityScenario) runCherryPickCarryover(dir string) (forceAccepted, cherryPickClean, trailersPreserved bool, err error) {
	if err := runGit(dir, "checkout", "-q", "branch-a"); err != nil { //coverage:ignore defensive: checking out a branch this scenario's own Setup just cut has no realistic failure mode
		return false, false, false, fmt.Errorf("checking out branch-a: %w", err)
	}
	forceEnv, forceErr := runAiwfJSON(s.aiwfBin, dir, "promote", s.milestoneID, "done", "--force", "--reason", "legitimate override on branch-a")
	if forceErr != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return false, false, false, fmt.Errorf("force-promoting the milestone on branch-a: %w", forceErr)
	}
	forceAccepted = forceEnv.Status == "ok"
	if !forceAccepted { //coverage:ignore defensive: a fresh draft milestone's own force-promote to done, immediately after Setup created it, has no realistic refusal mode here; see the seeding-refusal test referenced above
		return false, false, false, nil
	}
	fSHA, shaErr := headSHA(dir)
	if shaErr != nil { //coverage:ignore defensive: see headSHA's own rationale
		return false, false, false, fmt.Errorf("reading the force-promote commit SHA: %w", shaErr)
	}
	wantForce, trailerErr := commitTrailerValue(dir, fSHA, gitops.TrailerForce)
	if trailerErr != nil { //coverage:ignore defensive: reading a trailer off a commit this scenario itself just created has no realistic failure mode
		return false, false, false, fmt.Errorf("reading the original aiwf-force trailer: %w", trailerErr)
	}
	wantActor, trailerErr := commitTrailerValue(dir, fSHA, gitops.TrailerActor)
	if trailerErr != nil { //coverage:ignore defensive: see the aiwf-force read above
		return false, false, false, fmt.Errorf("reading the original aiwf-actor trailer: %w", trailerErr)
	}

	if checkoutErr := runGit(dir, "checkout", "-q", "branch-b"); checkoutErr != nil { //coverage:ignore defensive: checking out a branch this scenario's own Setup just cut has no realistic failure mode
		return forceAccepted, false, false, fmt.Errorf("checking out branch-b: %w", checkoutErr)
	}
	cherryPickClean = runGit(dir, "cherry-pick", fSHA) == nil
	if !cherryPickClean { //coverage:ignore defensive: branch-a and branch-b fork from the identical pre-force point and diverge only by this one commit, so the cherry-pick's target file is always at the exact base state the commit expects — no realistic conflict here
		return forceAccepted, false, false, nil
	}

	gotForce, trailerErr := commitTrailerValue(dir, "HEAD", gitops.TrailerForce)
	if trailerErr != nil { //coverage:ignore defensive: reading a trailer off the cherry-picked HEAD this scenario itself just produced has no realistic failure mode
		return forceAccepted, cherryPickClean, false, fmt.Errorf("reading the cherry-picked aiwf-force trailer: %w", trailerErr)
	}
	gotActor, trailerErr := commitTrailerValue(dir, "HEAD", gitops.TrailerActor)
	if trailerErr != nil { //coverage:ignore defensive: see the aiwf-force read above
		return forceAccepted, cherryPickClean, false, fmt.Errorf("reading the cherry-picked aiwf-actor trailer: %w", trailerErr)
	}
	trailersPreserved = gotForce == wantForce && gotActor == wantActor
	return forceAccepted, cherryPickClean, trailersPreserved, nil
}

// Verify returns every violation Run collected.
func (s *ForceOverrideDurabilityScenario) Verify(_ string) []Violation {
	return s.violations
}

// headSHA returns dir's current HEAD commit SHA.
func headSHA(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil { //coverage:ignore defensive: reading HEAD in a repo this scenario itself just committed to has no realistic failure mode
		return "", fmt.Errorf("reading HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// commitTrailerValue returns the value of trailer on commit ref (a SHA
// or "HEAD"), or "" if the commit carries no such trailer.
func commitTrailerValue(dir, ref, trailer string) (string, error) {
	cmd := exec.Command("git", "log", "-1", ref, "--format=%(trailers:key="+trailer+",valueonly,unfold=true)")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil { //coverage:ignore defensive: reading a trailer off a commit this scenario itself just produced has no realistic failure mode
		return "", fmt.Errorf("reading trailer %s off %s: %w", trailer, ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// hasFindingSubcodeForEntity reports whether findings contains one
// with the given code, subcode, and entity id.
func hasFindingSubcodeForEntity(findings []verbEnvelopeFinding, code, subcode, entityID string) bool {
	for _, f := range findings {
		if f.Code == code && f.Subcode == subcode && f.EntityID == entityID {
			return true
		}
	}
	return false
}

// classifyForceOverrideDurability judges one force-override-durability
// attempt. Item 5: the premise (illegal-transition flagged before the
// ack, suppressed after it) must hold, and a revival after the rebase
// that drops just the ack commit is itself a confirmed violation — a
// real audit-trail regression. Item 6: only premise breaks (the force
// wasn't accepted, the cherry-pick conflicted, or its trailers weren't
// preserved) count; the cherry-picked commit going unflagged on its
// new branch is the current, by-design trust model, not a violation.
func classifyForceOverrideDurability(preAckFlagged, postAckFlagged, postRebaseFlagged, forceAccepted, cherryPickClean, trailersPreserved bool) []Violation {
	var violations []Violation
	if !preAckFlagged {
		violations = append(violations, Violation{Message: "the manual illegal-transition edit was never flagged before acknowledging it — the scenario's premise did not hold"})
	}
	if postAckFlagged {
		violations = append(violations, Violation{Message: "acknowledge illegal did not suppress the illegal-transition finding"})
	}
	if postRebaseFlagged {
		violations = append(violations, Violation{Message: "confirmed: a rebase dropping just the acknowledgment commit (keeping the originally-flagged commit reachable) silently revives the illegal-transition finding — the ack's protection is not durable against history rewrites"})
	}
	if !forceAccepted {
		violations = append(violations, Violation{Message: "the original force-promote on branch-a was not accepted"})
	}
	if !cherryPickClean {
		violations = append(violations, Violation{Message: "cherry-picking the force-promote commit onto branch-b produced an unexpected conflict"})
	}
	if !trailersPreserved {
		violations = append(violations, Violation{Message: "the cherry-picked commit's aiwf-force/aiwf-actor trailers did not match the original — the scenario's premise about cherry-pick's trailer-preservation did not hold"})
	}
	return violations
}
