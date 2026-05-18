package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

func TestRun_NoArgs_UsageError(t *testing.T) {
	t.Parallel()
	if got := cli.Execute(nil); got != cliutil.ExitUsage {
		t.Errorf("cli.Execute(nil) = %d, want %d", got, cliutil.ExitUsage)
	}
}

func TestRun_UnknownVerb_UsageError(t *testing.T) {
	t.Parallel()
	if got := cli.Execute([]string{"yodel"}); got != cliutil.ExitUsage {
		t.Errorf("run(yodel) = %d, want %d", got, cliutil.ExitUsage)
	}
}

func TestRun_HelpVariants(t *testing.T) {
	t.Parallel()
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			if got := cli.Execute([]string{arg}); got != cliutil.ExitOK {
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
			testutil.CaptureStdout(t, func() {
				if rc := cli.Execute(args); rc != cliutil.ExitOK {
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
			if got := cli.Execute([]string{arg}); got != cliutil.ExitOK {
				t.Errorf("run(%q) = %d, want %d", arg, got, cliutil.ExitOK)
			}
		})
	}
}

func TestRun_CheckEmptyRepo_OK(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if got := cli.Execute([]string{"check", "--root=" + root}); got != cliutil.ExitOK {
		t.Errorf("run(check on empty) = %d, want %d", got, cliutil.ExitOK)
	}
}

func TestRun_CheckBadFormat_UsageError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if got := cli.Execute([]string{"check", "--root=" + root, "--format=xml"}); got != cliutil.ExitUsage {
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
	if got := cli.Execute([]string{"check", "--root=" + root}); got != cliutil.ExitFindings {
		t.Errorf("got %d, want %d (findings)", got, cliutil.ExitFindings)
	}
}

// TestRun_AddVerbThroughDispatcher verifies the `add` subcommand wires
// through main's dispatcher: flags parse, actor resolves, the verb
// runs, and a commit lands.
func TestRun_AddVerbThroughDispatcher(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	got := cli.Execute([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root})
	if got != cliutil.ExitOK {
		t.Fatalf("run(add epic) = %d, want %d", got, cliutil.ExitOK)
	}
	if _, err := os.Stat(filepath.Join(root, "work", "epics", "E-0001-foundations", "epic.md")); err != nil {
		t.Errorf("epic.md missing after add: %v", err)
	}
	if got := cli.Execute([]string{"check", "--root", root}); got != cliutil.ExitOK {
		t.Errorf("post-add check = %d, want %d", got, cliutil.ExitOK)
	}
}

// TestRun_AddThenPromoteThenCancel exercises the verb chain through
// the dispatcher to confirm flag handling and commit ordering.
func TestRun_AddThenPromoteThenCancel(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	if rc := cli.Execute([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := cli.Execute([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"}); rc != cliutil.ExitOK {
		t.Fatalf("promote: %d", rc)
	}
	if rc := cli.Execute([]string{"cancel", "--actor", "human/test", "--root", root, "E-0001"}); rc != cliutil.ExitOK {
		t.Fatalf("cancel: %d", rc)
	}
	if rc := cli.Execute([]string{"check", "--root", root}); rc != cliutil.ExitOK {
		t.Errorf("final check: %d", rc)
	}
}

// TestRun_AddBadKind reports a usage error without touching the repo.
func TestRun_AddBadKind(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if got := cli.Execute([]string{"add", "widget", "--title", "X", "--actor", "human/test", "--root", root}); got != cliutil.ExitUsage {
		t.Errorf("got %d, want %d", got, cliutil.ExitUsage)
	}
}

// TestRun_PromoteMissingArgs reports a usage error.
func TestRun_PromoteMissingArgs(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if got := cli.Execute([]string{"promote", "--root", root, "M-0001"}); got != cliutil.ExitUsage {
		t.Errorf("got %d, want %d (missing new-status)", got, cliutil.ExitUsage)
	}
}
