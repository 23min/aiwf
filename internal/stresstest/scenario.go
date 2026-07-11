package stresstest

import (
	"fmt"
	"os"
)

// Scenario is one unit of the stress harness's catalog. It sets up
// its own disposable repo, drives its behavior, and verifies an
// invariant afterward. Real scenarios (the tiered catalog in
// docs/initiatives/robustness-correctness-stress-testing.md) land in
// later milestones; this milestone only builds the interface and the
// driver that runs one.
type Scenario interface {
	Setup(dir string) error
	Run(dir string) error
	Verify(dir string) []Violation
}

// Violation is one invariant breach a scenario's Verify step found.
type Violation struct {
	Message string
}

// RunResult is the outcome of RunScenario. Dir names the scenario's
// temp directory when it survives on disk (any failure); Dir is empty
// when the scenario passed and its directory was already cleaned up.
type RunResult struct {
	Dir        string
	Passed     bool
	Violations []Violation
}

// RunScenario creates a fresh temp directory under baseDir, runs
// Setup/Run/Verify against it, and applies the harness's cleanup
// discipline: a passing scenario (no error, no violations) removes
// its own temp dir; a failing one — a Setup or Run error, or a
// non-empty Verify result — preserves it, since the on-disk state at
// failure time is RCA material a human might want to open directly.
func RunScenario(s Scenario, baseDir string) (RunResult, error) {
	dir, err := os.MkdirTemp(baseDir, "scenario-")
	if err != nil {
		return RunResult{}, fmt.Errorf("creating scenario temp dir under %s: %w", baseDir, err)
	}

	if err := s.Setup(dir); err != nil {
		return RunResult{Dir: dir}, fmt.Errorf("scenario setup: %w", err)
	}
	if err := s.Run(dir); err != nil {
		return RunResult{Dir: dir}, fmt.Errorf("scenario run: %w", err)
	}

	violations := s.Verify(dir)
	if len(violations) > 0 {
		return RunResult{Dir: dir, Violations: violations}, nil
	}

	if err := os.RemoveAll(dir); err != nil {
		return RunResult{Dir: dir, Passed: true}, fmt.Errorf("cleaning up passing scenario dir %s: %w", dir, err)
	}
	return RunResult{Passed: true}, nil
}
