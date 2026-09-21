package initrepo

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/config"
)

func TestInit_ProjectGuidanceDoesNotAdoptPolicyByDefault(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{SkipHook: true}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.GuidanceEnabled() || cfg.Guidance.Packs != nil {
		t.Fatalf("default guidance configuration: %+v", cfg.Guidance)
	}
	if _, err := os.Stat(filepath.Join(root, ".guidance", "index.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("init adopted policy without selection: %v", err)
	}
}

func TestInit_ProjectGuidancePreservesExistingConfiguration(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	original := []byte("# Keep this comment.\nhosts: []\ncustom_setting: retained\nguidance: {enabled: false, packs: [], ignored: [go/cobra], wire_claudemd: false, wire_agentsmd: false}\n")
	path := filepath.Join(root, config.FileName)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{SkipHook: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("init changed existing configuration: %s (%v)", got, err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Guidance.Packs == nil || len(*cfg.Guidance.Packs) != 0 || cfg.GuidanceEnabled() {
		t.Fatalf("explicit disabled empty selection lost: %+v", cfg.Guidance)
	}
}
