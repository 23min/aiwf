package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDevcontainerHooksPathRepair runs .devcontainer/project/hooks-path.sh
// in throwaway repositories. It unsets core.hooksPath only when the
// repository's own config holds the host's path to this checkout's hooks
// directory (AIWF_HOST_CHECKOUT/.git/hooks) and that path is missing — the
// value that leaves every hook dead in the container and whose removal
// changes nothing on the host. Every other value, scope and input is left as
// it was, and the script never fails.
func TestDevcontainerHooksPathRepair(t *testing.T) {
	t.Parallel()
	script := filepath.Join(repoRoot(t), ".devcontainer", "project", "hooks-path.sh")
	const host = "/Users/nobody/Projects/aiwf"
	existing := filepath.Join(t.TempDir(), "present")
	if err := os.MkdirAll(filepath.Join(existing, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name        string
		hostPath    string   // AIWF_HOST_CHECKOUT; "" leaves it unset
		local       []string // core.hooksPath values in the repository's config
		global      string   // core.hooksPath in the global config; "" for none
		wantLocal   string   // local value afterwards; "" means unset
		wantGlobal  string
		wantMessage bool
	}{
		{name: "host-checkout-hooks-missing-is-unset", hostPath: host, local: []string{host + "/.git/hooks"}, wantMessage: true},
		{name: "trailing-slash-is-unset", hostPath: host, local: []string{host + "/.git/hooks/"}, wantMessage: true},
		{name: "set-twice-is-fully-unset", hostPath: host, local: []string{host + "/.git/hooks", host + "/.git/hooks"}, wantMessage: true},
		{name: "another-clones-hooks-kept", hostPath: host, local: []string{"/Users/nobody/Projects/other/.git/hooks"}, wantLocal: "/Users/nobody/Projects/other/.git/hooks"},
		{name: "existing-path-kept", hostPath: existing, local: []string{existing + "/.git/hooks"}, wantLocal: existing + "/.git/hooks"},
		{name: "relative-kept", hostPath: host, local: []string{".githooks"}, wantLocal: ".githooks"},
		{name: "no-host-path-leaves-it", local: []string{host + "/.git/hooks"}, wantLocal: host + "/.git/hooks"},
		{name: "global-scope-left-alone", hostPath: host, global: host + "/.git/hooks", wantGlobal: host + "/.git/hooks"},
		{name: "unset-stays-unset", hostPath: host},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := t.TempDir()
			globalConfig := filepath.Join(t.TempDir(), "gitconfig")
			if err := os.WriteFile(globalConfig, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			env := append(os.Environ(), "GIT_CONFIG_GLOBAL="+globalConfig, "GIT_CONFIG_NOSYSTEM=1")
			run := func(name string, extraEnv []string, args ...string) (string, error) {
				cmd := exec.Command(name, args...)
				cmd.Dir = repo
				cmd.Env = append(append([]string{}, env...), extraEnv...)
				out, err := cmd.CombinedOutput()
				return strings.TrimSpace(string(out)), err
			}
			if out, err := run("git", nil, "init", "-q"); err != nil {
				t.Fatalf("git init: %v: %s", err, out)
			}
			for _, v := range tc.local {
				if out, err := run("git", nil, "config", "--add", "core.hooksPath", v); err != nil {
					t.Fatalf("git config: %v: %s", err, out)
				}
			}
			if tc.global != "" {
				if out, err := run("git", nil, "config", "--global", "core.hooksPath", tc.global); err != nil {
					t.Fatalf("git config --global: %v: %s", err, out)
				}
			}
			var scriptEnv []string
			if tc.hostPath != "" {
				scriptEnv = []string{"AIWF_HOST_CHECKOUT=" + tc.hostPath}
			}
			out, err := run("bash", scriptEnv, script)
			if err != nil {
				t.Fatalf("hooks-path.sh failed: %v: %s", err, out)
			}
			if got := strings.Contains(out, "Unset core.hooksPath"); got != tc.wantMessage {
				t.Errorf("reported an unset = %v, want %v; output: %q", got, tc.wantMessage, out)
			}
			if got, _ := run("git", nil, "config", "--local", "--get", "core.hooksPath"); got != tc.wantLocal {
				t.Errorf("local core.hooksPath = %q, want %q", got, tc.wantLocal)
			}
			if got, _ := run("git", nil, "config", "--global", "--get", "core.hooksPath"); got != tc.wantGlobal {
				t.Errorf("global core.hooksPath = %q, want %q", got, tc.wantGlobal)
			}
		})
	}
}
