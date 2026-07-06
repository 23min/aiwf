package policies

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/skills"
)

// M-0236/AC-3: a subprocess-level policy test mirroring
// TestAgentIsolationHook_* (agent_isolation_hook_test.go) — pins the
// worktree-rituals-check.sh hook script's real, end-to-end contract
// rather than just the Go-level --check-rituals mechanism AC-1's own
// tests cover.

var (
	worktreeHookBinDirOnce sync.Once
	worktreeHookBinDir     string
	worktreeHookBinErr     error
)

// worktreeHookTestBinDir builds the aiwf binary once per test binary run
// into a directory the tests in this file prepend to PATH — the hook
// script execs `aiwf doctor --check-rituals`, so a real binary must be
// reachable for the subprocess to exercise the actual contract, not a
// stubbed one. Uses os.MkdirTemp (not t.TempDir()) since the directory
// must outlive whichever individual test happens to build it first.
// // do not mutate
func worktreeHookTestBinDir(t *testing.T) string {
	t.Helper()
	worktreeHookBinDirOnce.Do(func() {
		dir, err := os.MkdirTemp("", "aiwf-hook-test-bin-")
		if err != nil {
			worktreeHookBinErr = err
			return
		}
		build := exec.Command("go", "build", "-o", filepath.Join(dir, "aiwf"), "./cmd/aiwf")
		build.Dir = repoRoot(t)
		build.Env = append(os.Environ(), "CGO_ENABLED=0")
		if out, buildErr := build.CombinedOutput(); buildErr != nil {
			worktreeHookBinErr = fmt.Errorf("go build ./cmd/aiwf: %w\n%s", buildErr, out)
			return
		}
		worktreeHookBinDir = dir
	})
	if worktreeHookBinErr != nil {
		t.Fatalf("worktreeHookTestBinDir: %v", worktreeHookBinErr)
	}
	return worktreeHookBinDir
}

// writeWorktreeHookScript writes the embedded hook script's real bytes
// (the same ones skills.ShippedHooks materializes) to an executable file
// inside dir, returning its path.
func writeWorktreeHookScript(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "worktree-rituals-check.sh")
	if err := os.WriteFile(path, skills.WorktreeRitualsCheckScript, 0o755); err != nil {
		t.Fatalf("writing hook script: %v", err)
	}
	return path
}

// runWorktreeHook execs the hook script with cwd=workDir and PATH
// prefixed with the built aiwf binary's directory, returning stdout,
// stderr, and the exit code.
func runWorktreeHook(t *testing.T, scriptPath, workDir, binDir string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(scriptPath)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	if runErr == nil {
		return outBuf.String(), errBuf.String(), 0
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		return outBuf.String(), errBuf.String(), exitErr.ExitCode()
	}
	t.Fatalf("running hook script: %v", runErr)
	return "", "", -1
}

// gitInit runs `git init` (plus identity config) in dir so
// `git rev-parse --show-toplevel` resolves inside it, matching the real
// shape of a worktree checkout.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "aiwf-test@example.com")
	run("config", "user.name", "aiwf-test")
}

// TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently pins the
// primary, most-common case: a cwd that isn't under .claude/worktrees/
// (the main checkout) never invokes aiwf at all — exit 0, no output.
func TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently(t *testing.T) {
	t.Parallel()
	binDir := worktreeHookTestBinDir(t)
	scriptDir := t.TempDir()
	scriptPath := writeWorktreeHookScript(t, scriptDir)

	workDir := t.TempDir() // no .claude/worktrees/ segment, not even a git repo
	stdout, stderr, exitCode := runWorktreeHook(t, scriptPath, workDir, binDir)

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0; stderr=%q", exitCode, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("expected silent exit for a non-worktree cwd; stdout=%q stderr=%q", stdout, stderr)
	}
}

// TestWorktreeRitualsCheckHook_HealthyWorktreeExitsZeroSilently pins the
// healthy case: a .claude/worktrees/ checkout whose rituals are fully
// materialized exits 0 with no output.
func TestWorktreeRitualsCheckHook_HealthyWorktreeExitsZeroSilently(t *testing.T) {
	t.Parallel()
	binDir := worktreeHookTestBinDir(t)
	scriptDir := t.TempDir()
	scriptPath := writeWorktreeHookScript(t, scriptDir)

	root := t.TempDir()
	workDir := filepath.Join(root, ".claude", "worktrees", "milestone-fixture-branch")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitInit(t, workDir)
	if _, err := initrepo.Init(context.Background(), workDir, initrepo.Options{SkipHook: true}); err != nil {
		t.Fatalf("initrepo.Init: %v", err)
	}

	stdout, stderr, exitCode := runWorktreeHook(t, scriptPath, workDir, binDir)

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0; stderr=%q", exitCode, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("expected silent exit for a fully-materialized worktree; stdout=%q stderr=%q", stdout, stderr)
	}
}

// TestWorktreeRitualsCheckHook_StaleWorktreeExitsNonzeroWithActionableStderr
// pins the stale/missing case: a .claude/worktrees/ checkout that never
// ran `aiwf init`/`update` (the primary target scenario — an interrupted
// or forgotten `aiwf worktree add`) exits nonzero with a stderr message
// pointing at the remedy.
func TestWorktreeRitualsCheckHook_StaleWorktreeExitsNonzeroWithActionableStderr(t *testing.T) {
	t.Parallel()
	binDir := worktreeHookTestBinDir(t)
	scriptDir := t.TempDir()
	scriptPath := writeWorktreeHookScript(t, scriptDir)

	root := t.TempDir()
	workDir := filepath.Join(root, ".claude", "worktrees", "milestone-fixture-branch")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitInit(t, workDir)
	// Deliberately no initrepo.Init: nothing under .claude/ is materialized.

	stdout, stderr, exitCode := runWorktreeHook(t, scriptPath, workDir, binDir)

	if exitCode == 0 {
		t.Fatalf("exitCode = 0, want nonzero for an unmaterialized worktree; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Errorf("expected no stdout output, got %q", stdout)
	}
	if !strings.Contains(stderr, "aiwf update") {
		t.Errorf("stderr = %q, want it to mention `aiwf update`", stderr)
	}
	// Pins that the script resolved workDir itself — exactly, with no
	// extra trailing path segment — as the root passed to
	// --check-rituals. A broken `git rev-parse --show-toplevel` (e.g. an
	// invalid flag, which git silently echoes back as a bogus relative
	// path rather than erroring) resolves to workDir plus a spurious
	// suffix; a substring-only check on workDir's own name would pass
	// for that wrong reason too, since the bogus path is still prefixed
	// by workDir. The " —" delimiter from the message's own format
	// ("... not materialized under %s — run `aiwf update` ...") pins
	// the path exactly, ruling that out.
	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q): %v", workDir, err)
	}
	if !strings.Contains(stderr, "under "+resolvedWorkDir+" —") {
		t.Errorf("stderr = %q, want it to name exactly %q as the root", stderr, resolvedWorkDir)
	}
}
