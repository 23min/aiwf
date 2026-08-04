package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// noSuchSubverb is the token these tests hand a parent command in place
// of a real subverb. Chosen to be unmistakably absent from every verb
// group so a match in the reported output can only have come from the
// argument the test supplied.
const noSuchSubverb = "zzz-no-such-subverb"

// captureExecuteStreams runs Execute with args, swapping both os.Stdout
// and os.Stderr to a pipe so the test can assert on what the operator
// sees. Both streams are collected together because the split between
// them is not what these tests pin — a usage rejection reaches stderr,
// while the help body a verb group prints on a bare invocation goes
// through Cobra's OutOrStderr. os.Stdout and os.Stderr are mutated, so
// callers sit on the serial-skip list in setup_test.go.
func captureExecuteStreams(t *testing.T, args []string) (combined string, exitCode int) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStdout, origStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	defer func() { os.Stdout, os.Stderr = origStdout, origStderr }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	exitCode = Execute(args)
	_ = w.Close()
	return <-done, exitCode
}

// parentArgvPaths walks the tree rooted at cmd and returns the argv
// prefix reaching every command that carries subcommands — the root
// itself (an empty prefix), the verb groups, and the runnable parents
// (`add`, `render`).
//
// Prefixes are always non-nil: Execute passes its argument straight to
// Cobra's SetArgs, which reads os.Args[1:] instead when handed nil, and
// under `go test` that would be the test binary's own flags.
func parentArgvPaths(cmd *cobra.Command, prefix []string) [][]string {
	var out [][]string
	if cmd.HasSubCommands() {
		out = append(out, append(make([]string, 0, len(prefix)), prefix...))
	}
	for _, child := range cmd.Commands() {
		childPrefix := append(append(make([]string, 0, len(prefix)+1), prefix...), child.Name())
		out = append(out, parentArgvPaths(child, childPrefix)...)
	}
	return out
}

// TestExecute_ParentCommandsRejectIncompleteInvocations pins the one
// exit-code rule that covers every command carrying subcommands: naming
// a parent without reaching one of its subverbs completes no operation,
// so it reports a usage error (exit 2) — whether the subverb is missing
// entirely or is a name the parent does not have.
//
// The rule is worth a test rather than a reading of the command
// literals because a verb group's Args constraint is not what enforces
// it. Cobra returns flag.ErrHelp for a command that is not Runnable
// before it validates arguments (command.go, Runnable check ahead of
// ValidateArgs), so a group whose behavior lives entirely in its
// children has no reachable argument check: it prints help and reports
// success for a subverb that does not exist. Every group here is
// Runnable for exactly that reason — the RunE is what lets the Args
// constraint fire.
//
// The walk is deliberately universal rather than a list of group names,
// so a parent added later is covered without being enrolled. Cobra's
// own auto-added `completion` and `help` parents are out of scope: they
// enter the tree during Execute, after the registered-verbs snapshot
// NewRootCmd takes, and the framework owns their behavior.
//
// Closes G-0528.
func TestExecute_ParentCommandsRejectIncompleteInvocations(t *testing.T) {
	paths := parentArgvPaths(NewRootCmd(""), []string{})
	if len(paths) < 6 {
		t.Fatalf("walk found only %d parent command(s); the tree carries the root, the verb groups, add and render", len(paths))
	}

	for _, path := range paths {
		label := strings.TrimSpace("aiwf " + strings.Join(path, " "))

		t.Run(label+" (no subverb)", func(t *testing.T) {
			out, code := captureExecuteStreams(t, path)
			if code != cliutil.ExitUsage {
				t.Errorf("%q exit code = %d, want %d (naming a parent completes no operation)",
					label, code, cliutil.ExitUsage)
			}
			if strings.TrimSpace(out) == "" {
				t.Errorf("%q produced no output; the operator gets no route to the subverbs", label)
			}
		})

		t.Run(label+" "+noSuchSubverb, func(t *testing.T) {
			out, code := captureExecuteStreams(t, append(append([]string{}, path...), noSuchSubverb))
			if code != cliutil.ExitUsage {
				t.Errorf("%q %s exit code = %d, want %d (an unrecognized subverb is a usage error)",
					label, noSuchSubverb, code, cliutil.ExitUsage)
			}
			if !strings.Contains(out, noSuchSubverb) {
				t.Errorf("%q %s: output never names the rejected argument, so exit %d came from something else:\n%s",
					label, noSuchSubverb, cliutil.ExitUsage, out)
			}
		})
	}
}
