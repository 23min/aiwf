package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProjectGuidanceConfiguration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input string
		enabled     bool
		source      string
		packs       *[]string
		ignored     []string
	}{
		{name: "absent", enabled: true},
		{name: "null source", input: "guidance: {source: null}", enabled: true},
		{name: "aliases", input: "repo: &repo /tmp/local\npack: &pack go/cobra\npacks: &packs [*pack]\nguidance: {source: *repo, packs: *packs}", enabled: true, source: "/tmp/local", packs: &[]string{"go/cobra"}},
		{name: "merged fields", input: "defaults: &defaults {source: /tmp/merged, packs: [code-health]}\nguidance: {<<: *defaults}", enabled: true, source: "/tmp/merged", packs: &[]string{"code-health"}},
		{name: "null selection", input: "guidance: {packs: null}", enabled: true},
		{name: "empty adoption", input: "guidance: {packs: []}", enabled: true, packs: &[]string{}},
		{name: "selected and ignored", input: "guidance: {packs: [go/cobra], ignored: [python/astral]}", enabled: true, packs: &[]string{"go/cobra"}, ignored: []string{"python/astral"}},
		{name: "disabled retains selection", input: "guidance: {enabled: false, packs: [code-health]}", packs: &[]string{"code-health"}},
		{name: "custom source", input: "guidance: {enabled: true, source: /tmp/custom guidance}", enabled: true, source: "/tmp/custom guidance"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(tc.input), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			source := tc.source
			if source == "" {
				source = "https://github.com/23min/engineering-guidance.git"
			}
			assert := func(c *Config) {
				t.Helper()
				if c.GuidanceEnabled() != tc.enabled || c.GuidanceSource() != source {
					t.Fatalf("effective guidance = %v, %q", c.GuidanceEnabled(), c.GuidanceSource())
				}
				if diff := cmp.Diff(tc.packs, c.Guidance.Packs); diff != "" {
					t.Fatal(diff)
				}
				if diff := cmp.Diff(tc.ignored, c.Guidance.Ignored); diff != "" {
					t.Fatal(diff)
				}
			}
			assert(cfg)
			other := t.TempDir()
			if writeErr := Write(other, cfg); writeErr != nil {
				t.Fatal(writeErr)
			}
			again, err := Load(other)
			if err != nil {
				t.Fatal(err)
			}
			assert(again)
		})
	}
	var cfg *Config
	if !cfg.GuidanceEnabled() || cfg.GuidanceSource() != "https://github.com/23min/engineering-guidance.git" {
		t.Fatal("nil configuration must use effective defaults")
	}
}

func TestProjectGuidanceRejectsInvalidSelection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, input, key string }{
		{"overlap", "packs: [go/cobra], ignored: [go/cobra]", "ignored"},
		{"duplicate selection", "packs: [code-health, code-health]", "packs"},
		{"duplicate ignore", "ignored: [go/cobra, go/cobra]", "ignored"},
		{"unsafe id", "packs: [../go]", "packs"},
		{"invalid ignored id", "ignored: [Go]", "ignored"},
		{"empty id", "packs: ['']", "packs"},
		{"blank source", "source: '  '", "source"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte("guidance: {"+tc.input+"}\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(root)
			if !errors.Is(err, ErrInvalidGuidance) || !strings.Contains(err.Error(), "guidance."+tc.key) {
				t.Fatalf("expected actionable %s rejection, got %v", tc.key, err)
			}
		})
	}
}

func TestProjectGuidanceRejectsWrongTypes(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"enabled: []", "packs: go/cobra", "ignored: {}", "source: []", "source: 123", "packs: [123]", "ignored: [true]", "<<: {packs: [123]}", "<<: {source: 123}"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte("guidance: {"+input+"}\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root); err == nil {
				t.Fatal("accepted invalid guidance field type")
			}
		})
	}
}

func TestProjectGuidanceRejectsNonMapping(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("guidance: [go/cobra]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("accepted non-mapping guidance")
	}
}
