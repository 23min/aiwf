package testutil

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// RunGit invokes git in workdir and returns combined output. The
// command runs with a fixed deterministic identity (GIT_AUTHOR_*
// env vars) so tests don't depend on the developer's git config.
func RunGit(workdir string, args ...string) (string, error) {
	return RunGitWithExtraEnv(workdir, nil, args...)
}

// RunGitWithExtraEnv is RunGit plus a slice of additional env
// entries appended AFTER the fixed identity defaults. Since
// exec processes env entries last-wins for duplicate keys, an
// extraEnv entry like "GIT_COMMITTER_EMAIL=human@example.com"
// overrides the default test identity for THIS subprocess only.
//
// Used by integration tests that need to vary committer identity
// per-call (e.g., M-0159/AC-6 cherry-pick scenarios that need
// committer != author to exercise the rule's gap-detection
// suppression contract). The default RunGit's "-c user.email=X"
// override would NOT achieve this — git evaluates GIT_*_EMAIL
// env vars with higher precedence than -c config overrides, so
// the env-var path is the only way to actually flip the
// committer identity inside a subprocess whose parent sets
// GIT_COMMITTER_EMAIL.
func RunGitWithExtraEnv(workdir string, extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// MustExec runs name with args in workdir; failure t.Fatals. The
// caller's location is reported via t.Helper().
func MustExec(t *testing.T, workdir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

// ExitedWithCode reports whether err is an *exec.ExitError with the
// given exit code. Used to tolerate non-zero exit codes that are
// documented contract (e.g., `aiwf doctor` returns 1 ExitFindings
// when aiwf.yaml is missing — that's expected, not a test failure).
func ExitedWithCode(err error, code int) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == code
	}
	return false
}

// buildOnce / builtPath / buildErr coordinate the one-time aiwf
// binary build that AiwfBinary returns. Per-process sync.Once so
// concurrent test goroutines share the result.
var (
	buildOnce sync.Once
	builtPath string
	buildErr  error
)

// AiwfBinary returns the absolute path to a built `aiwf` binary,
// compiling on the first call. The binary lives in a per-process
// temp dir so concurrent `go test` runs don't fight over it.
//
// macOS Sonoma 14.8.x has a syspolicyd crash on unsigned Mach-O
// headers (G-0128). Ad-hoc-signing the binary post-build routes
// around it.
func AiwfBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "aiwf-int-build-")
		if err != nil {
			buildErr = err
			return
		}
		bin := filepath.Join(dir, "aiwf")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		root := repoRootForTest(t)
		cmd := exec.Command("go", "build", "-o", bin, "./cmd/aiwf")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			buildErr = &buildError{err: err, output: string(out)}
			return
		}
		// macOS Gatekeeper crashes on unsigned Mach-O headers (Sonoma
		// 14.8.x syspolicyd bug per G-0128); ad-hoc sign to route around.
		if runtime.GOOS == "darwin" {
			if signOut, err := exec.Command("codesign", "--sign", "-", "--force", bin).CombinedOutput(); err != nil {
				buildErr = &buildError{err: err, output: "codesign: " + string(signOut)}
				return
			}
		}
		builtPath = bin
		// Prepend the just-built binary's dir to process PATH so the
		// G-0218 commit-msg hook (fired by `git commit` subprocesses
		// inside test fixtures) resolves `command -v aiwf` to *this*
		// build, not a stale system /go/bin/aiwf. Confined to the
		// per-package test binary's process.
		_ = os.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	return builtPath
}

type buildError struct {
	err    error
	output string
}

func (e *buildError) Error() string { return e.err.Error() + "\n" + e.output }

// repoRootForTest walks up from the test's cwd looking for go.mod
// and returns the absolute directory containing it. The test binary
// runs in the package directory; the repo root is some number of
// levels up depending on where the test lives.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate repo root (no go.mod in 8 parents from %q)", dir)
	return ""
}

// RunBin runs the built binary with args in workdir, prepending
// extraPath onto PATH. Returns combined stdout+stderr and exit error.
// The binary is built once per test process via AiwfBinary's
// sync.Once.
func RunBin(t *testing.T, workdir, extraPath string, env []string, args ...string) (string, error) {
	t.Helper()
	bin := AiwfBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"PATH="+extraPath+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunBinStdin is the stdin-bearing variant of RunBin: pipes the
// supplied reader to the binary's stdin so tests can exercise
// `--body-file -` and similar shorthands. Otherwise identical to
// RunBin (env, working dir, combined stdout+stderr).
func RunBinStdin(t *testing.T, workdir, extraPath string, stdin io.Reader, args ...string) (string, error) {
	t.Helper()
	bin := AiwfBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"PATH="+extraPath+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	cmd.Stdin = stdin
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// SkipIfShortOrUnsupported gates binary integration tests: requires
// `go` on PATH, skipped under `-short`, skipped on Windows (aiwf is
// unix-only).
func SkipIfShortOrUnsupported(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping binary integration test (-short); requires go build")
	}
	if runtime.GOOS == "windows" {
		t.Skip("aiwf is unix-only; binary integration test follows suit")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go not on PATH: %v", err)
	}
}

// BuildBinary compiles ./cmd/aiwf into tmp/aiwf with the given extra
// `go build` args (typically `-ldflags=…`) and returns the path. Use
// when AiwfBinary's cached default-build won't do (e.g., the
// ldflags-stamped-Version test path needs a per-invocation build).
//
// Builds happen from the repo root so the relative package path
// resolves regardless of which package the test runs in. The same
// G-0128 codesigning fix applies as AiwfBinary.
func BuildBinary(t *testing.T, tmp string, extraArgs ...string) string {
	t.Helper()
	out := filepath.Join(tmp, "aiwf")
	args := append([]string{"build"}, extraArgs...)
	args = append(args, "-o", out, "./cmd/aiwf")
	cmd := exec.Command("go", args...)
	cmd.Dir = repoRootForTest(t)
	if buildOut, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, buildOut)
	}
	// macOS Gatekeeper crashes on unsigned Mach-O headers (Sonoma 14.8.x
	// syspolicyd bug per G-0128); ad-hoc sign to route around.
	if runtime.GOOS == "darwin" {
		if signOut, err := exec.Command("codesign", "--sign", "-", "--force", out).CombinedOutput(); err != nil {
			t.Fatalf("codesign: %v\n%s", err, signOut)
		}
	}
	return out
}

// RunBinary invokes bin with args and returns combined stdout+stderr.
// Combined output is what a user sees, so the assertions read the
// same bytes the user would. Sibling of RunBin but doesn't add env
// vars — the caller controls the environment.
func RunBinary(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// RunBinaryAt is the workdir-bearing variant of RunBinary. Used by
// tests where the binary needs to run inside a specific repo (e.g.
// after init).
func RunBinaryAt(workdir, bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = workdir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// ExtractRow returns the first line of haystack whose prefix (after
// trimming leading whitespace) matches prefix. Empty if not found.
// Used by tests that parse line-oriented output like `aiwf doctor`.
func ExtractRow(haystack, prefix string) string {
	for _, line := range strings.Split(haystack, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), prefix) {
			return line
		}
	}
	return ""
}

// SetupGitRepoWithUpstream creates a fresh git repo with a bare
// origin set up as its upstream, then pushes an empty seed commit
// so the working repo has a tracked branch. Returns the absolute
// path of the working repo. Tests that exercise provenance audit
// scopes (`@{u}..HEAD` ranges) need an upstream configured.
func SetupGitRepoWithUpstream(t *testing.T, email string) string {
	t.Helper()
	upstream := t.TempDir()
	if out, err := RunGit(upstream, "init", "--bare", "-q"); err != nil {
		t.Fatalf("git init bare: %v\n%s", err, out)
	}
	root := t.TempDir()
	if out, err := RunGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", email},
		{"config", "user.name", "Test User"},
		{"remote", "add", "origin", upstream},
	} {
		if out, err := RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := RunGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit seed: %v\n%s", err, out)
	}
	if out, err := RunGit(root, "push", "-u", "origin", "HEAD:main"); err != nil {
		t.Fatalf("git push -u: %v\n%s", err, out)
	}
	return root
}

// ReadFileT reads the file at path and returns its contents as a
// string; t.Fatals on error. Trivial wrapper around os.ReadFile that
// keeps test bodies free of repeated error-check boilerplate.
func ReadFileT(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
