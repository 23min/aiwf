package policies_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDevcontainerGuidanceMount runs the real initialize.sh against a fixture
// HOME. It pins the personal-guidance rules: in every case the container's
// ~/.guidance mount is read-write and resolves to the host's ~/.guidance; a
// checkout's build runs, and a missing checkout or a failed build never blocks
// a start; the container-only Codex home's AGENTS.md is linked to the guidance
// build only once the build has produced ~/.guidance/AGENTS.md, replacing any
// link but never anything else.
// Each case names the messages it expects, and the others must be absent.
func TestDevcontainerGuidanceMount(t *testing.T) {
	t.Parallel()
	root := repoRootForHook(t)
	cases := []struct {
		name     string
		setup    func(t *testing.T, home string)
		messages []string
		check    func(t *testing.T, home string)
	}{
		{
			name:     "no checkout creates no Codex link",
			messages: []string{"setup hint"},
			check: func(t *testing.T, home string) {
				t.Helper()
				if _, err := os.Lstat(codexRules(home)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("Codex AGENTS.md created before any build: %v", err)
				}
			},
		},
		{
			name: "checkout builds and Codex reads the build",
			setup: func(t *testing.T, home string) {
				t.Helper()
				guidanceBuild(t, home, `printf 'built\n' > "$HOME/.guidance/AGENTS.md"`)
			},
			check: func(t *testing.T, home string) {
				t.Helper()
				codexLinksToGuidance(t, home)
				if got, err := os.ReadFile(codexRules(home)); err != nil || string(got) != "built\n" {
					t.Errorf("Codex does not read the built AGENTS.md: %q, %v", got, err)
				}
			},
		},
		{
			// The target exists, so only the guard's symlink test replaces it:
			// the state every start after a successful build finds.
			name: "live link replaced",
			setup: func(t *testing.T, home string) {
				t.Helper()
				guidanceBuild(t, home, `printf 'built\n' > "$HOME/.guidance/AGENTS.md"`)
				earlier := filepath.Join(home, "earlier-AGENTS.md")
				if err := os.WriteFile(earlier, []byte("earlier\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				codexLink(t, home, earlier)
			},
			check: func(t *testing.T, home string) {
				t.Helper()
				codexLinksToGuidance(t, home)
			},
		},
		{
			// A link whose target is missing on the host, as a container-side
			// link into ~/.agents is when initialize.sh runs.
			name: "dangling link replaced",
			setup: func(t *testing.T, home string) {
				t.Helper()
				guidanceBuild(t, home, `printf 'built\n' > "$HOME/.guidance/AGENTS.md"`)
				codexLink(t, home, filepath.Join(home, "missing", "AGENTS.md"))
			},
			check: func(t *testing.T, home string) {
				t.Helper()
				codexLinksToGuidance(t, home)
			},
		},
		{
			name: "failed build keeps the previous output in use",
			setup: func(t *testing.T, home string) {
				t.Helper()
				guidanceBuild(t, home, "exit 1")
				if err := os.WriteFile(filepath.Join(home, ".guidance", "AGENTS.md"), []byte("previous\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			messages: []string{"build failed"},
			check: func(t *testing.T, home string) {
				t.Helper()
				codexLinksToGuidance(t, home)
				if got, err := os.ReadFile(codexRules(home)); err != nil || string(got) != "previous\n" {
					t.Errorf("Codex does not read the previous build: %q, %v", got, err)
				}
			},
		},
		{
			name: "failed first build keeps the existing Codex link",
			setup: func(t *testing.T, home string) {
				t.Helper()
				guidanceBuild(t, home, "exit 1")
				codexLink(t, home, filepath.Join(home, "earlier", "AGENTS.md"))
			},
			messages: []string{"build failed"},
			check: func(t *testing.T, home string) {
				t.Helper()
				if got, err := os.Readlink(codexRules(home)); err != nil || got != filepath.Join(home, "earlier", "AGENTS.md") {
					t.Errorf("existing Codex link changed although no build output exists: %q, %v", got, err)
				}
			},
		},
		{
			name: "regular file kept",
			setup: func(t *testing.T, home string) {
				t.Helper()
				guidanceBuild(t, home, `printf 'built\n' > "$HOME/.guidance/AGENTS.md"`)
				mustMkdir(t, filepath.Dir(codexRules(home)))
				if err := os.WriteFile(codexRules(home), []byte("mine\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			messages: []string{"move aside"},
			check: func(t *testing.T, home string) {
				t.Helper()
				info, err := os.Lstat(codexRules(home))
				if err != nil {
					t.Fatal(err)
				}
				if !info.Mode().IsRegular() {
					t.Fatalf("regular file replaced by %v", info.Mode())
				}
				if got, err := os.ReadFile(codexRules(home)); err != nil || string(got) != "mine\n" {
					t.Errorf("regular file changed: %q, %v", got, err)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			mustMkdir(t, bin)
			mustMkdir(t, filepath.Join(dir, "mounts"))
			devcontainerExecutable(t, filepath.Join(bin, "ln"), devcontainerLnStub)
			if tc.setup != nil {
				tc.setup(t, dir)
			}
			cmd := exec.CommandContext(t.Context(), "bash", filepath.Join(root, ".devcontainer", "initialize.sh"))
			cmd.Dir = dir
			cmd.Env = []string{"HOME=" + dir, "PATH=" + bin + ":/usr/bin:/bin", "FIXTURE_DIR=" + dir}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("initialize host: %v\n%s", err, out)
			}
			devcontainerMountResolves(t, root, dir, "/home/vscode/.guidance", filepath.Join(dir, ".guidance"))
			guidanceMessages(t, dir, string(out), tc.messages)
			tc.check(t, dir)
		})
	}
}

// guidanceMessages checks that initialize.sh printed exactly the named
// messages among those the guidance step can print.
func guidanceMessages(t *testing.T, home, out string, want []string) {
	t.Helper()
	guidance := filepath.Join(home, ".guidance")
	all := map[string]string{
		"setup hint":   "git clone https://github.com/23min/guidance.git " + guidance + " && " + filepath.Join(guidance, "install"),
		"build failed": filepath.Join(guidance, "build") + " failed",
		"move aside":   "move it aside",
	}
	expected := make(map[string]bool, len(want))
	for _, name := range want {
		if _, ok := all[name]; !ok {
			t.Fatalf("unknown message %q", name)
		}
		expected[name] = true
	}
	for name, text := range all {
		if got := strings.Contains(out, text); got != expected[name] {
			t.Errorf("message %q (%q) printed = %v, want %v:\n%s", name, text, got, expected[name], out)
		}
	}
}

func codexRules(home string) string {
	return filepath.Join(home, ".codex-linux", "AGENTS.md")
}

func codexLink(t *testing.T, home, target string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(codexRules(home)))
	if err := os.Symlink(target, codexRules(home)); err != nil {
		t.Fatal(err)
	}
}

func codexLinksToGuidance(t *testing.T, home string) {
	t.Helper()
	if got, err := os.Readlink(codexRules(home)); err != nil || got != "../.guidance/AGENTS.md" {
		t.Errorf("Codex AGENTS.md link = %q, %v; want ../.guidance/AGENTS.md", got, err)
	}
}

func guidanceBuild(t *testing.T, home, body string) {
	t.Helper()
	mustMkdir(t, filepath.Join(home, ".guidance"))
	devcontainerExecutable(t, filepath.Join(home, ".guidance", "build"), "#!/bin/sh\n"+body+"\n")
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
}
