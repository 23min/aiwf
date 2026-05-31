package policies

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/initcmd"
	"github.com/23min/aiwf/internal/cli/update"
)

// TestM0156_AC2_WireSettingsFlagOnInitAndUpdate asserts M-0156/AC-2:
// `aiwf init` and `aiwf update` each grow a `--wire-settings` boolean
// flag. The flag gates consent-free settings writes in non-TTY / JSON
// contexts per ADR-0015's non-interactive consent mechanism.
//
// Boolean flags have no closed value set, so no FixedCompletions wiring
// is needed — the completion-drift test auto-skips booleans.
func TestM0156_AC2_WireSettingsFlagOnInitAndUpdate(t *testing.T) {
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
			f := tc.cmd.Flags().Lookup("wire-settings")
			if f == nil {
				t.Errorf("AC-2: `aiwf %s` must register a --wire-settings flag", tc.name)
				return
			}
			if f.DefValue != "false" {
				t.Errorf("AC-2: `aiwf %s` --wire-settings default must be false (got %q)", tc.name, f.DefValue)
			}
		})
	}
}
