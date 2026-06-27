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
// and combined stderr. extraEnv entries ("KEY=value") override any
// same-keyed variable inherited from the test process — duplicates
// are filtered out rather than appended, because which duplicate
// getenv returns is libc-dependent.
func runPrepushHook(t *testing.T, repoDir, stdin, lintCmd string, extraEnv ...string) (exitCode int, stderr string) {
	t.Helper()
	cmd := exec.Command(prepushHookPath(t))
	cmd.Dir = repoDir
	cmd.Stdin = strings.NewReader(stdin)
	// Both pre-push gates default to no-op stand-ins so a test exercises
	// exactly the gate it overrides: the lint cases don't trip the
	// (ungated) gitleaks scan, and the gitleaks cases don't trip the lint
	// gate. A caller's extraEnv entry for either key wins via the
	// last-occurrence dedup below.
	overrides := append([]string{
		"AIWF_PREPUSH_LINT_CMD=" + lintCmd,
		"AIWF_PREPUSH_GITLEAKS_CMD=true",
	}, extraEnv...)
	// Collapse duplicate keys, last occurrence winning, so a caller's
	// extraEnv replaces a default rather than both landing in env (which
	// duplicate getenv returns is libc-dependent).
	seen := map[string]int{}
	deduped := overrides[:0:0]
	for _, o := range overrides {
		key, _, _ := strings.Cut(o, "=")
		if i, ok := seen[key]; ok {
			deduped[i] = o
			continue
		}
		seen[key] = len(deduped)
		deduped = append(deduped, o)
	}
	overrides = deduped
	env := os.Environ()[:0:0]
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		overridden := false
		for _, o := range overrides {
			if oKey, _, _ := strings.Cut(o, "="); oKey == key {
				overridden = true
				break
			}
		}
		if !overridden {
			env = append(env, kv)
		}
	}
	env = append(env, overrides...)
	cmd.Env = env
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

// TestPrepushSecretScanHook_Decision pins the G-0291 secret-scan gate
// added to the same hook. Unlike the lint gate it is UNGATED — it scans
// every pushed range regardless of file type — so a docs-only push that
// skips lint still runs gitleaks. The AIWF_PREPUSH_GITLEAKS_CMD override
// stands in for the real gitleaks run (`true` clean / `false` finding).
func TestPrepushSecretScanHook_Decision(t *testing.T) {
	t.Parallel()
	fx := newPrepushFixture(t) // shared read-only across subtests — do not mutate

	const zero = "0000000000000000000000000000000000000000"
	refLine := func(localSha, remoteSha string) string {
		return "refs/heads/main " + localSha + " refs/heads/main " + remoteSha + "\n"
	}

	tests := []struct {
		name        string
		stdin       string
		gitleaksCmd string
		// setup, when set, builds a private fixture and returns
		// (dir, stdin), overriding the shared fixture and the stdin
		// field — for cases that must mutate repo state.
		setup      func(t *testing.T) (dir, stdin string)
		wantExit   int
		wantStderr []string
	}{
		{
			// The defining behavior: the scan is ungated. A docs-only
			// range skips lint but still scans for secrets — red blocks.
			name:        "docs-only range still scans, red blocks",
			stdin:       refLine(fx.docs, fx.base),
			gitleaksCmd: "false",
			wantExit:    1,
			wantStderr:  []string{"G-0291", "--no-verify"},
		},
		{
			name:        "docs-only range, green passes",
			stdin:       refLine(fx.docs, fx.base),
			gitleaksCmd: "true",
			wantExit:    0,
		},
		{
			name:        "go change range scans too, red blocks",
			stdin:       refLine(fx.goChange, fx.docs),
			gitleaksCmd: "false",
			wantExit:    1,
		},
		{
			name:        "remote ref delete skips the scan",
			stdin:       refLine(zero, fx.goChange),
			gitleaksCmd: "false",
			wantExit:    0,
		},
		{
			name:        "new ref without origin/main scans full history",
			stdin:       refLine(fx.goChange, zero),
			gitleaksCmd: "false",
			wantExit:    1,
		},
		{
			name:        "new ref with empty merge-base range skips the scan",
			gitleaksCmd: "false",
			setup: func(t *testing.T) (string, string) {
				// Own fixture: with origin/main at the docs commit, a new
				// ref at docs has an empty range — no commits to scan.
				t.Helper()
				own := newPrepushFixture(t)
				gitInFixture(t, own.dir, "update-ref", "refs/remotes/origin/main", own.docs)
				return own.dir, refLine(own.docs, zero)
			},
			wantExit: 0,
		},
		{
			name:        "unresolvable range scans full history",
			stdin:       refLine(fx.goChange, strings.Repeat("deadbeef", 5)),
			gitleaksCmd: "false",
			wantExit:    1,
		},
		{
			name:        "missing gitleaks tolerated with warning",
			stdin:       refLine(fx.docs, fx.base),
			gitleaksCmd: "aiwf-no-such-gitleaks-7d3f run",
			wantExit:    0,
			wantStderr:  []string{"local secret-scan skipped"},
		},
		{
			name:        "blank stdin lines ignored",
			stdin:       "\n" + refLine(zero, fx.goChange),
			gitleaksCmd: "false",
			wantExit:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, stdin := fx.dir, tt.stdin
			if tt.setup != nil {
				dir, stdin = tt.setup(t)
			}
			exitCode, stderr := runPrepushHook(t, dir, stdin, "true", "AIWF_PREPUSH_GITLEAKS_CMD="+tt.gitleaksCmd)
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

// TestPrepushSecretScanHook_ScanScope pins WHAT gitleaks is told to
// scan, which the decision table above cannot: its true/false stand-ins
// ignore their arguments. A normal push must scan only the pushed range
// (via --log-opts); the conservative fallback (new ref without
// origin/main) must scan the full history (no --log-opts). Without this,
// a mutant that drops --log-opts — scanning full history on every push,
// defeating the per-push performance goal — survives the table green.
// Uses an arg-recording stub, like TestPrepushLintHook_CacheIsolation.
func TestPrepushSecretScanHook_ScanScope(t *testing.T) {
	t.Parallel()
	fx := newPrepushFixture(t)

	const zero = "0000000000000000000000000000000000000000"
	refLine := func(localSha, remoteSha string) string {
		return "refs/heads/main " + localSha + " refs/heads/main " + remoteSha + "\n"
	}

	// Stub that records the gitleaks args it saw. A script (not an inline
	// sh -c) because the hook word-splits $gitleaks_cmd.
	stub := filepath.Join(fx.dir, "gitleaksstub.sh")
	stubBody := "#!/bin/sh\necho \"stub-args=$*\" >&2\nexit 0\n"
	if err := os.WriteFile(stub, []byte(stubBody), 0o755); err != nil {
		t.Fatalf("writing gitleaks stub: %v", err)
	}

	t.Run("range push scans only the pushed range", func(t *testing.T) {
		t.Parallel()
		stdin := refLine(fx.goChange, fx.docs)
		exitCode, stderr := runPrepushHook(t, fx.dir, stdin, "true", "AIWF_PREPUSH_GITLEAKS_CMD="+stub)
		if exitCode != 0 {
			t.Fatalf("exit = %d, want 0\nstderr: %s", exitCode, stderr)
		}
		want := "stub-args=--log-opts=" + fx.docs + ".." + fx.goChange
		if !strings.Contains(stderr, want) {
			t.Errorf("range push must scan the pushed range via --log-opts;\nwant %q in stderr:\n%s", want, stderr)
		}
	})

	t.Run("conservative fallback scans full history without --log-opts", func(t *testing.T) {
		t.Parallel()
		// New ref, no origin/main → scan_all → full-history scan.
		stdin := refLine(fx.goChange, zero)
		exitCode, stderr := runPrepushHook(t, fx.dir, stdin, "true", "AIWF_PREPUSH_GITLEAKS_CMD="+stub)
		if exitCode != 0 {
			t.Fatalf("exit = %d, want 0\nstderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stderr, "stub-args=") {
			t.Fatalf("stub should have run; stderr:\n%s", stderr)
		}
		if strings.Contains(stderr, "--log-opts") {
			t.Errorf("full-history fallback must NOT pass --log-opts; stderr:\n%s", stderr)
		}
	})
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

// TestPrepushLintHook_CacheIsolation pins the per-working-tree
// GOLANGCI_LINT_CACHE scoping. golangci-lint's default user-level
// cache replays raw issues carrying the absolute paths of whichever
// checkout linted a content-identical package; once that worktree is
// deleted, the nolint/filter processors can't re-read the files,
// fail open, and leak suppressed findings into the gate (observed
// false-blocking a main push after the G-0221 worktree was removed).
// The hook must derive a cache under the checkout's own git dir when
// the variable is unset, and respect an operator-set value.
func TestPrepushLintHook_CacheIsolation(t *testing.T) {
	t.Parallel()
	fx := newPrepushFixture(t)

	// Stub linter that reports the cache env var it saw. A script
	// (not an inline sh -c) because the hook word-splits $lint_cmd.
	stub := filepath.Join(fx.dir, "lintstub.sh")
	stubBody := "#!/bin/sh\necho \"stub-cache=$GOLANGCI_LINT_CACHE\" >&2\nexit 0\n"
	if err := os.WriteFile(stub, []byte(stubBody), 0o755); err != nil {
		t.Fatalf("writing lint stub: %v", err)
	}

	// Stdin that triggers the lint path (go change in range).
	stdin := "refs/heads/main " + fx.goChange + " refs/heads/main " + fx.docs + "\n"

	t.Run("unset derives per-git-dir cache", func(t *testing.T) {
		t.Parallel()
		exitCode, stderr := runPrepushHook(t, fx.dir, stdin, stub, "GOLANGCI_LINT_CACHE=")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0\nstderr: %s", exitCode, stderr)
		}
		want := "stub-cache=" + gitInFixture(t, fx.dir, "rev-parse", "--absolute-git-dir") + "/golangci-lint-cache"
		if !strings.Contains(stderr, want) {
			t.Errorf("hook must export a per-git-dir GOLANGCI_LINT_CACHE; want %q in stderr:\n%s", want, stderr)
		}
	})

	t.Run("pre-set value is respected", func(t *testing.T) {
		t.Parallel()
		exitCode, stderr := runPrepushHook(t, fx.dir, stdin, stub, "GOLANGCI_LINT_CACHE=/tmp/operator-cache-7d3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0\nstderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stderr, "stub-cache=/tmp/operator-cache-7d3f") {
			t.Errorf("hook must respect a pre-set GOLANGCI_LINT_CACHE; stderr:\n%s", stderr)
		}
	})

	t.Run("Makefile lint target carries the same scoping", func(t *testing.T) {
		t.Parallel()
		makefile, err := os.ReadFile(filepath.Join(repoRootForHook(t), "Makefile"))
		if err != nil {
			t.Fatalf("reading Makefile: %v", err)
		}
		wantLine := `GOLANGCI_LINT_CACHE="$${GOLANGCI_LINT_CACHE:-$$(git rev-parse --absolute-git-dir)/golangci-lint-cache}" golangci-lint run`
		if !strings.Contains(string(makefile), wantLine) {
			t.Errorf("Makefile lint recipe must scope the lint cache per working tree; missing line: %s", wantLine)
		}
	})
}
