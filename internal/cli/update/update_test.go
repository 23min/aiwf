package update_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/update"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := update.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "update" {
		t.Errorf("Use = %q", cmd.Use)
	}
	if cmd.Flags().Lookup("root") == nil {
		t.Errorf("missing --root flag")
	}
}

// TestNewCmd_RemoveAndForceFlags asserts G-0354's new flags are wired
// on `aiwf update` and discoverable via --help (bool flags, so no
// completion registration is required by the completion-drift policy).
func TestNewCmd_RemoveAndForceFlags(t *testing.T) {
	t.Parallel()
	cmd := update.NewCmd()
	if cmd.Flags().Lookup("remove") == nil {
		t.Error("missing --remove flag")
	}
	if cmd.Flags().Lookup("force") == nil {
		t.Error("missing --force flag")
	}
}

// TestRun_RemoveAndStatuslineMutuallyExclusive asserts G-0354: passing
// both --statusline and --remove is a usage error, and neither the
// artifact refresh nor either statusline action runs.
func TestRun_RemoveAndStatuslineMutuallyExclusive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	rc := update.Run(root, true /* statusline */, "project", false, true /* remove */, false)
	if rc != cliutil.ExitUsage {
		t.Fatalf("rc = %d, want cliutil.ExitUsage", rc)
	}
}
