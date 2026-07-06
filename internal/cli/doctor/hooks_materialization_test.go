package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/skills"
)

// TestAppendHookMaterializationReport_EmptyRegistryReportsOk covers
// today's real state: skills.ShippedHooks is empty until M-0236, so
// the report is a quiet ok line, not an omitted row.
func TestAppendHookMaterializationReport_EmptyRegistryReportsOk(t *testing.T) {
	t.Parallel()
	out, problems := appendHookMaterializationReport(nil, nil, t.TempDir(), nil)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "hooks:") || !strings.Contains(joined, "no hooks registered yet") {
		t.Errorf("expected an empty-registry ok line, got:\n%s", joined)
	}
	if len(problems) != 0 {
		t.Errorf("expected no problems for an empty registry, got %v", problems)
	}
}

// TestAppendHookMaterializationReport_MissingAiwfYamlTreatsAllUndecided
// covers the graceful-default path: a repo with no aiwf.yaml yet is
// "every hook undecided", not an error (mirrors appendRenderReport's
// config.Load-missing handling).
func TestAppendHookMaterializationReport_MissingAiwfYamlTreatsAllUndecided(t *testing.T) {
	t.Parallel()
	root := t.TempDir() // deliberately no aiwf.yaml
	hooks := []skills.HookDef{{Name: "h1.sh", Description: "does a thing", Content: []byte("x")}}
	out, problems := appendHookMaterializationReport(nil, nil, root, hooks)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "undecided: h1.sh") {
		t.Errorf("expected h1.sh reported undecided, got:\n%s", joined)
	}
	for _, p := range problems {
		if p.Severity == SeverityError {
			t.Errorf("missing aiwf.yaml must not be an error, got %v", p)
		}
	}
}

// TestAppendHookMaterializationReport_AllSyncedReportsOk covers the
// clean state: a hook decided true, materialized, and wired reports
// no drift.
func TestAppendHookMaterializationReport_AllSyncedReportsOk(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAMLWithHookDecision(t, root, "h1.sh", true)
	hooks := []skills.HookDef{{Name: "h1.sh", Description: "does a thing", Content: []byte("script")}}
	if err := skills.MaterializeHooks(root, skills.ClaudeTarget, hooks, map[string]bool{"h1.sh": true}); err != nil {
		t.Fatalf("MaterializeHooks: %v", err)
	}
	settingsPath := filepath.Join(root, skills.SharedSettingsRelPath)
	command := skills.ClaudeTarget.HooksDir + "/h1.sh"
	if _, err := skills.WireHookSettings(settingsPath, command, []string{"SessionStart"}); err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}

	out, problems := appendHookMaterializationReport(nil, nil, root, hooks)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "ok (1 hooks synced)") {
		t.Errorf("expected a clean sync line, got:\n%s", joined)
	}
	if len(problems) != 0 {
		t.Errorf("expected no problems for a fully-synced hook, got %v", problems)
	}
}

// TestAppendHookMaterializationReport_DriftReportsWarning covers a
// decided-true hook whose script was never materialized or wired —
// drift the report must surface as a warning, not silently.
func TestAppendHookMaterializationReport_DriftReportsWarning(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAMLWithHookDecision(t, root, "h1.sh", true)
	hooks := []skills.HookDef{{Name: "h1.sh", Description: "does a thing", Content: []byte("script")}}

	out, problems := appendHookMaterializationReport(nil, nil, root, hooks)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "drift:") {
		t.Errorf("expected a drift line, got:\n%s", joined)
	}
	foundWarn := false
	for _, p := range problems {
		if p.Severity == SeverityWarn {
			foundWarn = true
		}
		if p.Severity == SeverityError {
			t.Errorf("hook drift is advisory, not an error: %v", p)
		}
	}
	if !foundWarn {
		t.Errorf("expected a warning problem for hook drift, got %v", problems)
	}
}

// TestAppendHookMaterializationReport_WiredButStaleReportsWarning
// covers the third drift class end-to-end through the doctor report:
// a hook decided false whose command is still wired into settings.json
// must surface a "wired-but-stale" line and a warning problem.
func TestAppendHookMaterializationReport_WiredButStaleReportsWarning(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAMLWithHookDecision(t, root, "h1.sh", false)
	settingsPath := filepath.Join(root, skills.SharedSettingsRelPath)
	command := skills.ClaudeTarget.HooksDir + "/h1.sh"
	if _, err := skills.WireHookSettings(settingsPath, command, []string{"SessionStart"}); err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	hooks := []skills.HookDef{{Name: "h1.sh", Description: "does a thing", Content: []byte("script")}}

	out, problems := appendHookMaterializationReport(nil, nil, root, hooks)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "wired-but-stale: h1.sh") {
		t.Errorf("expected a wired-but-stale line, got:\n%s", joined)
	}
	foundWarn := false
	for _, p := range problems {
		if p.Severity == SeverityWarn {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Errorf("expected a warning problem for wired-but-stale drift, got %v", problems)
	}
}

// TestAppendHookMaterializationReport_MalformedTopLevelAiwfYamlReturnsError
// covers the default branch: aiwf.yaml exists but fails to parse for a
// reason other than not-exist (here, a top-level YAML sequence rather
// than a mapping) — a genuine Read error, distinct from the missing
// -file and unknown-hooks-field cases already covered.
func TestAppendHookMaterializationReport_MalformedTopLevelAiwfYamlReturnsError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	raw := "- not\n- a\n- mapping\n"
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []skills.HookDef{{Name: "h1.sh", Description: "does a thing", Content: []byte("script")}}

	_, problems := appendHookMaterializationReport(nil, nil, root, hooks)
	foundErr := false
	for _, p := range problems {
		if p.Severity == SeverityError {
			foundErr = true
		}
	}
	if !foundErr {
		t.Errorf("expected an error problem for a non-mapping aiwf.yaml, got %v", problems)
	}
}

// TestAppendHookMaterializationReport_MalformedAiwfYamlReturnsError
// covers the propagated decode error: a hand-edited hooks: block with
// an unrecognized field fails the strict decode, and that must surface
// as an error, not be swallowed.
func TestAppendHookMaterializationReport_MalformedAiwfYamlReturnsError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	raw := "hooks:\n  h1.sh:\n    unknown_field: true\n"
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []skills.HookDef{{Name: "h1.sh", Description: "does a thing", Content: []byte("script")}}

	_, problems := appendHookMaterializationReport(nil, nil, root, hooks)
	foundErr := false
	for _, p := range problems {
		if p.Severity == SeverityError {
			foundErr = true
		}
	}
	if !foundErr {
		t.Errorf("expected an error problem for a malformed aiwf.yaml, got %v", problems)
	}
}

// TestAppendHookMaterializationReport_MalformedSettingsReturnsError
// covers the second propagated error path: a valid decision but a
// malformed shared settings.json.
func TestAppendHookMaterializationReport_MalformedSettingsReturnsError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAMLWithHookDecision(t, root, "h1.sh", true)
	settingsPath := filepath.Join(root, skills.SharedSettingsRelPath)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []skills.HookDef{{Name: "h1.sh", Description: "does a thing", Content: []byte("script")}}

	_, problems := appendHookMaterializationReport(nil, nil, root, hooks)
	foundErr := false
	for _, p := range problems {
		if p.Severity == SeverityError {
			foundErr = true
		}
	}
	if !foundErr {
		t.Errorf("expected an error problem for malformed settings.json, got %v", problems)
	}
}

// writeAiwfYAMLWithHookDecision writes a minimal aiwf.yaml recording a
// single hook decision, matching the hooks: block shape
// internal/aiwfyaml decodes (config.Hook{Enabled *bool}).
func writeAiwfYAMLWithHookDecision(t *testing.T, root, name string, enabled bool) {
	t.Helper()
	raw := "hooks:\n  " + name + ":\n    enabled: " + boolYAML(enabled) + "\n"
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(raw), 0o644); err != nil {
		t.Fatalf("writing aiwf.yaml fixture: %v", err)
	}
}

func boolYAML(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
