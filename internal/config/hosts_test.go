package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/testsupport"
)

// Serial: PATH is process-wide; parallel configuration tests run after this test.
func TestResolveHosts_UsesExecutablePATHOrExplicitSelection(t *testing.T) {
	for _, tc := range []struct {
		name, yaml string
		commands   map[string]bool
		want       HostSelection
	}{
		{"neither", "", nil, HostSelection{Hosts: []Host{}, Source: HostsDetected}},
		{"claude", "", map[string]bool{"claude": true}, HostSelection{Hosts: []Host{HostClaudeCode}, Source: HostsDetected}},
		{"codex", "", map[string]bool{"codex": true}, HostSelection{Hosts: []Host{HostCodex}, Source: HostsDetected}},
		{"both", "", map[string]bool{"claude": true, "codex": true}, HostSelection{Hosts: []Host{HostClaudeCode, HostCodex}, Source: HostsDetected}},
		{"nonexecutables", "", map[string]bool{"claude": false, "codex": false}, HostSelection{Hosts: []Host{}, Source: HostsDetected}},
		{"mixed permissions", "", map[string]bool{"claude": false, "codex": true}, HostSelection{Hosts: []Host{HostCodex}, Source: HostsDetected}},
		{"explicit absent tools", "hosts: [codex, claude-code, codex, claude-code]\n", nil, HostSelection{Hosts: []Host{HostClaudeCode, HostCodex}, Source: HostsConfigured}},
		{"explicit claude", "hosts: [claude-code]\n", map[string]bool{"codex": true}, HostSelection{Hosts: []Host{HostClaudeCode}, Source: HostsConfigured}},
		{"explicit codex", "hosts: [codex]\n", map[string]bool{"claude": true}, HostSelection{Hosts: []Host{HostCodex}, Source: HostsConfigured}},
		{"explicit none", "hosts: []\n", map[string]bool{"claude": true, "codex": true}, HostSelection{Hosts: []Host{}, Source: HostsConfigured}},
		{"null is unset", "hosts: null\n", map[string]bool{"codex": true}, HostSelection{Hosts: []Host{HostCodex}, Source: HostsDetected}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			marker := filepath.Join(t.TempDir(), "launched")
			for name, executable := range tc.commands {
				path := filepath.Join(bin, name)
				if executable {
					if err := testsupport.WriteExecutable(path, fmt.Appendf(nil, "#!/bin/sh\n: > %q\nexit 99\n", marker)); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(path, []byte("not executable\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin)
			var cfg Config
			if err := yaml.Unmarshal([]byte(tc.yaml), &cfg); err != nil {
				t.Fatal(err)
			}
			before, err := yaml.Marshal(&cfg)
			if err != nil {
				t.Fatal(err)
			}
			configs := []*Config{&cfg}
			if tc.yaml == "" {
				configs = append(configs, nil)
			}
			for _, candidate := range configs {
				got, resolveErr := candidate.ResolveHosts(context.Background())
				if resolveErr != nil {
					t.Fatal(resolveErr)
				}
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Fatalf("selection (-want +got):\n%s", diff)
				}
			}
			after, err := yaml.Marshal(&cfg)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("resolution changed configuration: %s -> %s (%v)", before, after, err)
			}
			if _, err := os.Stat(marker); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("host command was launched: %v", err)
			}
		})
	}
}

func TestHosts_RoundTripPreservesAbsentEmptyAndExplicit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		want  *[]string
	}{
		{"", nil},
		{"hosts: []\n", &[]string{}},
		{"hosts: [codex]\n", &[]string{"codex"}},
		{"hosts: [codex, claude-code, codex]\n", &[]string{"codex", "claude-code", "codex"}},
	} {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(tc.input), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			other := t.TempDir()
			if writeErr := Write(other, cfg); writeErr != nil {
				t.Fatal(writeErr)
			}
			again, err := Load(other)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, again.Hosts); diff != "" {
				t.Fatalf("hosts after save/load (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHosts_ConfigEditsPreserveSelection(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "hosts: []\n", "hosts: [codex, claude-code, codex]\n"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, FileName)
			if err := os.WriteFile(path, []byte(input+"actor: human/legacy\naiwf_version: old\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			before, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, strip := range []func(string) (bool, error){StripLegacyActor, StripLegacyAiwfVersion} {
				if changed, stripErr := strip(root); stripErr != nil || !changed {
					t.Fatalf("legacy edit = %v, %v", changed, stripErr)
				}
			}
			doc, _, err := aiwfyaml.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			doc.SetHooks(map[string]bool{"example.sh": false})
			if writeErr := doc.Write(path); writeErr != nil {
				t.Fatal(writeErr)
			}
			after, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(before.Hosts, after.Hosts); diff != "" {
				t.Fatalf("host selection changed after config edits (-want +got):\n%s", diff)
			}
			if enabled, decided := after.HookDecision("example.sh"); enabled || !decided {
				t.Fatalf("hook edit missing: %v, %v", enabled, decided)
			}
			if after.LegacyActor != "" || after.LegacyAiwfVersion != "" {
				t.Fatal("legacy fields were not removed")
			}
		})
	}
}

func TestHosts_RejectUnknownValuesBeforeResolutionOrWrite(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"other", "claude", "Codex", "", " codex"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			cfg := Config{Hosts: &[]string{"claude-code", value}}
			if err := cfg.Validate(); !errors.Is(err, ErrInvalidHost) {
				t.Fatalf("Validate error = %v", err)
			}
			got, err := cfg.ResolveHosts(context.Background())
			if !errors.Is(err, ErrInvalidHost) || !cmp.Equal(HostSelection{}, got) {
				t.Fatalf("partial selection = %+v, error = %v", got, err)
			}
			root := t.TempDir()
			if err := Write(root, &cfg); !errors.Is(err, ErrInvalidHost) {
				t.Fatalf("Write error = %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, FileName)); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("invalid config written: %v", err)
			}
		})
	}
}

func TestHosts_LoadRejectsInvalidValuesAndShapes(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"hosts: [other]\n", "hosts: codex\n", "hosts: {codex: true}\n", "hosts: [1]\n", "hosts: [[codex]]\n", "hosts: [codex\n"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(input), 0o644); err != nil {
				t.Fatal(err)
			}
			if got, err := Load(root); err == nil || got != nil {
				t.Fatalf("invalid config accepted: %+v, %v", got, err)
			}
		})
	}
}

func TestResolveHosts_CanceledContextReturnsNoSelection(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var cfg *Config
	got, err := cfg.ResolveHosts(ctx)
	if !errors.Is(err, context.Canceled) || !cmp.Equal(HostSelection{}, got) {
		t.Fatalf("canceled selection = %+v, %v", got, err)
	}
}
