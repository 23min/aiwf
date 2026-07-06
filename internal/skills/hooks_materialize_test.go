package skills

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHookSkillsFrom_ConvertsNameAndContent covers the pure HookDef ->
// Skill conversion the materialization category's lister uses.
func TestHookSkillsFrom_ConvertsNameAndContent(t *testing.T) {
	t.Parallel()
	hooks := []HookDef{
		{Name: "b-hook", Description: "second", Content: []byte("script-b")},
		{Name: "a-hook", Description: "first", Content: []byte("script-a")},
	}
	got := HookSkillsFrom(hooks)
	if len(got) != 2 {
		t.Fatalf("HookSkillsFrom: got %d skills, want 2", len(got))
	}
	if got[0].Name != "b-hook" || string(got[0].Content) != "script-b" {
		t.Errorf("HookSkillsFrom[0] = %+v, want {b-hook script-b}", got[0])
	}
	if got[1].Name != "a-hook" || string(got[1].Content) != "script-a" {
		t.Errorf("HookSkillsFrom[1] = %+v, want {a-hook script-a}", got[1])
	}
}

// TestHookSkillsFrom_EmptyInputReturnsEmpty covers the boundary: an
// empty registry (ShippedHooks today) converts to an empty slice, not
// an error.
func TestHookSkillsFrom_EmptyInputReturnsEmpty(t *testing.T) {
	t.Parallel()
	got := HookSkillsFrom(nil)
	if len(got) != 0 {
		t.Errorf("HookSkillsFrom(nil) = %v, want empty", got)
	}
}

// TestMaterializeHooks_WritesEnabledHookExecutable covers the write
// branch: a hook decided true gets its content written under
// target.HooksDir, executable (mirrors the real
// .claude/hooks/validate-agent-isolation.sh mode).
func TestMaterializeHooks_WritesEnabledHookExecutable(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := []HookDef{{Name: "check.sh", Content: []byte("#!/bin/sh\necho hi\n")}}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, map[string]bool{"check.sh": true}); err != nil {
		t.Fatalf("MaterializeHooks: %v", err)
	}
	path := filepath.Join(root, ClaudeTarget.HooksDir, "check.sh")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("hook not materialized: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("materialized hook %s is not executable: mode %v", path, info.Mode())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading materialized hook: %v", err)
	}
	if string(got) != "#!/bin/sh\necho hi\n" {
		t.Errorf("materialized hook content = %q, want script body", got)
	}
}

// TestMaterializeHooks_RemovesDeclinedHook covers the remove branch: a
// hook decided false, previously materialized, is deleted.
func TestMaterializeHooks_RemovesDeclinedHook(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, map[string]bool{"check.sh": true}); err != nil {
		t.Fatalf("materializing: %v", err)
	}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, map[string]bool{"check.sh": false}); err != nil {
		t.Fatalf("declining: %v", err)
	}
	path := filepath.Join(root, ClaudeTarget.HooksDir, "check.sh")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("declined hook still present: stat err = %v", err)
	}
}

// TestMaterializeHooks_DecliningAbsentHookIsNoOp covers the remove
// branch's idempotent case: declining a hook that was never
// materialized does not error.
func TestMaterializeHooks_DecliningAbsentHookIsNoOp(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, map[string]bool{"check.sh": false}); err != nil {
		t.Fatalf("MaterializeHooks: %v", err)
	}
	path := filepath.Join(root, ClaudeTarget.HooksDir, "check.sh")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected no file, stat err = %v", err)
	}
}

// TestMaterializeHooks_UndecidedHookUntouched covers the skip branch:
// a hook absent from decisions is neither written nor removed — the
// consent gate runs before this function, so an undecided hook here
// is left exactly as found.
func TestMaterializeHooks_UndecidedHookUntouched(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, map[string]bool{}); err != nil {
		t.Fatalf("MaterializeHooks: %v", err)
	}
	path := filepath.Join(root, ClaudeTarget.HooksDir, "check.sh")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("undecided hook should not be materialized: stat err = %v", err)
	}
}

// TestMaterializeHooks_MultipleHooksOnlyTouchesDecided covers
// composition across several hooks in one call — enabled writes,
// declined removes, undecided skips — with a foreign file (not in the
// registry) left alone.
func TestMaterializeHooks_MultipleHooksOnlyTouchesDecided(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooksDir := filepath.Join(root, ClaudeTarget.HooksDir)
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(hooksDir, "foreign.sh")
	if err := os.WriteFile(foreign, []byte("mine"), 0o755); err != nil {
		t.Fatal(err)
	}
	hooks := []HookDef{
		{Name: "enabled.sh", Content: []byte("a")},
		{Name: "declined.sh", Content: []byte("b")},
		{Name: "undecided.sh", Content: []byte("c")},
	}
	decisions := map[string]bool{"enabled.sh": true, "declined.sh": false}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, decisions); err != nil {
		t.Fatalf("MaterializeHooks: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hooksDir, "enabled.sh")); err != nil {
		t.Errorf("enabled hook not materialized: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hooksDir, "declined.sh")); !os.IsNotExist(err) {
		t.Errorf("declined hook should be absent: stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(hooksDir, "undecided.sh")); !os.IsNotExist(err) {
		t.Errorf("undecided hook should be absent: stat err = %v", err)
	}
	got, err := os.ReadFile(foreign)
	if err != nil || string(got) != "mine" {
		t.Errorf("foreign file disturbed: content=%q err=%v", got, err)
	}
}

// TestMaterializeHooks_RemoveErrorSurfaces covers the remove branch's
// real-error arm: os.Remove on a non-empty directory fails with a
// genuine (not-IsNotExist) error, deterministically reproducible
// without any disk fault — that error must propagate, not be
// swallowed alongside the IsNotExist no-op case.
func TestMaterializeHooks_RemoveErrorSurfaces(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooksDir := filepath.Join(root, ClaudeTarget.HooksDir)
	nonEmptyDir := filepath.Join(hooksDir, "check.sh")
	if err := os.MkdirAll(nonEmptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "child"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, map[string]bool{"check.sh": false}); err == nil {
		t.Error("expected an error removing a non-empty directory, got nil")
	}
}

// TestMaterializeHooks_MkdirAllErrorSurfaces covers the write branch's
// directory-creation error: a path component that already exists as a
// regular file (not a directory) makes MkdirAll fail deterministically.
func TestMaterializeHooks_MkdirAllErrorSurfaces(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	claudeDir := filepath.Join(root, ".claude")
	if err := os.WriteFile(claudeDir, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, map[string]bool{"check.sh": true}); err == nil {
		t.Error("expected an error creating hooks dir under a file, got nil")
	}
}

// TestHookDrift_UndecidedHookReported covers the undecided class: a
// registry hook with no aiwf.yaml decision at all.
func TestHookDrift_UndecidedHookReported(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	report, err := HookDrift(root, ClaudeTarget, hooks, map[string]bool{}, settingsPath)
	if err != nil {
		t.Fatalf("HookDrift: %v", err)
	}
	if len(report.Undecided) != 1 || report.Undecided[0] != "check.sh" {
		t.Errorf("Undecided = %v, want [check.sh]", report.Undecided)
	}
	if len(report.MaterializedNotWired) != 0 || len(report.WiredButStale) != 0 {
		t.Errorf("expected only Undecided populated, got %+v", report)
	}
}

// TestHookDrift_FullySyncedHookReportsClean covers the clean state: a
// hook enabled, materialized, and wired reports in none of the three
// drift classes.
func TestHookDrift_FullySyncedHookReportsClean(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	decisions := map[string]bool{"check.sh": true}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, decisions); err != nil {
		t.Fatalf("MaterializeHooks: %v", err)
	}
	command := ClaudeTarget.HooksDir + "/check.sh"
	if _, err := WireHookSettings(settingsPath, command, []string{"SessionStart"}); err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	report, err := HookDrift(root, ClaudeTarget, hooks, decisions, settingsPath)
	if err != nil {
		t.Fatalf("HookDrift: %v", err)
	}
	if len(report.Undecided) != 0 || len(report.MaterializedNotWired) != 0 || len(report.WiredButStale) != 0 {
		t.Errorf("expected a clean report, got %+v", report)
	}
}

// TestHookDrift_MaterializedButNotWiredReported covers the second
// class: enabled and materialized on disk, but settings.json carries
// no matching command.
func TestHookDrift_MaterializedButNotWiredReported(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	decisions := map[string]bool{"check.sh": true}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, decisions); err != nil {
		t.Fatalf("MaterializeHooks: %v", err)
	}
	report, err := HookDrift(root, ClaudeTarget, hooks, decisions, settingsPath)
	if err != nil {
		t.Fatalf("HookDrift: %v", err)
	}
	if len(report.MaterializedNotWired) != 1 || report.MaterializedNotWired[0] != "check.sh" {
		t.Errorf("MaterializedNotWired = %v, want [check.sh]", report.MaterializedNotWired)
	}
	if len(report.Undecided) != 0 || len(report.WiredButStale) != 0 {
		t.Errorf("expected only MaterializedNotWired populated, got %+v", report)
	}
}

// TestHookDrift_WiredDespiteDeclinedReported covers the third class:
// settings.json still carries the command even though the recorded
// decision is false (a sync that hasn't caught up yet).
func TestHookDrift_WiredDespiteDeclinedReported(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	command := ClaudeTarget.HooksDir + "/check.sh"
	if _, err := WireHookSettings(settingsPath, command, []string{"SessionStart"}); err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	report, err := HookDrift(root, ClaudeTarget, hooks, map[string]bool{"check.sh": false}, settingsPath)
	if err != nil {
		t.Fatalf("HookDrift: %v", err)
	}
	if len(report.WiredButStale) != 1 || report.WiredButStale[0] != "check.sh" {
		t.Errorf("WiredButStale = %v, want [check.sh]", report.WiredButStale)
	}
}

// TestHookDrift_EnabledButUnmaterializedReportsNotFullySynced covers
// an enabled hook whose script is missing from disk despite already
// being wired into settings.json (e.g. deleted out from under aiwf).
// This is still "not fully synced toward on", so it buckets under
// MaterializedNotWired — the same remedy (`aiwf update`) applies
// regardless of which half (script or settings entry) is the one
// missing.
func TestHookDrift_EnabledButUnmaterializedReportsNotFullySynced(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	command := ClaudeTarget.HooksDir + "/check.sh"
	if _, err := WireHookSettings(settingsPath, command, []string{"SessionStart"}); err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	report, err := HookDrift(root, ClaudeTarget, hooks, map[string]bool{"check.sh": true}, settingsPath)
	if err != nil {
		t.Fatalf("HookDrift: %v", err)
	}
	if len(report.MaterializedNotWired) != 1 || report.MaterializedNotWired[0] != "check.sh" {
		t.Errorf("MaterializedNotWired = %v, want [check.sh]", report.MaterializedNotWired)
	}
	if len(report.WiredButStale) != 0 {
		t.Errorf("expected no WiredButStale, got %v", report.WiredButStale)
	}
}

// TestHookDrift_DeclinedButMaterializedReportsStale covers the other
// WiredButStale trigger: a hook decided false whose script is still
// present on disk (never wired, but the leftover file itself is the
// staleness) must be flagged, not treated as clean.
func TestHookDrift_DeclinedButMaterializedReportsStale(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	hooksDir := filepath.Join(root, ClaudeTarget.HooksDir)
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "check.sh"), []byte("leftover"), 0o755); err != nil {
		t.Fatal(err)
	}
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	report, err := HookDrift(root, ClaudeTarget, hooks, map[string]bool{"check.sh": false}, settingsPath)
	if err != nil {
		t.Fatalf("HookDrift: %v", err)
	}
	if len(report.WiredButStale) != 1 || report.WiredButStale[0] != "check.sh" {
		t.Errorf("WiredButStale = %v, want [check.sh]", report.WiredButStale)
	}
}

// TestHookDrift_DeclinedAndUnsyncedIsClean covers the negative case: a
// hook decided false, never materialized, never wired reports in no
// drift class — there is nothing to reconcile.
func TestHookDrift_DeclinedAndUnsyncedIsClean(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	report, err := HookDrift(root, ClaudeTarget, hooks, map[string]bool{"check.sh": false}, settingsPath)
	if err != nil {
		t.Fatalf("HookDrift: %v", err)
	}
	if len(report.Undecided) != 0 || len(report.MaterializedNotWired) != 0 || len(report.WiredButStale) != 0 {
		t.Errorf("expected a clean report, got %+v", report)
	}
}

// TestHookDrift_MultipleHooksSortedAndIndependent covers composition:
// several hooks in one call are each classified independently and the
// result lists are sorted.
func TestHookDrift_MultipleHooksSortedAndIndependent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	hooks := []HookDef{
		{Name: "z-hook.sh", Content: []byte("z")},
		{Name: "a-hook.sh", Content: []byte("a")},
	}
	decisions := map[string]bool{"z-hook.sh": true, "a-hook.sh": true}
	if err := MaterializeHooks(root, ClaudeTarget, hooks, decisions); err != nil {
		t.Fatalf("MaterializeHooks: %v", err)
	}
	report, err := HookDrift(root, ClaudeTarget, hooks, decisions, settingsPath)
	if err != nil {
		t.Fatalf("HookDrift: %v", err)
	}
	want := []string{"a-hook.sh", "z-hook.sh"}
	if len(report.MaterializedNotWired) != 2 || report.MaterializedNotWired[0] != want[0] || report.MaterializedNotWired[1] != want[1] {
		t.Errorf("MaterializedNotWired = %v, want sorted %v", report.MaterializedNotWired, want)
	}
}

// TestHookDrift_MalformedSettingsSurfacesError covers the error path:
// a malformed settings.json propagates as an error rather than being
// silently treated as "not wired".
func TestHookDrift_MalformedSettingsSurfacesError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := []HookDef{{Name: "check.sh", Content: []byte("script")}}
	if _, err := HookDrift(root, ClaudeTarget, hooks, map[string]bool{"check.sh": true}, settingsPath); err == nil {
		t.Error("expected an error from malformed settings.json, got nil")
	}
}
