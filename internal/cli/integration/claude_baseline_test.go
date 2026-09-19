package integration

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
)

// TestClaudeArtifacts_MatchBaseline compares real CLI output with a frozen
// inventory captured before host rendering changes. The expected bytes never
// come from the renderer or its embedded sources during the test.
func TestClaudeArtifacts_MatchBaseline(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	bin := testutil.BuildBinary(t, t.TempDir(), "-ldflags=-X github.com/23min/aiwf/internal/version.Stamp=v0.36.0")
	for _, tc := range []struct {
		name   string
		config string
		flags  []string
	}{
		{name: "undecided"},
		{name: "enabled", flags: []string{"--enable-hook", "worktree-rituals-check.sh"}},
		{name: "declined", config: "hooks:\n  worktree-rituals-check.sh:\n    enabled: false\n"},
		{name: "statusline-unwired", flags: []string{"--statusline"}},
		{name: "statusline-wired", flags: []string{"--statusline", "--wire-settings"}},
		{
			name: "configured",
			config: "guidance:\n  wire_claudemd: false\nagents:\n  builder:\n    model: sonnet\n    effort: high\n" +
				"hooks:\n  worktree-rituals-check.sh:\n    enabled: true\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, home := t.TempDir(), t.TempDir()
			git := func(args ...string) {
				t.Helper()
				claudeBaselineCommand(t, root, home, bin, "git", args...)
			}
			git("init", "-q", "-b", "main")
			git("config", "user.name", "aiwf-test")
			git("config", "user.email", "test@example.com")
			claudeBaselineWrite(t, home, ".claude/settings.json", "{\"permissions\":{\"allow\":[\"Read\"]}}\n")
			for path, content := range map[string]string{
				"CLAUDE.md":                             "User instructions before.\n\nUser instructions after.\n",
				".gitignore":                            "user-cache/\n",
				".claude/skills/aiwf-personal/SKILL.md": "Personal skill.\n",
				".claude/agents/personal.md":            "Personal agent.\n",
				".claude/templates/personal.md":         "Personal template.\n",
				".claude/settings.json":                 "{\"permissions\":{\"allow\":[\"Read\"]}}\n",
			} {
				claudeBaselineWrite(t, root, path, content)
			}
			if tc.config != "" {
				claudeBaselineWrite(t, root, "aiwf.yaml", tc.config)
			}
			args := append([]string{"init", "--no-prompt"}, tc.flags...)
			claudeBaselineCommand(t, root, home, bin, bin, args...)
			want := string(readFile(t, filepath.Join("testdata", "claude-baseline", tc.name+".golden")))
			assertSnapshot := func(checkout, stage string) {
				t.Helper()
				if diff := cmp.Diff(want, claudeBaselineSnapshot(t, checkout, home, filepath.Join(root, ".git", "hooks"))); diff != "" {
					t.Errorf("%s artifacts differ from the Claude baseline (-want +got):\n%s", stage, diff)
				}
			}
			assertSnapshot(root, "init")
			claudeBaselineCommand(t, root, home, bin, bin, "update")
			assertSnapshot(root, "update")

			// Track user files explicitly so Git transports them into the new
			// worktree; all owned adapters must be materialized by aiwf there.
			git("add", "-A")
			git("add", "-f", ".claude/skills/aiwf-personal/SKILL.md", ".claude/agents/personal.md", ".claude/templates/personal.md", ".claude/settings.json")
			git("commit", "-q", "-m", "chore: seed consumer configuration")
			worktree := filepath.Join(t.TempDir(), "checkout")
			claudeBaselineCommand(t, root, home, bin, bin, "worktree", "add", "feature/baseline", worktree)
			assertSnapshot(worktree, "worktree add")
		})
	}
}

func claudeBaselineWrite(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func claudeBaselineCommand(t *testing.T, root, home, bin, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+home,
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"PATH="+hostLifecyclePATH(t, bin, []config.Host{config.HostClaudeCode}),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func claudeBaselineSnapshot(t *testing.T, root, home, hooks string) string {
	t.Helper()
	var lines []string
	add := func(path, relative string) {
		t.Helper()
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("baseline artifact %s is not a regular file: %v", relative, info.Mode())
		}
		lines = append(lines, fmt.Sprintf("%04o %x %s", info.Mode().Perm(), sha256.Sum256(readFile(t, path)), relative))
	}
	err := filepath.WalkDir(filepath.Join(root, ".claude"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		// Health is runtime metadata: its timestamp and host diagnostics are
		// environment-dependent. No generated adapter content is normalized.
		if relative != ".claude/health.aiwf.json" {
			add(path, relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(home, ".claude"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		add(filepath.Join(home, ".claude", entry.Name()), "user/.claude/"+entry.Name())
	}
	for _, relative := range []string{"CLAUDE.md", "aiwf.yaml", "aiwf.example.yaml", ".gitignore"} {
		add(filepath.Join(root, relative), relative)
	}
	for _, name := range []string{"pre-push", "pre-commit", "commit-msg", "post-commit"} {
		add(filepath.Join(hooks, name), "git-hooks/"+name)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}
