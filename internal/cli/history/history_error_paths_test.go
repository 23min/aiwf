package history_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/history"
)

// M-0255/AC-1 backfill: Run's ResolveRoot guard and ReadHistoryChain's
// git-log guard, plus its trailing JSON envelope write, are
// `//coverage:ignore`d in history.go itself. The remaining flagged
// branch — the --format validation guard — gets a real test below.

// TestRun_BadFormat covers Run's --format validation branch: an
// unrecognized value returns ExitUsage before any root/tree work.
func TestRun_BadFormat(t *testing.T) {
	t.Parallel()
	rc := history.Run("E-0001", "", testutil.InvalidFormat, false, false, "")
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
