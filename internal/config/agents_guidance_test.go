package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWireAgentsMd_DefaultAndPersistentOverride(t *testing.T) {
	t.Parallel()
	var absent *Config
	if !absent.WireAgentsMd() {
		t.Fatal("nil config must default on")
	}
	for _, tc := range []struct {
		name, yaml string
		want       bool
	}{
		{"absent", "", true},
		{"empty guidance", "guidance: {}\n", true},
		{"explicit on", "guidance:\n  wire_agentsmd: true\n", true},
		{"explicit off", "guidance:\n  wire_agentsmd: false\n", false},
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
			if cfg.WireAgentsMd() != tc.want {
				t.Fatalf("loaded value = %v", cfg.WireAgentsMd())
			}
			other := t.TempDir()
			if writeErr := Write(other, cfg); writeErr != nil {
				t.Fatal(writeErr)
			}
			again, err := Load(other)
			if err != nil {
				t.Fatal(err)
			}
			if again.WireAgentsMd() != tc.want {
				t.Fatalf("roundtrip value = %v", again.WireAgentsMd())
			}
		})
	}
}

func TestWireAgentsMd_RejectsInvalidValue(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("guidance:\n  wire_agentsmd: [false]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("accepted non-boolean opt-out")
	}
}
