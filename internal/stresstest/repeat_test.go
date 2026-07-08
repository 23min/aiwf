package stresstest

import (
	"encoding/json"
	"errors"
	"testing"
)

func seedSequence(seeds ...int64) func() int64 {
	i := 0
	return func() int64 {
		s := seeds[i]
		i++
		return s
	}
}

func TestRunRepeated_RunsNAttemptsWithDistinctSeeds(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	cw := &countingWriter{}
	rw := newReportWriter(cw)

	var seedsSeenByNewScenario []int64
	newScenario := func(seed int64) Scenario {
		seedsSeenByNewScenario = append(seedsSeenByNewScenario, seed)
		return &fakeScenario{}
	}
	results, err := RunRepeated(newScenario, base, 3, seedSequence(10, 20, 30), rw)
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	wantSeen := []int64{10, 20, 30}
	if len(seedsSeenByNewScenario) != len(wantSeen) {
		t.Fatalf("newScenario called %d times, want %d", len(seedsSeenByNewScenario), len(wantSeen))
	}
	for i, got := range seedsSeenByNewScenario {
		if got != wantSeen[i] {
			t.Fatalf("newScenario call %d received seed %d, want %d — the seed must reach scenario construction, not just the logged event", i, got, wantSeen[i])
		}
	}
	for _, r := range results {
		if !r.Passed {
			t.Fatalf("expected every attempt to pass, got %+v", r)
		}
	}

	if len(cw.calls) != 3 {
		t.Fatalf("expected 3 logged events, got %d", len(cw.calls))
	}
	wantSeeds := []int64{10, 20, 30}
	for i, call := range cw.calls {
		var ev RepeatEvent
		if err := json.Unmarshal(call[:len(call)-1], &ev); err != nil {
			t.Fatalf("event %d is not valid JSON: %v\n%s", i, err, call)
		}
		if ev.Attempt != i {
			t.Fatalf("event %d: Attempt = %d, want %d", i, ev.Attempt, i)
		}
		if ev.Seed != wantSeeds[i] {
			t.Fatalf("event %d: Seed = %d, want %d", i, ev.Seed, wantSeeds[i])
		}
		if !ev.Passed {
			t.Fatalf("event %d: expected Passed true, got false", i)
		}
	}
}

func TestRunRepeated_ContinuesPastAScenarioFailure(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	rw := newReportWriter(&countingWriter{})

	attempt := 0
	newScenario := func(seed int64) Scenario {
		s := &fakeScenario{}
		if attempt == 1 {
			s.violations = []Violation{{Message: "found it"}}
		}
		attempt++
		return s
	}

	results, err := RunRepeated(newScenario, base, 3, seedSequence(1, 2, 3), rw)
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected all 3 attempts to run despite attempt 1 failing, got %d", len(results))
	}
	if !results[0].Passed || results[1].Passed || !results[2].Passed {
		t.Fatalf("expected pass/fail/pass, got %+v", results)
	}
}

func TestRunRepeated_RejectsNonPositiveRepeatCount(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	rw := newReportWriter(&countingWriter{})
	newScenario := func(seed int64) Scenario { return &fakeScenario{} }

	if _, err := RunRepeated(newScenario, base, 0, seedSequence(), rw); err == nil {
		t.Fatal("expected RunRepeated to reject a repeat count of 0")
	}
}

func TestRunRepeated_AbortsOnScenarioSetupError(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	rw := newReportWriter(&countingWriter{})

	attempt := 0
	newScenario := func(seed int64) Scenario {
		s := &fakeScenario{}
		if attempt == 1 {
			s.setupErr = errors.New("simulated setup failure")
		}
		attempt++
		return s
	}

	results, err := RunRepeated(newScenario, base, 3, seedSequence(1, 2, 3), rw)
	if err == nil {
		t.Fatal("expected RunRepeated to abort and return an error on a Setup failure")
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 completed result before the abort, got %d", len(results))
	}
}

func TestRunRepeated_AbortsOnLoggingError(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	rw := newReportWriter(erroringWriter{})
	newScenario := func(seed int64) Scenario { return &fakeScenario{} }

	if _, err := RunRepeated(newScenario, base, 2, seedSequence(1, 2), rw); err == nil {
		t.Fatal("expected RunRepeated to propagate a raw-report logging failure")
	}
}
