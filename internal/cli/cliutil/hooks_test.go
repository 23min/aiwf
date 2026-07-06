package cliutil

import (
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

func TestGateHookDecisions_EmptyRegistry(t *testing.T) {
	t.Parallel()
	got := GateHookDecisions(nil, nil, false)
	if len(got) != 0 {
		t.Errorf("GateHookDecisions(nil, ...) = %#v, want empty", got)
	}
}

// TestGateHookDecisions_EnableHookFlagBypassesPrompt: a hook named via
// --enable-hook is enabled without needing a TTY or interactive answer —
// the non-TTY consent escape hatch (ADR-0032), mirroring --wire-settings.
func TestGateHookDecisions_EnableHookFlagBypassesPrompt(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{{Name: "hook-a", Description: "does a thing"}}
	got := GateHookDecisions(hooks, []string{"hook-a"}, false)
	want := map[string]bool{"hook-a": true}
	if len(got) != 1 || got["hook-a"] != true {
		t.Errorf("GateHookDecisions(...) = %#v, want %#v", got, want)
	}
}

// TestGateHookDecisions_NonTTYDeclinesByDefault: under `go test`, stdin is
// never a real TTY, so a hook not named via --enable-hook silently
// declines rather than hanging on a prompt.
func TestGateHookDecisions_NonTTYDeclinesByDefault(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{{Name: "hook-a", Description: "does a thing"}}
	got := GateHookDecisions(hooks, nil, false)
	if got["hook-a"] != false {
		t.Errorf("GateHookDecisions(...)[\"hook-a\"] = %v, want false (non-TTY, not enabled via flag)", got["hook-a"])
	}
}

// TestGateHookDecisions_FormatJSONForcesNonInteractive pins the
// !formatJSON short-circuit explicitly (mirrors the statusline gate's
// !opts.FormatJSON check) rather than relying only on go test's
// never-a-TTY stdin to reach the decline path.
func TestGateHookDecisions_FormatJSONForcesNonInteractive(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{{Name: "hook-a", Description: "does a thing"}}
	got := GateHookDecisions(hooks, nil, true)
	if got["hook-a"] != false {
		t.Errorf("GateHookDecisions(..., formatJSON=true)[\"hook-a\"] = %v, want false", got["hook-a"])
	}
}

// TestGateHookDecisions_MultipleHooksIndependentDecisions: each hook in the
// registry gets its own decision — one named via --enable-hook, the other
// left to the non-TTY default decline.
func TestGateHookDecisions_MultipleHooksIndependentDecisions(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{
		{Name: "hook-a", Description: "a"},
		{Name: "hook-b", Description: "b"},
	}
	got := GateHookDecisions(hooks, []string{"hook-a"}, false)
	want := map[string]bool{"hook-a": true, "hook-b": false}
	for name, wantVal := range want {
		if got[name] != wantVal {
			t.Errorf("GateHookDecisions(...)[%q] = %v, want %v", name, got[name], wantVal)
		}
	}
}

// TestGateHookDecisions_EnableHookNameNotInRegistry: an --enable-hook value
// naming a hook absent from the registry is simply inert — it neither
// errors nor affects any registry hook's own decision. Registry membership
// validation (rejecting an unknown --enable-hook name) is a CLI-layer
// concern for the flag itself, not this pure decision function.
func TestGateHookDecisions_EnableHookNameNotInRegistry(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{{Name: "hook-a", Description: "a"}}
	got := GateHookDecisions(hooks, []string{"nonexistent-hook"}, false)
	want := map[string]bool{"hook-a": false}
	if diffVal := got["hook-a"]; diffVal != want["hook-a"] {
		t.Errorf("GateHookDecisions(...)[\"hook-a\"] = %v, want %v", diffVal, want["hook-a"])
	}
	if len(got) != 1 {
		t.Errorf("GateHookDecisions(...) = %#v, want exactly the registry's own hooks, not the flag's typo'd name", got)
	}
}
