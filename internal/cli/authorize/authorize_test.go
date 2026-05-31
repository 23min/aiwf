package authorize_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/authorize"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := authorize.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "authorize <id>" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"actor", "root", "to", "pause", "resume", "reason", "force", "branch"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

// TestRun_BranchWithPauseRejected (M-0102/AC-1, cli-layer gate): --branch
// is meaningful only on the open path. --branch + --pause is rejected
// upfront so the operator sees the misuse rather than silently dropping
// the flag. Mirrors the existing --reason + --pause guard.
func TestRun_BranchWithPauseRejected(t *testing.T) {
	t.Parallel()
	// pause supplies the reason; --branch must NOT be combined.
	rc := authorize.Run(
		"E-0001",          // id
		"human/test",      // actor
		"",                // root (unused; we fail before tree load)
		"",                // to
		"blocked by E-09", // pause
		"",                // resume
		"",                // reason
		"epic/E-0001-eng", // branch
		false,             // force
		cliutil.OutputFormat{},
	)
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage (%d)", rc, cliutil.ExitUsage)
	}
}

// TestRun_BranchWithResumeRejected: mirror of the pause case for the
// resume mode.
func TestRun_BranchWithResumeRejected(t *testing.T) {
	t.Parallel()
	rc := authorize.Run(
		"E-0001",
		"human/test",
		"",
		"",
		"",                // pause
		"resume work now", // resume
		"",
		"epic/E-0001-eng",
		false,
		cliutil.OutputFormat{},
	)
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage (%d)", rc, cliutil.ExitUsage)
	}
}
