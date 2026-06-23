package integration

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/list"
	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/cli/status"
)

// TestAreaCompletion_WiredOnReadVerbs pins M-0174/AC-4: the --area flag
// on list, show, and status each has a completion function wired that
// offers exactly the declared areas.members. This is the focused
// companion to the live-tree TestPolicy_FlagsHaveCompletion drift gate —
// that gate proves *a* func is registered; this proves it returns the
// declared set. Serial: t.Chdir mutates process-wide cwd, which
// CompleteAreaFlag reads via ResolveRoot("").
func TestAreaCompletion_WiredOnReadVerbs(t *testing.T) {
	root := setupAreaRepo(t) // declares {platform, billing}
	t.Chdir(root)

	cases := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"list", list.NewCmd()},
		{"show", show.NewCmd()},
		{"status", status.NewCmd()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, ok := tc.cmd.GetFlagCompletionFunc("area")
			if !ok {
				t.Fatalf("%s --area has no completion func registered", tc.name)
			}
			got, directive := fn(tc.cmd, nil, "")
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("%s --area directive = %d, want ShellCompDirectiveNoFileComp", tc.name, directive)
			}
			want := map[string]bool{"platform": true, "billing": true}
			if len(got) != len(want) {
				t.Fatalf("%s --area completion = %v, want exactly platform, billing", tc.name, got)
			}
			for _, g := range got {
				if !want[g] {
					t.Errorf("%s --area offered unexpected %q (want only platform, billing)", tc.name, g)
				}
			}
		})
	}
}
