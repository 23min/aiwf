package doctor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/initrepo"
)

func hostDoctorFixture(t *testing.T, host config.Host) string {
	t.Helper()
	root := t.TempDir()
	if err := config.Write(root, &config.Config{Hosts: &[]string{string(host)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{SkipHook: true, ActorOverride: "human/test"}); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestDoctor_SelectedHostArtifactAbsenceAndDrift(t *testing.T) {
	t.Parallel()
	for _, host := range []config.Host{config.HostClaudeCode, config.HostCodex} {
		for _, family := range []string{"skills", "rituals", "templates", "guidance"} {
			for _, missing := range []bool{false, true} {
				name := string(host) + "/" + family + "/drift"
				if missing {
					name = string(host) + "/" + family + "/missing"
				}
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					root := hostDoctorFixture(t, host)
					base, templates, guidance := ".claude/skills", ".claude/templates", ".claude/aiwf-guidance.md"
					if host == config.HostCodex {
						base, templates, guidance = ".agents/skills", ".agents/aiwf/templates", "AGENTS.md"
					}
					relative := map[string]string{"skills": base + "/aiwf-check/SKILL.md", "rituals": base + "/wf-tdd-cycle/SKILL.md", "templates": templates + "/epic-spec.md", "guidance": guidance}[family]
					path := filepath.Join(root, relative)
					if missing {
						if err := os.Remove(path); err != nil {
							t.Fatal(err)
						}
					} else {
						before, err := os.ReadFile(path)
						if err != nil {
							t.Fatal(err)
						}
						changed := append([]byte("changed artifact\n"), before...)
						// Native guidance owns only its marked block, not the surrounding user text.
						if host == config.HostCodex && family == "guidance" {
							lines := strings.Split(string(before), "\n")
							lines[1] = "changed managed guidance"
							changed = []byte(strings.Join(lines, "\n"))
						}
						if err := os.WriteFile(path, changed, 0o644); err != nil {
							t.Fatal(err)
						}
					}
					_, problems := DoctorReport(root, DoctorOptions{})
					wantSeverity := SeverityWarn
					if family == "skills" {
						wantSeverity = SeverityError
					}
					found := false
					for _, p := range problems {
						if p.Host == host && p.Path == relative {
							found = true
							if p.Severity != wantSeverity {
								t.Errorf("severity = %s, want %s: %+v", p.Severity, wantSeverity, p)
							}
							if !strings.Contains(p.Message, "aiwf update") {
								t.Errorf("missing remediation: %+v", p)
							}
						}
					}
					if !found {
						t.Fatalf("no %s problem for %s: %+v", host, relative, problems)
					}
				})
			}
		}
	}
}

func TestDoctor_UserGuidanceAndUnselectedFilesAreNotDrift(t *testing.T) {
	t.Parallel()
	for _, host := range []config.Host{config.HostClaudeCode, config.HostCodex} {
		t.Run(string(host), func(t *testing.T) {
			t.Parallel()
			root := hostDoctorFixture(t, host)
			selected, other := "CLAUDE.md", "AGENTS.md"
			if host == config.HostCodex {
				selected, other = other, selected
			}
			path := filepath.Join(root, selected)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			content = append([]byte("Personal instructions before.\n\n"), content...)
			content = append(content, []byte("\nPersonal instructions after.\n")...)
			if err := os.WriteFile(path, content, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, other), []byte("unrelated user content"), 0o644); err != nil {
				t.Fatal(err)
			}
			_, problems := DoctorReport(root, DoctorOptions{})
			for _, p := range problems {
				if p.Host != "" && p.Path != "" {
					t.Errorf("healthy selected or unrelated user content flagged: %+v", p)
				}
			}
		})
	}
}

func TestDoctor_GuidanceRefusalsAreWarningsAndReadOnly(t *testing.T) {
	t.Parallel()
	for _, host := range []config.Host{config.HostClaudeCode, config.HostCodex} {
		for _, condition := range []string{"symlink", "directory", "alias", "dangling peer", "markers", "unreadable"} {
			t.Run(string(host)+"/"+condition, func(t *testing.T) {
				t.Parallel()
				root := hostDoctorFixture(t, host)
				relative, peer := "CLAUDE.md", "AGENTS.md"
				if host == config.HostCodex {
					relative, peer = peer, relative
				}
				path := filepath.Join(root, relative)
				before, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				external := filepath.Join(t.TempDir(), "instructions")
				switch condition {
				case "symlink", "directory":
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if condition == "directory" {
						if err := os.Mkdir(path, 0o755); err != nil {
							t.Fatal(err)
						}
					} else {
						if err := os.WriteFile(external, before, 0o600); err != nil {
							t.Fatal(err)
						}
						if err := os.Symlink(external, path); err != nil {
							t.Fatal(err)
						}
					}
				case "alias":
					if err := os.Link(path, filepath.Join(root, peer)); err != nil {
						t.Fatal(err)
					}
				case "dangling peer":
					if err := os.Symlink("missing", filepath.Join(root, peer)); err != nil {
						t.Fatal(err)
					}
				case "markers":
					before = []byte(strings.ReplaceAll(string(before), "<!-- aiwf:guidance:END -->", ""))
					if err := os.WriteFile(path, before, 0o644); err != nil {
						t.Fatal(err)
					}
				case "unreadable":
					if err := os.Chmod(path, 0); err != nil {
						t.Fatal(err)
					}
					if _, err := os.ReadFile(path); err == nil {
						t.Skip("process bypasses permissions")
					}
				}
				_, problems := appendHostGuidanceReport(nil, nil, root, host, nil)
				found := false
				for _, problem := range problems {
					if problem.Host == host && problem.Path == relative && problem.Severity == SeverityWarn && strings.Contains(problem.Message, "blocked") {
						found = true
					}
				}
				if !found {
					t.Fatalf("guidance refusal missing: %+v", problems)
				}
				if condition == "directory" {
					return
				}
				if condition == "unreadable" {
					if err := os.Chmod(path, 0o644); err != nil {
						t.Fatal(err)
					}
				}
				after, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if !bytes.Equal(after, before) {
					t.Fatal("doctor changed guidance")
				}
				if condition == "symlink" {
					info, err := os.Lstat(path)
					if err != nil || info.Mode()&os.ModeSymlink == 0 {
						t.Fatalf("doctor replaced link: %v", err)
					}
				}
			})
		}
	}
}

func TestDoctor_ReportsBothHostsAndAllConcurrentDrift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := config.Write(root, &config.Config{Hosts: &[]string{"claude-code", "codex"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{SkipHook: true, ActorOverride: "human/test"}); err != nil {
		t.Fatal(err)
	}
	paths := []string{".claude/skills/aiwf-check/SKILL.md", ".agents/skills/aiwf-check/SKILL.md", ".claude/templates/epic-spec.md", ".agents/aiwf/templates/epic-spec.md"}
	for i, path := range paths {
		var err error
		if i < 2 {
			err = os.Remove(filepath.Join(root, path))
		} else {
			err = os.WriteFile(filepath.Join(root, path), []byte("drift"), 0o644)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	_, problems := DoctorReport(root, DoctorOptions{})
	for _, path := range paths {
		found := false
		for _, problem := range problems {
			if problem.Path == path {
				found = true
			}
		}
		if !found {
			t.Errorf("concurrent problem omitted: %s", path)
		}
	}
}

func TestDoctor_ConfiguredAgentTiersAreExpectedBytes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\nagents:\n  builder:\n    model: sonnet\n    effort: high\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{SkipHook: true, ActorOverride: "human/test"}); err != nil {
		t.Fatal(err)
	}
	_, problems := DoctorReport(root, DoctorOptions{})
	for _, p := range problems {
		if p.Path == ".claude/agents/builder.md" {
			t.Fatalf("configured agent incorrectly drifted: %+v", p)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "agents", "builder.md"), []byte("changed card"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, problems = DoctorReport(root, DoctorOptions{})
	found := false
	for _, p := range problems {
		if p.Path == ".claude/agents/builder.md" && p.Severity == SeverityWarn {
			found = true
		}
	}
	if !found {
		t.Fatal("agent card drift not diagnosed")
	}
}
