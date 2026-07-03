package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestRun_DoctorWriteHealth drives the `aiwf doctor --write-health` CLI seam
// (the RunE branch + runWriteHealth) and asserts a schema-valid
// .claude/health.aiwf.json lands.
func TestRun_DoctorWriteHealth(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"doctor", "--write-health", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("doctor --write-health = %d, want %d", rc, cliutil.ExitOK)
	}
	assertHealthFile(t, filepath.Join(root, ".claude", "health.aiwf.json"))
}

// TestRun_UpdateWritesHealth drives the `aiwf update` seam that refreshes the
// health file as its final step.
func TestRun_UpdateWritesHealth(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	health := filepath.Join(root, ".claude", "health.aiwf.json")
	_ = os.Remove(health) // ensure `update` is what writes it
	if rc := cli.Execute([]string{"update", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("update = %d, want %d", rc, cliutil.ExitOK)
	}
	assertHealthFile(t, health)
}

// TestRun_UpdateStatuslineRefreshesHealthDespiteSettingsConflict pins G-0347:
// `aiwf update --statusline` must still refresh .claude/health.aiwf.json when the
// settings-wiring step reports a finding (a pre-existing, differing statusLine
// key). The statusline script + version marker are written before the wiring
// conflict, so the health stoplight must reflect the just-marked statusline —
// not a stale pre-mark warning. Before the fix an early `return rc` on the
// findings code skipped the WriteHealth step. Project scope keeps the settings
// file under the test root (no $HOME juggling), so the test stays parallel.
func TestRun_UpdateStatuslineRefreshesHealthDespiteSettingsConflict(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}

	// Seed the project-scope settings file (settings.local.json) with a DIFFERENT
	// statusLine command so the wiring step returns a findings code (not
	// idempotent, not overwritten).
	settings := filepath.Join(root, ".claude", "settings.local.json")
	if err := os.WriteFile(settings, []byte(`{"statusLine":{"type":"command","command":"/foreign/statusline.sh"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	health := filepath.Join(root, ".claude", "health.aiwf.json")
	_ = os.Remove(health) // ensure the update run is what (re)writes it

	rc := cli.Execute([]string{"update", "--statusline", "--scope", "project", "--wire-settings", "--root", root})
	if rc != cliutil.ExitFindings {
		t.Fatalf("update --statusline with a conflicting statusLine key = %d, want ExitFindings %d (the conflict must still be reported)", rc, cliutil.ExitFindings)
	}

	// G-0347: the early return on the findings rc used to skip WriteHealth,
	// leaving the stoplight reading a stale pre-mark warning. Health must be
	// refreshed despite the wiring finding.
	assertHealthFile(t, health)
}

// TestRun_DoctorWriteHealth_NonGitErrors: on a non-git dir the main checkout
// can't be resolved, so --write-health surfaces an internal error — covering
// runWriteHealth's WriteHealth-failure branch.
func TestRun_DoctorWriteHealth_NonGitErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir() // no git repo → MainCheckoutRoot fails
	if rc := cli.Execute([]string{"doctor", "--write-health", "--root", root}); rc != cliutil.ExitInternal {
		t.Fatalf("doctor --write-health on non-git dir = %d, want %d (internal)", rc, cliutil.ExitInternal)
	}
}

// assertHealthFile confirms the file exists and carries the fixed ai-dotfiles
// schema (a generated_at stamp and an aiwf source on every finding).
func assertHealthFile(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("health.aiwf.json not written: %v", err)
	}
	var hf struct {
		GeneratedAt string `json:"generated_at"`
		Findings    []struct {
			Source   string `json:"source"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(raw, &hf); err != nil {
		t.Fatalf("health.aiwf.json is not valid JSON: %v\n%s", err, raw)
	}
	if hf.GeneratedAt == "" {
		t.Errorf("health.aiwf.json has an empty generated_at")
	}
	for _, f := range hf.Findings {
		if f.Source != "aiwf" {
			t.Errorf("finding source = %q, want aiwf", f.Source)
		}
	}
}
