package integration

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/testsupport"
)

func TestHostLifecycle_DeselectedAndDisappearedHostsAreRetained(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir(), "-ldflags=-X github.com/23min/aiwf/internal/version.Stamp=v0.36.0")
	for _, selected := range [][]config.Host{{}, {config.HostClaudeCode}, {config.HostCodex}} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/explicit=%v", selected, explicit), func(t *testing.T) {
				t.Parallel()
				root, home := t.TempDir(), t.TempDir()
				path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode, config.HostCodex})
				run := func(args ...string) string { return hostLifecycleRun(t, root, home, path, args...) }
				run("git", "init", "-q", "-b", "main")
				run("aiwf", "init", "--actor", "human/test", "--no-prompt", "--enable-hook", "worktree-rituals-check.sh", "--statusline", "--scope", "project", "--wire-settings")
				run("aiwf", "update")
				content := string(readFile(t, filepath.Join(root, "aiwf.yaml")))
				source := config.HostsDetected
				if explicit {
					source = config.HostsConfigured
					names := make([]string, len(selected))
					for i, host := range selected {
						names[i] = string(host)
					}
					content += "hosts: [" + strings.Join(names, ", ") + "]\n"
					claudeBaselineWrite(t, root, "aiwf.yaml", content)
				} else {
					for _, host := range []config.Host{config.HostClaudeCode, config.HostCodex} {
						if !slices.Contains(selected, host) {
							command := "codex"
							if host == config.HostClaudeCode {
								command = "claude"
							}
							if err := os.Remove(filepath.Join(path, command)); err != nil {
								t.Fatal(err)
							}
						}
					}
				}
				var retained []string
				if !slices.Contains(selected, config.HostClaudeCode) {
					retained = append(retained, ".claude", "CLAUDE.md")
					// Old marked scripts must not auto-refresh in either scope.
					for _, dir := range []string{root, home} {
						claudeBaselineWrite(t, dir, ".claude/statusline.sh", "#!/bin/sh\n# aiwf-statusline version: v0.35.0\nexit 0\n")
						claudeBaselineWrite(t, dir, ".claude/settings.local.json", "{\"personal\":true}\n")
					}
				}
				if !slices.Contains(selected, config.HostCodex) {
					retained = append(retained, ".agents", "AGENTS.md")
				}
				before := retainedPathSnapshot(t, root, retained, true)
				homeBefore := retainedPathSnapshot(t, home, []string{".claude"}, true)
				for _, verb := range []string{"init", "update"} {
					// Core Git hooks must still be refreshed when no assistant is selected.
					hook := filepath.Join(root, ".git", "hooks", "pre-push")
					if err := testsupport.WriteExecutable(hook, []byte("#!/bin/sh\n# aiwf:pre-push\nexit 99\n")); err != nil {
						t.Fatal(err)
					}
					output := run("aiwf", verb)
					assertHostSelectionReport(t, output, config.HostSelection{Hosts: selected, Source: source})
					for _, path := range retained {
						assertRetainedLedgerPath(t, output, path)
					}
					if diff := cmp.Diff(before, retainedPathSnapshot(t, root, retained, false)); diff != "" {
						t.Fatalf("%s rewrote retained artifacts (-want +got):\n%s", verb, diff)
					}
					if diff := cmp.Diff(homeBefore, retainedPathSnapshot(t, home, []string{".claude"}, false)); diff != "" {
						t.Fatalf("%s rewrote user artifacts (-want +got):\n%s", verb, diff)
					}
					if diff := cmp.Diff(content, string(readFile(t, filepath.Join(root, "aiwf.yaml")))); diff != "" {
						t.Fatalf("%s rewrote stored decisions (-want +got):\n%s", verb, diff)
					}
					if strings.Contains(string(readFile(t, hook)), "exit 99") {
						t.Fatal("core Git hook was not refreshed")
					}
				}
			})
		}
	}
}

// Compare bytes, modes and fixed modification times, including health files.
// Fixing times makes even an identical-byte rewrite observable.
func retainedPathSnapshot(t *testing.T, root string, paths []string, fixTimes bool) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, relative := range paths {
		base := filepath.Join(root, relative)
		if _, err := os.Lstat(base); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				name, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				info, err := entry.Info()
				if err != nil {
					return err
				}
				result[name] = info.Mode().String()
				return nil
			}
			if fixTimes {
				fixed := time.Unix(946684800, 0)
				if err := os.Chtimes(path, fixed, fixed); err != nil {
					return err
				}
			}
			info, err := os.Lstat(path)
			if err != nil {
				return err
			}
			name, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			result[name] = fmt.Sprintf("%s %s %s", info.Mode(), info.ModTime().UTC().Format(time.RFC3339Nano), readFile(t, path))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func assertRetainedLedgerPath(t *testing.T, output, path string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "preserved" && fields[1] == path {
			return
		}
	}
	t.Errorf("retained path %s missing from ledger:\n%s", path, output)
}

func TestHostLifecycle_UnselectedClaudeSecondaryPathsStayUntouched(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir(), "-ldflags=-X github.com/23min/aiwf/internal/version.Stamp=v0.36.0")
	for _, hosts := range []string{"[]", "[codex]"} {
		for _, existing := range []bool{false, true} {
			for _, enabled := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/existing=%v/enabled=%v", hosts, existing, enabled), func(t *testing.T) {
					t.Parallel()
					root, home := t.TempDir(), t.TempDir()
					path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode})
					run := func(args ...string) string { return hostLifecycleRun(t, root, home, path, args...) }
					run("git", "init", "-q", "-b", "main")
					content := fmt.Sprintf("hosts: %s\nhooks:\n  worktree-rituals-check.sh:\n    enabled: %v\n", hosts, enabled)
					claudeBaselineWrite(t, root, "aiwf.yaml", content)
					if existing {
						for _, dir := range []string{root, home} {
							for _, relative := range []string{".claude/settings.json", ".claude/settings.local.json", ".claude/health.aiwf.json", ".claude/aiwf-guidance.md", "CLAUDE.md", ".claude/hooks/worktree-rituals-check.sh"} {
								// Invalid settings ensure an accidental hook-sync path fails loudly.
								claudeBaselineWrite(t, dir, relative, "user data: leave untouched\n")
							}
							claudeBaselineWrite(t, dir, ".claude/statusline.sh", "#!/bin/sh\n# aiwf-statusline version: v0.35.0\nexit 0\n")
						}
					}
					paths := []string{".claude", "CLAUDE.md"}
					before := retainedPathSnapshot(t, root, paths, true)
					homeBefore := retainedPathSnapshot(t, home, paths, true)
					for _, verb := range []string{"init", "update"} {
						run("aiwf", verb)
						if diff := cmp.Diff(before, retainedPathSnapshot(t, root, paths, false)); diff != "" {
							t.Fatalf("%s changed Claude files (-want +got):\n%s", verb, diff)
						}
						if diff := cmp.Diff(homeBefore, retainedPathSnapshot(t, home, paths, false)); diff != "" {
							t.Fatalf("%s changed home (-want +got):\n%s", verb, diff)
						}
						if diff := cmp.Diff(content, string(readFile(t, filepath.Join(root, "aiwf.yaml")))); diff != "" {
							t.Fatalf("%s changed config (-want +got):\n%s", verb, diff)
						}
					}
					run("git", "add", "-A")
					run("git", "commit", "-q", "-m", "chore: seed retained host files")
					checkout := filepath.Join(t.TempDir(), "checkout")
					args := []string{"aiwf", "worktree", "add", "feature/retained", checkout}
					if enabled {
						args = append(args, "--format=json")
					}
					output := run(args...)
					if enabled {
						var envelope struct {
							Result struct {
								Steps []struct {
									What   string `json:"what"`
									Action string `json:"action"`
								} `json:"steps"`
							} `json:"result"`
						}
						if err := json.Unmarshal([]byte(output), &envelope); err != nil {
							t.Fatal(err)
						}
						if len(envelope.Result.Steps) == 0 {
							t.Fatal("worktree JSON omitted artifact ledger")
						}
						if existing {
							for _, path := range paths {
								found := false
								for _, step := range envelope.Result.Steps {
									if step.What == path && step.Action == "preserved" {
										found = true
									}
								}
								if !found {
									t.Errorf("worktree JSON omitted retained path %s", path)
								}
							}
						}
					} else if existing {
						for _, path := range paths {
							assertRetainedLedgerPath(t, output, path)
						}
					}
					// Git assigns new checkout times; normalize only for this cross-checkout comparison.
					if diff := cmp.Diff(before, retainedPathSnapshot(t, checkout, paths, true)); diff != "" {
						t.Fatalf("worktree changed tracked Claude files (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(homeBefore, retainedPathSnapshot(t, home, paths, false)); diff != "" {
						t.Fatalf("worktree changed home (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(content, string(readFile(t, filepath.Join(checkout, "aiwf.yaml")))); diff != "" {
						t.Fatalf("worktree changed config (-want +got):\n%s", diff)
					}
				})
			}
		}
	}
}

func TestHostLifecycle_ExplicitClaudeRequestsPreserveUnselectedState(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, hosts := range []string{"[]", "[codex]"} {
		t.Run(hosts, func(t *testing.T) {
			t.Parallel()
			root, home := t.TempDir(), t.TempDir()
			path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode})
			hostLifecycleRun(t, root, home, path, "git", "init", "-q")
			claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: "+hosts+"\nhooks:\n  worktree-rituals-check.sh:\n    enabled: true\n")
			claudeBaselineWrite(t, root, ".claude/settings.json", "user data\n")
			before := retainedPathSnapshot(t, root, []string{"aiwf.yaml", ".claude"}, true)
			for _, args := range [][]string{
				{"init", "--statusline"},
				{"init", "--enable-hook", "worktree-rituals-check.sh"},
				{"update", "--statusline"},
				{"update", "--remove"},
				{"update", "--enable-hook", "worktree-rituals-check.sh"},
			} {
				hostLifecycleRunExit(t, root, home, path, 2, append([]string{"aiwf"}, args...)...)
				if diff := cmp.Diff(before, retainedPathSnapshot(t, root, []string{"aiwf.yaml", ".claude"}, false)); diff != "" {
					t.Fatalf("%v changed unselected state (-want +got):\n%s", args, diff)
				}
				entries, err := os.ReadDir(root)
				if err != nil {
					t.Fatal(err)
				}
				var names []string
				for _, entry := range entries {
					names = append(names, entry.Name())
				}
				if diff := cmp.Diff([]string{".claude", ".git", "aiwf.yaml"}, names); diff != "" {
					t.Fatalf("%v wrote artifacts (-want +got):\n%s", args, diff)
				}
				if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 {
					t.Fatalf("%v changed home: %v %v", args, entries, err)
				}
			}
		})
	}
}
