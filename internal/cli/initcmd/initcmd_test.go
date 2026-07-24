package initcmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/initcmd"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/skills"
)

func freshGitRepo(t *testing.T) string {
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
	return root
}

// TestRun_HooksGatedAndBakedIntoFreshAiwfYaml pins M-0235/AC-2's core
// claim: a registry hook named via --enable-hook is baked into the
// freshly-written aiwf.yaml as `enabled: true`, and the full commented
// schema reference initrepo.Init already wrote survives untouched — the
// consent gate runs as a separate aiwfyaml splice after the main init
// pipeline, never through config.Write's marshal-fallback path, so it
// cannot silently drop the schema documentation the way populating
// cfg.Hooks before a raw yaml.Marshal(cfg) would.
func TestRun_HooksGatedAndBakedIntoFreshAiwfYaml(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooks := []skills.HookDef{{Name: "test-hook", Description: "does a thing"}}

	rc := initcmd.Run(root, "", false, true, false, "", false, false, false, []string{"test-hook"}, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	enabled, decided := cfg.HookDecision("test-hook")
	if !decided || !enabled {
		t.Errorf("HookDecision(test-hook) = (%v, %v), want (true, true)", enabled, decided)
	}

	raw, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	if !strings.Contains(string(raw), "# status_md:") {
		t.Errorf("aiwf.yaml lost its commented schema reference:\n%s", raw)
	}
}

// TestRun_HookLeftUndecidedWithoutEnableFlag pins G-0446: a registry hook
// not named via --enable-hook, gated where no interactive answer is
// available (here go test's non-TTY stdin), is left UNDECIDED — absent from
// aiwf.yaml's hooks: map, never recorded as a false decline. Absent-not-false
// is what surfaces it as a doctor "undecided" warning (yellow in the
// statusline) so a human decides it later, rather than a silent decline that
// hides the missed config.
func TestRun_HookLeftUndecidedWithoutEnableFlag(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooks := []skills.HookDef{{Name: "test-hook", Description: "does a thing"}}

	rc := initcmd.Run(root, "", false, true, false, "", false, false, false, nil, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if _, decided := cfg.HookDecision("test-hook"); decided {
		enabled, _ := cfg.HookDecision("test-hook")
		t.Errorf("HookDecision(test-hook) recorded a decision (enabled=%v), want it left undecided (absent)", enabled)
	}
}

// TestRun_HonorsExistingDecisionAndDefersNewHookUnderNoPrompt pins G-0446's
// core fix through the Run seam, simulating a container rebuild: init runs a
// second time (initrepo.Init preserves the existing aiwf.yaml), and under
// --no-prompt an already-decided hook is carried forward untouched — never
// re-prompted or re-defaulted to false — while a newly-registered hook the
// gate cannot decide is left undecided (absent) rather than silently
// declined. On the pre-G-0446 code this failed both ways: existing-hook was
// re-gated and clobbered to false, and new-hook was defaulted to false.
func TestRun_HonorsExistingDecisionAndDefersNewHookUnderNoPrompt(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	existingHook := skills.HookDef{Name: "existing-hook", Description: "already decided"}
	newHook := skills.HookDef{Name: "new-hook", Description: "just added to the registry"}

	// First run: enable existing-hook via the flag so it's recorded true.
	rc := initcmd.Run(root, "", false, true, false, "", false, false, true, []string{"existing-hook"}, []skills.HookDef{existingHook})
	if rc != cliutil.ExitOK {
		t.Fatalf("first Run() = %d, want ExitOK", rc)
	}

	// Second run (the rebuild): the registry now also carries new-hook;
	// --no-prompt, no --enable-hook. existing-hook must stay true (honored);
	// new-hook must be left undecided (absent).
	rc = initcmd.Run(root, "", false, true, false, "", false, false, true, nil, []skills.HookDef{existingHook, newHook})
	if rc != cliutil.ExitOK {
		t.Fatalf("second Run() = %d, want ExitOK", rc)
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if enabled, decided := cfg.HookDecision("existing-hook"); !decided || !enabled {
		t.Errorf("HookDecision(existing-hook) = (%v, %v), want (true, true) — honored across the rebuild", enabled, decided)
	}
	if _, decided := cfg.HookDecision("new-hook"); decided {
		enabled, _ := cfg.HookDecision("new-hook")
		t.Errorf("HookDecision(new-hook) recorded a decision (enabled=%v), want it left undecided so doctor surfaces it", enabled)
	}
}

// TestRun_DryRunSkipsHookGatingEntirely: --dry-run writes nothing at all,
// so the hook consent gate must not run (there is no aiwf.yaml yet to
// splice into).
func TestRun_DryRunSkipsHookGatingEntirely(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooks := []skills.HookDef{{Name: "test-hook", Description: "does a thing"}}

	rc := initcmd.Run(root, "", true, true, false, "", false, false, false, []string{"test-hook"}, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}
	if _, err := os.Stat(filepath.Join(root, config.FileName)); !os.IsNotExist(err) {
		t.Errorf("aiwf.yaml exists after --dry-run: %v", err)
	}
}

// TestRun_EmptyRegistrySkipsGatingEntirely pins today's real production
// behavior: with the shipped registry empty (no concrete hook has landed —
// M-0236), aiwf.yaml gets no hooks: block at all, unchanged from before
// this milestone.
func TestRun_EmptyRegistrySkipsGatingEntirely(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)

	rc := initcmd.Run(root, "", false, true, false, "", false, false, false, nil, nil)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}
	raw, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	if strings.Contains(string(raw), "hooks:\n") {
		t.Errorf("aiwf.yaml has a live hooks: block with an empty registry:\n%s", raw)
	}
}

// TestRun_HookMaterializesScriptAndWiresSettingsWhenEnabled pins
// M-0236/AC-4's core claim through the actual Run seam (not just the
// cliutil.SyncHookMaterialization unit): a hook enabled via
// --enable-hook gets its script written to disk under the target's
// hooks dir and its command wired into every one of its Events in
// .claude/settings.json.
func TestRun_HookMaterializesScriptAndWiresSettingsWhenEnabled(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooks := []skills.HookDef{{
		Name:    "test-hook.sh",
		Content: []byte("#!/bin/sh\necho hi\n"),
		Events:  []string{"SessionStart", "SubagentStart"},
	}}

	rc := initcmd.Run(root, "", false, true, false, "", false, false, false, []string{"test-hook.sh"}, hooks)
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

// TestRun_HookNotMaterializedWhenDeclinedByDefault pins the negative
// case: a registry hook not named via --enable-hook (declines by
// default, non-TTY) gets no script and no settings.json entry.
func TestRun_HookNotMaterializedWhenDeclinedByDefault(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooks := []skills.HookDef{{
		Name:    "test-hook.sh",
		Content: []byte("#!/bin/sh\necho hi\n"),
		Events:  []string{"SessionStart"},
	}}

	rc := initcmd.Run(root, "", false, true, false, "", false, false, false, nil, hooks)
	if rc != cliutil.ExitOK {
		t.Fatalf("Run() = %d, want ExitOK", rc)
	}

	scriptPath := filepath.Join(root, skills.ClaudeTarget.HooksDir, "test-hook.sh")
	if _, statErr := os.Stat(scriptPath); !os.IsNotExist(statErr) {
		t.Errorf("expected no script for a declined hook, stat err=%v", statErr)
	}
}

// TestNewCmd_EnableHookFlagParsesAndReachesRun exercises the actual Cobra
// wiring — flag registration through the RunE closure — rather than
// calling initcmd.Run directly the way the other Run-level tests do. The
// shipped registry is empty in production, so this cannot observe a hook
// actually being gated (that's what TestRun_HooksGatedAndBakedIntoFreshAiwfYaml
// pins); it proves --enable-hook parses without error and the command
// completes, i.e. the flag-to-Run wiring itself works.
func TestNewCmd_EnableHookFlagParsesAndReachesRun(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	cmd := initcmd.NewCmd()
	cmd.SetArgs([]string{"--root", root, "--skip-hook", "--enable-hook", "some-hook"})
	cmd.SetOut(os.Stderr)
	cmd.SetErr(os.Stderr)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute(): %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, config.FileName)); err != nil {
		t.Errorf("aiwf.yaml not written: %v", err)
	}
}

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := initcmd.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "init" {
		t.Errorf("Use = %q", cmd.Use)
	}
}

// TestNewCmd_HelpDocumentsIdempotentReRun: `aiwf init --help` (the
// command's Long description) must state the re-run is idempotent and
// name every artifact init never overwrites (M-0232/AC-5). Scoped to
// the Long field specifically — the one Cobra surface --help actually
// renders this prose from — not a blind grep over the file.
func TestNewCmd_HelpDocumentsIdempotentReRun(t *testing.T) {
	t.Parallel()
	cmd := initcmd.NewCmd()
	help := cmd.Long
	if !strings.Contains(help, "idempotent") {
		t.Errorf("Long missing an idempotent re-run statement: %q", help)
	}
	for _, never := range []string{"aiwf.yaml", ".claude/settings.json", "git hooks"} {
		if !strings.Contains(help, never) {
			t.Errorf("Long missing %q from the never-overwritten list: %q", never, help)
		}
	}
}
