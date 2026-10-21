package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFenceInstructionCommits_DefaultsOn is M-0350 AC-4's setting: the
// instruction-file fence runs unless aiwf.yaml turns it off.
func TestFenceInstructionCommits_DefaultsOn(t *testing.T) {
	t.Parallel()
	var absent *Config
	if !absent.FenceInstructionCommits() {
		t.Fatal("nil config must default on")
	}
	for _, tc := range []struct {
		name, yaml string
		want       bool
	}{
		{"absent", "", true},
		{"empty guidance", "guidance: {}\n", true},
		{"explicit on", "guidance:\n  fence_instruction_commits: true\n", true},
		{"explicit off", "guidance:\n  fence_instruction_commits: false\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.FenceInstructionCommits(); got != tc.want {
				t.Errorf("FenceInstructionCommits = %v, want %v", got, tc.want)
			}
		})
	}
}
