package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// Serial: CaptureRun redirects process stdout/stderr. These drive the dispatcher
// in-process so its host decisions are measured by the suite's coverage profile.
func TestHostDispatch_SelectionAndClaudeOnlyRequests(t *testing.T) {
	for _, hosts := range []string{"[]", "[codex]"} {
		t.Run(hosts, func(t *testing.T) {
			root := setupCLITestRepo(t)
			claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: "+hosts+"\n")
			for _, verb := range []string{"init", "update"} {
				rc, out, stderr := testutil.CaptureRun(t, func() int {
					args := []string{verb, "--root", root}
					if verb == "init" {
						args = append(args, "--no-prompt")
					}
					return cli.Execute(args)
				})
				if rc != cliutil.ExitOK || !strings.Contains(out, "Hosts (configured):") {
					t.Fatalf("%s: exit=%d, stdout=%s, stderr=%s", verb, rc, out, stderr)
				}
				if hosts == "[]" && !strings.Contains(out, "Hosts (configured): none") {
					t.Fatalf("empty selection not reported: %s", out)
				}
			}
			for _, args := range [][]string{{"init", "--statusline"}, {"update", "--statusline"}, {"doctor", "--write-health"}} {
				rc, _, stderr := testutil.CaptureRun(t, func() int {
					return cli.Execute(append(args, "--root", root))
				})
				if rc != cliutil.ExitUsage || !strings.Contains(stderr, "claude-code") {
					t.Fatalf("%v: exit=%d, stderr=%s", args, rc, stderr)
				}
			}
		})
	}
}

func TestHostDispatch_MalformedRitualConfigReportsFailure(t *testing.T) {
	root := setupCLITestRepo(t)
	claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: [\n")
	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"doctor", "--root", root, "--check-rituals"})
	})
	if rc != cliutil.ExitInternal || !strings.Contains(stderr, "aiwf.yaml") {
		t.Fatalf("malformed config: exit=%d, stderr=%s", rc, stderr)
	}
}

func TestHostDispatch_WorktreeFailuresRemoveCheckoutAndBranch(t *testing.T) {
	for _, failure := range []string{"hook settings", "skill ownership"} {
		t.Run(failure, func(t *testing.T) {
			root := setupCLITestRepo(t)
			claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: [claude-code]\nhooks:\n  worktree-rituals-check.sh:\n    enabled: true\n")
			rc, _, stderr := testutil.CaptureRun(t, func() int {
				return cli.Execute([]string{"init", "--root", root, "--no-prompt"})
			})
			if rc != cliutil.ExitOK {
				t.Fatalf("init exit=%d: %s", rc, stderr)
			}
			if failure == "hook settings" {
				claudeBaselineWrite(t, root, ".claude/settings.json", "invalid JSON\n")
			} else {
				claudeBaselineWrite(t, root, ".claude/skills/.aiwf-owned", "../escape\n")
				if err := osExec(t, root, "git", "add", "-f", ".claude/skills/.aiwf-owned"); err != nil {
					t.Fatal(err)
				}
			}
			for _, args := range [][]string{{"add", "-A"}, {"-c", "core.hooksPath=/dev/null", "commit", "-qm", "chore: seed invalid host settings"}} {
				if err := osExec(t, root, "git", args...); err != nil {
					t.Fatal(err)
				}
			}
			checkout := filepath.Join(t.TempDir(), "checkout")
			rc, _, stderr = testutil.CaptureRun(t, func() int {
				return cli.Execute([]string{"worktree", "add", "feature/bad-settings", checkout, "--root", root})
			})
			if rc != cliutil.ExitInternal {
				t.Fatalf("worktree exit=%d: %s", rc, stderr)
			}
			if _, err := os.Stat(checkout); !os.IsNotExist(err) {
				t.Fatalf("failed checkout remains: %v", err)
			}
			if err := osExec(t, root, "git", "show-ref", "--verify", "refs/heads/feature/bad-settings"); err == nil {
				t.Fatal("failed worktree branch remains")
			}
		})
	}
}
