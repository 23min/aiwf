package cliutil_test

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/initcmd"
	"github.com/23min/aiwf/internal/cli/update"
	"github.com/23min/aiwf/internal/cli/upgrade"
)

// TestNoPromptFlag_HelpComesFromOneSource pins every registered --no-prompt
// flag to the shared usage, so the verbs cannot describe it differently.
func TestNoPromptFlag_HelpComesFromOneSource(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		cmd      *cobra.Command
		isSuffix bool
	}{
		{name: "init", cmd: initcmd.NewCmd()},
		{name: "update", cmd: update.NewCmd()},
		{name: "upgrade", cmd: upgrade.NewCmd(), isSuffix: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			flag := tc.cmd.Flags().Lookup("no-prompt")
			if flag == nil {
				t.Fatal("no --no-prompt flag registered")
			}
			if tc.isSuffix && !strings.HasSuffix(flag.Usage, ": "+cliutil.NoPromptUsage) || !tc.isSuffix && flag.Usage != cliutil.NoPromptUsage {
				t.Fatalf("--no-prompt usage %q does not come from the shared usage %q", flag.Usage, cliutil.NoPromptUsage)
			}
		})
	}
}
