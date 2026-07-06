package skills

import (
	"bytes"
	"testing"
)

// TestShippedHooks_RegistersExactlyTheWorktreeRitualsCheckHook pins
// M-0236/AC-2's registration claim: the registry ships exactly one entry —
// the worktree-materialization-check hook — registered for both
// SessionStart and SubagentStart, with its Content byte-equal to the
// embedded script AC-1 shipped (never a re-authored copy).
func TestShippedHooks_RegistersExactlyTheWorktreeRitualsCheckHook(t *testing.T) {
	t.Parallel()
	if len(ShippedHooks) != 1 {
		t.Fatalf("ShippedHooks = %#v, want exactly 1 entry", ShippedHooks)
	}
	h := ShippedHooks[0]
	if h.Name == "" {
		t.Error("Name is empty")
	}
	if h.Description == "" {
		t.Error("Description is empty")
	}
	if !bytes.Equal(h.Content, WorktreeRitualsCheckScript) {
		t.Error("Content is not byte-equal to WorktreeRitualsCheckScript")
	}
	wantEvents := map[string]bool{"SessionStart": false, "SubagentStart": false}
	for _, e := range h.Events {
		if _, known := wantEvents[e]; !known {
			t.Errorf("unexpected event %q in Events %v", e, h.Events)
			continue
		}
		wantEvents[e] = true
	}
	for e, seen := range wantEvents {
		if !seen {
			t.Errorf("Events %v missing %q", h.Events, e)
		}
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
