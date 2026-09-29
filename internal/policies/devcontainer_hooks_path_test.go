package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDevcontainerHooksPathRepair runs .devcontainer/project/hooks-path.sh
// in throwaway repositories. It unsets core.hooksPath only when the value is
// a missing absolute path to a .git/hooks directory — the host spelling of
// the repository's default, which leaves every hook dead in the container —
// and leaves every other value as it was.
func TestDevcontainerHooksPathRepair(t *testing.T) {
	t.Parallel()
	script := filepath.Join(repoRoot(t), ".devcontainer", "project", "hooks-path.sh")
	existing := filepath.Join(t.TempDir(), "elsewhere", ".git", "hooks")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		value string // "" leaves core.hooksPath unset
		want  string // "" means unset afterwards
	}{
		{"host-default-missing-is-unset", "/Users/nobody/repo/.git/hooks", ""},
		{"relative-kept", ".githooks", ".githooks"},
		{"existing-absolute-kept", existing, existing},
		{"missing-custom-directory-kept", "/Users/nobody/my-hooks", "/Users/nobody/my-hooks"},
		{"unset-stays-unset", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := t.TempDir()
			git := func(args ...string) (string, error) {
				cmd := exec.Command("git", args...)
				cmd.Dir = repo
				out, err := cmd.CombinedOutput()
				return strings.TrimSpace(string(out)), err
			}
			if out, err := git("init", "-q"); err != nil {
				t.Fatalf("git init: %v: %s", err, out)
			}
			if tc.value != "" {
				if out, err := git("config", "core.hooksPath", tc.value); err != nil {
					t.Fatalf("git config: %v: %s", err, out)
				}
			}
			cmd := exec.Command("bash", script)
			cmd.Dir = repo
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("hooks-path.sh: %v: %s", err, out)
			}
			got, _ := git("config", "--get", "core.hooksPath")
			if got != tc.want {
				t.Errorf("core.hooksPath = %q, want %q", got, tc.want)
			}
		})
	}
}
