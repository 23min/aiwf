package stresstest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/23min/aiwf/internal/check"
)

// concurrent_writer_at_scale.go — M-0244/AC-1: ConcurrentWriterAtScaleScenario
// proves ADR-0017 Decision #5's O_APPEND/one-Write()-per-record safety
// under real, separate-OS-process concurrency — not the package-level
// goroutine simulation internal/logger's own
// TestConcurrentAppend_NoInterleavingOrTearing (M-0237) already covers,
// which shares one process's memory even though each writer opens its
// own file handle. n real `aiwf cancel` subprocesses, each instrumented
// via AIWF_LOG=debug/AIWF_LOG_FORMAT=json/AIWF_LOG_FILE pointed at one
// shared log file, run concurrently against n distinct pre-seeded gaps.
// Every invocation's diagnostic logger binds run_id to the same
// correlation id its own --format=json envelope reports
// (metadata.correlation_id — M-0239/AC-1), so
// classifyConcurrentWriterAtScale can assert not just "no torn or
// interleaved line" but "every line's run_id matches exactly one real
// invocation" — extending the existing single-process correlation
// guarantee to concurrent, multi-process load.

// concurrentWriterAtScaleExpectedWarnings is the baseline of finding
// codes this scenario's post-run check is expected to carry
// (M-0257/AC-1), beyond the shared-log-file assertion
// classifyConcurrentWriterAtScale already pins directly:
//
//   - archive-sweep-pending / terminal-entity-not-archived: every one
//     of the n seeded gaps is cancelled (a terminal status) by this
//     scenario's own actors, and it never runs `aiwf archive` — both
//     are advisory-only sweep reminders, not evidence of anything
//     this scenario probes.
//   - provenance-untrailered-scope-undefined: this scenario's
//     disposable repo never configures an upstream remote.
//
// Any OTHER finding — any error-severity finding, or a warning with a
// code not in this set — is a real violation.
var concurrentWriterAtScaleExpectedWarnings = map[string]bool{
	check.CodeArchiveSweepPending:               true,
	check.CodeTerminalEntityNotArchived:         true,
	check.CodeProvenanceUntrailedScopeUndefined: true,
}

// ConcurrentWriterAtScaleScenario implements Scenario.
type ConcurrentWriterAtScaleScenario struct {
	aiwfBin    string
	n          int
	gapIDs     []string
	violations []Violation
}

// NewConcurrentWriterAtScaleScenario builds a scenario driving n
// concurrent `aiwf cancel` subprocesses, each writing its diagnostic
// log line to one shared file. seed matches RunRepeated's
// newScenario(seed int64) Scenario signature (M-0240/AC-5) but is
// otherwise unused — this scenario's write-ordering jitter comes from
// real OS process scheduling, not seeded pseudo-randomness.
func NewConcurrentWriterAtScaleScenario(aiwfBin string, n int, _ int64) *ConcurrentWriterAtScaleScenario {
	return &ConcurrentWriterAtScaleScenario{aiwfBin: aiwfBin, n: n}
}

// Setup git-inits dir and pre-seeds n gap entities — one per actor Run
// will later cancel — capturing each allocated id from its own `aiwf
// add` envelope rather than assuming a fixed id-width format.
func (s *ConcurrentWriterAtScaleScenario) Setup(dir string) error {
	if err := gitInitAndConfig(dir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}
	s.gapIDs = make([]string, 0, s.n)
	for i := 0; i < s.n; i++ {
		env, err := runAiwfJSON(s.aiwfBin, dir, "add", "gap", "--title", fmt.Sprintf("concurrent-writer probe %d", i), "--body", "concurrent-writer-at-scale stress gap")
		if err != nil {
			return fmt.Errorf("seeding gap %d: %w", i, err)
		}
		if env.Status != "ok" { //coverage:ignore defensive: allocating the i-th gap in a repo this scenario itself just created, with no other writer present, has no realistic failure mode
			return fmt.Errorf("seeding gap %d: aiwf add did not report ok: %+v", i, env)
		}
		s.gapIDs = append(s.gapIDs, env.Metadata.EntityID)
	}
	return nil
}

// diagLogLine is the one field classifyConcurrentWriterAtScale needs
// out of a parsed diagnostic log line.
type diagLogLine struct {
	RunID string `json:"run_id"`
}

// launchCancelActor runs one `aiwf cancel <id>` invocation with
// diagnostic logging enabled and pointed at logPath, shared across
// every concurrent actor, and returns its envelope so Run can read
// back the run_id (metadata.correlation_id) this invocation should
// have logged.
func (s *ConcurrentWriterAtScaleScenario) launchCancelActor(dir, logPath, id string) (verbEnvelope, error) {
	cmd := exec.Command(s.aiwfBin, "cancel", id, "--reason", "concurrent-writer-at-scale probe", "--format=json") //nolint:gosec // s.aiwfBin is a path this package's own BuildBinary just produced, not attacker-controlled input
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "AIWF_LOG=debug", "AIWF_LOG_FORMAT=json", "AIWF_LOG_FILE="+logPath)
	out, err := cmd.Output()
	if err != nil {
		return verbEnvelope{}, fmt.Errorf("running aiwf cancel %s: %w", id, err)
	}
	return parseVerbEnvelope([]string{"cancel", id}, out)
}

// Run launches one `aiwf cancel` subprocess per pre-seeded gap,
// concurrently, all pointed at one shared diagnostic log file, then
// classifies the resulting file against every actor's own reported
// run_id. A verb outcome's status (ok or otherwise) is irrelevant to
// this scenario's own claim — EmitVerbOutcome's deferred call fires
// regardless of exit code, so even a non-ok invocation still logs a
// real run_id this scenario expects to find exactly once.
func (s *ConcurrentWriterAtScaleScenario) Run(dir string) error {
	logPath := filepath.Join(dir, "diag.log")

	envs := make([]verbEnvelope, s.n)
	errs := make([]error, s.n)
	var wg sync.WaitGroup
	for i, id := range s.gapIDs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			envs[i], errs[i] = s.launchCancelActor(dir, logPath, id)
		}(i, id)
	}
	wg.Wait()

	wantRunIDs := make([]string, s.n)
	for i, err := range errs {
		if err != nil {
			return fmt.Errorf("actor %d: %w", i, err)
		}
		wantRunIDs[i] = envs[i].Metadata.CorrelationID
	}

	raw, err := os.ReadFile(logPath)
	if err != nil { //coverage:ignore defensive: every one of the n actors above returned a parsed envelope, and EmitVerbOutcome always writes its outcome line via a defer registered before repolock acquisition even runs — the shared log file this scenario itself pointed AIWF_LOG_FILE at always exists by this point
		return fmt.Errorf("reading shared diagnostic log %s: %w", logPath, err)
	}

	parseFailures, logRunIDs, err := parseDiagLog(raw)
	if err != nil { //coverage:ignore defensive: bufio.Scanner over a bytes.Reader wrapping an already-successfully os.ReadFile'd byte slice has no realistic failure mode
		return fmt.Errorf("scanning shared diagnostic log: %w", err)
	}

	s.violations = classifyConcurrentWriterAtScale(parseFailures, logRunIDs, wantRunIDs)

	// M-0257/AC-1: alongside the shared-log-file assertion above,
	// confirm the resulting tree stays check-clean beyond baseline
	// noise — this scenario never ran `aiwf check` at all before.
	checkEnv, err := runAiwfJSON(s.aiwfBin, dir, "check")
	if err != nil { //coverage:ignore defensive: same launch-failure class other scenarios pin at runAiwfJSON's own source; the actor loop above already exercised this binary successfully by the time this call runs
		return fmt.Errorf("running aiwf check after the concurrent writers: %w", err)
	}
	s.violations = append(s.violations, classifyAgainstBaseline(checkEnv.Findings, concurrentWriterAtScaleExpectedWarnings)...)
	return nil
}

// parseDiagLog splits raw (a diagnostic log file's bytes) into lines,
// decoding each as a diagLogLine: a line that fails to parse is
// recorded verbatim in parseFailures (evidence of interleaving or
// tearing) rather than aborting the scan, so one bad line doesn't hide
// the rest of the file's outcome. Split out of Run so the malformed-
// line path is directly unit-testable without a real subprocess run —
// genuine O_APPEND tearing isn't reproducible on demand.
func parseDiagLog(raw []byte) (parseFailures, logRunIDs []string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Bytes()
		var decoded diagLogLine
		if jsonErr := json.Unmarshal(line, &decoded); jsonErr != nil {
			parseFailures = append(parseFailures, string(line))
			continue
		}
		logRunIDs = append(logRunIDs, decoded.RunID)
	}
	if scanErr := scanner.Err(); scanErr != nil { //coverage:ignore defensive: bufio.Scanner over a bytes.Reader wrapping an in-memory byte slice has no realistic failure mode short of a single line exceeding bufio.MaxScanTokenSize, which a one-line JSON diagnostic event never does
		return nil, nil, scanErr
	}
	return parseFailures, logRunIDs, nil
}

// Verify returns every violation Run collected.
func (s *ConcurrentWriterAtScaleScenario) Verify(_ string) []Violation {
	return s.violations
}

// classifyConcurrentWriterAtScale judges one concurrent-writer run: a
// parse failure means the shared log file interleaved or tore a line;
// a run_id count other than exactly 1 for a wanted actor means that
// actor's own diagnostic event went missing or was duplicated; a
// logged run_id absent from wantRunIDs is a foreign or corrupted
// value. Every check runs independently and reports rather than
// short-circuits, so one broken run can surface more than one kind of
// violation at once.
func classifyConcurrentWriterAtScale(parseFailures, logRunIDs, wantRunIDs []string) []Violation {
	var violations []Violation
	for _, raw := range parseFailures {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"shared diagnostic log line did not parse as valid JSON (interleaved or torn): %q", raw)})
	}

	counts := make(map[string]int, len(logRunIDs))
	for _, id := range logRunIDs {
		counts[id]++
	}
	want := make(map[string]bool, len(wantRunIDs))
	for _, id := range wantRunIDs {
		want[id] = true
		if counts[id] != 1 {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"run_id %s (one real aiwf cancel invocation's own correlation id) appears %d times in the shared diagnostic log, want exactly 1", id, counts[id])})
		}
	}
	for id, count := range counts {
		if !want[id] {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"run_id %s appears %d time(s) in the shared diagnostic log but does not match any of this run's actors' own correlation ids", id, count)})
		}
	}
	return violations
}
