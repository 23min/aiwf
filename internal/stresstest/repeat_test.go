package stresstest

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func seedSequence(seeds ...int64) func() int64 {
	i := 0
	return func() int64 {
		s := seeds[i]
		i++
		return s
	}
}

// logWritingScenario simulates a scenario whose Run drives real aiwf
// subprocesses that each append one diagnostic-log line — used only
// to pin RunRepeated's per-attempt correlation-id attribution without
// a real subprocess.
type logWritingScenario struct {
	logPath string
	lines   []string
}

func (s *logWritingScenario) Setup(_ string) error { return nil }

func (s *logWritingScenario) Run(_ string) error {
	f, err := os.OpenFile(s.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	for _, line := range s.lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func (s *logWritingScenario) Verify(_ string) []Violation { return nil }

// unmarshalEvent decodes call (as WriteEvent wrote it: JSON plus a
// trailing newline) into a RepeatEvent, failing the test on error.
func unmarshalEvent(t *testing.T, call []byte) RepeatEvent {
	t.Helper()
	var ev RepeatEvent
	if err := json.Unmarshal(call[:len(call)-1], &ev); err != nil {
		t.Fatalf("event is not valid JSON: %v\n%s", err, call)
	}
	return ev
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
	results, err := RunRepeated(newScenario, base, 3, seedSequence(10, 20, 30), rw, "")
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
	cw := &countingWriter{}
	rw := newReportWriter(cw)

	attempt := 0
	newScenario := func(seed int64) Scenario {
		s := &fakeScenario{}
		if attempt == 1 {
			s.violations = []Violation{{Message: "found it"}}
		}
		attempt++
		return s
	}

	results, err := RunRepeated(newScenario, base, 3, seedSequence(1, 2, 3), rw, "")
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected all 3 attempts to run despite attempt 1 failing, got %d", len(results))
	}
	if !results[0].Passed || results[1].Passed || !results[2].Passed {
		t.Fatalf("expected pass/fail/pass, got %+v", results)
	}

	// Pin that the logged event's Passed field actually reflects each
	// attempt's outcome, not just the returned RunResult — a
	// hardcoded Passed: true in the event construction would satisfy
	// every assertion above while still silently misleading a report
	// reader about which attempt failed.
	if len(cw.calls) != 3 {
		t.Fatalf("expected 3 logged events, got %d", len(cw.calls))
	}
	wantPassed := []bool{true, false, true}
	for i, call := range cw.calls {
		var ev RepeatEvent
		if err := json.Unmarshal(call[:len(call)-1], &ev); err != nil {
			t.Fatalf("event %d is not valid JSON: %v\n%s", i, err, call)
		}
		if ev.Passed != wantPassed[i] {
			t.Fatalf("event %d: Passed = %v, want %v", i, ev.Passed, wantPassed[i])
		}
	}
}

func TestRunRepeated_RejectsNonPositiveRepeatCount(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	rw := newReportWriter(&countingWriter{})
	newScenario := func(seed int64) Scenario { return &fakeScenario{} }

	if _, err := RunRepeated(newScenario, base, 0, seedSequence(), rw, ""); err == nil {
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

	results, err := RunRepeated(newScenario, base, 3, seedSequence(1, 2, 3), rw, "")
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

	if _, err := RunRepeated(newScenario, base, 2, seedSequence(1, 2), rw, ""); err == nil {
		t.Fatal("expected RunRepeated to propagate a raw-report logging failure")
	}
}

// TestRunRepeated_LogsDirOnFailingAttempt pins M-0249/AC-2's own
// finding: RunResult.Dir was already populated in memory on a failing
// attempt but never reached the logged event. A passing attempt's Dir
// stays empty (RunScenario already removed the dir).
func TestRunRepeated_LogsDirOnFailingAttempt(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	cw := &countingWriter{}
	rw := newReportWriter(cw)

	attempt := 0
	newScenario := func(seed int64) Scenario {
		s := &fakeScenario{}
		if attempt == 1 {
			s.violations = []Violation{{Message: "found it"}}
		}
		attempt++
		return s
	}

	results, err := RunRepeated(newScenario, base, 3, seedSequence(1, 2, 3), rw, "")
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}

	events := make([]RepeatEvent, len(cw.calls))
	for i, call := range cw.calls {
		events[i] = unmarshalEvent(t, call)
	}
	if events[0].Dir != "" {
		t.Errorf("event 0 (passed): Dir = %q, want empty", events[0].Dir)
	}
	if events[1].Dir == "" {
		t.Error("event 1 (failed): Dir is empty, want the preserved scenario dir")
	}
	if events[1].Dir != results[1].Dir {
		t.Errorf("event 1: Dir = %q, want RunResult.Dir %q", events[1].Dir, results[1].Dir)
	}
	if events[2].Dir != "" {
		t.Errorf("event 2 (passed): Dir = %q, want empty", events[2].Dir)
	}
}

// TestRunRepeated_LogsCorrelationIDsFromDiagnosticLog pins the
// per-attempt correlation-id attribution: each attempt's own
// diagnostic-log lines (written by logWritingScenario standing in for
// the real aiwf subprocesses a scenario drives) land on that attempt's
// own event, never bleeding into a neighboring attempt's.
func TestRunRepeated_LogsCorrelationIDsFromDiagnosticLog(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "diagnostic.log")
	cw := &countingWriter{}
	rw := newReportWriter(cw)

	attempt := 0
	linesByAttempt := [][]string{
		{`{"run_id":"aaa"}`, `{"run_id":"bbb"}`, `{"run_id":"aaa"}`},
		{`{"run_id":"ccc"}`},
	}
	newScenario := func(seed int64) Scenario {
		s := &logWritingScenario{logPath: logPath, lines: linesByAttempt[attempt]}
		attempt++
		return s
	}

	if _, err := RunRepeated(newScenario, base, 2, seedSequence(1, 2), rw, logPath); err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}

	events := make([]RepeatEvent, len(cw.calls))
	for i, call := range cw.calls {
		events[i] = unmarshalEvent(t, call)
	}
	if diff := cmp.Diff([]string{"aaa", "bbb"}, events[0].CorrelationIDs); diff != "" {
		t.Errorf("event 0 CorrelationIDs mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"ccc"}, events[1].CorrelationIDs); diff != "" {
		t.Errorf("event 1 CorrelationIDs mismatch (-want +got):\n%s", diff)
	}
}

// TestRunRepeated_ToleratesMissingDiagnosticLog pins that a
// diagnosticLogPath naming a file that never gets created (diagnostic
// logging effectively produced nothing this run) is not itself an
// error — CorrelationIDs is just empty.
func TestRunRepeated_ToleratesMissingDiagnosticLog(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	cw := &countingWriter{}
	rw := newReportWriter(cw)
	newScenario := func(seed int64) Scenario { return &fakeScenario{} }

	if _, err := RunRepeated(newScenario, base, 1, seedSequence(1), rw, filepath.Join(t.TempDir(), "never-created.log")); err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	ev := unmarshalEvent(t, cw.calls[0])
	if len(ev.CorrelationIDs) != 0 {
		t.Errorf("CorrelationIDs = %v, want empty", ev.CorrelationIDs)
	}
}

// TestRunRepeated_SkipsAMalformedDiagnosticLogLineAndContinues pins
// D-0035's own resolution: a complete-but-malformed JSON line in the
// diagnostic log (the shape a concurrent-write interleave under
// O_APPEND's PIPE_BUF-sized write guarantee, or genuine corruption,
// leaves behind) does NOT abort the campaign — a single scenario
// attempt's own replayability (its seed) must not be gated behind an
// optional correlation-id harvest's success. The malformed line
// simply contributes no id; a well-formed line elsewhere in the same
// scan still does.
func TestRunRepeated_SkipsAMalformedDiagnosticLogLineAndContinues(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "diagnostic.log")
	content := "not-json\n{\"run_id\":\"good\"}\n"
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatalf("seed diagnostic log: %v", err)
	}
	cw := &countingWriter{}
	rw := newReportWriter(cw)
	newScenario := func(seed int64) Scenario { return &fakeScenario{} }

	if _, err := RunRepeated(newScenario, base, 1, seedSequence(1), rw, logPath); err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	ev := unmarshalEvent(t, cw.calls[0])
	if ev.Seed != 1 {
		t.Errorf("Seed = %d, want 1 (the attempt's own event must still be logged)", ev.Seed)
	}
	if diff := cmp.Diff([]string{"good"}, ev.CorrelationIDs); diff != "" {
		t.Errorf("CorrelationIDs mismatch (-want +got):\n%s", diff)
	}
}

func TestCorrelationIDsSince_MissingFileReturnsNilUnchanged(t *testing.T) {
	t.Parallel()
	ids, offset, err := correlationIDsSince(filepath.Join(t.TempDir(), "does-not-exist.log"), 5)
	if err != nil {
		t.Fatalf("correlationIDsSince: %v", err)
	}
	if ids != nil {
		t.Errorf("ids = %v, want nil", ids)
	}
	if offset != 5 {
		t.Errorf("offset = %d, want unchanged 5", offset)
	}
}

func TestCorrelationIDsSince_DedupesWithinOneScan(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diagnostic.log")
	content := "{\"run_id\":\"one\"}\n{\"run_id\":\"two\"}\n{\"run_id\":\"one\"}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ids, offset, err := correlationIDsSince(path, 0)
	if err != nil {
		t.Fatalf("correlationIDsSince: %v", err)
	}
	if diff := cmp.Diff([]string{"one", "two"}, ids); diff != "" {
		t.Errorf("ids mismatch (-want +got):\n%s", diff)
	}
	if offset != int64(len(content)) {
		t.Errorf("offset = %d, want %d (full file consumed)", offset, len(content))
	}
}

// TestCorrelationIDsSince_LeavesATrailingPartialLineUnconsumed pins
// the resumable-cursor contract: a line with no trailing newline yet
// (a subprocess still mid-write) is not parsed and not consumed — the
// returned offset stops right before it, so a later call starting
// from that offset picks up the now-complete line.
func TestCorrelationIDsSince_LeavesATrailingPartialLineUnconsumed(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diagnostic.log")
	complete := "{\"run_id\":\"one\"}\n"
	partial := "{\"run_id\":\"two\""
	if err := os.WriteFile(path, []byte(complete+partial), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ids, offset, err := correlationIDsSince(path, 0)
	if err != nil {
		t.Fatalf("correlationIDsSince: %v", err)
	}
	if diff := cmp.Diff([]string{"one"}, ids); diff != "" {
		t.Errorf("ids mismatch (-want +got):\n%s", diff)
	}
	if offset != int64(len(complete)) {
		t.Errorf("offset = %d, want %d (stops before the partial line)", offset, len(complete))
	}

	// Completing the partial line and re-scanning from the returned
	// offset picks it up.
	if writeErr := os.WriteFile(path, []byte(complete+partial+"}\n"), 0o600); writeErr != nil {
		t.Fatalf("rewrite: %v", writeErr)
	}
	ids2, _, err := correlationIDsSince(path, offset)
	if err != nil {
		t.Fatalf("correlationIDsSince (resume): %v", err)
	}
	if diff := cmp.Diff([]string{"two"}, ids2); diff != "" {
		t.Errorf("resumed ids mismatch (-want +got):\n%s", diff)
	}
}

func TestCorrelationIDsSince_ResumesFromANonZeroOffset(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diagnostic.log")
	first := "{\"run_id\":\"one\"}\n"
	second := "{\"run_id\":\"two\"}\n"
	if err := os.WriteFile(path, []byte(first+second), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ids, _, err := correlationIDsSince(path, int64(len(first)))
	if err != nil {
		t.Fatalf("correlationIDsSince: %v", err)
	}
	if diff := cmp.Diff([]string{"two"}, ids); diff != "" {
		t.Errorf("ids mismatch (-want +got):\n%s", diff)
	}
}

func TestCorrelationIDsSince_SkipsBlankLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diagnostic.log")
	content := "{\"run_id\":\"one\"}\n\n{\"run_id\":\"two\"}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ids, offset, err := correlationIDsSince(path, 0)
	if err != nil {
		t.Fatalf("correlationIDsSince: %v", err)
	}
	if diff := cmp.Diff([]string{"one", "two"}, ids); diff != "" {
		t.Errorf("ids mismatch (-want +got):\n%s", diff)
	}
	if offset != int64(len(content)) {
		t.Errorf("offset = %d, want %d", offset, len(content))
	}
}

func TestCorrelationIDsSince_SkipsLinesWithoutARunID(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diagnostic.log")
	content := "{\"level\":\"info\",\"msg\":\"no run_id here\"}\n{\"run_id\":\"one\"}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ids, _, err := correlationIDsSince(path, 0)
	if err != nil {
		t.Fatalf("correlationIDsSince: %v", err)
	}
	if diff := cmp.Diff([]string{"one"}, ids); diff != "" {
		t.Errorf("ids mismatch (-want +got):\n%s", diff)
	}
}

// TestCorrelationIDsSince_SkipsAMalformedCompleteLineRatherThanErroring
// pins D-0035's own resolution: a complete line that isn't valid JSON
// at all (not merely missing run_id) is skipped, not an error — the
// shape a benign concurrent-write interleave under O_APPEND's
// PIPE_BUF-sized write guarantee can produce. The offset still
// advances past it (it's fully consumed, not left as a retry
// candidate — unlike the genuinely-incomplete trailing-line case),
// and a well-formed line on either side is still picked up.
func TestCorrelationIDsSince_SkipsAMalformedCompleteLineRatherThanErroring(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diagnostic.log")
	content := "{\"run_id\":\"before\"}\nnot-valid-json-at-all\n{\"run_id\":\"after\"}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ids, offset, err := correlationIDsSince(path, 0)
	if err != nil {
		t.Fatalf("correlationIDsSince: %v", err)
	}
	if diff := cmp.Diff([]string{"before", "after"}, ids); diff != "" {
		t.Errorf("ids mismatch (-want +got):\n%s", diff)
	}
	if offset != int64(len(content)) {
		t.Errorf("offset = %d, want %d (the malformed line is still fully consumed)", offset, len(content))
	}
}
