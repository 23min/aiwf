package stresstest

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mid_write_kill_retry_test.go — how MidWriteKillScenario.Run disposes
// of an attempt, and how LockKillScenario.Run disposes of a holder
// that never speaks. Both drive a stand-in rather than the real aiwf
// binary, because the behavior under test is *timing the real thing
// cannot be asked for*: a promote that finishes before the poller
// samples it, and a holder that neither reports nor exits. The real
// binary's own runs stay in mid_write_kill_test.go and
// lock_kill_test.go, which is where the scenario is proved end to end;
// nothing here starts a race, so it belongs in the lane that runs on
// every push.

// fakeAiwf writes a stand-in for the aiwf binary that serves the two
// verbs this scenario calls. `add gap` seeds the entity file Setup
// expects; a promote in the control repo rewrites it, so the
// before/after oracle has a real difference to distinguish; a promote
// in a target repo runs targetBehavior, which is where a test dictates
// whether the write is sampleable.
func fakeAiwf(t *testing.T, dir, targetBehavior string) (bin, countFile string) {
	t.Helper()
	bin = filepath.Join(dir, "fake-aiwf")
	countFile = filepath.Join(dir, "target-promotes")
	script := `#!/bin/sh
set -e
gapfile=work/gaps/G-0001-midwrite.md
case "$(basename "$PWD")" in target*) target=1 ;; *) target=0 ;; esac
case "$1" in
  add)
    mkdir -p work/gaps
    printf -- '---\nid: G-0001\nstatus: open\n---\nbody\n' > "$gapfile"
    echo '{"tool":"aiwf","status":"ok"}'
    ;;
  promote)
    if [ "$target" = 1 ]; then
      echo x >> '` + countFile + `'
      n=$(wc -l < '` + countFile + `')
      ` + targetBehavior + `
    else
      printf -- '---\nid: G-0001\nstatus: wontfix\n---\nbody\n' > "$gapfile"
    fi
    echo '{"tool":"aiwf","status":"ok"}'
    ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil { //nolint:gosec // an executable stand-in is the point, and it lives in this test's own t.TempDir
		t.Fatalf("writing the aiwf stand-in: %v", err)
	}
	return bin, countFile
}

// countPromotes reports how many target promotes the stand-in served.
func countPromotes(t *testing.T, countFile string) int {
	t.Helper()
	data, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatalf("reading the promote counter: %v", err)
	}
	return strings.Count(string(data), "\n")
}

// TestMidWriteKillRun_RetriesAnUnsampledPromote pins the disposition
// of a miss: a promote that completes before the poller sees its temp
// file is retried, bounded, and reported only once every attempt has
// missed. Without the retry one unlucky sample fails the scenario,
// which is the flake G-0468 removes; without the bound a repo that is
// never sampled would spin.
func TestMidWriteKillRun_RetriesAnUnsampledPromote(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// ":" is the shell's no-op: the stand-in returns without ever
	// writing, so every attempt completes unsampled.
	bin, countFile := fakeAiwf(t, dir, ":")

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	err := s.Run(dir)
	if !errors.Is(err, errMidWriteAllUnsampled) {
		t.Fatalf("expected errMidWriteAllUnsampled, got: %v", err)
	}
	if got := countPromotes(t, countFile); got != midWriteAttempts {
		t.Errorf("the target promote ran %d times, want %d", got, midWriteAttempts)
	}
}

// TestMidWriteKillRun_RecoversOnALaterAttempt pins the other half: a
// first attempt that misses does not lose the run. The stand-in
// completes unsampled once, then leaves a temp file in place and
// blocks, so the second attempt is caught in flight and the scenario
// reaches its real verdict — against a fresh repo, since the first
// attempt's promote already committed its own change.
func TestMidWriteKillRun_RecoversOnALaterAttempt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// exec so the SIGKILL lands on the blocking process itself rather
	// than on a shell that would leave it orphaned.
	behavior := `if [ "$n" -ge 2 ]; then touch work/gaps/probe.aiwf-tmp-fake; exec sleep 30; fi`
	bin, countFile := fakeAiwf(t, dir, behavior)

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if err := s.Run(dir); err != nil {
		t.Fatalf("Run should have recovered on the second attempt: %v", err)
	}
	if violations := s.Verify(dir); len(violations) != 0 {
		t.Errorf("Verify = %+v, want none: the entity file was never written, so it still matches the pre-write bytes", violations)
	}
	if got := countPromotes(t, countFile); got != 2 {
		t.Errorf("the target promote ran %d times, want 2 — one miss, then one observed", got)
	}

	// The oracle compares a later attempt's file against bytes read
	// once, from the first attempt's repo, so a reseed that produced
	// anything different would have the scenario judging a retry
	// against the wrong baseline.
	first, err := readGapFile(targetDirForAttempt(dir, 1), "G-0001")
	if err != nil {
		t.Fatalf("reading the first attempt's entity file: %v", err)
	}
	second, err := readGapFile(targetDirForAttempt(dir, 2), "G-0001")
	if err != nil {
		t.Fatalf("reading the retry's entity file: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("a reseeded repo differs from the first attempt's; the retry would be judged against the wrong baseline")
	}
}

// TestMidWriteKillSetup_SurfacesASeedingRefusal pins that a refused
// seed stops Setup rather than leaving a repo the scenario would go on
// to measure.
func TestMidWriteKillSetup_SurfacesASeedingRefusal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin := filepath.Join(dir, "refusing-aiwf")
	script := "#!/bin/sh\necho '{\"tool\":\"aiwf\",\"status\":\"error\",\"error\":{\"code\":\"refused\",\"message\":\"no\"}}'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil { //nolint:gosec // an executable stand-in is the point, and it lives in this test's own t.TempDir
		t.Fatalf("writing the refusing stand-in: %v", err)
	}

	s := NewMidWriteKillScenario(bin)
	err := s.Setup(dir)
	if err == nil || !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected Setup to surface the refusal, got: %v", err)
	}
}

// TestMidWriteKillRun_SurfacesAPromoteThatFailed pins the difference
// between a promote that finished unsampled and one that finished
// badly: the first is retried, the second is reported, because a
// failing promote is evidence about aiwf rather than about sampling.
func TestMidWriteKillRun_SurfacesAPromoteThatFailed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin, _ := fakeAiwf(t, dir, "exit 1")

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	err := s.Run(dir)
	if !errors.Is(err, errMidWritePromoteFailed) {
		t.Fatalf("expected errMidWritePromoteFailed rather than a retry, got: %v", err)
	}
}

// TestMidWriteKillRun_ErrorsWhenTheHangGuardFires pins the backstop: a
// promote that neither writes nor exits is killed and reported rather
// than waited on forever.
func TestMidWriteKillRun_ErrorsWhenTheHangGuardFires(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin, _ := fakeAiwf(t, dir, "exec sleep 30")

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	s.hangGuard = 50 * time.Millisecond

	if err := s.Run(dir); !errors.Is(err, errMidWriteHangGuard) {
		t.Fatalf("expected errMidWriteHangGuard, got: %v", err)
	}
}

// TestLockKillRun_HangGuardFiresOnASilentHolder pins the one branch a
// healthy holder never reaches: a process that neither reports
// ACQUIRED nor closes its stdout. A holder that dies is caught by its
// stdout closing at any speed of machine, so this is all the clock is
// left to catch.
func TestLockKillRun_HangGuardFiresOnASilentHolder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := gitInitAndConfig(dir); err != nil {
		t.Fatalf("gitInitAndConfig: %v", err)
	}
	holder := filepath.Join(dir, "silent-holder")
	// Never prints, never exits — and holds no lock, which it does not
	// need to: the guard fires before any probe.
	if err := os.WriteFile(holder, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil { //nolint:gosec // an executable stand-in is the point, and it lives in this test's own t.TempDir
		t.Fatalf("writing the silent holder: %v", err)
	}

	s := &LockKillScenario{lockHolderBin: holder, hangGuard: 50 * time.Millisecond}
	if err := s.Run(dir); !errors.Is(err, errHolderWedged) {
		t.Fatalf("expected errHolderWedged from a holder that never speaks, got: %v", err)
	}
}
