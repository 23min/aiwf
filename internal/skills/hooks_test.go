package skills

import "testing"

// TestShippedHooks_EmptyUntilAConcreteHookRegisters pins the current,
// deliberate state (M-0235/AC-2): the hook registry ships with zero entries
// until a later milestone (M-0236) registers its first concrete hook. A
// non-empty registry here would mean a hook shipped without its own
// milestone's TDD cycle covering it.
func TestShippedHooks_EmptyUntilAConcreteHookRegisters(t *testing.T) {
	t.Parallel()
	if len(ShippedHooks) != 0 {
		t.Errorf("ShippedHooks = %#v, want empty (no concrete hook has landed yet)", ShippedHooks)
	}
}

// TestHookNames_SortedAndDerivedFromRegistry pins HookNames() as the
// single-source derivation from ShippedHooks (mirroring AgentNames()),
// exercised against a synthetic registry since the real one is empty.
func TestHookNames_SortedAndDerivedFromRegistry(t *testing.T) {
	t.Parallel()
	hooks := []HookDef{
		{Name: "zeta-hook", Description: "z"},
		{Name: "alpha-hook", Description: "a"},
	}
	got := HookNamesFrom(hooks)
	want := []string{"alpha-hook", "zeta-hook"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("HookNamesFrom(...) = %v, want %v", got, want)
	}
}

// TestHookNamesFrom_Empty pins the zero-registry case returns an empty,
// non-nil-or-nil-both-acceptable slice — the real production call site
// today, since ShippedHooks is empty.
func TestHookNamesFrom_Empty(t *testing.T) {
	t.Parallel()
	got := HookNamesFrom(nil)
	if len(got) != 0 {
		t.Errorf("HookNamesFrom(nil) = %v, want empty", got)
	}
}
