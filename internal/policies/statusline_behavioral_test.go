package policies

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// --- M-0191 / G-0187: behavioral harness for .claude/statusline.sh ----------
//
// The M-0153 assertions in statusline_content_test.go are *structural* — they
// grep the source for robust shell forms (the `tail -r || tac` portability
// chain, the default-IFS sync parse, the GIT_OPTIONAL_LOCKS export). Those
// guard cross-platform / reflow-robustness properties a single-OS behavioral
// run cannot exercise, so they stay.
//
// This file adds the axis G-0187 says is missing: it *runs* the script against
// a hermetic git repo + transcript fixture + a stubbed `gh`, strips ANSI, and
// asserts the rendered segments — token count, ahead/behind sync, repo/branch,
// and the CI glyph. The CI cases drive the G-0189 stale-after-push fix:
//   - AC-2: a run whose headSha differs from local HEAD must render the
//     stale-pending glyph (…), not the previous commit's verdict.
//   - AC-3: folding HEAD into the cache key must make a new commit invalidate
//     a cached verdict (otherwise the 45s TTL serves the pre-commit result).

// statuslineANSI strips ANSI SGR escape sequences from rendered output.
var statuslineANSI = regexp.MustCompile("\x1b\\[[0-9;]*m")

// statuslineScript is the absolute path to the worktree's statusline.sh.
func statuslineScript(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), ".claude", "statusline.sh")
}

// gitIn runs git in dir, failing the test on error, returning trimmed stdout.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// newStatuslineRepo builds a temp git repo on `main` wired to a bare upstream
// so the sync segment resolves, then makes the local branch one commit ahead
// (renders ↑1). Returns the repo dir.
func newStatuslineRepo(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	repo := filepath.Join(base, "myrepo")
	bare := filepath.Join(base, "origin.git")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, base, "init", "--bare", bare)
	gitIn(t, repo, "init", "-b", "main")
	gitIn(t, repo, "remote", "add", "origin", bare)
	writeFile(t, filepath.Join(repo, "f0"), "0\n")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-m", "init")
	gitIn(t, repo, "push", "-u", "origin", "main")
	// One commit ahead of upstream -> sync renders ↑1.
	writeFile(t, filepath.Join(repo, "f1"), "1\n")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-m", "ahead")
	return repo
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeTranscript writes a JSONL transcript whose last usage-bearing line sums
// to 6000 tokens (1000 input + 5000 cache_read), rendered as "6k". Lives
// outside the repo so it does not dirty the working tree.
func writeTranscript(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "transcript.jsonl")
	lines := []string{
		`{"message":{"role":"user"}}`,
		`{"message":{"usage":{"input_tokens":1000,"cache_read_input_tokens":5000,"cache_creation_input_tokens":0}}}`,
	}
	writeFile(t, p, strings.Join(lines, "\n")+"\n")
	return p
}

// writeGhStub writes a fake `gh` to a fresh dir (returned for PATH prepend).
// For `gh run list --branch <b> ...` it prints $STUB_GH_JSON, except it prints
// "[]" when <b> equals $STUB_GH_EMPTY_BRANCH (used to force the no-runs →
// main-fallback path). It exits 0 for everything else, so the script's
// `command -v gh` guard passes.
func writeGhStub(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	stub := "#!/usr/bin/env bash\n" +
		"[ \"$1\" = run ] && [ \"$2\" = list ] || exit 0\n" +
		"shift 2\n" +
		"br=\"\"\n" +
		"while [ $# -gt 0 ]; do\n" +
		"  case \"$1\" in --branch) br=\"$2\"; shift 2 ;; *) shift ;; esac\n" +
		"done\n" +
		"if [ -n \"$STUB_GH_EMPTY_BRANCH\" ] && [ \"$br\" = \"$STUB_GH_EMPTY_BRANCH\" ]; then\n" +
		"  printf '%s' '[]'\n" +
		"else\n" +
		"  printf '%s' \"${STUB_GH_JSON:-[]}\"\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// statuslineStdin builds the session-context JSON Claude Code feeds on stdin.
func statuslineStdin(transcriptPath, repoDir string) string {
	return `{"model":{"display_name":"Opus 4.8 (1M context)"},` +
		`"transcript_path":"` + transcriptPath + `",` +
		`"workspace":{"current_dir":"` + repoDir + `"},` +
		`"effort":{"level":"xhigh"}}`
}

// runStatuslineCache runs the script in repoDir with the given stdin, the given
// CI cache dir, and extra env (e.g. STUB_GH_JSON=...), returning ANSI-stripped
// stdout. A fresh `gh` stub is placed on PATH for each call.
func runStatuslineCache(t *testing.T, repoDir, cacheDir, stdinJSON string, extraEnv ...string) string {
	t.Helper()
	ghDir := writeGhStub(t)
	cmd := exec.Command("bash", statuslineScript(t))
	cmd.Dir = repoDir
	cmd.Stdin = strings.NewReader(stdinJSON)

	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") || strings.HasPrefix(e, "AIWF_STATUSLINE_CACHE_DIR=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env,
		"PATH="+ghDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"AIWF_STATUSLINE_CACHE_DIR="+cacheDir,
	)
	env = append(env, extraEnv...)
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("statusline.sh: %v\nstderr: %s", err, stderr.String())
	}
	return statuslineANSI.ReplaceAllString(stdout.String(), "")
}

// runStatusline runs the script with a fresh per-call CI cache dir.
func runStatusline(t *testing.T, repoDir, stdinJSON string, extraEnv ...string) string {
	t.Helper()
	return runStatuslineCache(t, repoDir, t.TempDir(), stdinJSON, extraEnv...)
}

func ghRunJSON(headSha, conclusion, status string) string {
	return `STUB_GH_JSON=[{"headSha":"` + headSha + `","conclusion":"` + conclusion + `","status":"` + status + `"}]`
}

// TestStatusline_M0191_AC1_RendersRealSegments establishes the behavioral
// harness (G-0187): it runs statusline.sh end-to-end and asserts the rendered
// output reflects the fixture — token count from the transcript, ahead/behind
// sync, repo + branch names, and a green CI glyph for a success run at HEAD.
func TestStatusline_M0191_AC1_RendersRealSegments(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	tr := writeTranscript(t)
	head := gitIn(t, repo, "rev-parse", "HEAD")

	out := runStatusline(t, repo, statuslineStdin(tr, repo), ghRunJSON(head, "success", "completed"))

	checks := []struct{ want, why string }{
		{"6k", "token count summed from the transcript usage (1000+5000)"},
		{"main↑1", "branch + sync, contiguous: on main, one commit ahead of upstream"},
		{"myrepo", "repo name from git toplevel"},
		{"✓ ci", "success run whose headSha == HEAD"},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.want) {
			t.Errorf("AC-1: expected %q (%s) in rendered output\n got: %q", c.want, c.why, out)
		}
	}
}

// TestStatusline_M0191_AC2_StaleCIShowsPending drives the G-0189 fix: when the
// latest CI run is for a different commit than local HEAD, its verdict does not
// apply to what is checked out. The statusline must render stale-pending "… ci"
// rather than the run's "✓".
func TestStatusline_M0191_AC2_StaleCIShowsPending(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	tr := writeTranscript(t)
	const otherSha = "0000000000000000000000000000000000000000"

	out := runStatusline(t, repo, statuslineStdin(tr, repo), ghRunJSON(otherSha, "success", "completed"))

	if strings.Contains(out, "✓ ci") {
		t.Errorf("AC-2: a success run for a non-HEAD commit must not render \"✓ ci\" (stale)\n got: %q", out)
	}
	if !strings.Contains(out, "… ci") {
		t.Errorf("AC-2: a run whose headSha != local HEAD must render stale-pending \"… ci\"\n got: %q", out)
	}
}

// TestStatusline_M0191_AC3_CacheKeyIncludesHEAD proves HEAD is folded into the
// CI cache key. Run 1 caches a ✓ for commit A. After a new commit B (with CI
// now reporting a failure for B), a HEAD-keyed cache misses and re-fetches,
// rendering ✗; a branch-only cache key would serve the stale ✓ inside the TTL.
func TestStatusline_M0191_AC3_CacheKeyIncludesHEAD(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	tr := writeTranscript(t)
	cache := t.TempDir()

	headA := gitIn(t, repo, "rev-parse", "HEAD")
	out1 := runStatuslineCache(t, repo, cache, statuslineStdin(tr, repo), ghRunJSON(headA, "success", "completed"))
	if !strings.Contains(out1, "✓ ci") {
		t.Fatalf("AC-3 precondition: run 1 should render \"✓ ci\" for a success run at HEAD\n got: %q", out1)
	}

	writeFile(t, filepath.Join(repo, "f2"), "2\n")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-m", "second")
	headB := gitIn(t, repo, "rev-parse", "HEAD")

	out2 := runStatuslineCache(t, repo, cache, statuslineStdin(tr, repo), ghRunJSON(headB, "failure", "completed"))
	if strings.Contains(out2, "✓ ci") {
		t.Errorf("AC-3: after a new commit the cached \"✓ ci\" must be invalidated (HEAD in cache key)\n got stale: %q", out2)
	}
	if !strings.Contains(out2, "✗ ci") {
		t.Errorf("AC-3: after a new commit the fresh fetch should render \"✗ ci\" for the new HEAD\n got: %q", out2)
	}
}

// TestStatusline_M0191_CacheHitServesWithinTTL pins the positive cache path:
// a second render at the same HEAD within the TTL serves the cached verdict
// rather than re-fetching. It complements AC-3 — which proves a HEAD change
// invalidates the cache — by proving the cache actually caches. Without it, a
// fully-disabled cache (never reads/writes) would still satisfy AC-3.
func TestStatusline_M0191_CacheHitServesWithinTTL(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	tr := writeTranscript(t)
	cache := t.TempDir()
	head := gitIn(t, repo, "rev-parse", "HEAD")

	out1 := runStatuslineCache(t, repo, cache, statuslineStdin(tr, repo), ghRunJSON(head, "success", "completed"))
	if !strings.Contains(out1, "✓ ci") {
		t.Fatalf("precondition: run 1 should render \"✓ ci\"\n got: %q", out1)
	}
	// Same HEAD + shared cache, but gh now reports failure. Within the TTL the
	// cached ✓ must win (no re-fetch).
	out2 := runStatuslineCache(t, repo, cache, statuslineStdin(tr, repo), ghRunJSON(head, "failure", "completed"))
	if !strings.Contains(out2, "✓ ci") {
		t.Errorf("cache: a same-HEAD re-render within TTL must serve the cached \"✓ ci\", not re-fetch\n got: %q", out2)
	}
}

// TestStatusline_M0191_MainFallbackSkipsStaleness exercises the CI
// main-fallback path: on a branch with no runs of its own, the script falls
// back to main's status with an "m:" prefix, passing expected_sha="" so the
// staleness check is skipped (main's HEAD is not the current checkout's HEAD).
// This is the empty-expected_sha arm of the staleness guard added by this
// change — the current-branch AC tests never take it.
func TestStatusline_M0191_MainFallbackSkipsStaleness(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	tr := writeTranscript(t)
	gitIn(t, repo, "checkout", "-b", "feature/x")
	head := gitIn(t, repo, "rev-parse", "HEAD")

	// feature/x has no runs (STUB_GH_EMPTY_BRANCH) -> fall back to main, which
	// has a success run. The "m:" prefix marks the proxy.
	out := runStatusline(t, repo, statuslineStdin(tr, repo),
		"STUB_GH_EMPTY_BRANCH=feature/x",
		ghRunJSON(head, "success", "completed"))

	if !strings.Contains(out, "✓ m:ci") {
		t.Errorf("main-fallback: a branch with no runs should render \"✓ m:ci\" from main\n got: %q", out)
	}
}
