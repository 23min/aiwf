package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDevcontainerHooksPathRepair runs .devcontainer/project/hooks-path.sh,
// which runs on the host before every container start, in throwaway
// repositories standing in for the host checkout. It unsets core.hooksPath
// only when every value set is the repository's own hooks directory by its
// absolute path and none comes from the global config — the one case where
// removing it changes nothing on the host while restoring the hooks in the
// container. It exits 0 in every case, including when the unset fails.
func TestDevcontainerHooksPathRepair(t *testing.T) {
	t.Parallel()
	script := filepath.Join(repoRoot(t), ".devcontainer", "project", "hooks-path.sh")
	const own = "<own>" // replaced by the repository's absolute hooks directory
	cases := []struct {
		name       string
		local      []string // core.hooksPath values in the repository's config
		global     string   // core.hooksPath in the global config; "" for none
		gitRepo    bool
		lockConfig bool   // hold .git/config.lock so the unset fails
		wantLocal  string // local values afterwards, one per line; "" means unset
		wantUnset  bool   // the script reports that it unset the value
	}{
		{name: "own-hooks-is-unset", gitRepo: true, local: []string{own}, wantUnset: true},
		{name: "trailing-slash-is-unset", gitRepo: true, local: []string{own + "/"}, wantUnset: true},
		{name: "set-twice-is-fully-unset", gitRepo: true, local: []string{own, own}, wantUnset: true},
		{name: "global-setting-keeps-local", gitRepo: true, local: []string{own}, global: "/elsewhere/hooks", wantLocal: own},
		{name: "another-directory-kept", gitRepo: true, local: []string{"/Users/x/Projects/other/.git/hooks"}, wantLocal: "/Users/x/Projects/other/.git/hooks"},
		{name: "own-plus-another-kept", gitRepo: true, local: []string{own, "/elsewhere/hooks"}, wantLocal: own + "\n/elsewhere/hooks"},
		{name: "relative-kept", gitRepo: true, local: []string{".githooks"}, wantLocal: ".githooks"},
		{name: "unset-stays-unset", gitRepo: true},
		{name: "failed-unset-still-exits-0", gitRepo: true, local: []string{own}, lockConfig: true, wantLocal: own},
		{name: "not-a-repository", gitRepo: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir, runIn := hooksPathSandbox(t)
			run := func(name string, args ...string) (string, error) { return runIn(dir, name, args...) }
			ownHooks := filepath.Join(dir, ".git", "hooks")
			expand := func(v string) string { return strings.ReplaceAll(v, own, ownHooks) }
			if tc.gitRepo {
				if out, err := run("git", "init", "-q"); err != nil {
					t.Fatalf("git init: %v: %s", err, out)
				}
				for _, v := range tc.local {
					if out, err := run("git", "config", "--add", "core.hooksPath", expand(v)); err != nil {
						t.Fatalf("git config: %v: %s", err, out)
					}
				}
				if tc.global != "" {
					if out, err := run("git", "config", "--global", "core.hooksPath", tc.global); err != nil {
						t.Fatalf("git config --global: %v: %s", err, out)
					}
				}
				if tc.lockConfig {
					if err := os.WriteFile(filepath.Join(dir, ".git", "config.lock"), nil, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			out, err := run("bash", script)
			if err != nil {
				t.Fatalf("hooks-path.sh exited non-zero, which would stop the container start: %v: %s", err, out)
			}
			if got := strings.Contains(out, "initialize: unset core.hooksPath="); got != tc.wantUnset {
				t.Errorf("reported an unset = %v, want %v; output: %q", got, tc.wantUnset, out)
			}
			if !tc.gitRepo {
				return
			}
			got, _ := run("git", "config", "--local", "--get-all", "core.hooksPath")
			if want := expand(tc.wantLocal); got != want {
				t.Errorf("local core.hooksPath = %q, want %q", got, want)
			}
			if tc.global != "" {
				if got, _ := run("git", "config", "--global", "--get", "core.hooksPath"); got != tc.global {
					t.Errorf("global core.hooksPath = %q, want it left as %q", got, tc.global)
				}
			}
		})
	}
}

// TestDevcontainerHooksPathRepairRunsOnTheHost pins that the project's
// initialize hook, which the kit runs on the host before every start, runs
// the repair against its own checkout whatever directory it is started from.
func TestDevcontainerHooksPathRepairRunsOnTheHost(t *testing.T) {
	t.Parallel()
	repo, run := hooksPathSandbox(t)
	elsewhere := t.TempDir()
	project := filepath.Join(repo, ".devcontainer", "project")
	for _, name := range []string{"initialize.sh", "hooks-path.sh"} {
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".devcontainer", "project", name))
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(project, name), string(raw))
	}
	if out, err := run(repo, "git", "init", "-q"); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if out, err := run(repo, "git", "config", "core.hooksPath", filepath.Join(repo, ".git", "hooks")); err != nil {
		t.Fatalf("git config: %v: %s", err, out)
	}
	if out, err := run(elsewhere, "bash", filepath.Join(project, "initialize.sh")); err != nil {
		t.Fatalf("initialize.sh started outside the checkout exited non-zero, which would stop the container start: %v: %s", err, out)
	}
	if got, _ := run(repo, "git", "config", "--local", "--get", "core.hooksPath"); got != "" {
		t.Errorf("initialize.sh left core.hooksPath = %q; it must run the repair against its own checkout", got)
	}
}

// hooksPathSandbox returns an empty directory for a throwaway repository and
// a runner for commands in it or beside it. The directory is resolved through
// symlinks, as git reports a repository's own paths, so a value written with
// it compares equal on hosts whose temporary directory sits behind one (macOS
// reaches it through /var, a link to /private/var). The runner's git reads an
// empty global config, no system config, and no repository above the directory.
func hooksPathSandbox(t *testing.T) (dir string, run func(in, name string, args ...string) (string, error)) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(globalConfig, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "GIT_CONFIG_GLOBAL="+globalConfig, "GIT_CONFIG_NOSYSTEM=1", "GIT_CEILING_DIRECTORIES="+filepath.Dir(dir))
	run = func(in, name string, args ...string) (string, error) {
		cmd := exec.Command(name, args...)
		cmd.Dir = in
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	return dir, run
}
