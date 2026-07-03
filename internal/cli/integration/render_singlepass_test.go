package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// render_singlepass_test.go — E-0054 / M-0221 AC-1 + AC-2 runtime count
// seam. render's per-entity history fan-out (a `git log --grep aiwf-entity`
// per epic/milestone/AC) and the per-milestone provenance greps (a second
// history walk + the repo-wide authorize-opener grep + per-SHA `git show`
// dates) collapse into ONE HEAD walk. This drives the real binary over a
// fixture that WOULD have triggered all of those on the old path, through a
// git wrapper that records every git invocation, and counts:
//
//   - AC-1: zero per-entity `--grep aiwf-entity` / `aiwf-prior-entity`
//     history greps, and exactly one full-HEAD walk (the shared pass,
//     identified by its unique AIWF-HEADREC pretty-format marker).
//   - AC-2: zero authorize-opener greps (`--grep aiwf-verb: authorize`) and
//     zero per-SHA scope-date `git show` lookups — the provenance/scope
//     views now derive from the shared pass.
//
// The fixture carries an epic, a milestone with an AC, and an OPEN scope
// (authorize) so the old path would have run every one of these greps —
// the zero-counts are non-vacuous.

func TestRenderSinglePass_OneHeadWalkZeroPerEntityGreps(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	bin := testutil.AiwfBinary(t)

	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, kv := range [][]string{{"user.email", "test@example.com"}, {"user.name", "test"}, {"commit.gpgsign", "false"}} {
		if out, err := exec.Command("git", "-C", repo, "config", kv[0], kv[1]).CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v\n%s", kv[0], err, out)
		}
	}
	// Build a tree that exercises every old-path grep family: an epic, a
	// milestone (history greps), an AC (composite grep), and an authorize
	// scope on the milestone (the authorize-opener grep + scope-date shows).
	for _, args := range [][]string{
		{"init", "--root", repo, "--actor", "human/test"},
		{"add", "epic", "--root", repo, "--actor", "human/test", "--title", "Foundations"},
		{"add", "milestone", "--tdd", "none", "--root", repo, "--actor", "human/test", "--epic", "E-0001", "--title", "Schema parser"},
		{"add", "ac", "--root", repo, "--actor", "human/test", "M-0001", "--title", "Engine starts"},
		{"promote", "--root", repo, "--actor", "human/test", "M-0001/AC-1", "--phase", "red"},
		{"promote", "--root", repo, "--actor", "human/test", "M-0001/AC-1", "--phase", "green", "--tests", "pass=3 fail=0 skip=0"},
	} {
		if out, err := testutil.RunBinary(bin, args...); err != nil {
			t.Fatalf("aiwf %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	// A ritual branch satisfies the authorize AI-target preflight, then open
	// a scope so the old path's authorize-opener grep would fire.
	if out, err := exec.Command("git", "-C", repo, "checkout", "-b", "milestone/M-0001-schema-parser").CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b: %v\n%s", err, out)
	}
	if out, err := testutil.RunBinary(bin, "authorize", "--root", repo, "--actor", "human/test", "M-0001", "--to", "ai/claude"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}

	// git-trace wrapper: log every invocation's args, then exec the real git.
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("locating real git: %v", err)
	}
	wrapDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "git-invocations.log")
	wrapper := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$AIWF_GIT_LOG\"\nexec " + realGit + " \"$@\"\n"
	if werr := os.WriteFile(filepath.Join(wrapDir, "git"), []byte(wrapper), 0o755); werr != nil {
		t.Fatalf("writing git wrapper: %v", werr)
	}

	siteDir := filepath.Join(t.TempDir(), "site")
	out, err := testutil.RunBin(t, repo, wrapDir, []string{"AIWF_GIT_LOG=" + logFile},
		"render", "--root", repo, "--format", "html", "--out", siteDir)
	if err != nil {
		t.Fatalf("aiwf render (traced): %v\n%s", err, out)
	}

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading git trace log: %v", err)
	}
	lines := strings.Split(string(logBytes), "\n")

	var entityGreps, authorizeGreps, headWalks, scopeDateShows int
	for _, ln := range lines {
		if !strings.HasPrefix(ln, "log") && !strings.HasPrefix(ln, "show") {
			continue
		}
		switch {
		case strings.HasPrefix(ln, "log") && strings.Contains(ln, "--grep") &&
			(strings.Contains(ln, "aiwf-entity:") || strings.Contains(ln, "aiwf-prior-entity:")):
			entityGreps++
		case strings.HasPrefix(ln, "log") && strings.Contains(ln, "--grep") && strings.Contains(ln, "aiwf-verb: authorize"):
			authorizeGreps++
		case strings.HasPrefix(ln, "show") && strings.Contains(ln, "%aI"):
			scopeDateShows++
		}
		if strings.Contains(ln, "AIWF-HEADREC") {
			headWalks++
		}
	}

	// AC-1: zero per-entity history greps; exactly one shared HEAD walk.
	if entityGreps != 0 {
		t.Errorf("AC-1: render issued %d per-entity `--grep aiwf-entity` history walks, want 0\ntrace:\n%s", entityGreps, logBytes)
	}
	if headWalks != 1 {
		t.Errorf("AC-1: render issued %d full-HEAD walks (AIWF-HEADREC marker), want exactly 1\ntrace:\n%s", headWalks, logBytes)
	}
	// AC-2: zero authorize-opener greps; zero per-SHA scope-date lookups.
	if authorizeGreps != 0 {
		t.Errorf("AC-2: render issued %d authorize-opener greps, want 0\ntrace:\n%s", authorizeGreps, logBytes)
	}
	if scopeDateShows != 0 {
		t.Errorf("AC-2: render issued %d per-SHA `git show %%aI` scope-date lookups, want 0\ntrace:\n%s", scopeDateShows, logBytes)
	}
}

// TestRenderSinglePass_FailsLoudOnUnreadableHistory pins the deliberate
// error semantic (M-0221): when the single shared HEAD walk cannot read
// history (a corrupt/partial repo), render fails loud with a clear error
// rather than silently emitting a site whose every history/provenance
// section is blank. The old per-entity path degraded one tab per failed
// walk; the shared pass would blank *every* page, so a silent degrade
// would be strictly worse and undetectable.
func TestRenderSinglePass_FailsLoudOnUnreadableHistory(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	bin := testutil.AiwfBinary(t)

	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, kv := range [][]string{{"user.email", "test@example.com"}, {"user.name", "test"}, {"commit.gpgsign", "false"}, {"gc.auto", "0"}} {
		if out, err := exec.Command("git", "-C", repo, "config", kv[0], kv[1]).CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v\n%s", kv[0], err, out)
		}
	}
	// A tree tree.Load can read, plus ≥2 commits so a non-HEAD ancestor's
	// object can be removed to make the ancestry walk unreadable.
	for _, args := range [][]string{
		{"init", "--root", repo, "--actor", "human/test"},
		{"add", "epic", "--root", repo, "--actor", "human/test", "--title", "Foundations"},
	} {
		if out, err := testutil.RunBinary(bin, args...); err != nil {
			t.Fatalf("aiwf %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	// Remove the root commit's loose object: HEAD still resolves (so the
	// walk starts), but `git log --reverse HEAD` cannot complete the
	// ancestry — the same repro the check-side fail-loud test uses.
	rootOut, err := exec.Command("git", "-C", repo, "rev-list", "--max-parents=0", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list root: %v\n%s", err, rootOut)
	}
	root := strings.TrimSpace(strings.SplitN(string(rootOut), "\n", 2)[0])
	if len(root) < 3 {
		t.Fatalf("unexpected root sha %q", root)
	}
	obj := filepath.Join(repo, ".git", "objects", root[:2], root[2:])
	if rmErr := os.Remove(obj); rmErr != nil {
		t.Fatalf("removing root commit object %s: %v", obj, rmErr)
	}

	siteDir := filepath.Join(t.TempDir(), "site")
	out, err := testutil.RunBinary(bin, "render", "--root", repo, "--format", "html", "--out", siteDir)
	if err == nil {
		t.Fatalf("render on corrupt history: want fail-loud non-zero exit, got success:\n%s", out)
	}
	if !strings.Contains(out, "reading history") {
		t.Errorf("render on corrupt history: want a 'reading history' error, got:\n%s", out)
	}
}

// TestRenderSinglePass_FailsLoudInProcess is the in-process companion to the
// subprocess fail-loud test above: it drives render.RunSite via cli.Execute
// so the ExitInternal branch is exercised in the coverage profile (a
// subprocess exit code is invisible to it). Serial — CaptureRun swaps the
// process stdout/stderr fds.
func TestRenderSinglePass_FailsLoudInProcess(t *testing.T) {
	repo := t.TempDir()
	if err := osExec(t, repo, "git", "init", "-q"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	for _, kv := range [][]string{{"user.email", "test@example.com"}, {"user.name", "test"}, {"commit.gpgsign", "false"}, {"gc.auto", "0"}} {
		if err := osExec(t, repo, "git", "config", kv[0], kv[1]); err != nil {
			t.Fatalf("git config %s: %v", kv[0], err)
		}
	}
	// Two empty commits: an empty tree still loads cleanly (no entities),
	// so RunSite reaches the shared history walk, which then fails on the
	// removed ancestor object.
	for _, msg := range []string{"root", "child"} {
		if err := osExec(t, repo, "git", "commit", "-q", "--allow-empty", "-m", msg); err != nil {
			t.Fatalf("git commit %s: %v", msg, err)
		}
	}
	rootOut, err := exec.Command("git", "-C", repo, "rev-list", "--max-parents=0", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list root: %v\n%s", err, rootOut)
	}
	root := strings.TrimSpace(strings.SplitN(string(rootOut), "\n", 2)[0])
	if rmErr := os.Remove(filepath.Join(repo, ".git", "objects", root[:2], root[2:])); rmErr != nil {
		t.Fatalf("removing root object: %v", rmErr)
	}

	siteDir := filepath.Join(t.TempDir(), "site")
	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"render", "--root", repo, "--format", "html", "--out", siteDir})
	})
	if rc != cliutil.ExitInternal {
		t.Fatalf("render on corrupt history: rc = %d, want ExitInternal (%d)\nstderr:\n%s", rc, cliutil.ExitInternal, stderr)
	}
	if !strings.Contains(stderr, "reading history") {
		t.Errorf("render on corrupt history: stderr missing 'reading history':\n%s", stderr)
	}
}
