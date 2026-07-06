package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/23min/aiwf/internal/pathutil"
)

// HookSettingsWriteResult reports what WireHookSettings did.
type HookSettingsWriteResult struct {
	// Path is the absolute path to the settings file that was written.
	Path string

	// BackupPath is the absolute path to the .bak file written before
	// editing. Empty when no edit was needed (every requested event was
	// already wired, or the file didn't exist yet).
	BackupPath string

	// Wrote is true when at least one requested event gained a new
	// entry and the file was written.
	Wrote bool

	// WiredEvents lists the events (subset of the requested ones) that
	// were newly wired this call. An event already carrying command is
	// left untouched and excluded here.
	WiredEvents []string
}

// hookMatcherEntry is one matcher-group entry in a hooks.<event> array —
// the shape Claude Code expects (see .claude/settings.json for a live
// example): a matcher plus the list of commands it runs.
type hookMatcherEntry struct {
	Matcher string             `json:"matcher"`
	Hooks   []hookCommandEntry `json:"hooks"`
}

// hookCommandEntry is one command-type hook inside a matcher-group.
type hookCommandEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// WireHookSettings wires command as a command-type hook under every
// named event in the shared settings file at settingsPath (ADR-0032).
// The caller is responsible for consent gating — this function does
// the mechanical JSON edit unconditionally.
//
// Behavior:
//   - Append-only: an event whose array doesn't yet contain command
//     gains one new matcher-group entry (empty matcher — matches
//     unconditionally). An event that already contains command
//     anywhere in its array is left untouched (idempotent per event).
//   - No pre-existing entry, foreign or aiwf's own, is ever edited or
//     removed — composing across multiple hook-event arrays never
//     clobbers what another hook (or a hand-authored entry) already
//     registered for the same event.
//   - Every unrelated top-level key and every event not named in
//     events is preserved untouched.
//   - A .bak of the pre-edit file is written once, only when an actual
//     edit happens (mirrors WireStatuslineSettings) — an empty
//     events slice, or a fully idempotent call, writes nothing.
func WireHookSettings(settingsPath, command string, events []string) (HookSettingsWriteResult, error) {
	res := HookSettingsWriteResult{Path: settingsPath}
	if len(events) == 0 {
		return res, nil
	}

	existing, readErr := os.ReadFile(settingsPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return res, fmt.Errorf("reading %s: %w", settingsPath, readErr)
	}

	obj, parseErr := parseSettingsJSON(existing)
	if parseErr != nil {
		return res, fmt.Errorf("parsing %s: %w", settingsPath, parseErr)
	}

	hooks, hooksErr := parseHooksKey(obj)
	if hooksErr != nil {
		return res, fmt.Errorf("parsing %s hooks key: %w", settingsPath, hooksErr)
	}

	var wired []string
	for _, event := range events {
		if hookCommandPresent(hooks[event], command) {
			continue
		}
		hooks[event] = append(hooks[event], hookMatcherEntry{
			Hooks: []hookCommandEntry{{Type: "command", Command: command}},
		})
		wired = append(wired, event)
	}

	if len(wired) == 0 {
		return res, nil
	}
	sort.Strings(wired)
	res.WiredEvents = wired

	if len(existing) > 0 {
		bakPath := settingsPath + ".bak"
		if wErr := pathutil.AtomicWriteFile(bakPath, existing, 0o644); wErr != nil {
			return res, fmt.Errorf("writing backup %s: %w", bakPath, wErr) //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce (mirrors WireStatuslineSettings's identical, equally-untested shape)
		}
		res.BackupPath = bakPath
	}

	hooksBytes, mErr := json.Marshal(hooks)
	if mErr != nil { //coverage:ignore hooks is built entirely from json.Marshal-safe types (strings, structs); marshaling cannot fail
		return res, fmt.Errorf("marshaling hooks: %w", mErr)
	}
	obj["hooks"] = hooksBytes

	out, mErr := json.MarshalIndent(obj, "", "  ")
	if mErr != nil { //coverage:ignore obj's remaining values are json.RawMessage already proven valid; re-marshaling never fails
		return res, fmt.Errorf("marshaling settings: %w", mErr)
	}
	out = append(out, '\n')

	if mkErr := os.MkdirAll(filepath.Dir(settingsPath), 0o755); mkErr != nil {
		return res, fmt.Errorf("creating directory for %s: %w", settingsPath, mkErr) //coverage:ignore MkdirAll fails only on filesystem faults (permission, a path segment that's a file); every test's settingsPath sits directly in an already-writable t.TempDir()
	}
	if wErr := pathutil.AtomicWriteFile(settingsPath, out, 0o644); wErr != nil {
		return res, fmt.Errorf("writing %s: %w", settingsPath, wErr) //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce (mirrors WireStatuslineSettings's identical, equally-untested shape)
	}
	res.Wrote = true
	return res, nil
}

// parseHooksKey decodes obj's "hooks" key into the event-name-keyed map
// WireHookSettings composes over. A missing key returns an empty map
// (there's nothing to preserve yet); a present key of the wrong shape
// is a parse error rather than a silent clobber.
func parseHooksKey(obj map[string]json.RawMessage) (map[string][]hookMatcherEntry, error) {
	raw, ok := obj["hooks"]
	if !ok {
		return make(map[string][]hookMatcherEntry), nil
	}
	var hooks map[string][]hookMatcherEntry
	if err := json.Unmarshal(raw, &hooks); err != nil {
		return nil, err
	}
	if hooks == nil {
		hooks = make(map[string][]hookMatcherEntry)
	}
	return hooks, nil
}

// hookCommandPresent reports whether command already appears in any
// matcher-group of entries — the identity check WireHookSettings uses
// to decide idempotent-skip vs. append.
func hookCommandPresent(entries []hookMatcherEntry, command string) bool {
	for _, entry := range entries {
		for _, h := range entry.Hooks {
			if h.Command == command {
				return true
			}
		}
	}
	return false
}

// HookSettingsRemoveResult reports what UnwireHookSettings did.
type HookSettingsRemoveResult struct {
	// Path is the absolute path to the settings file that was written.
	Path string

	// BackupPath is the absolute path to the .bak file written before
	// editing. Empty when no edit was needed (the command wasn't wired
	// anywhere, or the file didn't exist yet).
	BackupPath string

	// Wrote is true when at least one event lost a matching entry and
	// the file was written.
	Wrote bool

	// RemovedFromEvents lists the events (sorted) that had at least one
	// matching entry removed this call.
	RemovedFromEvents []string
}

// UnwireHookSettings removes every settings.json hook-event entry whose
// command equals command (ADR-0032's "remove both when false" half —
// the counterpart to WireHookSettings, MaterializeHooks's own decline
// path already handles the on-disk script removal). Idempotent: a
// command not currently wired anywhere, or a missing settings file, is
// a silent no-op — mirrors HookCommandWired's identical
// not-yet-materialized handling. A matcher-group entry left with zero
// commands after removal is dropped entirely rather than kept as
// residue; a foreign command sharing the same matcher-group's array
// survives untouched.
func UnwireHookSettings(settingsPath, command string) (HookSettingsRemoveResult, error) {
	res := HookSettingsRemoveResult{Path: settingsPath}

	existing, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return res, nil
		}
		return res, fmt.Errorf("reading %s: %w", settingsPath, readErr)
	}

	obj, parseErr := parseSettingsJSON(existing)
	if parseErr != nil {
		return res, fmt.Errorf("parsing %s: %w", settingsPath, parseErr)
	}

	hooks, hooksErr := parseHooksKey(obj)
	if hooksErr != nil {
		return res, fmt.Errorf("parsing %s hooks key: %w", settingsPath, hooksErr)
	}

	var removedFrom []string
	for event, entries := range hooks {
		filtered, changed := removeCommandFromEntries(entries, command)
		if !changed {
			continue
		}
		if len(filtered) == 0 {
			// No matcher-groups left under this event — drop the key
			// entirely rather than leaving it as a JSON `null` (an
			// empty-but-present slice marshals to `null`, not `[]`).
			delete(hooks, event)
		} else {
			hooks[event] = filtered
		}
		removedFrom = append(removedFrom, event)
	}

	if len(removedFrom) == 0 {
		return res, nil
	}
	sort.Strings(removedFrom)
	res.RemovedFromEvents = removedFrom

	bakPath := settingsPath + ".bak"
	if wErr := pathutil.AtomicWriteFile(bakPath, existing, 0o644); wErr != nil {
		return res, fmt.Errorf("writing backup %s: %w", bakPath, wErr) //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce (mirrors WireHookSettings's identical, equally-untested shape)
	}
	res.BackupPath = bakPath

	hooksBytes, mErr := json.Marshal(hooks)
	if mErr != nil { //coverage:ignore hooks is built entirely from json.Marshal-safe types (strings, structs); marshaling cannot fail
		return res, fmt.Errorf("marshaling hooks: %w", mErr)
	}
	obj["hooks"] = hooksBytes

	out, mErr := json.MarshalIndent(obj, "", "  ")
	if mErr != nil { //coverage:ignore obj's remaining values are json.RawMessage already proven valid; re-marshaling never fails
		return res, fmt.Errorf("marshaling settings: %w", mErr)
	}
	out = append(out, '\n')

	if wErr := pathutil.AtomicWriteFile(settingsPath, out, 0o644); wErr != nil {
		return res, fmt.Errorf("writing %s: %w", settingsPath, wErr) //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce (mirrors WireHookSettings's identical, equally-untested shape)
	}
	res.Wrote = true
	return res, nil
}

// removeCommandFromEntries returns entries with every hookCommandEntry
// matching command dropped, and any matcher-group left with zero
// commands dropped entirely (no empty-group residue). changed reports
// whether anything was actually removed.
func removeCommandFromEntries(entries []hookMatcherEntry, command string) (filtered []hookMatcherEntry, changed bool) {
	for _, entry := range entries {
		var keptHooks []hookCommandEntry
		for _, h := range entry.Hooks {
			if h.Command == command {
				changed = true
				continue
			}
			keptHooks = append(keptHooks, h)
		}
		if len(keptHooks) == 0 {
			continue
		}
		entry.Hooks = keptHooks
		filtered = append(filtered, entry)
	}
	return filtered, changed
}

// HookCommandWired reports whether command is wired anywhere in the
// hooks key of the settings file at settingsPath — in any event's
// matcher-group array, not just one named event (ADR-0032's drift
// check: a hook's materialized-vs-wired state doesn't depend on which
// event it's registered under). A missing settings file reports false
// with no error, mirroring WireHookSettings's own not-yet-materialized
// case.
func HookCommandWired(settingsPath, command string) (bool, error) {
	existing, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", settingsPath, readErr)
	}

	obj, parseErr := parseSettingsJSON(existing)
	if parseErr != nil {
		return false, fmt.Errorf("parsing %s: %w", settingsPath, parseErr)
	}

	hooks, hooksErr := parseHooksKey(obj)
	if hooksErr != nil {
		return false, fmt.Errorf("parsing %s hooks key: %w", settingsPath, hooksErr)
	}

	for _, entries := range hooks {
		if hookCommandPresent(entries, command) {
			return true, nil
		}
	}
	return false, nil
}
