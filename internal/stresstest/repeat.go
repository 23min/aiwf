package stresstest

import "fmt"

// RepeatEvent is the raw-report event one --repeat attempt logs. The
// seed is what makes a violation found on a given attempt replayable:
// rerunning newScenario with that exact seed reproduces the same
// actor-start jitter and randomized delays the attempt used.
type RepeatEvent struct {
	Attempt int   `json:"attempt"`
	Seed    int64 `json:"seed"`
	Passed  bool  `json:"passed"`
}

// RunRepeated runs a scenario n times against baseDir. newScenario
// builds one Scenario per attempt from that attempt's seed (so
// scenario code can thread the seed into whatever randomness it
// uses); seedFn supplies each attempt's seed — production callers
// inject a real random source, tests inject a deterministic
// sequence. Every attempt's outcome is logged via rw as a
// RepeatEvent before the next attempt starts, so a run that's killed
// mid-repeat still leaves every completed attempt's seed on disk.
//
// A scenario attempt that fails verification (RunScenario returns
// Passed: false) does not stop the repeat loop — a single pass of a
// concurrency-shaped scenario proves nothing about a rare race, so
// --repeat's whole point is running the full count regardless,
// buying statistical coverage across all n attempts. A mechanical
// failure in RunScenario itself, or a failure to log an event, aborts
// the loop immediately: the harness's own machinery is broken, so
// continuing would only produce more failures of the same kind.
func RunRepeated(newScenario func(seed int64) Scenario, baseDir string, n int, seedFn func() int64, rw *ReportWriter) ([]RunResult, error) {
	if n <= 0 {
		return nil, fmt.Errorf("repeat count must be positive, got %d", n)
	}

	results := make([]RunResult, 0, n)
	for i := 0; i < n; i++ {
		seed := seedFn()
		result, err := RunScenario(newScenario(seed), baseDir)
		if err != nil {
			return results, fmt.Errorf("attempt %d (seed %d): %w", i, seed, err)
		}
		results = append(results, result)

		event := RepeatEvent{Attempt: i, Seed: seed, Passed: result.Passed}
		if err := rw.WriteEvent(event); err != nil {
			return results, fmt.Errorf("logging attempt %d (seed %d): %w", i, seed, err)
		}
	}
	return results, nil
}
