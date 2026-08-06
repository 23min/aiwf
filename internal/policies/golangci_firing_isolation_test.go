//go:build !windows

package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

// TestGolangciFixtureCmd_RunsWhileAnotherRunnerHoldsTheLock is the
// behavioral pin for the firing harness's isolation: the harness must
// reach a verdict about the config while another golangci-lint holds the
// start-up lock, because on this repo that is the ordinary case — several
// worktrees, a `make lint`, a pre-push hook, an editor integration.
//
// Without --allow-parallel-runners the child retries the lock until
// run.timeout and then exits carrying no findings, which the harness
// reads as every rule being dormant at once.
//
// The lock is routed to a private TMPDIR rather than the machine-global
// os.TempDir(): golangci-lint derives its lock path from the child's temp
// dir, so holding the real one would inflict on every concurrent
// golangci-lint on the machine exactly the stall this test exists to
// prevent. That routing is also what makes the test deterministic instead
// of dependent on a real second runner being scheduled.
//
// The `!windows` constraint follows internal/repolock's split: flock(2)
// is POSIX, and CI runs Linux only. The measures under test are a flag
// and an env var, and are platform-independent — only this way of holding
// the lock is not.
func TestGolangciFixtureCmd_RunsWhileAnotherRunnerHoldsTheLock(t *testing.T) {
	t.Parallel()

	bin := requireGolangci(t)

	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "golangci-lint.lock")

	held, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("creating the stand-in lock at %s: %v", lockPath, err)
	}
	defer func() {
		if err := held.Close(); err != nil {
			t.Errorf("closing the stand-in lock: %v", err)
		}
	}()

	//nolint:gosec // Fd returns a small fd; the int conversion is safe for any value the runtime returns (same rationale as internal/repolock)
	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("taking the stand-in lock at %s: %v", lockPath, err)
	}

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module fixture_isolation\n\ngo 1.24\n")
	mustWrite(t, filepath.Join(dir, "bad.go"),
		"package fixture\n\n// Boom is library code that must not panic.\nfunc Boom() {\n\tpanic(\"library code must not panic\")\n}\n")

	cmd := golangciFixtureCmd(bin, filepath.Join(repoRoot(t), ".golangci.yml"), dir, t.TempDir())
	// Appended last so it wins: os/exec documents that the last value for
	// a duplicated key is the one used. This is what puts the child's lock
	// at the path held above.
	cmd.Env = append(cmd.Env, "TMPDIR="+tempDir)

	out, _ := cmd.CombinedOutput()

	if golangciRefusedForConcurrency(string(out)) {
		t.Fatalf("golangci-lint refused to start while another runner held %s — the firing harness must not depend on being the only instance on the machine.\n--- golangci-lint output ---\n%s", lockPath, out)
	}
	if !strings.Contains(golangciFindingMessages(string(out)), "(forbidigo)") {
		t.Fatalf("the fixture's forbidigo violation must still be reported when the run is not refused; got:\n%s", out)
	}
}

// TestGolangciFixtureCmd_RefusesWithoutTheFlag is the negative control
// for the test above, and the only thing pinning golangciRefusedForConcurrency's
// literal to reality: it provokes a real refusal from the real binary and
// asserts the predicate recognizes those bytes.
//
// Without it the predicate is a hand-copied string checked against a
// second hand-copied copy of itself. Drift them together — or let
// golangci-lint reword the message — and the predicate goes permanently
// false, the harness silently returns to reporting a contended run as a
// dormant rule, and every test stays green. This is the independent
// derivation CLAUDE.md § "Contract tests for upstream-cached systems"
// asks for.
//
// It doubles as the pin on golangciLockPath()'s filename claim: the
// refusal only happens if the child really does look for its lock at
// <TMPDIR>/golangci-lint.lock.
func TestGolangciFixtureCmd_RefusesWithoutTheFlag(t *testing.T) {
	t.Parallel()

	bin := requireGolangci(t)

	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "golangci-lint.lock")

	held, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("creating the stand-in lock at %s: %v", lockPath, err)
	}
	defer func() {
		if err := held.Close(); err != nil {
			t.Errorf("closing the stand-in lock: %v", err)
		}
	}()

	//nolint:gosec // Fd returns a small fd; the int conversion is safe for any value the runtime returns (same rationale as internal/repolock)
	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("taking the stand-in lock at %s: %v", lockPath, err)
	}

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module fixture_refusal\n\ngo 1.24\n")
	mustWrite(t, filepath.Join(dir, "bad.go"),
		"package fixture\n\n// Boom is library code that must not panic.\nfunc Boom() {\n\tpanic(\"library code must not panic\")\n}\n")

	// The harness command minus the one flag under test, so the refusal
	// this provokes is the exact condition golangciFixtureCmd prevents.
	cmd := golangciFixtureCmd(bin, filepath.Join(repoRoot(t), ".golangci.yml"), dir, t.TempDir())
	cmd.Args = slices.DeleteFunc(cmd.Args, func(arg string) bool { return arg == "--allow-parallel-runners" })
	cmd.Env = append(cmd.Env, "TMPDIR="+tempDir)

	out, _ := cmd.CombinedOutput()

	if !golangciRefusedForConcurrency(string(out)) {
		t.Fatalf("golangci-lint %s did not produce output golangciRefusedForConcurrency recognizes when another runner held %s.\n\nEither upstream reworded the refusal — in which case the predicate is now permanently false and the harness will report a contended run as a dormant rule — or it no longer takes a lock at that path, in which case --allow-parallel-runners and this pair of tests are obsolete. Both need a human.\n--- golangci-lint output ---\n%s",
			golangciVersion(t, bin), lockPath, out)
	}
}

// golangciVersion reports the binary's version for a failure message, so
// a refusal that stops reproducing names the version it stopped on.
func golangciVersion(t *testing.T, bin string) string {
	t.Helper()

	out, err := exec.Command(bin, "version").CombinedOutput()
	if err != nil {
		return "(version unavailable: " + err.Error() + ")"
	}
	return strings.TrimSpace(string(out))
}
