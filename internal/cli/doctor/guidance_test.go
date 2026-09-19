package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
)

func guidanceFixture(t *testing.T, withGuidanceFile, withImport bool) string {
	t.Helper()
	root := hostDoctorFixture(t, config.HostClaudeCode)
	if !withGuidanceFile {
		if err := os.Remove(filepath.Join(root, ".claude", "aiwf-guidance.md")); err != nil {
			t.Fatal(err)
		}
	}
	if !withImport {
		if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# project\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestGuidanceReport_ClaudeStates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name            string
		fragment, wired bool
		want            int
	}{
		{"current", true, true, 0},
		{"unwired", true, false, 1},
		{"fragment absent", false, true, 1},
		{"fragment and import absent", false, false, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := guidanceFixture(t, tc.fragment, tc.wired)
			_, problems := appendHostGuidanceReport(nil, nil, root, config.HostClaudeCode, nil)
			if len(problems) != tc.want {
				t.Fatalf("got %+v, want %d problems", problems, tc.want)
			}
			for _, p := range problems {
				if p.Severity != SeverityWarn || p.Host != config.HostClaudeCode || !strings.Contains(p.Message, "aiwf update") {
					t.Errorf("guidance finding = %+v", p)
				}
			}
		})
	}
}

func TestGuidanceReport_OptOutIsExplicit(t *testing.T) {
	t.Parallel()
	for _, host := range []config.Host{config.HostClaudeCode, config.HostCodex} {
		t.Run(string(host), func(t *testing.T) {
			t.Parallel()
			root := hostDoctorFixture(t, host)
			cfg, err := config.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			disabled := false
			cfg.Guidance.WireClaudeMd = &disabled
			cfg.Guidance.WireAgentsMd = &disabled
			path := "CLAUDE.md"
			if host == config.HostCodex {
				path = "AGENTS.md"
			}
			if err := os.WriteFile(filepath.Join(root, path), []byte("personal instructions only"), 0o644); err != nil {
				t.Fatal(err)
			}
			lines, problems := appendHostGuidanceReport(nil, nil, root, host, cfg)
			if len(problems) != 0 || !strings.Contains(strings.Join(lines, "\n"), "opted out") {
				t.Fatalf("opt-out = %v %+v", lines, problems)
			}
		})
	}
}

func TestGuidanceReport_MissingRootInstructions(t *testing.T) {
	t.Parallel()
	for _, host := range []config.Host{config.HostClaudeCode, config.HostCodex} {
		t.Run(string(host), func(t *testing.T) {
			t.Parallel()
			root := hostDoctorFixture(t, host)
			path := "CLAUDE.md"
			if host == config.HostCodex {
				path = "AGENTS.md"
			}
			if err := os.Remove(filepath.Join(root, path)); err != nil {
				t.Fatal(err)
			}
			_, problems := appendHostGuidanceReport(nil, nil, root, host, nil)
			if len(problems) != 1 || problems[0].Path != path || !strings.Contains(problems[0].Message, "missing") {
				t.Fatalf("absent root instructions = %+v", problems)
			}
		})
	}
}
