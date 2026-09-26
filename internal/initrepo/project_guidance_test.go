package initrepo

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/testsupport"
)

func TestInit_WithoutAdoptGuidanceDoesNotAdoptPolicy(t *testing.T) {
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
	testsupport.IsolateGuidanceEnvironment(t)
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
	testsupport.IsolateGuidanceEnvironment(t)
	for _, kind := range []string{"cancelled inspection", "claude conflict", "codex conflict", "unavailable source", "installation conflict"} {
		t.Run(kind, func(t *testing.T) {
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
			step, _ := ensureProjectGuidance(ctx, root, cfg, hosts, RefreshOptions{WireClaudeMd: true})
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
	testsupport.IsolateGuidanceEnvironment(t)
	for _, kind := range []string{"missing config", "unselected", "disabled", "dry run", "dry run adopting", "host wiring disabled"} {
		t.Run(kind, func(t *testing.T) {
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
			case "dry run adopting":
				opts.DryRun, opts.AdoptGuidance = true, true
				cfg.Guidance.Packs = nil
			case "host wiring disabled":
				opts.WireClaudeMd = false
				cfg.Guidance.WireAgentsMd = &disabled
			}
			step, _ := ensureProjectGuidance(t.Context(), root, cfg, hosts, opts)
			if strings.HasPrefix(kind, "dry run") && (step == nil || strings.Contains(step.Detail, "adopt") != (kind == "dry run adopting")) {
				t.Fatalf("dry run preview does not match whether it adopts: %+v", step)
			}
			if kind == "host wiring disabled" {
				if step == nil || step.Action != ActionUpdated {
					t.Fatalf("selected files not installed: %+v", step)
				}
				repeated, _ := ensureProjectGuidance(t.Context(), root, cfg, hosts, opts)
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

func TestProjectGuidanceRefresh_RoutesOnlySelectedWiredHosts(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	for _, tc := range []struct {
		name          string
		hosts         []config.Host
		claude, codex bool
	}{
		{"claude only", []config.Host{config.HostClaudeCode}, true, true},
		{"codex only", []config.Host{config.HostCodex}, true, true},
		{"neither", nil, true, true},
		{"claude opted out", []config.Host{config.HostClaudeCode, config.HostCodex}, false, true},
		{"codex opted out", []config.Host{config.HostClaudeCode, config.HostCodex}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			packs := []string{"sample/base"}
			cfg := &config.Config{Guidance: config.Guidance{Source: testsupport.GuidanceSource(t), Packs: &packs, WireAgentsMd: &tc.codex}}
			step, _ := ensureProjectGuidance(t.Context(), root, cfg, config.HostSelection{Hosts: tc.hosts}, RefreshOptions{WireClaudeMd: tc.claude})
			if step == nil || step.Action != ActionUpdated {
				t.Fatalf("install: %+v", step)
			}
			for _, host := range []struct {
				host config.Host
				file string
				wire bool
			}{{config.HostClaudeCode, "CLAUDE.md", tc.claude}, {config.HostCodex, "AGENTS.md", tc.codex}} {
				content, err := os.ReadFile(filepath.Join(root, host.file))
				want := host.wire && slices.Contains(tc.hosts, host.host)
				if !want {
					if !errors.Is(err, fs.ErrNotExist) {
						t.Fatalf("unselected %s written: %v", host.file, err)
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				for _, path := range []string{".guidance/project.md", ".guidance/index.md"} {
					if !strings.Contains(string(content), "]("+path+")") {
						t.Errorf("%s does not link %s", host.file, path)
					}
				}
				if strings.Contains(string(content), "Initial upstream content.") {
					t.Fatal("corpus concatenated into routing")
				}
			}
		})
	}
}

func TestProjectGuidanceAdoption_RecordsSelectionOnlyWithInstallation(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	for _, kind := range []string{"adopts", "no applicable pack", "detection failure", "unreadable configuration", "unrecordable configuration", "installation conflict", "unwritable configuration"} {
		t.Run(kind, func(t *testing.T) {
			if kind == "unwritable configuration" && os.Geteuid() == 0 {
				t.Skip("root bypasses permission checks")
			}
			source := testsupport.GuidanceSource(t, "*.xyz")
			root := freshGitRepo(t)
			configuration := "hosts: []\nguidance:\n  source: " + source + "\n"
			switch kind {
			case "detection failure":
				root = t.TempDir()
			case "unrecordable configuration":
				configuration = "hosts: []\nshared: &guidance\n  source: " + source + "\nguidance: *guidance\n"
			}
			path := filepath.Join(root, config.FileName)
			if kind != "unreadable configuration" {
				if err := os.WriteFile(path, []byte(configuration), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "no applicable pack" {
				if err := os.WriteFile(filepath.Join(root, "source.xyz"), []byte("source"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "installation conflict" {
				if err := os.MkdirAll(filepath.Join(root, ".guidance"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, ".guidance", "index.md"), []byte("foreign\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "unwritable configuration" {
				// Installation writes only below .guidance; the configuration beside it cannot be replaced.
				if err := os.Mkdir(filepath.Join(root, ".guidance"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(root, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
			}
			cfg := &config.Config{Guidance: config.Guidance{Source: source}}
			step, outcome := ensureProjectGuidance(t.Context(), root, cfg, config.HostSelection{}, RefreshOptions{AdoptGuidance: true})
			if step == nil || outcome == nil {
				t.Fatal("adoption produced no step or outcome")
			}
			// A pack counts as adopted only once aiwf.yaml records it.
			if (kind == "adopts") != (outcome.Adopted != nil) {
				t.Fatalf("adopted %v for %s", outcome.Adopted, kind)
			}
			raw, readErr := os.ReadFile(path)
			_, installErr := os.Stat(filepath.Join(root, ".guidance", "packs", "sample", "base", "guide.md"))
			switch kind {
			case "adopts":
				recorded, err := config.Load(root)
				if err != nil {
					t.Fatal(err)
				}
				want := []string{"sample/base"}
				if step.Action != ActionUpdated || installErr != nil || recorded.Guidance.Packs == nil || !slices.Equal(*recorded.Guidance.Packs, want) {
					t.Fatalf("adoption not recorded with installation: %+v, %s, %v", step, raw, installErr)
				}
				return
			case "no applicable pack":
				if step.Detail != "no project guidance selected; existing delivery retained" {
					t.Fatalf("unexpected detail: %+v", step)
				}
			default:
				cause := map[string]string{
					"detection failure":          "detecting guidance",
					"unreadable configuration":   "recording adopted guidance: reading",
					"unrecordable configuration": "recording adopted guidance: expand guidance anchors and aliases so guidance packs can be recorded: aliases are not supported inside the guidance: block",
					"installation conflict":      "installation conflict",
					"unwritable configuration":   "recording adopted guidance: writing",
				}[kind]
				if step.Action != ActionSkipped || !strings.HasPrefix(step.Detail, "guidance incomplete:") || !strings.Contains(step.Detail, cause) {
					t.Fatalf("failed adoption claimed success or named the wrong cause (want %q): %+v", cause, step)
				}
			}
			if kind != "unreadable configuration" && (readErr != nil || string(raw) != configuration) {
				t.Fatalf("configuration changed: %s (%v)", raw, readErr)
			}
			if kind != "unwritable configuration" {
				if !errors.Is(installErr, fs.ErrNotExist) {
					t.Fatalf("pack installed without a recorded selection: %v", installErr)
				}
				return
			}
			// The pack installed before the write failed; the next run records it
			// and reports the configuration it changed.
			if err := os.Chmod(root, 0o755); err != nil {
				t.Fatal(err)
			}
			rerun, _ := ensureProjectGuidance(t.Context(), root, &config.Config{Guidance: config.Guidance{Source: source}}, config.HostSelection{}, RefreshOptions{AdoptGuidance: true})
			recorded, err := config.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if rerun == nil || rerun.Action != ActionUpdated || recorded.Guidance.Packs == nil || !slices.Equal(*recorded.Guidance.Packs, []string{"sample/base"}) {
				t.Fatalf("interrupted adoption did not converge: %+v, packs %v", rerun, recorded.Guidance.Packs)
			}
		})
	}
}

func TestProjectGuidanceRefresh_ReportsSelectedPacksMissingFromCatalogue(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	for _, kind := range []string{"adopting", "not adopting", "adopting with an installation conflict"} {
		t.Run(kind, func(t *testing.T) {
			root := freshGitRepo(t)
			selected := []string{"gone/pack", "sample/base"}
			cfg := &config.Config{Guidance: config.Guidance{Source: testsupport.GuidanceSource(t), Packs: &selected}}
			if kind == "adopting with an installation conflict" {
				if err := os.MkdirAll(filepath.Join(root, ".guidance"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, ".guidance", "index.md"), []byte("foreign\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			step, _ := ensureProjectGuidance(t.Context(), root, cfg, config.HostSelection{}, RefreshOptions{AdoptGuidance: kind != "not adopting"})
			if step == nil || !strings.Contains(step.Detail, "no longer has gone/pack") || !strings.Contains(step.Detail, "guidance.ignored") {
				t.Fatalf("vanished pack not reported with its fix: %+v", step)
			}
			_, installErr := os.Stat(filepath.Join(root, ".guidance", "packs", "sample", "base", "guide.md"))
			if kind == "adopting" {
				if step.Action != ActionUpdated || installErr != nil {
					t.Fatalf("the rest of the selection was not installed: %+v, %v", step, installErr)
				}
				return
			}
			// A run that does not adopt leaves installed guidance as it is rather
			// than retiring a pack it cannot replace.
			if step.Action != ActionSkipped || !strings.HasPrefix(step.Detail, "guidance incomplete:") || !errors.Is(installErr, fs.ErrNotExist) {
				t.Fatalf("guidance changed or claimed success: %+v, %v", step, installErr)
			}
		})
	}
}

func TestInit_FirstDryRunPreviewsAdoption(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	res, err := Init(t.Context(), root, Options{AdoptGuidance: true, DryRun: true, SkipHook: true})
	if err != nil {
		t.Fatal(err)
	}
	previewed := slices.ContainsFunc(res.Steps, func(step StepResult) bool {
		return step.What == ".guidance (project engineering guidance)" && strings.Contains(step.Detail, "adopt")
	})
	if !previewed {
		t.Fatalf("first dry run did not preview guidance adoption: %+v", res.Steps)
	}
	if _, err := os.Stat(filepath.Join(root, config.FileName)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("dry run wrote configuration: %v", err)
	}
}

func TestProjectGuidance_ReportsOutcome(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	t.Run("not considered without a selection or adoption", func(t *testing.T) {
		_, outcome := projectGuidance(t.Context(), freshGitRepo(t), &config.Config{}, config.HostSelection{}, RefreshOptions{})
		if outcome != nil {
			t.Fatalf("outcome for a run that did not consider guidance: %+v", outcome)
		}
	})
	t.Run("adoption reports only this run's additions and the installed index", func(t *testing.T) {
		root, source := freshGitRepo(t), testsupport.GuidanceSource(t, "*.xyz")
		// The existing selection names a pack the catalogue lacks, so it is
		// kept but neither adopted now nor installed.
		if err := os.WriteFile(filepath.Join(root, config.FileName), []byte("hosts: []\nguidance:\n  source: "+source+"\n  packs: [gone/pack]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "source.xyz"), []byte("source"), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg := &config.Config{Guidance: config.Guidance{Source: source, Packs: &[]string{"gone/pack"}, Ignored: []string{"other/pack"}}}
		_, outcome := projectGuidance(t.Context(), root, cfg, config.HostSelection{}, RefreshOptions{AdoptGuidance: true})
		want := []string{"sample/base"}
		if outcome == nil || outcome.Selection == nil || !slices.Equal(*outcome.Selection, []string{"gone/pack", "sample/base"}) || !slices.Equal(outcome.Adopted, want) ||
			!slices.Equal(outcome.Installed.Packs, want) || outcome.Installed.Revision == "" || !slices.Equal(outcome.Ignored, []string{"other/pack"}) {
			t.Fatalf("outcome = %+v", outcome)
		}
	})
	t.Run("a dry run reports nothing and previews nothing when maintenance is disabled", func(t *testing.T) {
		root := freshGitRepo(t)
		step, outcome := projectGuidance(t.Context(), root, &config.Config{}, config.HostSelection{}, RefreshOptions{AdoptGuidance: true, DryRun: true})
		if step == nil || outcome != nil {
			t.Fatalf("enabled dry run: step %+v, outcome %+v", step, outcome)
		}
		disabled := false
		step, outcome = projectGuidance(t.Context(), root, &config.Config{Guidance: config.Guidance{Enabled: &disabled}}, config.HostSelection{}, RefreshOptions{AdoptGuidance: true, DryRun: true})
		if step != nil || outcome != nil {
			t.Fatalf("disabled dry run: step %+v, outcome %+v", step, outcome)
		}
	})
	t.Run("disabled maintenance reports without changing anything", func(t *testing.T) {
		root := freshGitRepo(t)
		disabled := false
		cfg := &config.Config{Guidance: config.Guidance{Enabled: &disabled, Packs: &[]string{"sample/base"}}}
		step, outcome := projectGuidance(t.Context(), root, cfg, config.HostSelection{}, RefreshOptions{AdoptGuidance: true})
		if step != nil || outcome == nil || !outcome.Disabled || outcome.Adopted != nil {
			t.Fatalf("step %+v, outcome %+v", step, outcome)
		}
		if _, err := os.Stat(filepath.Join(root, ".guidance")); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("disabled run wrote guidance: %v", err)
		}
	})
	t.Run("a refused instruction file blocks with its reason once", func(t *testing.T) {
		root := freshGitRepo(t)
		if err := os.WriteFile(filepath.Join(root, "elsewhere.md"), []byte("rules\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("elsewhere.md", filepath.Join(root, "CLAUDE.md")); err != nil {
			t.Fatal(err)
		}
		cfg := &config.Config{Guidance: config.Guidance{Packs: &[]string{"sample/base"}}}
		step, outcome := projectGuidance(t.Context(), root, cfg, config.HostSelection{Hosts: []config.Host{config.HostClaudeCode}}, RefreshOptions{WireClaudeMd: true})
		if step == nil || outcome == nil || strings.Count(step.Detail, "guidance incomplete: ") != 1 ||
			!strings.HasPrefix(outcome.Blocker, "CLAUDE.md is a symlink") {
			t.Fatalf("step %+v, outcome %+v", step, outcome)
		}
	})
	t.Run("unreadable installed guidance is reported", func(t *testing.T) {
		root := freshGitRepo(t)
		if err := os.MkdirAll(filepath.Join(root, ".guidance"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".guidance", ".aiwf-owned"), []byte("not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		disabled := false
		cfg := &config.Config{Guidance: config.Guidance{Enabled: &disabled}}
		_, outcome := projectGuidance(t.Context(), root, cfg, config.HostSelection{}, RefreshOptions{AdoptGuidance: true})
		if outcome == nil || !strings.HasPrefix(outcome.InspectError, "cannot inspect installed policy: ") {
			t.Fatalf("outcome = %+v", outcome)
		}
	})
}
