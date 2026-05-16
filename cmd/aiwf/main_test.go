package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

func TestRun_NoArgs_UsageError(t *testing.T) {
	t.Parallel()
	if got := run(nil); got != cliutil.ExitUsage {
		t.Errorf("run(nil) = %d, want %d", got, cliutil.ExitUsage)
	}
}

func TestRun_UnknownVerb_UsageError(t *testing.T) {
	t.Parallel()
	if got := run([]string{"yodel"}); got != cliutil.ExitUsage {
		t.Errorf("run(yodel) = %d, want %d", got, cliutil.ExitUsage)
	}
}

func TestRun_HelpVariants(t *testing.T) {
	t.Parallel()
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			if got := run([]string{arg}); got != cliutil.ExitOK {
				t.Errorf("run(%q) = %d, want %d", arg, got, cliutil.ExitOK)
			}
		})
	}
}

// TestRun_SubverbHelpDoesNotRecurse pins the SetHelpFunc inheritance
// fix (M-061 AC-5). Pre-fix, `aiwf <subverb> --help` re-entered the
// root's SetHelpFunc through c.Help() and recursed until stack-
// overflow. The fix renders UsageString directly for non-root
// commands. A regression here would either crash the test (stack
// overflow) or return non-zero — both are caught.
//
// Cases cover one- and multi-level deep subverbs so the fix is
// exercised against every command nesting depth.
func TestRun_SubverbHelpDoesNotRecurse(t *testing.T) {
	cases := [][]string{
		{"check", "--help"},
		{"check", "-h"},
		{"add", "--help"},
		{"add", "ac", "--help"},
		{"promote", "--help"},
		{"render", "--help"},
		{"render", "roadmap", "--help"},
		{"contract", "--help"},
		{"contract", "verify", "--help"},
		{"contract", "recipe", "--help"},
		{"contract", "recipe", "show", "--help"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			captureStdout(t, func() {
				if rc := run(args); rc != cliutil.ExitOK {
					t.Errorf("run(%v) = %d, want cliutil.ExitOK", args, rc)
				}
			})
		})
	}
}

func TestRun_VersionVariants(t *testing.T) {
	t.Parallel()
	for _, arg := range []string{"version", "--version", "-v"} {
		t.Run(arg, func(t *testing.T) {
			if got := run([]string{arg}); got != cliutil.ExitOK {
				t.Errorf("run(%q) = %d, want %d", arg, got, cliutil.ExitOK)
			}
		})
	}
}

func TestRun_CheckEmptyRepo_OK(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if got := run([]string{"check", "--root=" + root}); got != cliutil.ExitOK {
		t.Errorf("run(check on empty) = %d, want %d", got, cliutil.ExitOK)
	}
}

func TestRun_CheckBadFormat_UsageError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if got := run([]string{"check", "--root=" + root, "--format=xml"}); got != cliutil.ExitUsage {
		t.Errorf("got %d, want %d", got, cliutil.ExitUsage)
	}
}

func TestRun_CheckFindsErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Create a milestone with a bad parent reference and a bad status.
	dir := filepath.Join(root, "work", "epics", "E-0001-foo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte(`---
id: E-01
title: Foo
status: active
---
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "M-0001-bar.md"), []byte(`---
id: M-001
title: Bar
status: bogus
parent: E-99
---
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := run([]string{"check", "--root=" + root}); got != cliutil.ExitFindings {
		t.Errorf("got %d, want %d (findings)", got, cliutil.ExitFindings)
	}
}

func TestResolveRoot_ExplicitWins(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	got, err := resolveRoot(tmp)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(tmp)
	if got != abs {
		t.Errorf("got %q, want %q", got, abs)
	}
}

func TestWalkUpFor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "marker.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := walkUpFor(deep, "marker.txt")
	if !ok {
		t.Fatal("not found")
	}
	if got != root {
		t.Errorf("got %q, want %q", got, root)
	}
	if _, ok := walkUpFor(deep, "nonsuch.txt"); ok {
		t.Errorf("nonsuch.txt should not be found")
	}
}

// setupCLITestRepo gives the test process a git identity and an
// initialized repo; returns the repo root.
//
// Hook discipline: every test calling `aiwf init` via this in-process
// dispatcher must pass `--skip-hook` unless it specifically wants to
// verify hook installation. The hook bakes in `os.Executable()`,
// which under `go test` resolves to the test binary — letting git
// then exec the test binary as a hook can hang or behave
// unpredictably (deadlocked `aiwf add` integration tests historically).
// Tests that need consumer-parity hook firing should use the
// runBin-style subprocess pattern (see auditonly_cmd_test.go,
// authorize_cmd_test.go) where a real aiwf binary is built and
// driven as a child process.
//
// Pre-G48 this helper redirected `core.hooksPath` to a non-existent
// directory so git's hook lookup would miss aiwf's hooks at
// `.git/hooks/`. That worked because `aiwf init` was buggy and
// installed to `.git/hooks/` regardless of `core.hooksPath`. After
// G48 the install path follows `core.hooksPath`, so the redirect
// would now actually surface aiwf's hooks at the configured path
// and re-introduce the test-binary-as-hook hazard. The discipline
// above (explicit `--skip-hook`) replaces it.
func setupCLITestRepo(t *testing.T) string {
	t.Helper()
	// GIT identity is seeded once by TestMain (setup_test.go) via
	// os.Setenv — t.Setenv would panic under t.Parallel.
	root := t.TempDir()
	if got := run([]string{"check", "--root=" + root}); got != cliutil.ExitOK {
		t.Fatalf("baseline check on tmpdir = %d", got)
	}
	// Initialize git repo so the verb can commit.
	if err := osExec(t, root, "git", "init", "-q"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return root
}

// osExec runs a command in workdir. Returns the error if any.
func osExec(t *testing.T, workdir, name string, args ...string) error {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("%s output: %s", name, out)
	}
	return err
}

// TestRun_AddVerbThroughDispatcher verifies the `add` subcommand wires
// through main's dispatcher: flags parse, actor resolves, the verb
// runs, and a commit lands.
func TestRun_AddVerbThroughDispatcher(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	got := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root})
	if got != cliutil.ExitOK {
		t.Fatalf("run(add epic) = %d, want %d", got, cliutil.ExitOK)
	}
	if _, err := os.Stat(filepath.Join(root, "work", "epics", "E-0001-foundations", "epic.md")); err != nil {
		t.Errorf("epic.md missing after add: %v", err)
	}
	if got := run([]string{"check", "--root", root}); got != cliutil.ExitOK {
		t.Errorf("post-add check = %d, want %d", got, cliutil.ExitOK)
	}
}

// TestRun_AddThenPromoteThenCancel exercises the verb chain through
// the dispatcher to confirm flag handling and commit ordering.
func TestRun_AddThenPromoteThenCancel(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"}); rc != cliutil.ExitOK {
		t.Fatalf("promote: %d", rc)
	}
	if rc := run([]string{"cancel", "--actor", "human/test", "--root", root, "E-0001"}); rc != cliutil.ExitOK {
		t.Fatalf("cancel: %d", rc)
	}
	if rc := run([]string{"check", "--root", root}); rc != cliutil.ExitOK {
		t.Errorf("final check: %d", rc)
	}
}

// TestRun_AddBadKind reports a usage error without touching the repo.
func TestRun_AddBadKind(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if got := run([]string{"add", "widget", "--title", "X", "--actor", "human/test", "--root", root}); got != cliutil.ExitUsage {
		t.Errorf("got %d, want %d", got, cliutil.ExitUsage)
	}
}

// TestRun_PromoteMissingArgs reports a usage error.
func TestRun_PromoteMissingArgs(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if got := run([]string{"promote", "--root", root, "M-0001"}); got != cliutil.ExitUsage {
		t.Errorf("got %d, want %d (missing new-status)", got, cliutil.ExitUsage)
	}
}
