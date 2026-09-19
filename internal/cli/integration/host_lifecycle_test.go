package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/testsupport"
)

func TestHostLifecycle_InitUpdateAndWorktreeUseResolvedHosts(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, hosts := range [][]config.Host{{}, {config.HostClaudeCode}, {config.HostCodex}, {config.HostClaudeCode, config.HostCodex}} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/explicit=%v", hosts, explicit), func(t *testing.T) {
				t.Parallel()
				installed := hosts
				source := config.HostsDetected
				content := "hooks:\n  worktree-rituals-check.sh:\n    enabled: true\n"
				if explicit {
					source = config.HostsConfigured
					// Explicit lists must work with no installed host commands.
					installed = nil
					names := make([]string, len(hosts))
					for i, host := range hosts {
						names[i] = string(host)
					}
					content += "hosts: [" + strings.Join(names, ", ") + "]\n"
				}
				root, home := t.TempDir(), t.TempDir()
				path := hostLifecyclePATH(t, binary, installed)
				run := func(dir string, args ...string) string { return hostLifecycleRun(t, dir, home, path, args...) }
				run(root, "git", "init", "-q", "-b", "main")
				claudeBaselineWrite(t, root, "aiwf.yaml", content)
				want := config.HostSelection{Hosts: hosts, Source: source}
				for _, verb := range []string{"init", "update"} {
					if verb == "update" {
						for _, host := range hosts {
							target := skills.CodexTarget()
							if host == config.HostClaudeCode {
								target = skills.ClaudeTarget
							}
							for _, relative := range []string{target.SkillsDir + "/aiwf-check/SKILL.md", target.TemplatesDir + "/epic-spec.md"} {
								if err := os.Remove(filepath.Join(root, relative)); err != nil {
									t.Fatal(err)
								}
							}
						}
					}
					output := run(root, "aiwf", verb)
					assertHostLifecycleArtifacts(t, root, hosts)
					assertHostSelectionReport(t, output, want)
					if diff := cmp.Diff(content, string(readFile(t, filepath.Join(root, "aiwf.yaml")))); diff != "" {
						t.Fatalf("%s persisted detection or rewrote configuration (-want +got):\n%s", verb, diff)
					}
				}
				run(root, "git", "add", "-A")
				run(root, "git", "commit", "-q", "-m", "chore: seed host configuration")
				worktree := filepath.Join(t.TempDir(), "checkout")
				output := run(root, "aiwf", "worktree", "add", "feature/hosts", worktree, "--format=json")
				var envelope struct {
					Result struct {
						Path          string               `json:"path"`
						HostSelection config.HostSelection `json:"host_selection"`
					} `json:"result"`
				}
				if err := json.Unmarshal([]byte(output), &envelope); err != nil {
					t.Fatal(err)
				}
				if envelope.Result.Path != worktree {
					t.Fatalf("worktree path = %q", envelope.Result.Path)
				}
				if diff := cmp.Diff(want, envelope.Result.HostSelection); diff != "" {
					t.Fatalf("worktree selection (-want +got):\n%s", diff)
				}
				assertHostLifecycleArtifacts(t, worktree, hosts)
			})
		}
	}
}

// Child commands receive only these tools and fake hosts; the developer's
// installed assistants and user-level settings never enter the fixture.
func hostLifecyclePATH(t *testing.T, binary string, hosts []config.Host) string {
	t.Helper()
	path := t.TempDir()
	for _, tool := range []string{"git", "sh", "dirname"} {
		resolved, err := exec.LookPath(tool)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(resolved, filepath.Join(path, tool)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(binary, filepath.Join(path, "aiwf")); err != nil {
		t.Fatal(err)
	}
	for _, host := range hosts {
		command := "codex"
		if host == config.HostClaudeCode {
			command = "claude"
		}
		if err := testsupport.WriteExecutable(filepath.Join(path, command), []byte("#!/bin/sh\nexit 99\n")); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func hostLifecycleRun(t *testing.T, root, home, path string, args ...string) string {
	t.Helper()
	return hostLifecycleRunExit(t, root, home, path, 0, args...)
}

func hostLifecycleRunExit(t *testing.T, root, home, path string, wantCode int, args ...string) string {
	t.Helper()
	cmd := exec.Command(filepath.Join(path, args[0]), args[1:]...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+path, "HOME="+home, "XDG_CONFIG_HOME="+home, "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	output, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("launching %v: %v", args, err)
		}
		code = exitErr.ExitCode()
	}
	if code != wantCode {
		t.Fatalf("%v: exit %d, want %d\n%s", args, code, wantCode, output)
	}
	return string(output)
}

func assertHostSelectionReport(t *testing.T, output string, selection config.HostSelection) {
	t.Helper()
	// The selection row is a CLI data record: source plus ordered host IDs.
	names := make([]string, len(selection.Hosts))
	for i, host := range selection.Hosts {
		names[i] = string(host)
	}
	value := strings.Join(names, ", ")
	if value == "" {
		value = "none"
	}
	want := fmt.Sprintf("Hosts (%s): %s", selection.Source, value)
	if !slices.Contains(strings.Split(output, "\n"), want) {
		t.Fatalf("selection record %q missing:\n%s", want, output)
	}
}

func assertHostLifecycleArtifacts(t *testing.T, root string, hosts []config.Host) {
	t.Helper()
	verbSkills, err := skills.List()
	if err != nil {
		t.Fatal(err)
	}
	rituals, err := skills.ListRituals()
	if err != nil {
		t.Fatal(err)
	}
	templates, err := skills.ListRitualTemplates()
	if err != nil {
		t.Fatal(err)
	}
	for _, host := range []config.Host{config.HostClaudeCode, config.HostCodex} {
		target, dir, guidance := skills.CodexTarget(), ".agents", "AGENTS.md"
		if host == config.HostClaudeCode {
			target, dir, guidance = skills.ClaudeTarget, ".claude", "CLAUDE.md"
		}
		if !slices.Contains(hosts, host) {
			for _, relative := range []string{dir, guidance} {
				if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
					t.Errorf("unselected host artifact %s: %v", relative, err)
				}
			}
			continue
		}
		paths := []string{guidance, target.SkillsDir + "/" + skills.ManifestFile, target.SkillsDir + "/" + skills.ProvenanceReadme, target.TemplatesDir + "/" + skills.ManifestFile}
		for _, skill := range verbSkills {
			paths = append(paths, target.SkillsDir+"/"+skill.Name+"/SKILL.md")
		}
		for _, ritual := range rituals {
			paths = append(paths, target.SkillsDir+"/"+ritual.Name+"/SKILL.md")
		}
		for _, template := range templates {
			paths = append(paths, target.TemplatesDir+"/"+template.Name)
		}
		if host == config.HostClaudeCode {
			agents, listErr := skills.ListRitualAgents()
			if listErr != nil {
				t.Fatal(listErr)
			}
			for _, agent := range agents {
				paths = append(paths, target.AgentsDir+"/"+agent.Name)
			}
			paths = append(paths, skills.GuidanceFile)
		}
		for _, relative := range paths {
			info, statErr := os.Stat(filepath.Join(root, relative))
			if statErr != nil || !info.Mode().IsRegular() || info.Size() == 0 {
				t.Errorf("selected host artifact %s missing or empty: %v", relative, statErr)
			}
		}
	}
	for _, relative := range []string{"aiwf.yaml", "aiwf.example.yaml", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Errorf("core artifact %s: %v", relative, err)
		}
	}
	// Generated host files stay derivable: Git must ignore the managed skill
	// while keeping root guidance eligible for version control.
	for _, host := range hosts {
		dir := ".agents/skills"
		if host == config.HostClaudeCode {
			dir = ".claude/skills"
		}
		cmd := exec.Command("git", "-C", root, "check-ignore", dir+"/aiwf-check/SKILL.md")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("managed skill not ignored: %v\n%s", err, output)
		}
	}
}

func TestHostLifecycle_AddingCodexPreservesClaudeArtifacts(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	root, home := t.TempDir(), t.TempDir()
	path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode})
	run := func(args ...string) string { return hostLifecycleRun(t, root, home, path, args...) }
	run("git", "init", "-q", "-b", "main")
	run("aiwf", "init", "--actor", "human/test", "--no-prompt", "--enable-hook", "worktree-rituals-check.sh")
	before := hostLifecycleClaudeSnapshot(t, root)
	beforeConfig := string(readFile(t, filepath.Join(root, "aiwf.yaml")))
	if err := testsupport.WriteExecutable(filepath.Join(path, "codex"), []byte("#!/bin/sh\nexit 99\n")); err != nil {
		t.Fatal(err)
	}
	output := run("aiwf", "update")
	want := config.HostSelection{Hosts: []config.Host{config.HostClaudeCode, config.HostCodex}, Source: config.HostsDetected}
	assertHostSelectionReport(t, output, want)
	assertHostLifecycleArtifacts(t, root, want.Hosts)
	if diff := cmp.Diff(before, hostLifecycleClaudeSnapshot(t, root)); diff != "" {
		t.Fatalf("adding Codex changed Claude artifacts (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(beforeConfig, string(readFile(t, filepath.Join(root, "aiwf.yaml")))); diff != "" {
		t.Fatalf("adding Codex persisted machine state (-want +got):\n%s", diff)
	}
}

func hostLifecycleClaudeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{"CLAUDE.md": string(readFile(t, filepath.Join(root, "CLAUDE.md")))}
	err := filepath.WalkDir(filepath.Join(root, ".claude"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() == "health.aiwf.json" {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		files[relative] = fmt.Sprintf("%04o %s", info.Mode().Perm(), readFile(t, path))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestHostLifecycle_ClaudeFlagsRequireClaudeSelection(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, args := range [][]string{
		{"init", "--statusline"},
		{"init", "--enable-hook", "worktree-rituals-check.sh"},
		{"update", "--statusline"},
		{"update", "--remove"},
		{"update", "--enable-hook", "worktree-rituals-check.sh"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			root, home := t.TempDir(), t.TempDir()
			path := hostLifecyclePATH(t, binary, []config.Host{config.HostClaudeCode})
			hostLifecycleRun(t, root, home, path, "git", "init", "-q")
			claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: []\n")
			hostLifecycleRunExit(t, root, home, path, 2, append([]string{"aiwf"}, args...)...)
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, entry := range entries {
				names = append(names, entry.Name())
			}
			if diff := cmp.Diff([]string{".git", "aiwf.yaml"}, names); diff != "" {
				t.Errorf("invalid host request wrote artifacts (-want +got):\n%s", diff)
			}
			if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
				t.Fatalf("incompatible flags created Claude artifacts: %v", err)
			}
		})
	}
}

func TestHostLifecycle_WorktreeArtifactFailuresRollBackCreation(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	for _, failure := range []string{"hook settings", "skill ownership"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			root, home := t.TempDir(), t.TempDir()
			path := hostLifecyclePATH(t, binary, nil)
			run := func(args ...string) string { return hostLifecycleRun(t, root, home, path, args...) }
			run("git", "init", "-q", "-b", "main")
			claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: [claude-code]\nhooks:\n  worktree-rituals-check.sh:\n    enabled: true\n")
			run("aiwf", "init", "--no-prompt")
			if failure == "hook settings" {
				claudeBaselineWrite(t, root, ".claude/settings.json", "invalid JSON\n")
			} else {
				claudeBaselineWrite(t, root, ".claude/skills/.aiwf-owned", "../escape\n")
				run("git", "add", "-f", ".claude/skills/.aiwf-owned")
			}
			run("git", "add", "-A")
			run("git", "commit", "-q", "-m", "chore: seed invalid host settings")
			worktree := filepath.Join(t.TempDir(), "checkout")
			hostLifecycleRunExit(t, root, home, path, 3, "aiwf", "worktree", "add", "feature/bad-settings", worktree)
			if _, err := os.Stat(worktree); !os.IsNotExist(err) {
				t.Fatalf("failed worktree remains: %v", err)
			}
			if branch := run("git", "branch", "--list", "feature/bad-settings"); branch != "" {
				t.Fatalf("failed branch remains: %q", branch)
			}
		})
	}
}

func TestHostLifecycle_SelfCheckWorksWithoutAssistantCommands(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	root, home := t.TempDir(), t.TempDir()
	path := hostLifecyclePATH(t, binary, nil)
	claudeBaselineWrite(t, home, ".gitconfig", "[user]\n  email = test@example.com\n  name = aiwf-test\n")
	cmd := exec.Command(filepath.Join(path, "aiwf"), "doctor", "--self-check")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+path, "HOME="+home, "XDG_CONFIG_HOME="+home, "GIT_CONFIG_GLOBAL="+filepath.Join(home, ".gitconfig"), "GIT_CONFIG_NOSYSTEM=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("self-check without assistants: %v\n%s", err, output)
	}
}
