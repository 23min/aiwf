package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestTrace_EmitsPhaseApplyTiming pins M-0239/AC-3: --trace forces a
// debug-level phase.apply event through the bound logger, without
// needing AIWF_LOG set separately (the whole point of the flag is
// "just show me this one run's timing").
func TestTrace_EmitsPhaseApplyTiming(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Stale probe", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)
	// Deliberately NOT setting AIWF_LOG — --trace must enable logging
	// on its own for this one invocation.

	if rc := cli.Execute([]string{"promote", "G-0001", "wontfix", "--actor", "human/test", "--root", root, "--trace"}); rc != cliutil.ExitOK {
		t.Fatalf("promote --trace: rc=%d", rc)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	var found bool
	for {
		var rec struct {
			Msg       string  `json:"msg"`
			Level     string  `json:"level"`
			ElapsedMs float64 `json:"elapsed_ms"`
		}
		if err := dec.Decode(&rec); err != nil {
			break
		}
		if rec.Msg == "phase.apply" {
			found = true
			if rec.Level != "DEBUG" {
				t.Errorf("phase.apply level = %q, want DEBUG", rec.Level)
			}
			if rec.ElapsedMs < 0 {
				t.Errorf("phase.apply elapsed_ms = %v, want >= 0", rec.ElapsedMs)
			}
		}
	}
	if !found {
		t.Fatalf("no phase.apply event found in log:\n%s", raw)
	}
}

// TestTrace_NoOpWithoutFlag confirms --trace's absence changes
// nothing: no phase.apply event, no log file at all (default-off,
// matching every other diagnostic-logging test in this package).
func TestTrace_NoOpWithoutFlag(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Stale probe", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG_FILE", logPath)

	mustRun(t, "promote", "G-0001", "wontfix", "--actor", "human/test", "--root", root)

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("diagnostic log file exists at %s despite no --trace and no AIWF_LOG (err=%v)", logPath, err)
	}
}
