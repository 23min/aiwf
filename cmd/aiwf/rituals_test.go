package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRitualsPluginInstalled_DetectsProjectScope verifies the heuristic
// finds an `aiwf-extensions` reference in .claude/settings.json.
func TestRitualsPluginInstalled_DetectsProjectScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	claude := filepath.Join(root, ".claude")
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(claude, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"enabledPlugins":{"aiwf-extensions@ai-workflow-rituals":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if !ritualsPluginInstalled(root) {
		t.Error("expected detection from project-scope settings.json")
	}
}

// TestRitualsPluginInstalled_DetectsLocalScope verifies detection in
// the local-scope settings.local.json file too.
func TestRitualsPluginInstalled_DetectsLocalScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	claude := filepath.Join(root, ".claude")
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(claude, "settings.local.json")
	if err := os.WriteFile(settings, []byte(`{"enabledPlugins":{"aiwf-extensions@ai-workflow-rituals":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if !ritualsPluginInstalled(root) {
		t.Error("expected detection from local-scope settings.local.json")
	}
}

// TestRitualsPluginInstalled_NoSettings: no settings files at all → not
// detected (the common case; user hasn't installed the plugin or has it
// at user scope).
func TestRitualsPluginInstalled_NoSettings(t *testing.T) {
	t.Parallel()
	if ritualsPluginInstalled(t.TempDir()) {
		t.Error("expected non-detection when no settings file exists")
	}
}

// TestRitualsPluginInstalled_OtherPluginsOnly: settings exists but
// references other plugins, not aiwf-extensions → not detected.
func TestRitualsPluginInstalled_OtherPluginsOnly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	claude := filepath.Join(root, ".claude")
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(claude, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"enabledPlugins":{"some-other-plugin@somewhere-else":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if ritualsPluginInstalled(root) {
		t.Error("expected non-detection when settings reference unrelated plugins")
	}
}

// TestPrintRitualsSuggestion_ContainsKeyLines: the suggestion output
// names the marketplace add command, both recommended plugins, and
// steers the operator to the interactive `/plugin` menu at PROJECT
// scope (per G-0069 — the CLI install form defaults to user scope and
// doesn't satisfy `aiwf doctor`'s recommended-plugins check).
func TestPrintRitualsSuggestion_ContainsKeyLines(t *testing.T) {
	out := captureStdout(t, func() {
		printRitualsSuggestion()
	})
	got := string(out)

	for _, want := range []string{
		"/plugin marketplace add 23min/ai-workflow-rituals",
		"aiwf-extensions@ai-workflow-rituals",
		"wf-rituals@ai-workflow-rituals",
		"Recommended next step",
		"Discover tab",
		"PROJECT scope",
		"aiwf doctor",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

// TestPrintRitualsSuggestion_DoesNotRecommendCLIInstallForm pins
// G-0069's fix: the nudge must not direct operators to the CLI install
// form `/plugin install <name>@<marketplace>` because that form
// defaults to user scope and leaves the recommended-plugins check
// stuck warning. The fix is to steer operators to the interactive
// menu instead.
func TestPrintRitualsSuggestion_DoesNotRecommendCLIInstallForm(t *testing.T) {
	out := captureStdout(t, func() {
		printRitualsSuggestion()
	})
	got := string(out)

	for _, forbidden := range []string{
		"/plugin install aiwf-extensions@ai-workflow-rituals",
		"/plugin install wf-rituals@ai-workflow-rituals",
	} {
		if strings.Contains(got, forbidden) {
			t.Errorf("nudge still recommends user-scope CLI form %q (G-0069):\n%s", forbidden, got)
		}
	}
}

// Pre-M-070 the doctor verb carried a hardcoded soft note for the
// `aiwf-extensions` plugin (greppable via `.claude/settings*.json`).
// M-070 replaced that with a config-driven check
// (doctor.recommended_plugins → installed_plugins.json) covered by
// TestDoctorReport_RecommendedPlugins_* in admin_cmd_test.go. The
// previous tests for the old block (TestDoctorReport_NotesMissingPlugin,
// TestDoctorReport_NotesPresentPlugin) were deleted alongside the
// block. The `ritualsPluginInstalled` / `printRitualsSuggestion`
// functions remain because `aiwf init` still calls them as a
// one-shot setup nudge — see the Init usage tests above.
