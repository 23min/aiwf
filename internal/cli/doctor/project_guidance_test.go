package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/projectguidance"
	"github.com/23min/aiwf/internal/testsupport"
)

func TestDoctor_ProjectGuidanceReportsInstalledRevisionLocally(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	source := testsupport.GuidanceSource(t)
	packs := []string{"sample/base"}
	snapshot, err := projectguidance.Retrieve(t.Context(), source, packs)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = projectguidance.Install(t.Context(), root, snapshot, projectguidance.InstallOptions{Source: source, Selected: packs}); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Hosts: &[]string{}, Guidance: config.Guidance{Source: "/unavailable-upstream", Packs: &packs}}
	if err = config.Write(root, cfg); err != nil {
		t.Fatal(err)
	}
	lines, _ := DoctorReport(root, DoctorOptions{})
	text := strings.Join(lines, "\n")
	for _, want := range []string{"selected: sample/base", "installed revision: " + snapshot.Commit, "upstream freshness not checked", ".guidance/packs/sample/base/guide.md"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in report:\n%s", want, text)
		}
	}
}

func TestDoctor_ProjectGuidanceStateAndWarnings(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"unset", "empty", "disabled", "not installed", "corrupt", "pending", "missing document", "selected claude", "selected codex", "unselected claude", "unselected codex", "opted out claude", "opted out codex"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			packs := []string{"sample/base"}
			cfg := &config.Config{Guidance: config.Guidance{Packs: &packs}}
			selection := config.HostSelection{Hosts: []config.Host{config.HostClaudeCode, config.HostCodex}}
			want := ""
			path := ""
			retained := false
			switch kind {
			case "unset":
				cfg = nil
				want = "not adopted"
			case "empty":
				packs = nil
				want = "none (explicit empty selection)"
			case "disabled":
				disabled := false
				cfg.Guidance.Enabled = &disabled
				want = "maintenance disabled"
			case "not installed":
				path = ".guidance/index.md"
			case "corrupt":
				path = ".guidance"
			case "pending":
				path = ".guidance/.aiwf-pending"
			case "missing document":
				path = ".guidance/packs/sample/base/guide.md"
			default:
				path = "AGENTS.md"
				if strings.HasSuffix(kind, "claude") {
					path = "CLAUDE.md"
				}
				if strings.HasPrefix(kind, "unselected") {
					selection.Hosts = nil
					retained = true
				}
				if strings.HasPrefix(kind, "opted out") {
					disabled := false
					cfg.Guidance.WireAgentsMd = &disabled
					cfg.Guidance.WireClaudeMd = &disabled
					retained = true
				}
			}
			if kind != "unset" && kind != "not installed" {
				source := testsupport.GuidanceSource(t)
				snapshot, err := projectguidance.Retrieve(t.Context(), source, packs)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = projectguidance.Install(t.Context(), root, snapshot, projectguidance.InstallOptions{Source: source, Selected: packs, HostFiles: []string{"AGENTS.md", "CLAUDE.md"}}); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "corrupt":
				writeDoctorGuidance(t, root, ".guidance/.aiwf-owned", []byte("invalid"))
			case "pending":
				writeDoctorGuidance(t, root, path, []byte("{}"))
			case "missing document":
				if err := os.Remove(filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
			default:
				if path == "AGENTS.md" || path == "CLAUDE.md" {
					data, err := os.ReadFile(filepath.Join(root, path))
					if err != nil {
						t.Fatal(err)
					}
					writeDoctorGuidance(t, root, path, []byte(strings.Replace(string(data), "Handwritten project overrides", "Local project overrides", 1)))
				}
			}
			lines, problems := appendProjectGuidanceReport(nil, nil, root, cfg, selection)
			if want != "" && !strings.Contains(strings.Join(lines, "\n"), want) {
				t.Fatalf("missing %q: %v", want, lines)
			}
			if path != "" {
				found := false
				for _, p := range problems {
					if p.Path == path {
						found = true
						if p.Severity != SeverityWarn {
							t.Fatalf("not warning: %+v", p)
						}
						if strings.Contains(p.Message, "retained unchanged") != retained {
							t.Fatalf("wrong retained status: %+v", p)
						}
					}
				}
				if !found {
					t.Fatalf("missing warning for %s: %+v", path, problems)
				}
			} else if len(problems) != 0 {
				t.Fatalf("unexpected warnings: %+v", problems)
			}
		})
	}
}

func writeDoctorGuidance(t *testing.T, root, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDoctor_ProjectGuidanceNewlySelectedHostNeedsRouting(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		hosts         []config.Host
		claude, codex bool
		want          []string
	}{
		{"claude", []config.Host{config.HostClaudeCode}, true, true, []string{"CLAUDE.md"}},
		{"codex", []config.Host{config.HostCodex}, true, true, []string{"AGENTS.md"}},
		{"claude optout", []config.Host{config.HostClaudeCode}, false, true, nil},
		{"codex optout", []config.Host{config.HostCodex}, true, false, nil},
		{"neither", nil, true, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			packs := []string{}
			snapshot, err := projectguidance.Retrieve(t.Context(), testsupport.GuidanceSource(t), packs)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = projectguidance.Install(t.Context(), root, snapshot, projectguidance.InstallOptions{}); err != nil {
				t.Fatal(err)
			}
			cfg := &config.Config{Guidance: config.Guidance{Packs: &packs, WireClaudeMd: &tc.claude, WireAgentsMd: &tc.codex}}
			_, problems := appendProjectGuidanceReport(nil, nil, root, cfg, config.HostSelection{Hosts: tc.hosts})
			var paths []string
			for _, p := range problems {
				paths = append(paths, p.Path)
			}
			if diff := cmp.Diff(tc.want, paths); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
