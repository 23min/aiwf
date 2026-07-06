package update

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestCompleteHookNames_EmptyRegistryReturnsNoCompletions pins today's real
// state: skills.ShippedHooks is empty until a milestone registers its first
// concrete hook (M-0236), so `aiwf update --enable-hook <TAB>` offers
// nothing yet — not an error, just an empty, valid completion list.
func TestCompleteHookNames_EmptyRegistryReturnsNoCompletions(t *testing.T) {
	t.Parallel()
	got, directive := completeHookNames(nil, nil, "")
	if len(got) != 0 {
		t.Errorf("completeHookNames() = %v, want empty (registry currently empty)", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
}
