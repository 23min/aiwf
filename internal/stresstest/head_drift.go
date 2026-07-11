package stresstest

import "fmt"

// head_drift.go — M-0243/AC-5: HeadDriftScenario reproduces the actual
// G-0269 incident, deterministically — no real concurrency is needed
// to demonstrate it, since the defect is a plain time-of-check to
// time-of-use gap between two SEQUENTIAL steps, not a timing race
// that must be won: a preflight reads the current branch, a parallel
// session's `git checkout` changes it, and the verb that follows
// commits against whatever branch is now checked out, silently
// landing on the wrong one. Per this milestone's own Constraints,
// this AC is allowed to end red (confirming the incident still
// reproduces) without failing the milestone — the acceptance is that
// the scenario exists and correctly reports it, not that the race is
// fixed. Once G-0269's own guard ships, a real run of this scenario
// starts reporting 0 violations instead — itself a meaningful signal.

// HeadDriftScenario implements Scenario.
type HeadDriftScenario struct {
	aiwfBin    string
	epicID     string
	violations []Violation
}

// NewHeadDriftScenario builds a scenario driving one epic activation
// through a simulated preflight-then-interloper-checkout sequence.
func NewHeadDriftScenario(aiwfBin string) *HeadDriftScenario {
	return &HeadDriftScenario{aiwfBin: aiwfBin}
}

// Setup creates one epic at its default (proposed) status — the
// entity a subsequent activation promote will target, mirroring the
// actual G-0269 incident's own `aiwf promote <epic> active` call.
func (s *HeadDriftScenario) Setup(dir string) error {
	epicID, err := seedActivationEpic(s.aiwfBin, dir, "headdrift", "epic for the head-drift scenario")
	if err != nil {
		return err
	}
	s.epicID = epicID
	return nil
}

// Run simulates the incident: a preflight reads the current branch, a
// parallel session's checkout switches to a different one, and the
// activation promote then runs — landing wherever HEAD now points,
// not the branch the preflight observed.
func (s *HeadDriftScenario) Run(dir string) error {
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

	var landedOnPreflightBranch, landedOnInterloperBranch bool
	if promEnv.Status == "ok" {
		sha, shaErr := headSHA(dir)
		if shaErr != nil { //coverage:ignore defensive: see headSHA's own rationale
			return fmt.Errorf("reading the promote commit SHA: %w", shaErr)
		}
		landedOnPreflightBranch = runGit(dir, "merge-base", "--is-ancestor", sha, preflightBranch) == nil
		landedOnInterloperBranch = runGit(dir, "merge-base", "--is-ancestor", sha, "interloper-branch") == nil
	}

	s.violations = classifyHeadDrift(promEnv.Status, landedOnPreflightBranch, landedOnInterloperBranch)
	return nil
}

// Verify returns every violation Run collected.
func (s *HeadDriftScenario) Verify(_ string) []Violation {
	return s.violations
}

// classifyHeadDrift judges one head-drift attempt. A refused promote
// (a guard blocked it outright) reports no violation. Otherwise: a
// commit landing on the interloper branch (not the preflight-observed
// one) confirms G-0269's incident still reproduces — a violation,
// expected per this milestone's own Constraints; landing on the
// preflight-observed branch instead means a guard now exists (good
// news, no violation); landing on neither is an unexpected outcome,
// itself reported as a violation.
func classifyHeadDrift(promStatus string, landedOnPreflightBranch, landedOnInterloperBranch bool) []Violation {
	if promStatus != "ok" {
		return nil
	}
	if landedOnPreflightBranch {
		return nil
	}
	if landedOnInterloperBranch {
		return []Violation{{Message: "confirmed (G-0269, expected-red until its guard lands): the promote commit landed on the interloper branch a parallel session's checkout drifted to between preflight and commit, not the branch the preflight observed — no guard yet blocks this"}}
	}
	return []Violation{{Message: "the promote commit landed on neither the preflight-observed branch nor the interloper branch — unexpected outcome"}}
}
