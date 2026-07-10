package stresstest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// RepeatEvent is the raw-report event one --repeat attempt logs. The
// seed is what makes a violation found on a given attempt replayable:
// rerunning newScenario with that exact seed reproduces the same
// actor-start jitter and randomized delays the attempt used. Dir
// names the preserved scenario dir on a failing attempt (mirrors
// RunResult.Dir; empty on a pass). CorrelationIDs holds every
// diagnostic-log run_id observed during this attempt's window (empty
// when diagnosticLogPath was never supplied, or nothing logged) — the
// same value each contributing subprocess's own --format=json output
// reports as metadata.correlation_id (internal/cli/root.go: "reused
// as the diagnostic logger's run_id"), so a failing attempt's Dir plus
// these ids is enough to find every diagnostic-log entry that
// subprocess wrote without re-running the campaign.
type RepeatEvent struct {
	Attempt        int      `json:"attempt"`
	Seed           int64    `json:"seed"`
	Passed         bool     `json:"passed"`
	Dir            string   `json:"dir,omitempty"`
	CorrelationIDs []string `json:"correlation_ids,omitempty"`
}

// RunRepeated runs a scenario n times against baseDir. newScenario
// builds one Scenario per attempt from that attempt's seed (so
// scenario code can thread the seed into whatever randomness it
// uses); seedFn supplies each attempt's seed — production callers
// inject a real random source, tests inject a deterministic
// sequence. Every attempt's outcome is logged via rw as a
// RepeatEvent before the next attempt starts, so a run that's killed
// mid-repeat still leaves every completed attempt's seed on disk.
// diagnosticLogPath, when non-empty, names aiwf's own diagnostic-log
// file (AIWF_LOG_FILE) the scenario's subprocesses write to; each
// attempt's RepeatEvent.CorrelationIDs is the set of run_ids that
// landed in the file during that attempt's own window, via a
// resumable byte-offset cursor so consecutive attempts never
// attribute the same log lines twice. An empty diagnosticLogPath (no
// diagnostic logging enabled for this run) skips attribution
// entirely — every event's CorrelationIDs stays empty.
//
// A scenario attempt that fails verification (RunScenario returns
// Passed: false) does not stop the repeat loop — a single pass of a
// concurrency-shaped scenario proves nothing about a rare race, so
// --repeat's whole point is running the full count regardless,
// buying statistical coverage across all n attempts. A mechanical
// failure in RunScenario itself, a failure to log an event, or a
// diagnostic-log line that's corrupted rather than merely still
// mid-write, aborts the loop immediately: the harness's own machinery
// is broken, so continuing would only produce more failures of the
// same kind.
func RunRepeated(newScenario func(seed int64) Scenario, baseDir string, n int, seedFn func() int64, rw *ReportWriter, diagnosticLogPath string) ([]RunResult, error) {
	if n <= 0 {
		return nil, fmt.Errorf("repeat count must be positive, got %d", n)
	}

	results := make([]RunResult, 0, n)
	var logOffset int64
	for i := 0; i < n; i++ {
		seed := seedFn()
		result, err := RunScenario(newScenario(seed), baseDir)
		if err != nil {
			return results, fmt.Errorf("attempt %d (seed %d): %w", i, seed, err)
		}
		results = append(results, result)

		var ids []string
		if diagnosticLogPath != "" {
			var scanErr error
			ids, logOffset, scanErr = correlationIDsSince(diagnosticLogPath, logOffset)
			if scanErr != nil {
				return results, fmt.Errorf("attempt %d (seed %d): %w", i, seed, scanErr)
			}
		}

		event := RepeatEvent{Attempt: i, Seed: seed, Passed: result.Passed, Dir: result.Dir, CorrelationIDs: ids}
		if err := rw.WriteEvent(event); err != nil {
			return results, fmt.Errorf("logging attempt %d (seed %d): %w", i, seed, err)
		}
	}
	return results, nil
}

// correlationIDsSince reads path from byte offset from to EOF and
// returns the distinct "run_id" field value of each complete
// (newline-terminated) JSON line, in first-seen order, plus the byte
// offset to resume from on the next call. A line with no trailing
// newline yet — a subprocess still mid-write — is left unconsumed:
// the returned offset stops before it, so a later call starting from
// that offset picks it up once it's complete. A complete line that
// still fails to parse is real corruption, not an in-flight write,
// and returns an error. A path that does not exist yet (diagnostic
// logging produced nothing so far) is not an error: returns (nil,
// from, nil) unchanged.
func correlationIDsSince(path string, from int64) (ids []string, offset int64, err error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, from, nil
	}
	if err != nil { //coverage:ignore not portably triggerable: an os.Open failure other than not-exist (e.g. permission denied) can't be forced deterministically here — a devcontainer commonly runs as root, which bypasses permission checks entirely, making a chmod-based test flaky-to-vacuous depending on the runtime user
		return nil, from, fmt.Errorf("opening diagnostic log %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	if _, err = f.Seek(from, io.SeekStart); err != nil { //coverage:ignore not portably triggerable: Seek on a freshly opened regular file with a non-negative offset has no realistic failure mode
		return nil, from, fmt.Errorf("seeking diagnostic log %s to offset %d: %w", path, from, err)
	}
	data, err := io.ReadAll(f)
	if err != nil { //coverage:ignore not portably triggerable: reading a regular, already-seeked file this function itself just opened has no realistic failure mode short of the file being altered out from under the process mid-call
		return nil, from, fmt.Errorf("reading diagnostic log %s from offset %d: %w", path, from, err)
	}

	seen := make(map[string]bool)
	offset = from
	rest := data
	for {
		idx := bytes.IndexByte(rest, '\n')
		if idx < 0 {
			break
		}
		line := rest[:idx]
		rest = rest[idx+1:]
		offset += int64(idx) + 1

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var rec struct {
			RunID string `json:"run_id"`
		}
		if err := json.Unmarshal(line, &rec); err != nil {
			return ids, offset, fmt.Errorf("diagnostic log %s: malformed record ending at offset %d: %w", path, offset, err)
		}
		if rec.RunID != "" && !seen[rec.RunID] {
			seen[rec.RunID] = true
			ids = append(ids, rec.RunID)
		}
	}
	return ids, offset, nil
}
