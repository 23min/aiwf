package update

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestCompleteHookNames_ListsRegisteredHooks pins today's real state
// (M-0236/AC-2): `aiwf update --enable-hook <TAB>` offers the names in
// skills.ShippedHooks, derived via HookNamesFrom rather than hardcoded.
func TestCompleteHookNames_ListsRegisteredHooks(t *testing.T) {
	t.Parallel()
	got, directive := completeHookNames(nil, nil, "")
	want := []string{"worktree-rituals-check.sh"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("completeHookNames() = %v, want %v", got, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
}
