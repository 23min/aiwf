package milestone_test

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/milestone"
)

// M-0253/AC-1 backfill: milestone.go's sole subcommand, depends-on,
// wires the standard ResolveRoot/ResolveActor/tree.Load guard shape
// shared with every entity-lifecycle verb inside its unexported
// runDependsOn, reached only through the depends-on subcommand's RunE
// closure — the same "unexported fn behind RunE" shape wave-1's
// add.go execExitCode pattern was built for. The ResolveRoot and
// tree.Load fatal-IO branches are `//coverage:ignore`d in
// milestone.go itself, mirroring the established internal/cli/archive
// and wave-1/wave-2 precedent. The one remaining flagged branch — the
// actor-resolution guard — gets a real test below.

// execExitCode drives cmd through Cobra's real Execute path (the only
// way to reach depends-on's RunE closure and the unexported
// runDependsOn) and unwraps the resulting *cliutil.ExitError.
func execExitCode(t *testing.T, cmd *cobra.Command, args []string) int {
	t.Helper()
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err == nil {
		return cliutil.ExitOK
	}
	var ee *cliutil.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("Execute() error = %v (%T), want *cliutil.ExitError", err, err)
	}
	return ee.Code
}

// TestDependsOnCmd_ResolveActorFailure covers runDependsOn's
// cliutil.ResolveActor guard using M-0252's BrokenGitIdentity fixture.
// Serial: BrokenGitIdentity uses t.Setenv, which panics under
// t.Parallel.
func TestDependsOnCmd_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	rc := execExitCode(t, milestone.NewCmd(""), []string{
		"depends-on", "M-0001", "--on", "M-0002", "--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
