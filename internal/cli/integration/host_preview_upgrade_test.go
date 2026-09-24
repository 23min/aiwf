package integration

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/testsupport"
)

func TestHostLifecycle_DryRunPreservesConsumerAndHome(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, hosts := range [][]config.Host{{}, {config.HostClaudeCode}, {config.HostCodex}, {config.HostClaudeCode, config.HostCodex}} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/explicit=%v", hosts, explicit), func(t *testing.T) {
				t.Parallel()
				root, home := t.TempDir(), t.TempDir()
				installed, source := hosts, config.HostsDetected
				if explicit {
					installed, source = nil, config.HostsConfigured
				}
				path := hostLifecyclePATH(t, binary, installed)
				hostLifecycleRun(t, root, home, path, "git", "init", "-q")
				if explicit {
					names := make([]string, len(hosts))
					for i, host := range hosts {
						names[i] = string(host)
					}
					claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: ["+strings.Join(names, ", ")+"]\n")
				}
				for _, base := range []string{root, home} {
					claudeBaselineWrite(t, base, ".claude/settings.json", "personal settings\n")
					claudeBaselineWrite(t, base, ".agents/personal.txt", "personal data\n")
				}
				before := retainedPathSnapshot(t, root, []string{"."}, true)
				homeBefore := retainedPathSnapshot(t, home, []string{"."}, true)
				args := []string{"aiwf", "init", "--dry-run", "--actor", "human/test"}
				for _, host := range hosts {
					if host == config.HostClaudeCode {
						args = append(args, "--statusline", "--wire-settings", "--enable-hook", "worktree-rituals-check.sh")
					}
				}
				output := hostLifecycleRun(t, root, home, path, args...)
				assertHostSelectionReport(t, output, config.HostSelection{Hosts: hosts, Source: source})
				for host, artifact := range map[config.Host]string{config.HostClaudeCode: ".claude/skills/aiwf-*", config.HostCodex: ".agents/skills and .agents/aiwf/templates"} {
					if strings.Contains(output, artifact) != slices.Contains(hosts, host) {
						t.Errorf("preview selected artifact ledger for %s disagrees with host selection: %s", host, output)
					}
				}
				if diff := cmp.Diff(before, retainedPathSnapshot(t, root, []string{"."}, false)); diff != "" {
					t.Fatalf("dry-run changed consumer (-before +after):\n%s", diff)
				}
				if diff := cmp.Diff(homeBefore, retainedPathSnapshot(t, home, []string{"."}, false)); diff != "" {
					t.Fatalf("dry-run changed home (-before +after):\n%s", diff)
				}
			})
		}
	}
}

func TestHostLifecycle_UpgradeReexecResolvesCheckoutHosts(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, hosts := range [][]config.Host{{}, {config.HostClaudeCode}, {config.HostCodex}, {config.HostClaudeCode, config.HostCodex}} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/explicit=%v", hosts, explicit), func(t *testing.T) {
				t.Parallel()
				root, home := t.TempDir(), t.TempDir()
				installed, source := hosts, config.HostsDetected
				content := "hooks:\n  worktree-rituals-check.sh:\n    enabled: false\n"
				if explicit {
					installed, source = nil, config.HostsConfigured
					names := make([]string, len(hosts))
					for i, host := range hosts {
						names[i] = string(host)
					}
					content += "hosts: [" + strings.Join(names, ", ") + "]\n"
				}
				path := hostLifecyclePATH(t, binary, installed)
				hostLifecycleRun(t, root, home, path, "git", "init", "-q", "-b", "main")
				// Main selects neither host; the linked checkout's own config is authoritative.
				claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: []\n")
				hostLifecycleRun(t, root, home, path, "git", "add", "aiwf.yaml")
				hostLifecycleRun(t, root, home, path, "git", "commit", "-q", "-m", "chore: seed main")
				checkout := filepath.Join(t.TempDir(), "linked checkout")
				hostLifecycleRun(t, root, home, path, "git", "worktree", "add", "-q", "-b", "feature/upgrade", checkout)
				claudeBaselineWrite(t, checkout, "aiwf.yaml", content)
				nested := filepath.Join(checkout, "nested")
				if err := os.Mkdir(nested, 0o755); err != nil {
					t.Fatal(err)
				}
				args := []string{"upgrade", "--version", "v0.1.0"}
				// Exercise both cwd discovery and an explicit relative root from elsewhere.
				cwd := nested
				if explicit {
					cwd = root
					relative, err := filepath.Rel(root, checkout)
					if err != nil {
						t.Fatal(err)
					}
					args = append(args, "--root", relative)
				}
				output := hostUpgradeRun(t, binary, cwd, home, path, false, 0, args...)
				assertHostSelectionReport(t, output, config.HostSelection{Hosts: hosts, Source: source})
				assertHostLifecycleArtifacts(t, checkout, hosts)
				for _, relative := range []string{".claude", "CLAUDE.md", ".agents", "AGENTS.md", "aiwf.example.yaml", ".gitignore"} {
					if _, err := os.Lstat(filepath.Join(root, relative)); !os.IsNotExist(err) {
						t.Errorf("unexpected output at %s: %v", relative, err)
					}
				}
				if diff := cmp.Diff(content, string(readFile(t, filepath.Join(checkout, "aiwf.yaml")))); diff != "" {
					t.Fatalf("upgrade rewrote host config:\n%s", diff)
				}
				if diff := cmp.Diff("hosts: []\n", string(readFile(t, filepath.Join(root, "aiwf.yaml")))); diff != "" {
					t.Fatalf("upgrade changed main config:\n%s", diff)
				}
			})
		}
	}
}

// TestHostLifecycle_UpgradeNoPromptReachesReexecutedUpdate runs upgrade with
// --no-prompt through the real re-exec: the re-executed update must accept the
// forwarded flag and complete.
func TestHostLifecycle_UpgradeNoPromptReachesReexecutedUpdate(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	root, home := t.TempDir(), t.TempDir()
	path := hostLifecyclePATH(t, binary, nil)
	hostLifecycleRun(t, root, home, path, "git", "init", "-q", "-b", "main")
	claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: []\n")
	output := hostUpgradeRun(t, binary, root, home, path, false, 0, "upgrade", "--version", "v0.1.0", "--no-prompt")
	if !strings.Contains(output, "update --root "+root+" --no-prompt\n") {
		t.Fatalf("upgrade did not re-execute update with --no-prompt:\n%s", output)
	}
}

func TestHostLifecycle_UpgradePropagatesRefreshAndExecFailures(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, tc := range []struct {
		name, config string
		brokenExec   bool
	}{
		{"unknown host", "hosts: [unsupported]\n", false},
		{"malformed config", "hosts: [\n", false},
		{"cannot execute replacement", "hosts: [codex]\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, home := t.TempDir(), t.TempDir()
			path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode})
			hostLifecycleRun(t, root, home, path, "git", "init", "-q")
			claudeBaselineWrite(t, root, "aiwf.yaml", tc.config)
			hostUpgradeRun(t, binary, root, home, path, tc.brokenExec, 3, "upgrade", "--version", "v0.1.0")
			for _, relative := range []string{".claude", "CLAUDE.md", ".agents", "AGENTS.md", "aiwf.example.yaml", ".gitignore"} {
				if _, err := os.Lstat(filepath.Join(root, relative)); !os.IsNotExist(err) {
					t.Errorf("unexpected output at %s: %v", relative, err)
				}
			}
			if diff := cmp.Diff(tc.config, string(readFile(t, filepath.Join(root, "aiwf.yaml")))); diff != "" {
				t.Fatalf("failed upgrade changed configuration:\n%s", diff)
			}
		})
	}
}

// Fake only installation: the replacement process runs the actual update command.
func hostUpgradeRun(t *testing.T, binary, cwd, home, path string, brokenExec bool, wantCode int, args ...string) string {
	t.Helper()
	installerRoot := t.TempDir()
	replacement := filepath.Join(installerRoot, "aiwf")
	if brokenExec {
		if err := os.WriteFile(replacement, []byte("not executable\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	} else if err := testsupport.WriteExecutable(replacement, []byte("#!/bin/sh\nexec \"$AIWF_TEST_BINARY\" \"$@\"\n")); err != nil {
		t.Fatal(err)
	}
	installer := filepath.Join(installerRoot, "go")
	script := `#!/bin/sh
case "$1" in
 install) printf '%s\n' "$2" > "$AIWF_TEST_INSTALL_LOG" ;;
 env) test "$2" = GOBIN || exit 91; printf '%s\n' "$AIWF_TEST_GOBIN" ;;
 *) exit 92 ;;
esac
`
	if err := testsupport.WriteExecutable(installer, []byte(script)); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(installerRoot, "install.log")
	cmd := exec.Command(binary, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "PATH="+path, "HOME="+home, "XDG_CONFIG_HOME="+home,
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GOPROXY=off", "AIWF_NO_REEXEC=",
		"AIWF_GO_BIN="+installer, "AIWF_TEST_BINARY="+binary, "AIWF_TEST_GOBIN="+installerRoot, "AIWF_TEST_INSTALL_LOG="+log)
	output, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal(err)
		}
		code = exitErr.ExitCode()
	}
	if code != wantCode {
		t.Fatalf("upgrade exit=%d want=%d: %s", code, wantCode, output)
	}
	if diff := cmp.Diff("github.com/23min/aiwf/cmd/aiwf@v0.1.0\n", string(readFile(t, log))); diff != "" {
		t.Fatalf("installer argument:\n%s", diff)
	}
	return string(output)
}

func TestHostLifecycle_DryRunInvalidConfigurationWritesNothing(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, content := range []string{"hosts: [unsupported]\n", "hosts: [\n"} {
		t.Run(content, func(t *testing.T) {
			t.Parallel()
			root, home := t.TempDir(), t.TempDir()
			path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode, config.HostCodex})
			hostLifecycleRun(t, root, home, path, "git", "init", "-q")
			claudeBaselineWrite(t, root, "aiwf.yaml", content)
			before := retainedPathSnapshot(t, root, []string{"."}, true)
			homeBefore := retainedPathSnapshot(t, home, []string{"."}, true)
			hostLifecycleRunExit(t, root, home, path, 3, "aiwf", "init", "--dry-run")
			if diff := cmp.Diff(before, retainedPathSnapshot(t, root, []string{"."}, false)); diff != "" {
				t.Fatalf("failed preview changed consumer:\n%s", diff)
			}
			if diff := cmp.Diff(homeBefore, retainedPathSnapshot(t, home, []string{"."}, false)); diff != "" {
				t.Fatalf("failed preview changed home:\n%s", diff)
			}
		})
	}
}
