package cliutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/skills"
)

// seedHookDecisionForSyncTest writes a minimal aiwf.yaml recording a
// single hook decision — the fixture shape SyncHookMaterialization
// reads back after the consent gate has already persisted it.
func seedHookDecisionForSyncTest(t *testing.T, root, name string, enabled bool) {
	t.Helper()
	configPath := filepath.Join(root, "aiwf.yaml")
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		if err := os.WriteFile(configPath, []byte("hosts: [claude-code]\n"), 0o644); err != nil {
			t.Fatalf("seeding aiwf.yaml: %v", err)
		}
	}
	doc, _, err := aiwfyaml.Read(configPath)
	if err != nil {
		t.Fatalf("aiwfyaml.Read: %v", err)
	}
	doc.SetHooks(map[string]bool{name: enabled})
	if err := doc.Write(configPath); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}
}

func TestGateHookDecisions_EmptyRegistry(t *testing.T) {
	t.Parallel()
	got := GateHookDecisions(nil, nil, false, false)
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
	got := GateHookDecisions(hooks, []string{"hook-a"}, false, false)
	want := map[string]bool{"hook-a": true}
	if len(got) != 1 || got["hook-a"] != true {
		t.Errorf("GateHookDecisions(...) = %#v, want %#v", got, want)
	}
}

// TestGateHookDecisions_NonTTYLeavesUndecided: under `go test`, stdin is
// never a real TTY, so a hook not named via --enable-hook is left UNDECIDED
// (G-0446) — its name absent from the map, never recorded as a false
// decline. Absent-not-false is what surfaces it as a doctor "undecided"
// warning rather than an honored decline.
func TestGateHookDecisions_NonTTYLeavesUndecided(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{{Name: "hook-a", Description: "does a thing"}}
	got := GateHookDecisions(hooks, nil, false, false)
	if _, decided := got["hook-a"]; decided {
		t.Errorf("GateHookDecisions(...) recorded a decision for \"hook-a\" = %v, want it left undecided (absent)", got["hook-a"])
	}
}

// TestGateHookDecisions_FormatJSONLeavesUndecided pins the !formatJSON
// short-circuit explicitly (mirrors the statusline gate's !opts.FormatJSON
// check) rather than relying only on go test's never-a-TTY stdin: under
// --format=json the hook is left undecided, not defaulted to false.
func TestGateHookDecisions_FormatJSONLeavesUndecided(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{{Name: "hook-a", Description: "does a thing"}}
	got := GateHookDecisions(hooks, nil, true, false)
	if _, decided := got["hook-a"]; decided {
		t.Errorf("GateHookDecisions(..., formatJSON=true) recorded a decision for \"hook-a\", want undecided (absent)")
	}
}

// TestGateHookDecisions_NoPromptLeavesUndecided pins the --no-prompt contract
// (G-0446): a hook not named via --enable-hook is left undecided rather than
// prompted or defaulted. (The TTY-suppression itself — noPrompt forcing the
// non-interactive path when stdin IS a real TTY — cannot be exercised without
// a pty library, the same untestable gap the promptYN arm carries; here
// go test's non-TTY stdin already yields the non-interactive path, so this
// pins the omit contract the flag guarantees.)
func TestGateHookDecisions_NoPromptLeavesUndecided(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{{Name: "hook-a", Description: "does a thing"}}
	got := GateHookDecisions(hooks, nil, false, true)
	if _, decided := got["hook-a"]; decided {
		t.Errorf("GateHookDecisions(..., noPrompt=true) recorded a decision for \"hook-a\", want undecided (absent)")
	}
}

// TestGateHookDecisions_MultipleHooksIndependentDecisions: each hook in the
// registry gets its own outcome — one enabled via --enable-hook, the other
// left undecided (absent) under the non-interactive default.
func TestGateHookDecisions_MultipleHooksIndependentDecisions(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{
		{Name: "hook-a", Description: "a"},
		{Name: "hook-b", Description: "b"},
	}
	got := GateHookDecisions(hooks, []string{"hook-a"}, false, false)
	if got["hook-a"] != true {
		t.Errorf("GateHookDecisions(...)[\"hook-a\"] = %v, want true (enabled via flag)", got["hook-a"])
	}
	if _, decided := got["hook-b"]; decided {
		t.Errorf("GateHookDecisions(...) recorded a decision for \"hook-b\", want it left undecided (absent)")
	}
}

// TestGateHookDecisions_EnableHookNameNotInRegistry: an --enable-hook value
// naming a hook absent from the registry is simply inert — it neither
// errors nor affects any registry hook's own outcome. The one registry hook
// is left undecided (absent), so the map is empty. Registry membership
// validation (rejecting an unknown --enable-hook name) is a CLI-layer
// concern for the flag itself, not this pure decision function.
func TestGateHookDecisions_EnableHookNameNotInRegistry(t *testing.T) {
	t.Parallel()
	hooks := []skills.HookDef{{Name: "hook-a", Description: "a"}}
	got := GateHookDecisions(hooks, []string{"nonexistent-hook"}, false, false)
	if _, decided := got["hook-a"]; decided {
		t.Errorf("GateHookDecisions(...) recorded a decision for \"hook-a\", want undecided (absent)")
	}
	if len(got) != 0 {
		t.Errorf("GateHookDecisions(...) = %#v, want empty (the typo'd name decides nothing, the registry hook is undecided)", got)
	}
}

// TestSyncHookMaterialization_EmptyRegistryIsNoOp pins the same
// empty-registry-no-op convention MaterializeHooks/HookDrift already
// use — no aiwf.yaml read even attempted, so it's safe to call with
// hooks=nil regardless of whether aiwf.yaml exists yet.
func TestSyncHookMaterialization_EmptyRegistryIsNoOp(t *testing.T) {
	t.Parallel()
	root := t.TempDir() // deliberately no aiwf.yaml
	if got := SyncHookMaterialization(root, skills.ClaudeTarget, nil); got != ExitOK {
		t.Errorf("SyncHookMaterialization(nil) = %d, want ExitOK", got)
	}
}

// TestSyncHookMaterialization_EnabledHookMaterializesAndWires pins
// M-0236/AC-4's core claim: a hook decided true gets its script written
// to disk and its command wired under every one of its Events.
func TestSyncHookMaterialization_EnabledHookMaterializesAndWires(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	seedHookDecisionForSyncTest(t, root, "h1.sh", true)
	hooks := []skills.HookDef{{
		Name:    "h1.sh",
		Content: []byte("#!/bin/sh\necho hi\n"),
		Events:  []string{"SessionStart", "SubagentStart"},
	}}

	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitOK {
		t.Fatalf("SyncHookMaterialization(...) = %d, want ExitOK", got)
	}

	scriptPath := filepath.Join(root, skills.ClaudeTarget.HooksDir, "h1.sh")
	if _, statErr := os.Stat(scriptPath); statErr != nil {
		t.Errorf("expected %s to exist, stat err=%v", scriptPath, statErr)
	}
	settingsPath := filepath.Join(root, skills.SharedSettingsRelPath)
	wired, wiredErr := skills.HookCommandWired(settingsPath, hooks[0].Command(skills.ClaudeTarget))
	if wiredErr != nil {
		t.Fatalf("HookCommandWired: %v", wiredErr)
	}
	if !wired {
		t.Error("expected the hook's command to be wired into settings.json")
	}
}

// TestSyncHookMaterialization_DeclinedHookRemovesScriptAndUnwires pins
// ADR-0032's "remove both when false" half: a hook previously
// materialized and wired, then flipped to decided-false, has both the
// script and the settings.json entry removed on sync.
func TestSyncHookMaterialization_DeclinedHookRemovesScriptAndUnwires(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := []skills.HookDef{{
		Name:    "h1.sh",
		Content: []byte("#!/bin/sh\necho hi\n"),
		Events:  []string{"SessionStart"},
	}}
	// Simulate a prior enabled+synced state.
	seedHookDecisionForSyncTest(t, root, "h1.sh", true)
	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitOK {
		t.Fatalf("priming SyncHookMaterialization(...) = %d, want ExitOK", got)
	}

	// Now decline it and sync again.
	seedHookDecisionForSyncTest(t, root, "h1.sh", false)
	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitOK {
		t.Fatalf("SyncHookMaterialization(...) = %d, want ExitOK", got)
	}

	scriptPath := filepath.Join(root, skills.ClaudeTarget.HooksDir, "h1.sh")
	if _, statErr := os.Stat(scriptPath); !os.IsNotExist(statErr) {
		t.Errorf("expected %s to be removed after declining, stat err=%v", scriptPath, statErr)
	}
	settingsPath := filepath.Join(root, skills.SharedSettingsRelPath)
	wired, wiredErr := skills.HookCommandWired(settingsPath, hooks[0].Command(skills.ClaudeTarget))
	if wiredErr != nil {
		t.Fatalf("HookCommandWired: %v", wiredErr)
	}
	if wired {
		t.Error("expected the hook's command to be unwired from settings.json after declining")
	}
}

// TestSyncHookMaterialization_UndecidedHookUntouched pins that a hook
// absent from aiwf.yaml's hooks: map (not yet gated) is left alone —
// no script, no wiring, no error. The consent gate runs before this
// function; an undecided hook here means the caller hasn't gated yet.
func TestSyncHookMaterialization_UndecidedHookUntouched(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []skills.HookDef{{
		Name:    "h1.sh",
		Content: []byte("#!/bin/sh\necho hi\n"),
		Events:  []string{"SessionStart"},
	}}

	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitOK {
		t.Fatalf("SyncHookMaterialization(...) = %d, want ExitOK", got)
	}

	scriptPath := filepath.Join(root, skills.ClaudeTarget.HooksDir, "h1.sh")
	if _, statErr := os.Stat(scriptPath); !os.IsNotExist(statErr) {
		t.Errorf("expected no script for an undecided hook, stat err=%v", statErr)
	}
}

// TestSyncHookMaterialization_MissingAiwfYamlReturnsInternal mirrors
// gateAndPersistHookDecisions's/gateAndSyncHookDecisions's identical
// shape: a non-empty registry with no aiwf.yaml to read decisions from
// is an internal error, not a silent no-op.
func TestSyncHookMaterialization_MissingAiwfYamlReturnsInternal(t *testing.T) {
	t.Parallel()
	root := t.TempDir() // deliberately no aiwf.yaml
	hooks := []skills.HookDef{{Name: "h1.sh", Content: []byte("x"), Events: []string{"SessionStart"}}}
	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitInternal {
		t.Errorf("SyncHookMaterialization(...) = %d, want ExitInternal", got)
	}
}

// TestSyncHookMaterialization_MalformedHooksBlockReturnsInternal covers
// doc.Hooks()'s own reachable decode error — distinct from
// aiwfyaml.Read's own error above — a hand-edited hooks: block
// carrying an unrecognized field inside one hook's entry fails the
// strict decode, mirroring
// TestGateAndSyncHookDecisions_UnknownFieldInExistingHooksBlockReturnsInternal's
// identical fixture shape.
func TestSyncHookMaterialization_MalformedHooksBlockReturnsInternal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	raw := "hooks:\n  h1.sh:\n    unknown_field: true\n"
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []skills.HookDef{{Name: "h1.sh", Content: []byte("x"), Events: []string{"SessionStart"}}}

	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitInternal {
		t.Errorf("SyncHookMaterialization(...) = %d, want ExitInternal", got)
	}
}

// TestSyncHookMaterialization_MaterializeHooksErrorPropagates covers
// the seam: an error from skills.MaterializeHooks itself (here, a
// non-empty directory blocking os.Remove for a declined hook —
// deterministically reproducible, mirroring
// TestMaterializeHooks_RemoveErrorSurfaces) must propagate as
// ExitInternal, not be swallowed.
func TestSyncHookMaterialization_MaterializeHooksErrorPropagates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	seedHookDecisionForSyncTest(t, root, "h1.sh", false)
	nonEmptyDir := filepath.Join(root, skills.ClaudeTarget.HooksDir, "h1.sh")
	if err := os.MkdirAll(nonEmptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "child"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []skills.HookDef{{Name: "h1.sh", Content: []byte("x"), Events: []string{"SessionStart"}}}

	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitInternal {
		t.Errorf("SyncHookMaterialization(...) = %d, want ExitInternal", got)
	}
}

// TestSyncHookMaterialization_WireHookSettingsErrorPropagates covers
// the seam: an error from skills.WireHookSettings (here, a malformed
// pre-existing settings.json) must propagate as ExitInternal.
func TestSyncHookMaterialization_WireHookSettingsErrorPropagates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	seedHookDecisionForSyncTest(t, root, "h1.sh", true)
	settingsPath := filepath.Join(root, skills.SharedSettingsRelPath)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"hooks": "not-an-object"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []skills.HookDef{{Name: "h1.sh", Content: []byte("x"), Events: []string{"SessionStart"}}}

	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitInternal {
		t.Errorf("SyncHookMaterialization(...) = %d, want ExitInternal", got)
	}
}

// TestSyncHookMaterialization_UnwireHookSettingsErrorPropagates covers
// the seam: an error from skills.UnwireHookSettings (the declined
// branch's counterpart to the wire-error test above) must propagate as
// ExitInternal.
func TestSyncHookMaterialization_UnwireHookSettingsErrorPropagates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	seedHookDecisionForSyncTest(t, root, "h1.sh", false)
	settingsPath := filepath.Join(root, skills.SharedSettingsRelPath)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"hooks": "not-an-object"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []skills.HookDef{{Name: "h1.sh", Content: []byte("x"), Events: []string{"SessionStart"}}}

	if got := SyncHookMaterialization(root, skills.ClaudeTarget, hooks); got != ExitInternal {
		t.Errorf("SyncHookMaterialization(...) = %d, want ExitInternal", got)
	}
}
