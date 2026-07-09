package stresstest

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// mid_write_kill.go — M-0242/AC-2: MidWriteKillScenario proves a
// process killed mid-write never leaves a half-written entity file.
// It drives two identical disposable repos (control, target) seeded
// with the same large-bodied gap entity: the control repo runs a real
// `aiwf promote` to completion, capturing the fully-written bytes;
// the target repo runs the same promote as a killable subprocess,
// watched from outside for pathutil.AtomicWriteFile's sibling temp
// file (the "<name>.aiwf-tmp-*" pattern its own doc comment
// documents — no code change to internal/pathutil, per M-0242/AC-3)
// to appear, and is SIGKILLed the instant it's observed. The oracle:
// the target's entity file afterward must be byte-identical to either
// the pre-write or the fully-written bytes — never a third value.
//
// The entity body is made large (see midWriteBodySize) so the write's
// temp-file-visible window is comfortably wide for a busy-poll to
// catch deterministically, not a matter of racing a near-instant
// syscall sequence.

// midWriteBodySize is calibrated (empirically, against
// pathutil.AtomicWriteFile directly) to give the temp-file-visible
// window tens of milliseconds of width — comfortably wide for
// waitForTempFile's busy-poll to catch reliably on any machine this
// runs on, not a matter of winning a microsecond-scale race.
const midWriteBodySize = 10_000_000

// defaultMidWriteReadyTimeout bounds how long Run waits to observe
// the sibling temp file before giving up. Generous: the write's
// visible window is calibrated to tens of milliseconds (see
// midWriteBodySize), so 5s is ample margin, not a tight budget.
const defaultMidWriteReadyTimeout = 5 * time.Second

// MidWriteKillScenario implements Scenario.
type MidWriteKillScenario struct {
	aiwfBin string
	// readyTimeout bounds how long Run waits to observe the sibling
	// temp file. Defaulted by the constructor; tests in this package
	// may set it directly (same-package struct literal) to force the
	// timeout branch deterministically and quickly.
	readyTimeout time.Duration
	violations   []Violation
}

// NewMidWriteKillScenario builds a scenario driving aiwfBin (the real
// compiled aiwf binary) against a large-bodied gap entity.
func NewMidWriteKillScenario(aiwfBin string) *MidWriteKillScenario {
	return &MidWriteKillScenario{aiwfBin: aiwfBin, readyTimeout: defaultMidWriteReadyTimeout}
}

// Setup creates dir/control and dir/target as two independent git
// repos, each seeded with one identically-bodied gap entity.
func (s *MidWriteKillScenario) Setup(dir string) error {
	bodyPath := filepath.Join(dir, "body.txt")
	if err := os.WriteFile(bodyPath, bytes.Repeat([]byte("x"), midWriteBodySize), 0o644); err != nil { //coverage:ignore defensive: writing a fresh file under this scenario's own os.MkdirTemp dir has no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("writing seed body: %w", err)
	}
	for _, repo := range []string{"control", "target"} {
		repoDir := filepath.Join(dir, repo)
		if err := os.MkdirAll(repoDir, 0o755); err != nil { //coverage:ignore defensive: repoDir is a fresh subdirectory of this scenario's own os.MkdirTemp dir, no realistic failure mode short of filesystem sabotage
			return fmt.Errorf("creating %s repo dir: %w", repo, err)
		}
		if err := gitInitAndConfig(repoDir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
			return err
		}
		addEnv, err := runAiwfJSON(s.aiwfBin, repoDir, "add", "gap", "--title", "midwrite", "--body-file", bodyPath)
		if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
			return fmt.Errorf("seeding %s repo: %w", repo, err)
		}
		if addEnv.Status != "ok" {
			return fmt.Errorf("seeding %s repo: aiwf did not report ok (status=%s, error=%+v)", repo, addEnv.Status, addEnv.Error)
		}
	}
	return nil
}

// Run promotes the control repo's gap to completion to learn the
// fully-written bytes, then kills the target repo's equivalent
// promote mid-write and classifies the result.
func (s *MidWriteKillScenario) Run(dir string) error {
	const id = "G-0001"
	controlDir := filepath.Join(dir, "control")
	targetDir := filepath.Join(dir, "target")

	wantOldBytes, err := readGapFile(targetDir, id)
	if err != nil {
		return fmt.Errorf("reading target's pre-write bytes: %w", err)
	}

	promoteEnv, err := runAiwfJSON(s.aiwfBin, controlDir, "promote", id, "wontfix")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("running the control promote: %w", err)
	}
	if promoteEnv.Status != "ok" {
		return fmt.Errorf("control promote did not report ok (status=%s, error=%+v)", promoteEnv.Status, promoteEnv.Error)
	}
	wantNewBytes, err := readGapFile(controlDir, id)
	if err != nil { //coverage:ignore defensive: readGapFile's own mismatch/glob branches are pinned directly at their source (TestReadGapFile_ErrorsWhenNoneOrMultipleMatch); reaching this specific call site requires the control repo's gap file to vanish or duplicate strictly between the promote call above and this read, a window no external black-box test can arrange without instrumenting Run itself
		return fmt.Errorf("reading control's fully-written bytes: %w", err)
	}
	if bytes.Equal(wantOldBytes, wantNewBytes) { //coverage:ignore defensive: this scenario's own hardcoded open->wontfix transition always changes the status field; guards against a future edit accidentally picking a no-op transition, which would silently defeat the before/after oracle
		return fmt.Errorf("control promote produced no byte change — the scenario's before/after oracle needs a real difference to distinguish")
	}

	targetCmd := exec.Command(s.aiwfBin, "promote", id, "wontfix") //nolint:gosec // aiwfBin is a path this package's own BuildBinary just produced, not attacker-controlled input
	targetCmd.Dir = targetDir
	if startErr := targetCmd.Start(); startErr != nil { //coverage:ignore defensive: same launch-failure class already pinned at its source (TestMidWriteKillScenario_RealBinary_ErrorsWhenBinaryMissing) — s.aiwfBin is the identical path Setup's own runAiwfJSON calls already proved fails identically when invalid
		return fmt.Errorf("starting target promote: %w", startErr)
	}

	found, err := waitForTempFile(filepath.Join(targetDir, "work", "gaps"), s.readyTimeout)
	if err != nil { //coverage:ignore defensive: os.ReadDir on a directory this scenario itself just created and is still driving has no realistic failure mode
		_ = targetCmd.Process.Kill()
		_ = targetCmd.Wait()
		return fmt.Errorf("watching for the sibling temp file: %w", err)
	}
	if !found {
		_ = targetCmd.Process.Kill()
		_ = targetCmd.Wait()
		return fmt.Errorf("timed out waiting to observe the sibling temp file — never caught the write in flight")
	}

	if killErr := targetCmd.Process.Kill(); killErr != nil { //coverage:ignore defensive: killing the process this scenario itself just started, confirmed alive moments ago, has no realistic failure mode on the unix platforms this package targets
		return fmt.Errorf("killing target promote: %w", killErr)
	}
	waitErr := targetCmd.Wait()
	if !processWasSignaled(waitErr) { //coverage:ignore defensive: processWasSignaled's own branches are pinned directly at their source (TestProcessWasSignaled); forcing THIS call site's false case needs the just-observed-writing subprocess to finish and exit cleanly in the narrow instant between detecting the temp file and this immediate Kill() call, not a race any test can win or lose on demand
		return fmt.Errorf("expected the target promote to terminate by signal (SIGKILL), got: %v", waitErr) //nolint:errorlint // waitErr may be nil (a clean exit has no cause to wrap); this is diagnostic text, not meant for errors.Is/As
	}

	gotBytes, err := readGapFile(targetDir, id)
	if err != nil { //coverage:ignore defensive: readGapFile's own mismatch/glob branches are pinned directly at their source (TestReadGapFile_ErrorsWhenNoneOrMultipleMatch); reaching this specific call site requires the target repo's gap file to vanish or duplicate strictly between the kill above and this read, a window no external black-box test can arrange without instrumenting Run itself
		return fmt.Errorf("reading target's post-kill bytes: %w", err)
	}
	s.violations = classifyMidWriteKillOutcome(wantOldBytes, wantNewBytes, gotBytes)
	return nil
}

// Verify returns the violations Run collected.
func (s *MidWriteKillScenario) Verify(_ string) []Violation {
	return s.violations
}

// readGapFile reads the one gap entity file id names under root's
// work/gaps/ directory, tolerating any slug.
func readGapFile(root, id string) ([]byte, error) {
	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", id+"-*.md"))
	if err != nil { //coverage:ignore defensive: the only error filepath.Glob returns is ErrBadPattern, and this package's own literal pattern is well-formed by construction
		return nil, fmt.Errorf("globbing for gap %s under %s: %w", id, root, err)
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("expected exactly one gap file for %s under %s, found %d: %v", id, root, len(matches), matches)
	}
	return os.ReadFile(matches[0])
}

// waitForTempFile busy-polls dir for a sibling temp file matching
// pathutil.AtomicWriteFile's own ".aiwf-tmp-" naming convention
// (already documented by that package — this reads an existing,
// unmodified side effect, per M-0242/AC-3), returning true the
// instant one appears or false if timeout elapses first. A tight,
// sleep-free loop is deliberate: the write's visible window is
// calibrated to tens of milliseconds (see midWriteBodySize), so
// maximizing poll frequency is what makes the catch reliable rather
// than a matter of luck.
func waitForTempFile(dir string, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return false, err
		}
		for _, e := range entries {
			if strings.Contains(e.Name(), ".aiwf-tmp-") {
				return true, nil
			}
		}
	}
	return false, nil
}

// classifyMidWriteKillOutcome judges one mid-write-kill attempt: got
// must match either the pre-write or the fully-written bytes exactly.
// Anything else is a half-written (or otherwise corrupted) file.
func classifyMidWriteKillOutcome(wantOldBytes, wantNewBytes, gotBytes []byte) []Violation {
	if bytes.Equal(gotBytes, wantOldBytes) || bytes.Equal(gotBytes, wantNewBytes) {
		return nil
	}
	return []Violation{{Message: fmt.Sprintf(
		"entity file after a mid-write kill matched neither the pre-write (%d bytes) nor the fully-written (%d bytes) content — got %d bytes: a half-written file",
		len(wantOldBytes), len(wantNewBytes), len(gotBytes))}}
}
