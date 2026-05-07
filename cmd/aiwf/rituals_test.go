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
	if ritualsPluginInstalled(t.TempDir()) {
		t.Error("expected non-detection when no settings file exists")
	}
}

// TestRitualsPluginInstalled_OtherPluginsOnly: settings exists but
// references other plugins, not aiwf-extensions → not detected.
func TestRitualsPluginInstalled_OtherPluginsOnly(t *testing.T) {
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
// names both plugins and the marketplace add command.
func TestPrintRitualsSuggestion_ContainsKeyLines(t *testing.T) {
	out := captureStdout(t, func() {
		printRitualsSuggestion()
	})
	got := string(out)

	for _, want := range []string{
		"/plugin marketplace add 23min/ai-workflow-rituals",
		"/plugin install aiwf-extensions@ai-workflow-rituals",
		"/plugin install wf-rituals@ai-workflow-rituals",
		"Recommended next step",
		"Optional",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
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
