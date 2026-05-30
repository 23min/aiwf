package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/doctor"
)

// initForDoctor inits a repo (materializing verb + ritual artifacts) and
// returns its root. --skip-hook keeps the test from installing the test
// binary as a git hook.
func initForDoctor(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	t.Setenv("HOME", t.TempDir())
	return root
}

// TestDoctorReport_RitualsMaterialized_OK covers M-0152 AC-1: after
// init materializes the rituals, doctor reports a `rituals:` ok line
// and no "not materialized" warning.
func TestDoctorReport_RitualsMaterialized_OK(t *testing.T) {
	root := initForDoctor(t)
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "rituals:") {
		t.Errorf("expected a rituals: line; got:\n%s", joined)
	}
	if strings.Contains(joined, "not materialized") {
		t.Errorf("freshly-inited repo should not warn about missing rituals; got:\n%s", joined)
	}
	if strings.Contains(joined, "artifacts materialized") == false {
		t.Errorf("expected rituals ok line naming materialized artifacts; got:\n%s", joined)
	}
}

// TestDoctorReport_RitualsMissing_WarnsSoft covers M-0152 AC-1: when
// ritual artifacts are absent, doctor emits a soft warning pointing at
// `aiwf update` and does NOT increment the problem count for it.
func TestDoctorReport_RitualsMissing_WarnsSoft(t *testing.T) {
	root := initForDoctor(t)
	// Capture the baseline problem count, then remove materialized
	// templates so the rituals check has something missing.
	_, before := doctor.DoctorReport(root, doctor.DoctorOptions{})
	if err := os.RemoveAll(filepath.Join(root, ".claude", "templates")); err != nil {
		t.Fatal(err)
	}
	lines, after := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "not materialized") || !strings.Contains(joined, "aiwf update") {
		t.Errorf("expected 'not materialized — run aiwf update' warning; got:\n%s", joined)
	}
	if after != before {
		t.Errorf("missing rituals must be a soft warning (problems unchanged): before=%d after=%d", before, after)
	}
}

// writeEnabledPlugins writes a .claude/settings.json declaring the given
// plugin enabled. Returns the file path.
func writeEnabledPlugins(t *testing.T, root, plugin string, enabled bool) string {
	t.Helper()
	dir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "settings.json")
	val := "false"
	if enabled {
		val = "true"
	}
	body := []byte(`{"enabledPlugins":{"` + plugin + `":` + val + `}}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestDoctorReport_MarketplaceOverlap_WarnsNoSettingsEdit covers M-0152
// AC-2: with rituals materialized AND a marketplace plugin enabled,
// doctor warns to disable the plugin and does NOT modify settings.json.
func TestDoctorReport_MarketplaceOverlap_WarnsNoSettingsEdit(t *testing.T) {
	root := initForDoctor(t)
	settingsPath := writeEnabledPlugins(t, root, "aiwf-extensions@ai-workflow-rituals", true)
	before, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "marketplace-rituals-overlap") {
		t.Errorf("expected marketplace-rituals-overlap warning; got:\n%s", joined)
	}
	if !strings.Contains(joined, "aiwf-extensions@ai-workflow-rituals") {
		t.Errorf("overlap warning should name the enabled plugin; got:\n%s", joined)
	}
	if !strings.Contains(joined, "disable") {
		t.Errorf("overlap warning should instruct disable; got:\n%s", joined)
	}
	after, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("doctor must not edit settings.json (ADR-0014 §5); before=%q after=%q", before, after)
	}
}

// TestDoctorReport_NoOverlap_WhenPluginDisabled covers AC-2's negative:
// plugin present but disabled → no overlap warning.
func TestDoctorReport_NoOverlap_WhenPluginDisabled(t *testing.T) {
	root := initForDoctor(t)
	writeEnabledPlugins(t, root, "aiwf-extensions@ai-workflow-rituals", false)
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	if strings.Contains(strings.Join(lines, "\n"), "marketplace-rituals-overlap") {
		t.Errorf("disabled plugin must not trigger overlap warning:\n%s", strings.Join(lines, "\n"))
	}
}

// TestDoctorReport_NoOverlap_WhenNotMaterialized covers AC-2's other
// negative: plugin enabled but rituals not materialized → no overlap
// (only one side of the duplication present).
func TestDoctorReport_NoOverlap_WhenNotMaterialized(t *testing.T) {
	root := initForDoctor(t)
	writeEnabledPlugins(t, root, "aiwf-extensions@ai-workflow-rituals", true)
	// Remove all materialized ritual artifacts.
	if err := os.RemoveAll(filepath.Join(root, ".claude", "agents")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".claude", "templates")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"aiwfx-plan-epic", "aiwfx-start-milestone", "aiwfx-wrap-epic", "aiwfx-plan-milestones", "aiwfx-start-epic", "aiwfx-wrap-milestone", "aiwfx-record-decision", "aiwfx-release", "aiwfx-whiteboard", "wf-tdd-cycle", "wf-review-code", "wf-doc-lint", "wf-patch"} {
		_ = os.RemoveAll(filepath.Join(root, ".claude", "skills", name))
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	if strings.Contains(strings.Join(lines, "\n"), "marketplace-rituals-overlap") {
		t.Errorf("no overlap expected when rituals are not materialized:\n%s", strings.Join(lines, "\n"))
	}
}

// TestDoctorReport_RitualsLine_SurfacesProvenance covers the
// discoverability enhancement: the rituals: ok line is followed by a
// note naming aiwf as the manager, the refresh verb, and the
// do-not-hand-edit contract — so an operator reading `aiwf doctor`
// learns these skills came from aiwf.
func TestDoctorReport_RitualsLine_SurfacesProvenance(t *testing.T) {
	root := initForDoctor(t)
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"managed by aiwf", "aiwf update", "do not hand-edit"} {
		if !strings.Contains(joined, want) {
			t.Errorf("doctor rituals provenance note missing %q; got:\n%s", want, joined)
		}
	}
}
