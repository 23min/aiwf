package update_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/update"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/skills"
)

// freshInitializedRepo builds a git repo with a real, freshly-written
// aiwf.yaml (via initrepo.Init directly, skipping the hook-consent gate
// that only aiwf init/update's own CLI layer runs) — the "existing
// aiwf.yaml" precondition AC-3 operates against.
func freshInitializedRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q", root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		c := exec.Command("git", args...)
		c.Dir = root
		if out, cErr := c.CombinedOutput(); cErr != nil {
			t.Fatalf("git %v: %v\n%s", args, cErr, out)
		}
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{SkipHook: true}); err != nil {
		t.Fatalf("initrepo.Init: %v", err)
	}
	return root
}

// seedHookDecisions splices decisions directly into the root's aiwf.yaml,
// bypassing the interactive gate — arranging the "already decided" fixture
// state AC-3's sync logic must leave untouched.
func seedHookDecisions(t *testing.T, root string, decisions map[string]bool) {
	t.Helper()
	configPath := filepath.Join(root, config.FileName)
	doc, _, err := aiwfyaml.Read(configPath)
	if err != nil {
		t.Fatalf("aiwfyaml.Read: %v", err)
	}
	doc.SetHooks(decisions)
	if err := doc.Write(configPath); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}
}

func hookDecision(t *testing.T, root, name string) (enabled, decided bool) {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg.HookDecision(name)
}

// TestRun_SyncsExistingHookDecisionSilentlyWithoutReprompt pins M-0235/AC-3's
// core claim: a hook already decided in aiwf.yaml is never re-gated on
// `aiwf update`, even when it wasn't named via --enable-hook — the sync
// carries the existing value forward untouched, not re-derives it from the
// current flags/TTY state. Seeded true deliberately (not false): the
// non-TTY default for a hook that WAS wrongly re-gated is always a
// decline (false), so a true seed is the one value that distinguishes
// "left alone" from "re-decided via the default" — a false seed would let
// that exact bug slip through unnoticed.
func TestRun_SyncsExistingHookDecisionSilentlyWithoutReprompt(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	seedHookDecisions(t, root, map[string]bool{"existing-hook": true})
	hooks := []skills.HookDef{{Name: "existing-hook", Description: "does a thing"}}

	rc := update.Run(root, false, "", false, false, false, false, nil, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}
	enabled, decided := hookDecision(t, root, "existing-hook")
	if !decided || !enabled {
		t.Errorf("HookDecision(existing-hook) = (%v, %v), want (true, true) — unchanged", enabled, decided)
	}
}

// TestRun_GatesOnlyNewlyIntroducedHooksAndPreservesExisting: on a registry
// with one already-decided hook and one newly-introduced hook, only the
// newly-introduced one is gated (via --enable-hook here); the pre-existing
// decision is carried through the same sync run unmodified.
func TestRun_GatesOnlyNewlyIntroducedHooksAndPreservesExisting(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	seedHookDecisions(t, root, map[string]bool{"existing-hook": false})
	hooks := []skills.HookDef{
		{Name: "existing-hook", Description: "does a thing"},
		{Name: "new-hook", Description: "does another thing"},
	}

	rc := update.Run(root, false, "", false, false, false, false, []string{"new-hook"}, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}

	if enabled, decided := hookDecision(t, root, "existing-hook"); !decided || enabled {
		t.Errorf("HookDecision(existing-hook) = (%v, %v), want (false, true) — unchanged", enabled, decided)
	}
	if enabled, decided := hookDecision(t, root, "new-hook"); !decided || !enabled {
		t.Errorf("HookDecision(new-hook) = (%v, %v), want (true, true) — gated via --enable-hook", enabled, decided)
	}
}

// TestRun_PreservesDecisionForHookRemovedFromRegistry pins the durability
// half of gateAndSyncHookDecisions's own doc comment: a decision for a hook
// no longer named in the current run's registry (e.g. one shipped by an
// older aiwf and since dropped) survives the union write untouched, rather
// than being silently dropped because it isn't in `hooks`. Every other
// test in this file keeps the existing-decision's hook present in the
// registry passed to Run, so this is the one case that actually exercises
// "existing survives when absent from hooks", not just "existing survives
// when also present in hooks".
func TestRun_PreservesDecisionForHookRemovedFromRegistry(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	seedHookDecisions(t, root, map[string]bool{"gone-hook": false, "kept-hook": true})
	hooks := []skills.HookDef{{Name: "kept-hook", Description: "does a thing"}}

	rc := update.Run(root, false, "", false, false, false, false, nil, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}
	if enabled, decided := hookDecision(t, root, "gone-hook"); !decided || enabled {
		t.Errorf("HookDecision(gone-hook) = (%v, %v), want (false, true) — preserved despite absence from the registry", enabled, decided)
	}
	if enabled, decided := hookDecision(t, root, "kept-hook"); !decided || !enabled {
		t.Errorf("HookDecision(kept-hook) = (%v, %v), want (true, true) — unchanged", enabled, decided)
	}
}

// TestRun_NewHookDeclinesByDefaultWithoutEnableFlag: a newly-introduced
// registry hook not named via --enable-hook declines (non-TTY default per
// ADR-0032) — recorded as decided=true/enabled=false, not left undecided
// (which would re-prompt every future run).
func TestRun_NewHookDeclinesByDefaultWithoutEnableFlag(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	hooks := []skills.HookDef{{Name: "new-hook", Description: "does a thing"}}

	rc := update.Run(root, false, "", false, false, false, false, nil, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}
	enabled, decided := hookDecision(t, root, "new-hook")
	if !decided || enabled {
		t.Errorf("HookDecision(new-hook) = (%v, %v), want (false, true)", enabled, decided)
	}
}

// TestRun_EmptyRegistryLeavesExistingHooksBlockUntouched pins today's real
// production behavior: with the shipped registry empty (M-0236 hasn't
// registered a concrete hook yet), `aiwf update` doesn't touch the hooks:
// block at all — an existing decision survives byte-for-byte.
func TestRun_EmptyRegistryLeavesExistingHooksBlockUntouched(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	seedHookDecisions(t, root, map[string]bool{"existing-hook": true})

	rc := update.Run(root, false, "", false, false, false, false, nil, nil)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}
	enabled, decided := hookDecision(t, root, "existing-hook")
	if !decided || !enabled {
		t.Errorf("HookDecision(existing-hook) = (%v, %v), want (true, true) — untouched", enabled, decided)
	}
}

// TestRun_HookMaterializesScriptAndWiresSettingsWhenEnabled pins
// M-0236/AC-4's core claim through the actual update.Run seam: a
// newly-introduced hook enabled via --enable-hook gets its script
// written to disk and its command wired into every one of its Events.
func TestRun_HookMaterializesScriptAndWiresSettingsWhenEnabled(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	hooks := []skills.HookDef{{
		Name:    "test-hook.sh",
		Content: []byte("#!/bin/sh\necho hi\n"),
		Events:  []string{"SessionStart", "SubagentStart"},
	}}

	rc := update.Run(root, false, "", false, false, false, false, []string{"test-hook.sh"}, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}

	scriptPath := filepath.Join(root, skills.ClaudeTarget.HooksDir, "test-hook.sh")
	if _, statErr := os.Stat(scriptPath); statErr != nil {
		t.Errorf("expected %s to exist, stat err=%v", scriptPath, statErr)
	}
	settingsPath := filepath.Join(root, skills.SharedSettingsRelPath)
	wired, wiredErr := skills.HookCommandWired(settingsPath, hooks[0].Command(skills.ClaudeTarget))
	if wiredErr != nil {
		t.Fatalf("HookCommandWired: %v", wiredErr)
	}
	if !wired {
		t.Error("expected the enabled hook's command to be wired into settings.json")
	}
}

// TestRun_HookRemovedWhenFlippedFromEnabledToDeclined pins ADR-0032's
// "remove both when false" half through the actual update.Run seam: a
// hook previously enabled+synced, then hand-edited to enabled: false
// in aiwf.yaml, has both its script and settings.json entry removed on
// the next `aiwf update` — no re-prompt, no --enable-hook needed.
func TestRun_HookRemovedWhenFlippedFromEnabledToDeclined(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	hooks := []skills.HookDef{{
		Name:    "test-hook.sh",
		Content: []byte("#!/bin/sh\necho hi\n"),
		Events:  []string{"SessionStart"},
	}}
	if rc := update.Run(root, false, "", false, false, false, false, []string{"test-hook.sh"}, hooks); rc != cliutil.ExitOK {
		t.Fatalf("priming Run() = %d, want ExitOK", rc)
	}
	scriptPath := filepath.Join(root, skills.ClaudeTarget.HooksDir, "test-hook.sh")
	if _, statErr := os.Stat(scriptPath); statErr != nil {
		t.Fatalf("priming Run() didn't materialize %s, stat err=%v — the removal assertion below would pass vacuously otherwise", scriptPath, statErr)
	}
	settingsPathBefore := filepath.Join(root, skills.SharedSettingsRelPath)
	wiredBefore, wiredBeforeErr := skills.HookCommandWired(settingsPathBefore, hooks[0].Command(skills.ClaudeTarget))
	if wiredBeforeErr != nil {
		t.Fatalf("HookCommandWired: %v", wiredBeforeErr)
	}
	if !wiredBefore {
		t.Fatal("priming Run() didn't wire the command into settings.json — the unwire assertion below would pass vacuously otherwise")
	}

	seedHookDecisions(t, root, map[string]bool{"test-hook.sh": false})
	if rc := update.Run(root, false, "", false, false, false, false, nil, hooks); rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}

	if _, statErr := os.Stat(scriptPath); !os.IsNotExist(statErr) {
		t.Errorf("expected %s to be removed after flipping to declined, stat err=%v", scriptPath, statErr)
	}
	wired, wiredErr := skills.HookCommandWired(settingsPathBefore, hooks[0].Command(skills.ClaudeTarget))
	if wiredErr != nil {
		t.Fatalf("HookCommandWired: %v", wiredErr)
	}
	if wired {
		t.Error("expected the flipped-to-declined hook's command to be unwired from settings.json")
	}
}

// TestNewCmd_EnableHookFlagParsesAndReachesRun exercises the actual Cobra
// wiring (flag registration through the RunE closure), not just a direct
// Run call. The shipped registry is empty in production, so this cannot
// observe a hook actually being gated; it proves --enable-hook parses
// without error and the command completes.
func TestNewCmd_EnableHookFlagParsesAndReachesRun(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	cmd := update.NewCmd()
	cmd.SetArgs([]string{"--root", root, "--enable-hook", "some-hook"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute(): %v", err)
	}
}
