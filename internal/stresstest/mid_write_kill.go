package stresstest

import (
	"bytes"
	"errors"
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
// temp-file-visible window is comfortably wide for the poll to catch
// deterministically, not a matter of racing a near-instant syscall
// sequence.
//
// Catching that window is a sampling problem, and failing to sample it
// says nothing about aiwf. So the observation is bounded by the
// watched process rather than by a clock — the poller stops when the
// promote writes or exits, whichever comes first — and a promote that
// finishes unsampled is retried from the same starting state rather
// than reported as a failure (G-0468).

// midWriteBodySize is calibrated (empirically, against
// pathutil.AtomicWriteFile directly) to give the temp-file-visible
// window tens of milliseconds of width — comfortably wide for
// observeTempFile's poll to catch reliably on any machine this runs
// on, not a matter of winning a microsecond-scale race.
const midWriteBodySize = 10_000_000

// defaultMidWriteHangGuard bounds one attempt so a wedged subprocess
// cannot hang the harness. It is not a budget for how long the write
// may take: the observation ends when the promote writes or exits,
// whichever comes first, so a loaded machine reaches its verdict at
// the pace the machine runs rather than failing at a deadline.
const defaultMidWriteHangGuard = 60 * time.Second

// midWriteAttempts bounds how many times Run restarts the promote
// after one runs to completion unsampled. Missing is a failure to
// observe rather than evidence about aiwf, so it is retried; a repo
// where every attempt slips past the poller reports that plainly
// instead of being retried forever.
const midWriteAttempts = 3

// midWritePollInterval paces the scan for the sibling temp file. The
// visible window is tens of milliseconds wide (see midWriteBodySize),
// so this samples it hundreds of times over, while leaving the CPU to
// the process being watched — a sleepless scan competes with the very
// write it is waiting for, which matters most on exactly the loaded
// machine the observation is hardest on.
const midWritePollInterval = 200 * time.Microsecond

// The conditions one attempt can end in, other than a clean
// observation. Each is a stable condition a caller or test names by
// identity rather than by matching its message.
var (
	// errMidWriteHangGuard reports a target promote that neither
	// reached its write nor exited within the hang guard.
	errMidWriteHangGuard = errors.New("the target promote neither wrote nor exited within the hang guard")
	// errMidWriteAllUnsampled reports every attempt completing before
	// the poller sampled its write.
	errMidWriteAllUnsampled = errors.New("the target promote ran to completion unsampled")
	// errMidWritePromoteFailed reports a target promote that exited
	// unsuccessfully, which is evidence about aiwf rather than about
	// sampling and so is never retried.
	errMidWritePromoteFailed = errors.New("the target promote failed before its write was observed")
)

// MidWriteKillScenario implements Scenario.
type MidWriteKillScenario struct {
	aiwfBin string
	// hangGuard bounds one attempt's observation. Defaulted by the
	// constructor; tests in this package may set it directly
	// (same-package struct literal) to force the guard branch
	// deterministically and quickly.
	hangGuard  time.Duration
	violations []Violation
}

// NewMidWriteKillScenario builds a scenario driving aiwfBin (the real
// compiled aiwf binary) against a large-bodied gap entity.
func NewMidWriteKillScenario(aiwfBin string) *MidWriteKillScenario {
	return &MidWriteKillScenario{aiwfBin: aiwfBin, hangGuard: defaultMidWriteHangGuard}
}

// Setup creates dir/control and dir/target as two independent git
// repos, each seeded with one identically-bodied gap entity.
func (s *MidWriteKillScenario) Setup(dir string) error {
	bodyPath := filepath.Join(dir, "body.txt")
	if err := os.WriteFile(bodyPath, bytes.Repeat([]byte("x"), midWriteBodySize), 0o644); err != nil { //coverage:ignore defensive: writing a fresh file under this scenario's own os.MkdirTemp dir has no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("writing seed body: %w", err)
	}
	for _, repo := range []string{"control", "target"} {
		if err := s.seedRepo(dir, filepath.Join(dir, repo)); err != nil {
			return err
		}
	}
	return nil
}

// seedRepo git-inits repoDir and adds the one large-bodied gap entity
// every repo in this scenario starts from, so a control repo, the
// first target, and a retry's fresh target are all seeded identically.
func (s *MidWriteKillScenario) seedRepo(dir, repoDir string) error {
	repo := filepath.Base(repoDir)
	if err := os.MkdirAll(repoDir, 0o755); err != nil { //coverage:ignore defensive: repoDir is a fresh subdirectory of this scenario's own os.MkdirTemp dir, no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("creating %s repo dir: %w", repo, err)
	}
	if err := gitInitAndConfig(repoDir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}
	addEnv, err := runAiwfJSON(s.aiwfBin, repoDir, "add", "gap", "--title", "midwrite", "--body-file", filepath.Join(dir, "body.txt"))
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("seeding %s repo: %w", repo, err)
	}
	if addEnv.Status != "ok" {
		return fmt.Errorf("seeding %s repo: aiwf did not report ok (status=%s, error=%+v)", repo, addEnv.Status, addEnv.Error)
	}
	return nil
}

// targetDirForAttempt names the repo one attempt drives. The first
// reuses Setup's target; a retry gets a freshly seeded sibling,
// because the promote being retried already committed its change and
// a second run against that repo would be a same-state NoOp that
// writes nothing at all.
func targetDirForAttempt(dir string, attempt int) string {
	if attempt == 1 {
		return filepath.Join(dir, "target")
	}
	return filepath.Join(dir, fmt.Sprintf("target-%d", attempt))
}

// Run promotes the control repo's gap to completion to learn the
// fully-written bytes, then kills the target repo's equivalent
// promote mid-write and classifies the result.
func (s *MidWriteKillScenario) Run(dir string) error {
	const id = "G-0001"
	controlDir := filepath.Join(dir, "control")

	wantOldBytes, err := readGapFile(targetDirForAttempt(dir, 1), id)
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

	for attempt := 1; ; attempt++ {
		gotBytes, observed, attemptErr := s.killTargetMidWrite(targetDirForAttempt(dir, attempt), id)
		if attemptErr != nil {
			return attemptErr
		}
		if observed {
			s.violations = classifyMidWriteKillOutcome(wantOldBytes, wantNewBytes, gotBytes)
			return nil
		}
		if attempt >= midWriteAttempts {
			return fmt.Errorf("%w on all %d attempts — either the poller never sampled the write, or aiwf no longer writes through a sibling temp file", errMidWriteAllUnsampled, midWriteAttempts)
		}
		if seedErr := s.seedRepo(dir, targetDirForAttempt(dir, attempt+1)); seedErr != nil { //coverage:ignore defensive: seedRepo's own branches are pinned at their source (TestMidWriteKillSetup_SurfacesASeedingRefusal); reaching this call site's failure needs the seeding to succeed for Setup's repos and fail for an identically-built sibling
			return fmt.Errorf("seeding a fresh target for attempt %d: %w", attempt+1, seedErr)
		}
	}
}

// killTargetMidWrite runs one attempt: it starts the target's promote,
// watches for AtomicWriteFile's sibling temp file, and SIGKILLs the
// process the instant one appears. observed reports whether the write
// was caught in flight; false means the promote ran to completion
// before the poller sampled it, which is a failure to observe rather
// than evidence about aiwf, and leaves the caller free to retry.
func (s *MidWriteKillScenario) killTargetMidWrite(targetDir, id string) (gotBytes []byte, observed bool, err error) {
	targetCmd := exec.Command(s.aiwfBin, "promote", id, "wontfix") //nolint:gosec // aiwfBin is a path this package's own BuildBinary just produced, not attacker-controlled input
	targetCmd.Dir = targetDir
	if startErr := targetCmd.Start(); startErr != nil { //coverage:ignore defensive: same launch-failure class already pinned at its source (TestMidWriteKillScenario_RealBinary_ErrorsWhenBinaryMissing) — s.aiwfBin is the identical path Setup's own runAiwfJSON calls already proved fails identically when invalid
		return nil, false, fmt.Errorf("starting target promote: %w", startErr)
	}
	exited := make(chan error, 1)
	go func() { exited <- targetCmd.Wait() }()

	seen, exitErr, watchErr := observeTempFile(filepath.Join(targetDir, "work", "gaps"), exited, s.hangGuard)
	if watchErr != nil {
		_ = targetCmd.Process.Kill()
		<-exited
		return nil, false, fmt.Errorf("watching for the sibling temp file: %w", watchErr)
	}
	if !seen {
		if exitErr != nil {
			return nil, false, fmt.Errorf("%w: %w", errMidWritePromoteFailed, exitErr)
		}
		return nil, false, nil
	}

	if killErr := targetCmd.Process.Kill(); killErr != nil { //coverage:ignore defensive: killing the process this scenario itself just started, confirmed alive moments ago, has no realistic failure mode on the unix platforms this package targets
		return nil, false, fmt.Errorf("killing target promote: %w", killErr)
	}
	if waitErr := <-exited; !processWasSignaled(waitErr) { //coverage:ignore defensive: processWasSignaled's own branches are pinned directly at their source (TestProcessWasSignaled); forcing THIS call site's false case needs the just-observed-writing subprocess to finish and exit cleanly in the narrow instant between detecting the temp file and this immediate Kill() call, not a race any test can win or lose on demand
		return nil, false, fmt.Errorf("expected the target promote to terminate by signal (SIGKILL), got: %v", waitErr) //nolint:errorlint // waitErr may be nil (a clean exit has no cause to wrap); this is diagnostic text, not meant for errors.Is/As
	}

	gotBytes, err = readGapFile(targetDir, id)
	if err != nil { //coverage:ignore defensive: readGapFile's own mismatch/glob branches are pinned directly at their source (TestReadGapFile_ErrorsWhenNoneOrMultipleMatch); reaching this specific call site requires the target repo's gap file to vanish or duplicate strictly between the kill above and this read, a window no external black-box test can arrange without instrumenting Run itself
		return nil, false, fmt.Errorf("reading target's post-kill bytes: %w", err)
	}
	return gotBytes, true, nil
}

// Verify returns the violations Run collected.
func (s *MidWriteKillScenario) Verify(_ string) []Violation {
	return s.violations
}

// observeTempFile polls dir for AtomicWriteFile's sibling temp file
// while the process feeding exited is still running, returning
// observed the instant one appears.
//
// The watched process bounds the wait, not a clock: a promote that
// finishes without the poller ever sampling its temp file returns
// observed false along with the process's own exit result, which is
// what makes a miss immediately distinguishable from a slow machine.
// hangGuard is the backstop for a process that neither writes nor
// exits, so it is set generously and fires only on a genuine wedge.
//
// Who drains exited is keyed on err, not on observed. When err is nil
// the exit result has been consumed and is handed back as exitErr, so
// the caller must not read the channel; when err is non-nil the
// process is still running and the caller drains it after killing.
func observeTempFile(dir string, exited <-chan error, hangGuard time.Duration) (observed bool, exitErr, err error) {
	guard := time.After(hangGuard)
	for {
		// Exit is checked ahead of the scan so a finished process is
		// never sent a signal: AtomicWriteFile renames its temp file
		// away before returning, so a promote that has exited has no
		// temp file left to observe.
		select {
		case waitErr := <-exited:
			return false, waitErr, nil
		case <-guard:
			return false, nil, errMidWriteHangGuard
		default:
		}
		present, readErr := tempFilePresent(dir)
		if readErr != nil {
			return false, nil, readErr
		}
		if present {
			return true, nil, nil
		}
		time.Sleep(midWritePollInterval)
	}
}

// tempFilePresent reports whether dir currently holds one of
// pathutil.AtomicWriteFile's sibling temp files, matching the
// ".aiwf-tmp-" naming convention that package's own doc comment
// documents — this reads an existing, unmodified side effect, per
// M-0242/AC-3.
func tempFilePresent(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".aiwf-tmp-") {
			return true, nil
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
