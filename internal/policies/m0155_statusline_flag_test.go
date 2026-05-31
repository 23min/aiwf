package policies

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/initcmd"
	"github.com/23min/aiwf/internal/cli/update"
)

// TestM0155_AC2_StatuslineFlagsOnInitAndUpdate asserts M-0155/AC-2:
// `aiwf init` and `aiwf update` each grow a `--statusline` boolean
// flag and a `--scope` string flag whose closed value set
// (`project|user`) is registered with Cobra via FixedCompletions, so
// shell tab-completion enumerates the valid values without an extra
// trip to the binary.
//
// The completion-wiring assertion is the load-bearing one — per
// CLAUDE.md's "auto-completion-friendly" principle, every closed-set
// flag value must be reachable via completion or the
// `TestPolicy_FlagsHaveCompletion` drift test (M-054) fires on CI. The
// per-flag check here catches a missing FixedCompletions before the
// drift test does, with a focused error message.
//
// `--statusline` is a boolean: no value to complete; the drift test
// auto-skips booleans, and this AC test does not check completion on it.
func TestM0155_AC2_StatuslineFlagsOnInitAndUpdate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		cmd  *cobra.Command
	}{
		{"init", initcmd.NewCmd()},
		{"update", update.NewCmd()},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.cmd.Flags().Lookup("statusline") == nil {
				t.Errorf("AC-2: `aiwf %s` must register a --statusline flag", tc.name)
			}
			scope := tc.cmd.Flags().Lookup("scope")
			if scope == nil {
				t.Errorf("AC-2: `aiwf %s` must register a --scope flag", tc.name)
				return
			}
			if _, ok := tc.cmd.GetFlagCompletionFunc("scope"); !ok {
				t.Errorf("AC-2: `aiwf %s` --scope must have a completion function registered (FixedCompletions of project|user)", tc.name)
			}
		})
	}
}
