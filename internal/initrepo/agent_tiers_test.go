package initrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAiwfYAML(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing aiwf.yaml: %v", err)
	}
}

// TestAgentTiersFromConfig walks every branch of the aiwf.yaml → tier
// translation (G-0353): missing file, malformed file, absent block, a mix of
// known and unknown keys, and an all-unknown block.
func TestAgentTiersFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("missing aiwf.yaml yields no tiers", func(t *testing.T) {
		t.Parallel()
		tiers, unknown, err := agentTiersFromConfig(t.TempDir())
		if err != nil || tiers != nil || unknown != nil {
			t.Fatalf("got (tiers=%v, unknown=%v, err=%v), want all nil", tiers, unknown, err)
		}
	})

	t.Run("malformed aiwf.yaml surfaces the load error", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAiwfYAML(t, root, "agents: [not-a-map\n")
		if _, _, err := agentTiersFromConfig(root); err == nil {
			t.Fatal("want a load error for malformed yaml")
		}
	})

	t.Run("no agents block yields no tiers", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAiwfYAML(t, root, "hosts: [claude-code]\n")
		tiers, unknown, err := agentTiersFromConfig(root)
		if err != nil || tiers != nil || unknown != nil {
			t.Fatalf("got (tiers=%v, unknown=%v, err=%v), want all nil", tiers, unknown, err)
		}
	})

	t.Run("known key tiered, unknown key reported", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAiwfYAML(t, root, "agents:\n  reviewer:\n    model: sonnet\n    effort: high\n  reviwer:\n    model: haiku\n")
		tiers, unknown, err := agentTiersFromConfig(root)
		if err != nil {
			t.Fatalf("agentTiersFromConfig: %v", err)
		}
		if got := tiers["reviewer"]; got.Model != "sonnet" || got.Effort != "high" {
			t.Errorf("reviewer tier = %+v, want {sonnet high}", got)
		}
		if _, ok := tiers["reviwer"]; ok {
			t.Error("typo key should not produce a tier")
		}
		if len(unknown) != 1 || unknown[0] != "reviwer" {
			t.Errorf("unknown = %v, want [reviwer]", unknown)
		}
	})

	t.Run("all-unknown block yields nil tiers and reports the keys", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAiwfYAML(t, root, "agents:\n  ghost:\n    model: opus\n")
		tiers, unknown, err := agentTiersFromConfig(root)
		if err != nil {
			t.Fatalf("agentTiersFromConfig: %v", err)
		}
		if tiers != nil {
			t.Errorf("tiers = %v, want nil when every key is unknown", tiers)
		}
		if len(unknown) != 1 || unknown[0] != "ghost" {
			t.Errorf("unknown = %v, want [ghost]", unknown)
		}
	})
}

// TestEnsureSkillsAgentTiers covers the two branches ensureSkills gained for
// the agents block (G-0353): the config-load error propagating out, and the
// unknown-key report reaching the operator-facing step ledger.
func TestEnsureSkillsAgentTiers(t *testing.T) {
	t.Parallel()

	t.Run("malformed aiwf.yaml fails ensureSkills", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAiwfYAML(t, root, "agents: [not-a-map\n")
		if _, err := ensureSkills(root, false); err == nil {
			t.Fatal("ensureSkills = nil error, want the malformed-config error to propagate")
		}
	})

	t.Run("unknown agent key is surfaced in the step detail", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAiwfYAML(t, root, "agents:\n  reviewer:\n    model: sonnet\n  ghost:\n    model: opus\n")
		step, err := ensureSkills(root, false)
		if err != nil {
			t.Fatalf("ensureSkills: %v", err)
		}
		const want = "ignored aiwf.yaml agents keys with no shipped agent: ghost"
		if !strings.Contains(step.Detail, want) {
			t.Errorf("step.Detail = %q, want it to contain %q", step.Detail, want)
		}
	})

	t.Run("materialize failure propagates from ensureSkills", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		// A regular file where the .claude dir must go makes the materializer's
		// MkdirAll fail, so MaterializeWithTiers errors and ensureSkills wraps
		// it. No aiwf.yaml here, so agentTiersFromConfig returns nil tiers and
		// the failure comes from materialization itself.
		if err := os.WriteFile(filepath.Join(root, ".claude"), []byte("x"), 0o644); err != nil {
			t.Fatalf("seeding .claude file: %v", err)
		}
		if _, err := ensureSkills(root, false); err == nil {
			t.Fatal("ensureSkills = nil error, want the materialize failure to propagate")
		}
	})
}
