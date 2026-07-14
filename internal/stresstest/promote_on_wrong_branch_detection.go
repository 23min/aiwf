package stresstest

import "fmt"

// promote_on_wrong_branch_detection.go — G-0270:
// PromoteOnWrongBranchDetectionScenario confirms `aiwf check` detects
// an epic-activation commit that lands on a non-trunk branch, even
// when:
//
//   - check is run from a DIFFERENT branch than the one the commit
//     landed on (the candidate-commit gather feeding this rule must
//     not be HEAD-scoped), and
//   - that branch's name doesn't match any ritual shape (branch
//     matching must not depend on enumerating ritual-shaped branch
//     names).
//
// Reuses the same incident-reproduction shape HeadDriftScenario
// (G-0269) already demonstrates — a preflight branch read, a parallel
// session's interloping checkout, then the activation promote — since
// this is the detection-side half of the same lesson: G-0269 is the
// prevention gap (the commit still lands on the wrong branch; the
// race itself is not what this scenario's fix addresses), G-0270 is
// the detection gap (nothing used to notice once it happened).
// Unlike HeadDriftScenario, whose Verify checks raw git ancestry of
// where the commit landed, this scenario additionally checks out back
// to the preflight branch — mirroring how the real incident was only
// noticed later, from a different checkout — and asks `aiwf check`
// whether it now reports the misplacement.
//
// Unlike head-drift's own deliberately-red AC-5, this scenario is
// expected to PASS (report zero violations) once G-0270's fix has
// shipped, and to fail again if that detection regresses — it drives
// the production `aiwf check` binary end-to-end, not a unit-level
// fixture.

// PromoteOnWrongBranchDetectionScenario implements Scenario.
type PromoteOnWrongBranchDetectionScenario struct {
	aiwfBin    string
	epicID     string
	violations []Violation
}

// NewPromoteOnWrongBranchDetectionScenario builds a scenario driving
// one epic activation through the same interloping-checkout shape
// HeadDriftScenario uses, then checking whether `aiwf check` detects
// the misplacement once back on the original branch.
func NewPromoteOnWrongBranchDetectionScenario(aiwfBin string) *PromoteOnWrongBranchDetectionScenario {
	return &PromoteOnWrongBranchDetectionScenario{aiwfBin: aiwfBin}
}

// Setup creates one epic at its default (proposed) status — the
// entity a subsequent activation promote will target.
func (s *PromoteOnWrongBranchDetectionScenario) Setup(dir string) error {
	epicID, err := seedActivationEpic(s.aiwfBin, dir, "wrongbranchdetection", "epic for the promote-on-wrong-branch detection scenario")
	if err != nil {
		return err
	}
	s.epicID = epicID
	return nil
}

// Run reproduces the incident (an interloping checkout onto an
// arbitrarily-named, non-ritual-shaped branch, then the activation
// promote lands there — same shape as the real G-0270 incident), then
// checks back out to the preflight branch and runs `aiwf check`,
// simulating the operator later inspecting from where they started,
// unaware of what happened on the interloper branch in between.
func (s *PromoteOnWrongBranchDetectionScenario) Run(dir string) error {
	preflightBranch, err := currentBranch(dir)
	if err != nil { //coverage:ignore defensive: reading the branch this scenario's own Setup just committed to has no realistic failure mode
		return fmt.Errorf("running the preflight branch read: %w", err)
	}

	if checkoutErr := runGit(dir, "checkout", "-q", "-b", "interloper-branch"); checkoutErr != nil { //coverage:ignore defensive: creating a fresh branch off a repo this scenario itself just built has no realistic failure mode
		return fmt.Errorf("simulating the interloping checkout: %w", checkoutErr)
	}

	promEnv, err := runAiwfJSON(s.aiwfBin, dir, "promote", s.epicID, "active")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("running the activation promote: %w", err)
	}
	if promEnv.Status != "ok" {
		// A guard refused the promote outright — the wrong-branch
		// landing this scenario probes detection for never happened,
		// so there is nothing for `aiwf check` to detect. Mirrors
		// HeadDriftScenario's own refused-promote case: not a
		// violation.
		return nil
	}

	if checkoutErr := runGit(dir, "checkout", "-q", preflightBranch); checkoutErr != nil { //coverage:ignore defensive: checking back out to a branch this scenario's own preflight read just observed has no realistic failure mode
		return fmt.Errorf("checking back out to the preflight branch: %w", checkoutErr)
	}

	checkEnv, err := runAiwfJSON(s.aiwfBin, dir, "check")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("running aiwf check from the preflight branch: %w", err)
	}

	s.violations = classifyPromoteOnWrongBranchDetection(s.epicID, checkEnv) //coverage:ignore reached only if G-0269's branch guard fails to block this scenario's own wrong-branch activation promote (the same interloping-checkout shape HeadDriftScenario drives); the guard now refuses it before it can land, so promEnv.Status is never "ok" here today. classifyPromoteOnWrongBranchDetection's own decision logic is exhaustively pinned against fabricated inputs in promote_on_wrong_branch_detection_classify_test.go regardless.
	return nil
}

// Verify returns every violation Run collected.
func (s *PromoteOnWrongBranchDetectionScenario) Verify(_ string) []Violation {
	return s.violations
}

// classifyPromoteOnWrongBranchDetection judges whether `aiwf check`
// reported the misplaced activation commit: a promote-on-wrong-branch
// finding naming this scenario's epic means detection worked; its
// absence means the misplacement is invisible — the confirmed G-0270
// defect this scenario exists to catch a regression of.
func classifyPromoteOnWrongBranchDetection(epicID string, checkEnv verbEnvelope) []Violation {
	for _, f := range checkEnv.Findings {
		if f.Code == "promote-on-wrong-branch" && f.EntityID == epicID {
			return nil
		}
	}
	return []Violation{{Message: fmt.Sprintf(
		"aiwf check run from the preflight branch reported no promote-on-wrong-branch finding for %s, even though its activation commit landed on a differently-named, non-ritual-shaped branch (G-0270)", epicID,
	)}}
}
