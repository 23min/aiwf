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
	"github.com/23min/aiwf/internal/testsupport"
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

func TestInit_InstallsExplicitProjectGuidance(t *testing.T) {
	t.Parallel()
	root, source := freshGitRepo(t), testsupport.GuidanceSource(t)
	cfg := "hosts: [claude-code, codex]\nguidance:\n  source: " + source + "\n  packs: [sample/base]\n"
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(t.Context(), root, Options{SkipHook: true}); err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(filepath.Join(root, ".guidance", "packs", "sample", "base", "guide.md"))
	if err != nil || string(installed) != "# Sample guidance\nInitial upstream content.\n" {
		t.Fatalf("installed = %s, %v", installed, err)
	}
}

func TestProjectGuidanceRefresh_ReportsFailuresWithoutClaimingInstallation(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"cancelled inspection", "claude conflict", "codex conflict", "unavailable source", "installation conflict"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			selected := []string{"sample/base"}
			cfg := &config.Config{Guidance: config.Guidance{Source: testsupport.GuidanceSource(t), Packs: &selected}}
			hosts := config.HostSelection{Hosts: []config.Host{config.HostClaudeCode, config.HostCodex}}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch kind {
			case "cancelled inspection":
				cancel()
			case "claude conflict":
				if err := os.Mkdir(filepath.Join(root, "CLAUDE.md"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "codex conflict":
				if err := os.Mkdir(filepath.Join(root, "AGENTS.md"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "unavailable source":
				cfg.Guidance.Source = filepath.Join(root, "absent")
			case "installation conflict":
				if err := os.Mkdir(filepath.Join(root, ".guidance"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, ".guidance", "index.md"), []byte("foreign\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			step := ensureProjectGuidance(ctx, root, cfg, hosts, RefreshOptions{WireClaudeMd: true})
			if step == nil || step.Action != ActionSkipped || step.Detail == "" {
				t.Fatalf("failed guidance claimed success: %+v", step)
			}
			if _, err := os.Stat(filepath.Join(root, ".guidance", "packs")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("failed refresh installed packs: %v", err)
			}
		})
	}
}

func TestProjectGuidanceRefresh_OptOutAndDryRun(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"missing config", "unselected", "disabled", "dry run", "host wiring disabled"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			selected := []string{"sample/base"}
			disabled := false
			cfg := &config.Config{Guidance: config.Guidance{Source: testsupport.GuidanceSource(t), Packs: &selected}}
			hosts := config.HostSelection{Hosts: []config.Host{config.HostClaudeCode, config.HostCodex}}
			opts := RefreshOptions{WireClaudeMd: true}
			switch kind {
			case "missing config":
				cfg = nil
			case "unselected":
				cfg.Guidance.Packs = nil
			case "disabled":
				cfg.Guidance.Enabled = &disabled
			case "dry run":
				opts.DryRun = true
			case "host wiring disabled":
				opts.WireClaudeMd = false
				cfg.Guidance.WireAgentsMd = &disabled
			}
			step := ensureProjectGuidance(t.Context(), root, cfg, hosts, opts)
			if kind == "host wiring disabled" {
				if step == nil || step.Action != ActionUpdated {
					t.Fatalf("selected files not installed: %+v", step)
				}
				repeated := ensureProjectGuidance(t.Context(), root, cfg, hosts, opts)
				if repeated == nil || repeated.Action != ActionPreserved {
					t.Fatalf("unchanged guidance not preserved: %+v", repeated)
				}
			} else {
				entries, err := os.ReadDir(root)
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 0 {
					t.Fatalf("opt-out/dry run wrote files: %v", entries)
				}
			}
			for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
				if _, err := os.Stat(filepath.Join(root, name)); !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("unexpected %s: %v", name, err)
				}
			}
		})
	}
}
