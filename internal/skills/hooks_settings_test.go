package skills

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// hookCommandsForEvent reads back settingsPath and returns the command
// strings wired under the named event, in file order, for assertions.
func hookCommandsForEvent(t *testing.T, settingsPath, event string) []string {
	t.Helper()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading %s: %v", settingsPath, err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("unmarshaling %s: %v", settingsPath, err)
	}
	var hooks map[string][]struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(obj["hooks"], &hooks); err != nil {
		t.Fatalf("unmarshaling hooks key: %v", err)
	}
	var commands []string
	for _, group := range hooks[event] {
		for _, h := range group.Hooks {
			commands = append(commands, h.Command)
		}
	}
	return commands
}

// TestWireHookSettings_CreatesFileWhenMissing asserts a missing
// settings file is created fresh with just the requested hook wired,
// no .bak (nothing existed to back up).
func TestWireHookSettings_CreatesFileWhenMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	res, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"})
	if err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	if !res.Wrote {
		t.Error("Wrote must be true when the file is created fresh")
	}
	if res.BackupPath != "" {
		t.Errorf("BackupPath must be empty when no prior file existed, got %q", res.BackupPath)
	}
	if got, want := res.WiredEvents, []string{"SessionStart"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("WiredEvents = %v, want %v", got, want)
	}

	got := hookCommandsForEvent(t, settingsPath, "SessionStart")
	if len(got) != 1 || got[0] != ".claude/hooks/foo.sh" {
		t.Errorf("SessionStart commands = %v, want [.claude/hooks/foo.sh]", got)
	}
}

// TestWireHookSettings_AppendsNewMatcherGroupPreservingExistingEntries
// asserts a pre-existing foreign entry under the same event, and
// unrelated top-level keys, survive untouched — the writer only ever
// appends a new matcher-group, never edits or removes what's there.
func TestWireHookSettings_AppendsNewMatcherGroupPreservingExistingEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	original := []byte(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Agent",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/validate-agent-isolation.sh" }
        ]
      }
    ]
  },
  "enabledPlugins": {},
  "extraKnownMarketplaces": {}
}
`)
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"PreToolUse"})
	if err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	if !res.Wrote {
		t.Error("Wrote must be true when a new entry is appended")
	}
	if res.BackupPath == "" {
		t.Error("BackupPath must be non-empty when editing a pre-existing file")
	}

	got := hookCommandsForEvent(t, settingsPath, "PreToolUse")
	want := []string{".claude/hooks/validate-agent-isolation.sh", ".claude/hooks/foo.sh"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("PreToolUse commands = %v, want %v (foreign entry preserved, ours appended)", got, want)
	}

	after, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var afterObj map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(after, &afterObj); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	for _, key := range []string{"enabledPlugins", "extraKnownMarketplaces"} {
		if _, ok := afterObj[key]; !ok {
			t.Errorf("unrelated top-level key %q was dropped", key)
		}
	}

	backup, err := os.ReadFile(res.BackupPath)
	if err != nil {
		t.Fatalf("reading backup %s: %v", res.BackupPath, err)
	}
	if !bytes.Equal(backup, original) {
		t.Errorf("backup content does not match pre-edit original\nbackup: %s\noriginal: %s", backup, original)
	}
}

// TestWireHookSettings_IdempotentOnRepeatRunNoDuplicateEntries asserts
// a second call with the same command+event is a no-op: no duplicate
// entry, no second .bak, Wrote=false.
func TestWireHookSettings_IdempotentOnRepeatRunNoDuplicateEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	if _, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"}); err != nil {
		t.Fatalf("first WireHookSettings: %v", err)
	}

	res, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"})
	if err != nil {
		t.Fatalf("second WireHookSettings: %v", err)
	}
	if res.Wrote {
		t.Error("Wrote must be false on an idempotent repeat run")
	}
	if len(res.WiredEvents) != 0 {
		t.Errorf("WiredEvents must be empty on an idempotent repeat run, got %v", res.WiredEvents)
	}
	if res.BackupPath != "" {
		t.Errorf("BackupPath must be empty on an idempotent repeat run, got %q", res.BackupPath)
	}

	got := hookCommandsForEvent(t, settingsPath, "SessionStart")
	if len(got) != 1 {
		t.Errorf("SessionStart commands = %v, want exactly one entry (no duplicate)", got)
	}
}

// TestWireHookSettings_ComposesAcrossMultipleEventArrays asserts one
// call wiring several events populates every named event array, and a
// repeat call adds nothing further to any of them.
func TestWireHookSettings_ComposesAcrossMultipleEventArrays(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	events := []string{"SessionStart", "SubagentStart"}
	if _, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", events); err != nil {
		t.Fatalf("first WireHookSettings: %v", err)
	}
	for _, event := range events {
		got := hookCommandsForEvent(t, settingsPath, event)
		if len(got) != 1 || got[0] != ".claude/hooks/foo.sh" {
			t.Errorf("event %s commands = %v, want exactly [.claude/hooks/foo.sh]", event, got)
		}
	}

	res, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", events)
	if err != nil {
		t.Fatalf("second WireHookSettings: %v", err)
	}
	if res.Wrote {
		t.Error("Wrote must be false when every requested event is already wired")
	}
	for _, event := range events {
		got := hookCommandsForEvent(t, settingsPath, event)
		if len(got) != 1 {
			t.Errorf("event %s commands = %v after repeat call, want exactly one entry (no duplicate)", event, got)
		}
	}
}

// TestWireHookSettings_MixedAlreadyWiredAndNewEventInSameCall asserts
// that when one requested event is already wired and another isn't,
// only the new one is reported as newly wired and only it gains an
// entry — the already-wired event is left exactly as it was.
func TestWireHookSettings_MixedAlreadyWiredAndNewEventInSameCall(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	if _, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"}); err != nil {
		t.Fatalf("priming WireHookSettings: %v", err)
	}

	res, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart", "SubagentStart"})
	if err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	if !res.Wrote {
		t.Error("Wrote must be true when at least one requested event is newly wired")
	}
	if len(res.WiredEvents) != 1 || res.WiredEvents[0] != "SubagentStart" {
		t.Errorf("WiredEvents = %v, want [SubagentStart] (SessionStart was already wired)", res.WiredEvents)
	}

	sessionStart := hookCommandsForEvent(t, settingsPath, "SessionStart")
	if len(sessionStart) != 1 {
		t.Errorf("SessionStart commands = %v, want exactly one entry (must not duplicate the already-wired event)", sessionStart)
	}
	subagentStart := hookCommandsForEvent(t, settingsPath, "SubagentStart")
	if len(subagentStart) != 1 || subagentStart[0] != ".claude/hooks/foo.sh" {
		t.Errorf("SubagentStart commands = %v, want exactly [.claude/hooks/foo.sh]", subagentStart)
	}
}

// TestWireHookSettings_UnrelatedEventArrayUntouched asserts an event
// not named in the call is left completely alone.
func TestWireHookSettings_UnrelatedEventArrayUntouched(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	original := []byte(`{"hooks":{"Stop":[{"matcher":"","hooks":[{"type":"command","command":".claude/hooks/other.sh"}]}]}}` + "\n")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"}); err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}

	got := hookCommandsForEvent(t, settingsPath, "Stop")
	if len(got) != 1 || got[0] != ".claude/hooks/other.sh" {
		t.Errorf("Stop commands = %v, want unchanged [.claude/hooks/other.sh]", got)
	}
}

// TestWireHookSettings_MalformedHooksKeyReturnsError asserts a
// pre-existing `hooks` key of the wrong JSON shape (not an
// event-name-keyed object) surfaces as an error rather than a silent
// clobber or panic.
func TestWireHookSettings_MalformedHooksKeyReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"hooks": "not-an-object"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"})
	if err == nil {
		t.Fatal("WireHookSettings: want error for malformed hooks key, got nil")
	}
}

// TestWireHookSettings_MalformedTopLevelJSONReturnsError asserts a
// settings file that isn't valid JSON at all surfaces as an error.
func TestWireHookSettings_MalformedTopLevelJSONReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{not valid json`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"})
	if err == nil {
		t.Fatal("WireHookSettings: want error for malformed top-level JSON, got nil")
	}
}

// TestWireHookSettings_ReadErrorSurfacesRatherThanTreatedAsMissing
// asserts a non-ENOENT read failure (settingsPath is a directory)
// surfaces as an error rather than being treated as "file absent"
// (mirrors TestStatuslineSettingsKeyStatus_ReadError).
func TestWireHookSettings_ReadErrorSurfacesRatherThanTreatedAsMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.Mkdir(settingsPath, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"})
	if err == nil {
		t.Fatal("WireHookSettings: want error when settingsPath is a directory, got nil")
	}
}

// TestWireHookSettings_NullHooksKeyTreatedAsEmpty asserts an explicit
// `"hooks": null` — valid JSON, unmarshals to a nil map without error —
// is treated the same as an absent hooks key, not a nil-map panic.
func TestWireHookSettings_NullHooksKeyTreatedAsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"hooks": null}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", []string{"SessionStart"})
	if err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	if !res.Wrote {
		t.Error("Wrote must be true when wiring the first entry after a null hooks key")
	}

	got := hookCommandsForEvent(t, settingsPath, "SessionStart")
	if len(got) != 1 || got[0] != ".claude/hooks/foo.sh" {
		t.Errorf("SessionStart commands = %v, want [.claude/hooks/foo.sh]", got)
	}
}

// TestWireHookSettings_NoEventsIsNoOp asserts calling with an empty
// events slice touches nothing and reports no write.
func TestWireHookSettings_NoEventsIsNoOp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	res, err := WireHookSettings(settingsPath, ".claude/hooks/foo.sh", nil)
	if err != nil {
		t.Fatalf("WireHookSettings: %v", err)
	}
	if res.Wrote {
		t.Error("Wrote must be false when events is empty")
	}
	if len(res.WiredEvents) != 0 {
		t.Errorf("WiredEvents must be empty when events is empty, got %v", res.WiredEvents)
	}
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Errorf("no file should be created when events is empty, stat err=%v", err)
	}
}
