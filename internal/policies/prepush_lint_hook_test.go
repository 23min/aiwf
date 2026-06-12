package policies_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// prepush_lint_hook_test.go — G-0179 chokepoint pins.
//
// Behavioral subprocess tests for scripts/git-hooks/pre-push (the
// lint boundary gate installed as .git/hooks/pre-push.local via
// `make install-hooks`), mirroring the agent_isolation_hook_test.go
// pattern: run the script against fabricated pre-push stdin inside
// a fixture git repo and assert the trigger/skip decision per
// branch. The AIWF_PREPUSH_LINT_CMD override stands in for the real
// golangci-lint run — `true` (lint green) / `false` (lint red) make
// the "did lint run?" decision observable without linting anything.
//
// Why these tests exist: G-0179's failure mode was lint debt
// accumulating invisibly on a long-lived unpushed branch because no
// mechanical gate ran the full linter before CI. The hook IS the
// mechanical fix; these tests are what keeps the hook's decision
// logic from silently rotting.

func prepushHookPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootForHook(t), "scripts", "git-hooks", "pre-push")
}

// gitInFixture runs a git command inside the fixture repo and
// returns trimmed stdout. Identity comes from the GIT_* env vars
// set once in setup_test.go's TestMain.
func gitInFixture(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v\nstderr: %s", args, err, errb.String())
	}
	return strings.TrimSpace(out.String())
}

// prepushFixture holds the shas of the fixture repo's commit chain:
// base (README.md) -> docs (docs.md) -> goChange (main.go) ->
// goMod (go.mod). Read-only after construction — do not mutate.
type prepushFixture struct {
	dir      string
	base     string
	docs     string
	goChange string
	goMod    string
}

func newPrepushFixture(t *testing.T) prepushFixture {
	t.Helper()
	dir := t.TempDir()
	gitInFixture(t, dir, "init", "-q", "-b", "main")

	commit := func(name, content, msg string) string {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
		gitInFixture(t, dir, "add", "-A")
		gitInFixture(t, dir, "commit", "-q", "-m", msg)
		return gitInFixture(t, dir, "rev-parse", "HEAD")
	}

	return prepushFixture{
		dir:      dir,
		base:     commit("README.md", "fixture\n", "base"),
		docs:     commit("docs.md", "prose only\n", "docs change"),
		goChange: commit("main.go", "package main\n", "go change"),
		goMod:    commit("go.mod", "module example.test/fixture\n", "go.mod change"),
	}
}

// runPrepushHook executes the hook script with the given pre-push
// stdin lines and lint-command override, returning the exit code
// and combined stderr.
func runPrepushHook(t *testing.T, repoDir, stdin, lintCmd string) (exitCode int, stderr string) {
	t.Helper()
	cmd := exec.Command(prepushHookPath(t))
	cmd.Dir = repoDir
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(), "AIWF_PREPUSH_LINT_CMD="+lintCmd)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), errb.String()
		}
		t.Fatalf("hook script run failed: %v", err)
	}
	return 0, errb.String()
}

func TestPrepushLintHook_Decision(t *testing.T) {
	t.Parallel()
	fx := newPrepushFixture(t) // shared read-only across subtests — do not mutate

	const zero = "0000000000000000000000000000000000000000"
	refLine := func(localSha, remoteSha string) string {
		return "refs/heads/main " + localSha + " refs/heads/main " + remoteSha + "\n"
	}

	tests := []struct {
		name    string
		stdin   string
		lintCmd string
		// setup, when set, builds a private fixture and returns
		// (dir, stdin), overriding the shared fixture and the
		// stdin field — for cases that must mutate repo state.
		setup      func(t *testing.T) (dir, stdin string)
		wantExit   int
		wantStderr []string
	}{
		{
			name:     "docs-only range skips lint",
			stdin:    refLine(fx.docs, fx.base),
			lintCmd:  "false", // would exit 1 if lint ran
			wantExit: 0,
		},
		{
			name:       "go change triggers lint, red blocks",
			stdin:      refLine(fx.goChange, fx.docs),
			lintCmd:    "false",
			wantExit:   1,
			wantStderr: []string{"G-0179", "--no-verify"},
		},
		{
			name:     "go change triggers lint, green passes",
			stdin:    refLine(fx.goChange, fx.docs),
			lintCmd:  "true",
			wantExit: 0,
		},
		{
			name:     "go.mod change triggers lint",
			stdin:    refLine(fx.goMod, fx.goChange),
			lintCmd:  "false",
			wantExit: 1,
		},
		{
			name:     "remote ref delete skips lint",
			stdin:    refLine(zero, fx.goChange),
			lintCmd:  "false",
			wantExit: 0,
		},
		{
			name:     "new ref without origin/main is conservative",
			stdin:    refLine(fx.goChange, zero),
			lintCmd:  "false",
			wantExit: 1,
		},
		{
			name:    "new ref diffs from origin/main merge-base",
			lintCmd: "false",
			setup: func(t *testing.T) (string, string) {
				// Own fixture: setting refs/remotes/origin/main
				// would mutate the shared repo. With origin/main
				// at the docs commit, a new ref at docs has an
				// empty range — no Go changes, lint skipped.
				t.Helper()
				own := newPrepushFixture(t)
				gitInFixture(t, own.dir, "update-ref", "refs/remotes/origin/main", own.docs)
				return own.dir, refLine(own.docs, zero)
			},
			wantExit: 0,
		},
		{
			name:     "unresolvable range is conservative",
			stdin:    refLine(fx.goChange, strings.Repeat("deadbeef", 5)),
			lintCmd:  "false",
			wantExit: 1,
		},
		{
			name:       "missing linter tolerated with warning",
			stdin:      refLine(fx.goChange, fx.docs),
			lintCmd:    "aiwf-no-such-linter-7d3f run",
			wantExit:   0,
			wantStderr: []string{"skipped"},
		},
		{
			name:     "blank stdin lines ignored",
			stdin:    "\n" + refLine(fx.docs, fx.base),
			lintCmd:  "false",
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, stdin := fx.dir, tt.stdin
			if tt.setup != nil {
				dir, stdin = tt.setup(t)
			}
			exitCode, stderr := runPrepushHook(t, dir, stdin, tt.lintCmd)
			if exitCode != tt.wantExit {
				t.Errorf("exit code = %d, want %d\nstderr: %s", exitCode, tt.wantExit, stderr)
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr should mention %q; got: %s", want, stderr)
				}
			}
		})
	}
}

// TestPrepushLintHook_InstallWiring pins the install path: the
// tracked script exists, is executable, and `make install-hooks`
// symlinks it into the G45 chain target. Drop any of the three and
// the gate silently stops being installable — the rot mode this pin
// exists to catch.
func TestPrepushLintHook_InstallWiring(t *testing.T) {
	t.Parallel()
	root := repoRootForHook(t)

	info, err := os.Stat(prepushHookPath(t))
	if err != nil {
		t.Fatalf("tracked hook script missing: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("scripts/git-hooks/pre-push must be executable; mode = %v", info.Mode())
	}

	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	wantLine := `ln -sfn ../../scripts/git-hooks/pre-push "$$HOOKS_DIR/pre-push.local"`
	if !strings.Contains(string(makefile), wantLine) {
		t.Errorf("Makefile install-hooks must symlink the pre-push hook into the G45 chain target; missing line: %s", wantLine)
	}
}
