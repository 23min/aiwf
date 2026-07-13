package show_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/show"
)

// M-0255/AC-1 backfill: Run's ResolveRoot and tree.Load guards, plus
// its trailing (success-path) JSON envelope write, are
// `//coverage:ignore`d in show.go itself, mirroring this verb's own
// established precedent at its two sibling json render branches. The
// remaining flagged branch — the --format validation guard — gets a
// real test below.

// TestRun_BadFormat covers Run's --format validation branch: an
// unrecognized value returns ExitUsage before any root/tree work.
func TestRun_BadFormat(t *testing.T) {
	t.Parallel()
	rc := show.Run("E-0001", "", testutil.InvalidFormat, "", false, 10, "")
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
