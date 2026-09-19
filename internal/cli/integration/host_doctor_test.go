package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
)

func TestHostLifecycle_DoctorUsesSelectionAndReportsUnavailableTools(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, hosts := range []string{"", "[]", "[codex]", "[claude-code]", "[claude-code, codex]"} {
		t.Run(hosts, func(t *testing.T) {
			t.Parallel()
			root, home := t.TempDir(), t.TempDir()
			path := hostLifecyclePATH(t, binary, nil)
			run := func(args ...string) string { return hostLifecycleRun(t, root, home, path, args...) }
			run("git", "init", "-q", "-b", "main")
			run("git", "config", "user.email", "test@example.com")
			content := "hooks:\n  worktree-rituals-check.sh:\n    enabled: false\n"
			if hosts != "" {
				content += "hosts: " + hosts + "\n"
			}
			claudeBaselineWrite(t, root, "aiwf.yaml", content)
			run("aiwf", "init")
			if !strings.Contains(hosts, "claude-code") {
				claudeBaselineWrite(t, root, ".claude/settings.json", "invalid retained settings\n")
				claudeBaselineWrite(t, root, "CLAUDE.md", "unselected instructions\n")
			}
			before := retainedPathSnapshot(t, root, []string{".claude", "CLAUDE.md", ".agents", "AGENTS.md", "aiwf.yaml"}, true)
			output := run("aiwf", "doctor")
			for _, host := range []string{"claude-code", "codex"} {
				if strings.Contains(hosts, host) {
					found := false
					for _, line := range strings.Split(output, "\n") {
						fields := strings.Fields(line)
						if len(fields) > 1 && fields[0] == "host-tool:" && fields[1] == host+":" {
							found = true
						}
					}
					if !found {
						t.Errorf("selected absent tool %s not reported:\n%s", host, output)
					}
				} else if strings.Contains(output, host+" skills:") {
					t.Errorf("unselected %s artifacts diagnosed:\n%s", host, output)
				}
			}
			if !strings.Contains(hosts, "claude-code") && !strings.Contains(output, "retained:") {
				t.Errorf("retained Claude paths omitted:\n%s", output)
			}
			run("aiwf", "doctor", "--check-rituals")
			if diff := cmp.Diff(before, retainedPathSnapshot(t, root, []string{".claude", "CLAUDE.md", ".agents", "AGENTS.md", "aiwf.yaml"}, false)); diff != "" {
				t.Fatalf("doctor mutated host files (-want +got):\n%s", diff)
			}
			if !strings.Contains(hosts, "claude-code") {
				hostLifecycleRunExit(t, root, home, path, 2, "aiwf", "doctor", "--write-health")
				if diff := cmp.Diff(before, retainedPathSnapshot(t, root, []string{".claude", "CLAUDE.md", ".agents", "AGENTS.md", "aiwf.yaml"}, false)); diff != "" {
					t.Fatalf("refused health write changed files (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestHostLifecycle_DoctorRitualDriftIsAdvisoryButFailsAutomationCheck(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	root, home := t.TempDir(), t.TempDir()
	path := hostLifecyclePATH(t, binary, []config.Host{config.HostCodex})
	run := func(args ...string) string { return hostLifecycleRun(t, root, home, path, args...) }
	run("git", "init", "-q", "-b", "main")
	run("git", "config", "user.email", "test@example.com")
	run("aiwf", "init", "--actor", "human/test")
	claudeBaselineWrite(t, root, ".agents/aiwf/templates/epic-spec.md", "stale template\n")
	output := run("aiwf", "doctor")
	if !strings.Contains(output, "codex templates: drifted .agents/aiwf/templates/epic-spec.md") {
		t.Fatalf("template drift omitted:\n%s", output)
	}
	hostLifecycleRunExit(t, root, home, path, 1, "aiwf", "doctor", "--check-rituals")
	if err := os.Remove(filepath.Join(root, ".agents", "skills", "aiwf-check", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	hostLifecycleRunExit(t, root, home, path, 1, "aiwf", "doctor")
	run("aiwf", "update")
	run("aiwf", "doctor")
	run("aiwf", "doctor", "--check-rituals")
}

func TestHostLifecycle_DoctorRejectsMalformedHostConfiguration(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	root, home := t.TempDir(), t.TempDir()
	path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode})
	hostLifecycleRun(t, root, home, path, "git", "init", "-q")
	claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: [unknown]\n")
	hostLifecycleRunExit(t, root, home, path, 1, "aiwf", "doctor")
	hostLifecycleRunExit(t, root, home, path, 3, "aiwf", "doctor", "--check-rituals")
	hostLifecycleRunExit(t, root, home, path, 3, "aiwf", "doctor", "--write-health")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if diff := cmp.Diff([]string{".git", "aiwf.yaml"}, names); diff != "" {
		t.Fatalf("invalid config wrote artifacts (-want +got):\n%s", diff)
	}
}
