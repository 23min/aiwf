package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
)

// Execute the YAML examples as configuration; prose and formatting within each
// example may change without changing the expected configuration behavior.
func TestHostDiscovery_HelpExamplesConfigureLifecycle(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, command := range [][]string{{"init"}, {"update"}, {"upgrade"}, {"doctor"}, {"worktree", "add"}} {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			t.Parallel()
			home := t.TempDir()
			path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode, config.HostCodex})
			help := hostLifecycleRun(t, t.TempDir(), home, path, append(append([]string{"aiwf"}, command...), "--help")...)
			var examples []string
			for _, section := range strings.Split(help, "```yaml\n")[1:] {
				body, _, ok := strings.Cut(section, "```")
				if !ok {
					t.Fatal("unterminated YAML example")
				}
				examples = append(examples, body)
			}
			if len(examples) != 3 {
				t.Fatalf("help offers %d configuration examples, want explicit hosts, none, and customized automatic detection", len(examples))
			}
			for i, body := range examples {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					t.Parallel()
					root := t.TempDir()
					claudeBaselineWrite(t, root, "aiwf.yaml", body)
					cfg, err := config.Load(root)
					if err != nil {
						t.Fatal(err)
					}
					var wantHosts *[]string
					source, hosts := config.HostsConfigured, []config.Host{config.HostClaudeCode, config.HostCodex}
					switch i {
					case 0:
						wantHosts = &[]string{"claude-code", "codex"}
					case 1:
						wantHosts = &[]string{}
						hosts = []config.Host{}
					case 2:
						source = config.HostsDetected
					}
					if diff := cmp.Diff(wantHosts, cfg.Hosts); diff != "" {
						t.Fatalf("example host config:\n%s", diff)
					}
					wantWire, wantDir := true, ".claude/worktrees"
					if i == 2 {
						wantWire, wantDir = false, ".worktrees"
					}
					if cfg.WireClaudeMd() != wantWire || cfg.WireAgentsMd() != wantWire || cfg.WorktreeDir() != wantDir {
						t.Fatalf("example guidance/worktree values: claude=%v codex=%v dir=%s", cfg.WireClaudeMd(), cfg.WireAgentsMd(), cfg.WorktreeDir())
					}
					hostLifecycleRun(t, root, home, path, "git", "init", "-q")
					output := hostLifecycleRun(t, root, home, path, "aiwf", "init", "--dry-run", "--skip-hook")
					assertHostSelectionReport(t, output, config.HostSelection{Hosts: hosts, Source: source})
				})
			}
		})
	}
}
