package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/pathutil"
)

// SettingsWriteResult reports what WireStatuslineSettings did.
type SettingsWriteResult struct {
	// Path is the absolute path to the settings file that was written.
	Path string

	// BackupPath is the absolute path to the .bak file written before
	// editing. Empty when no edit was needed (idempotent / no-clobber).
	BackupPath string

	// Wrote is true when the statusLine key was inserted and the file
	// was written. False on no-clobber (key exists with different value)
	// or idempotent no-op (key already matches).
	Wrote bool

	// Idempotent is true when the statusLine key already pointed at
	// the same command path — a re-run that required no changes.
	Idempotent bool

	// ExistingValue is non-empty when a pre-existing statusLine key
	// blocked the write (no-clobber). The caller uses this for merge
	// guidance.
	ExistingValue string
}

// statusLineValue is the JSON shape Claude Code expects for the
// statusLine settings key.
type statusLineValue struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// WireStatuslineSettings inserts the statusLine key into the
// scope-appropriate settings file. The caller is responsible for
// consent gating (TTY prompt or --wire-settings) — this function
// does the mechanical JSON edit unconditionally.
//
// Behavior:
//   - If the file does not exist, creates it with just the statusLine key.
//   - If the file exists but has no statusLine key, inserts it and writes
//     a .bak of the original.
//   - If the file exists and statusLine already points at cmdPath,
//     returns Idempotent=true without writing.
//   - If the file exists and statusLine points at something else,
//     returns Wrote=false with ExistingValue set (no-clobber).
func WireStatuslineSettings(settingsPath, cmdPath string) (SettingsWriteResult, error) {
	res := SettingsWriteResult{Path: settingsPath}

	existing, readErr := os.ReadFile(settingsPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return res, fmt.Errorf("reading %s: %w", settingsPath, readErr)
	}

	obj, parseErr := parseSettingsJSON(existing)
	if parseErr != nil {
		return res, fmt.Errorf("parsing %s: %w", settingsPath, parseErr)
	}

	if raw, ok := obj["statusLine"]; ok {
		return handleExistingKey(res, raw, cmdPath)
	}

	if len(existing) > 0 {
		bakPath := settingsPath + ".bak"
		if wErr := pathutil.AtomicWriteFile(bakPath, existing, 0o644); wErr != nil {
			return res, fmt.Errorf("writing backup %s: %w", bakPath, wErr)
		}
		res.BackupPath = bakPath
	}

	val := statusLineValue{Type: "command", Command: cmdPath}
	valBytes, mErr := json.Marshal(val)
	if mErr != nil {
		return res, fmt.Errorf("marshaling statusLine value: %w", mErr)
	}
	obj["statusLine"] = valBytes

	out, mErr := json.MarshalIndent(obj, "", "  ")
	if mErr != nil {
		return res, fmt.Errorf("marshaling settings: %w", mErr)
	}
	out = append(out, '\n')

	if mkErr := os.MkdirAll(filepath.Dir(settingsPath), 0o755); mkErr != nil {
		return res, fmt.Errorf("creating directory for %s: %w", settingsPath, mkErr)
	}
	if wErr := pathutil.AtomicWriteFile(settingsPath, out, 0o644); wErr != nil {
		return res, fmt.Errorf("writing %s: %w", settingsPath, wErr)
	}
	res.Wrote = true
	return res, nil
}

// parseSettingsJSON parses existing settings content or returns an
// empty map for a missing/empty file.
func parseSettingsJSON(data []byte) (map[string]json.RawMessage, error) {
	if len(data) == 0 {
		return make(map[string]json.RawMessage), nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// handleExistingKey decides between idempotent no-op (same command)
// and no-clobber (different command) when a statusLine key already exists.
func handleExistingKey(res SettingsWriteResult, raw json.RawMessage, cmdPath string) (SettingsWriteResult, error) {
	var cur statusLineValue
	if uErr := json.Unmarshal(raw, &cur); uErr == nil && cur.Command == cmdPath {
		res.Idempotent = true
		return res, nil
	}
	var pretty any
	_ = json.Unmarshal(raw, &pretty)
	b, _ := json.Marshal(pretty)
	res.ExistingValue = string(b)
	return res, nil
}

// StatuslineSettingsKeyStatus is the read-only inspection of a scope's
// statusLine settings key — G-0354's precondition check for
// `aiwf update --remove`. It never mutates anything, so a caller can
// inspect both the script and the settings key *before* deciding
// whether either mutation is authorized (see RemoveStatuslineSettingsKey).
//
//   - existed reports whether the settings file contained a statusLine
//     key at all.
//   - matches reports whether that key's command equals cmdPath — i.e.
//     it looks like aiwf's own wiring for this scope.
//   - existingValue is the pretty-printed key value when it existed but
//     did not match, for the caller's refusal message; empty otherwise.
func StatuslineSettingsKeyStatus(settingsPath, cmdPath string) (existed, matches bool, existingValue string, err error) {
	existing, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return false, false, "", nil
		}
		return false, false, "", fmt.Errorf("reading %s: %w", settingsPath, readErr)
	}

	obj, parseErr := parseSettingsJSON(existing)
	if parseErr != nil {
		return false, false, "", fmt.Errorf("parsing %s: %w", settingsPath, parseErr)
	}

	raw, ok := obj["statusLine"]
	if !ok {
		return false, false, "", nil
	}

	var cur statusLineValue
	if uErr := json.Unmarshal(raw, &cur); uErr != nil || cur.Command != cmdPath {
		var pretty any
		_ = json.Unmarshal(raw, &pretty)
		b, _ := json.Marshal(pretty)
		return true, false, string(b), nil
	}
	return true, true, "", nil
}

// RemoveStatuslineSettingsKey strips the statusLine key from
// settingsPath unconditionally — the caller (RunStatuslineRemove) must
// have already authorized this via StatuslineSettingsKeyStatus (a
// match, or an operator --force) before calling. No-op (removed=false)
// when the file doesn't exist or carries no statusLine key, so it's
// safe to call even when the inspection already reported nothing to do.
func RemoveStatuslineSettingsKey(settingsPath string) (removed bool, err error) {
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

	if _, ok := obj["statusLine"]; !ok {
		return false, nil
	}

	delete(obj, "statusLine")
	out, mErr := json.MarshalIndent(obj, "", "  ")
	if mErr != nil { //coverage:ignore unreachable: obj's remaining values are json.RawMessage already proven valid by the parseSettingsJSON unmarshal above, so re-marshaling never fails
		return false, fmt.Errorf("marshaling settings: %w", mErr)
	}
	out = append(out, '\n')
	if wErr := pathutil.AtomicWriteFile(settingsPath, out, 0o644); wErr != nil {
		return false, fmt.Errorf("writing %s: %w", settingsPath, wErr) //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce
	}
	return true, nil
}

// SettingsPathForScope returns the absolute path to the settings file
// the consent-gated wiring should target, based on scope.
//
// Project scope targets `.claude/settings.local.json` (personal,
// gitignored) — not the shared `.claude/settings.json`.
// User scope targets `~/.claude/settings.json`.
func SettingsPathForScope(root, home string, scope StatuslineScope) (string, error) {
	switch scope {
	case StatuslineScopeProject:
		return filepath.Join(root, ".claude", "settings.local.json"), nil
	case StatuslineScopeUser:
		return filepath.Join(home, ".claude", "settings.json"), nil
	default:
		return "", fmt.Errorf("unknown --scope %q", scope)
	}
}
