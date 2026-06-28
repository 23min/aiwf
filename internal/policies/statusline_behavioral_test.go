package policies

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

// --- M-0192 / G-0188: branch-contextual epic HUD ----------------------------
//
// The epic HUD answers a different question depending on the branch:
//   - non-ritual (e.g. main): the in-flight epic list — non-terminal epics
//     with the canonical glyph, terminal filtered, cap 3 + "+N" overflow.
//   - ritual (epic/E-*, milestone/M-*): ONLY the current epic — the one the
//     branch belongs to — plus its milestone inline on a milestone branch.
//
// Verifying the shipped "show epics on every branch" code surfaced the defect
// AC-2 pins: under the old show-all+accentuate shape the current epic was
// dropped into "+N" overflow whenever it sorted past the cap.

// statuslineEpicEntry matches a rendered epic-HUD entry ("<glyph> E-<n>"). It
// requires a status glyph before the id so the branch segment (".../E-1005…")
// is not mistaken for the HUD.
var statuslineEpicEntry = regexp.MustCompile(`[→○✓✗?] E-\d`)

// statuslineOverflow matches the "+N" epic-overflow marker.
var statuslineOverflow = regexp.MustCompile(`\+\d`)

// epicHUDSegment returns the " · "-delimited segment carrying the epic HUD, or
// "" if none is rendered. Scoping assertions to this segment (not the whole
// line) avoids the false positive where the branch segment itself contains the
// epic/milestone id (e.g. "epic/E-1005-echo" or "milestone/M-2001-work").
func epicHUDSegment(out string) string {
	for seg := range strings.SplitSeq(out, " · ") {
		if statuslineEpicEntry.MatchString(seg) {
			return seg
		}
	}
	return ""
}

// writeEpicFixture scaffolds work/epics/<id>-<slug>/epic.md with the status.
func writeEpicFixture(t *testing.T, repo, id, slug, status string) {
	t.Helper()
	dir := filepath.Join(repo, "work", "epics", id+"-"+slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "epic.md"),
		"---\nid: "+id+"\ntitle: "+slug+"\nstatus: "+status+"\n---\n## Deliverable\n\nfixture\n")
}

// writeMilestoneFixture scaffolds a milestone file under its parent epic dir.
func writeMilestoneFixture(t *testing.T, repo, epicID, epicSlug, id, slug, status string) {
	t.Helper()
	dir := filepath.Join(repo, "work", "epics", epicID+"-"+epicSlug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, id+"-"+slug+".md"),
		"---\nid: "+id+"\ntitle: "+slug+"\nstatus: "+status+"\nparent: "+epicID+"\n---\n## Deliverable\n\nfixture\n")
}

// commitFixtures commits scaffolded planning files so the ritual-branch
// milestone lookup (git ls-files) sees them.
func commitFixtures(t *testing.T, repo string) {
	t.Helper()
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-m", "fixtures")
}

// newEpicHUDRepo builds a repo with five active epics (E-1001..E-1005) and a
// milestone M-2001 under the last epic. The current epic (E-1005) sorts last,
// so the pre-reshape cap-3 list drops it into "+2" overflow.
func newEpicHUDRepo(t *testing.T) string {
	t.Helper()
	repo := newStatuslineRepo(t)
	writeEpicFixture(t, repo, "E-1001", "alpha", "active")
	writeEpicFixture(t, repo, "E-1002", "bravo", "active")
	writeEpicFixture(t, repo, "E-1003", "charlie", "active")
	writeEpicFixture(t, repo, "E-1004", "delta", "active")
	writeEpicFixture(t, repo, "E-1005", "echo", "active")
	writeMilestoneFixture(t, repo, "E-1005", "echo", "M-2001", "work", "in_progress")
	commitFixtures(t, repo)
	return repo
}

// TestStatusline_M0192_AC1_NonRitualRendersEpicList: on a non-ritual branch
// (main) the HUD renders the in-flight list — non-terminal epics with the
// canonical glyph, terminal epics filtered, and "+N" overflow past the cap.
func TestStatusline_M0192_AC1_NonRitualRendersEpicList(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t) // stays on main (non-ritual)
	tr := writeTranscript(t)

	// 4 non-terminal (cap 3 -> one overflow) + 2 terminal (filtered).
	writeEpicFixture(t, repo, "E-1001", "alpha", "active")
	writeEpicFixture(t, repo, "E-1002", "bravo", "proposed")
	writeEpicFixture(t, repo, "E-1003", "charlie", "active")
	writeEpicFixture(t, repo, "E-1004", "delta", "proposed")
	writeEpicFixture(t, repo, "E-1005", "echo", "done")
	writeEpicFixture(t, repo, "E-1006", "foxtrot", "cancelled")
	commitFixtures(t, repo)

	hud := epicHUDSegment(runStatusline(t, repo, statuslineStdin(tr, repo)))
	if hud == "" {
		t.Fatalf("AC-1: expected an epic HUD segment on main")
	}
	if !strings.Contains(hud, "→ E-1001") {
		t.Errorf("AC-1: active epic should render \"→ E-1001\"\n hud: %q", hud)
	}
	if !strings.Contains(hud, "○ E-1002") {
		t.Errorf("AC-1: proposed epic should render \"○ E-1002\"\n hud: %q", hud)
	}
	if strings.Contains(hud, "E-1005") {
		t.Errorf("AC-1: terminal (done) epic E-1005 must be filtered\n hud: %q", hud)
	}
	if strings.Contains(hud, "E-1006") {
		t.Errorf("AC-1: terminal (cancelled) epic E-1006 must be filtered\n hud: %q", hud)
	}
	if !strings.Contains(hud, "+1") {
		t.Errorf("AC-1: 4 non-terminal epics with cap 3 should render \"+1\" overflow\n hud: %q", hud)
	}
}

// TestStatusline_M0192_AC2_RitualShowsOnlyCurrentEpic: on a ritual branch the
// HUD shows ONLY the current epic (+ milestone inline on a milestone branch) —
// no other in-flight epic, no overflow — even with >cap epics in flight. Fails
// against the pre-reshape code, where the current epic (sorting last) is lost
// to "+2" overflow.
func TestStatusline_M0192_AC2_RitualShowsOnlyCurrentEpic(t *testing.T) {
	t.Parallel()

	others := []string{"E-1001", "E-1002", "E-1003", "E-1004"}

	t.Run("epic branch", func(t *testing.T) {
		t.Parallel()
		repo := newEpicHUDRepo(t)
		tr := writeTranscript(t)
		gitIn(t, repo, "checkout", "-b", "epic/E-1005-echo")

		hud := epicHUDSegment(runStatusline(t, repo, statuslineStdin(tr, repo)))
		if !strings.Contains(hud, "E-1005") {
			t.Errorf("AC-2 epic branch: HUD must show the current epic E-1005\n hud: %q", hud)
		}
		for _, o := range others {
			if strings.Contains(hud, o) {
				t.Errorf("AC-2 epic branch: HUD must not show other epic %s\n hud: %q", o, hud)
			}
		}
		if statuslineOverflow.MatchString(hud) {
			t.Errorf("AC-2 epic branch: ritual HUD must not render overflow\n hud: %q", hud)
		}
	})

	t.Run("milestone branch", func(t *testing.T) {
		t.Parallel()
		repo := newEpicHUDRepo(t)
		tr := writeTranscript(t)
		gitIn(t, repo, "checkout", "-b", "milestone/M-2001-work")

		hud := epicHUDSegment(runStatusline(t, repo, statuslineStdin(tr, repo)))
		if !strings.Contains(hud, "E-1005") {
			t.Errorf("AC-2 milestone branch: HUD must show the parent epic E-1005\n hud: %q", hud)
		}
		if !strings.Contains(hud, "M-2001") {
			t.Errorf("AC-2 milestone branch: HUD must show the milestone M-2001 inline\n hud: %q", hud)
		}
		for _, o := range others {
			if strings.Contains(hud, o) {
				t.Errorf("AC-2 milestone branch: HUD must not show other epic %s\n hud: %q", o, hud)
			}
		}
		if statuslineOverflow.MatchString(hud) {
			t.Errorf("AC-2 milestone branch: ritual HUD must not render overflow\n hud: %q", hud)
		}
	})
}

// TestStatusline_M0192_RitualEpicMissingFileFallsBackToUnknownGlyph pins the
// fail-soft arm of the ritual path: on an epic branch whose epic.md is absent
// or untracked (a stale or mistyped branch), the HUD still shows the id with
// the "?" unknown-status glyph rather than breaking under `set -u`. This is the
// reachable e_file="" branch the reviewer flagged as otherwise unexercised.
func TestStatusline_M0192_RitualEpicMissingFileFallsBackToUnknownGlyph(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	tr := writeTranscript(t)
	gitIn(t, repo, "checkout", "-b", "epic/E-9999-ghost") // no epic.md for E-9999

	hud := epicHUDSegment(runStatusline(t, repo, statuslineStdin(tr, repo)))
	if !strings.Contains(hud, "? E-9999") {
		t.Errorf("missing epic.md should fall back to \"? E-9999\" (unknown glyph)\n hud: %q", hud)
	}
}

// --- M-0193 / G-0290: statusline health glyph from a cached `--fast` probe ---
//
// The statusline prefixes ⚠ when `aiwf check --fast` reports error-severity
// findings (exit 1, per the check verb's contract). A clean tree — or one with
// only warnings (exit 0) — shows nothing, so the repo's always-present benign
// warnings never pin the light on. The verdict is cached with a TTL + HEAD-fold
// exactly like the CI segment: the hot render path reads the cache file and
// never runs a live check. The probe degrades silently (no glyph) when aiwf is
// absent or the tree is not an aiwf repo (no aiwf.yaml) — which is why the
// M-0191/M-0192 fixtures (no aiwf.yaml) render no health glyph and stay green.

// healthGlyph is the warning-prefix the health segment emits.
const healthGlyph = "⚠"

// writeAiwfStub writes a fake `aiwf` to a fresh dir (returned for PATH prepend).
// For `aiwf check --fast …` it exits with `code` (1 = errors present, 0 = clean
// or warnings-only, per the check verb's contract); everything else exits 0 so
// the script's `command -v aiwf` guard passes.
func writeAiwfStub(t *testing.T, code int) string {
	t.Helper()
	dir := t.TempDir()
	stub := "#!/usr/bin/env bash\n" +
		"if [ \"$1\" = check ]; then\n" +
		"  for a in \"$@\"; do [ \"$a\" = --fast ] && exit " + strconv.Itoa(code) + "; done\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "aiwf"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// newHealthRepo is newStatuslineRepo plus a committed aiwf.yaml, so the health
// segment's `-f aiwf.yaml` guard passes and the probe runs.
func newHealthRepo(t *testing.T) string {
	t.Helper()
	repo := newStatuslineRepo(t)
	writeFile(t, filepath.Join(repo, "aiwf.yaml"), "schema_version: 1\n")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-m", "aiwf.yaml")
	return repo
}

// runStatuslineHealth runs the script with both a `gh` stub and an `aiwf` stub
// (exiting aiwfExit for `check --fast`) on PATH, using the given CI/health cache
// dir. Mirrors runStatuslineCache; kept separate so the M-0191/M-0192 callers
// stay untouched.
func runStatuslineHealth(t *testing.T, repoDir, cacheDir, stdinJSON string, aiwfExit int, extraEnv ...string) string {
	t.Helper()
	ghDir := writeGhStub(t)
	aiwfDir := writeAiwfStub(t, aiwfExit)
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
		"PATH="+aiwfDir+string(os.PathListSeparator)+ghDir+string(os.PathListSeparator)+os.Getenv("PATH"),
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

// TestStatusline_M0193_AC2_FindingsRenderWarningPrefix: when `aiwf check --fast`
// reports errors (exit 1), the statusline leads with the ⚠ prefix.
func TestStatusline_M0193_AC2_FindingsRenderWarningPrefix(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	tr := writeTranscript(t)

	out := runStatuslineHealth(t, repo, t.TempDir(), statuslineStdin(tr, repo), 1)
	if !strings.HasPrefix(out, healthGlyph) {
		t.Errorf("AC-2: error findings must prefix the statusline with %q\n out: %q", healthGlyph, out)
	}
}

// TestStatusline_M0193_AC2_CleanTreeNoWarningPrefix: a clean tree (exit 0 —
// also the warnings-only case, which the check verb reports as exit 0) shows no
// ⚠ anywhere.
func TestStatusline_M0193_AC2_CleanTreeNoWarningPrefix(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	tr := writeTranscript(t)

	out := runStatuslineHealth(t, repo, t.TempDir(), statuslineStdin(tr, repo), 0)
	if strings.Contains(out, healthGlyph) {
		t.Errorf("AC-2: a clean / warnings-only tree must not render %q\n out: %q", healthGlyph, out)
	}
}

// TestStatusline_M0193_AC2_ProbeErrorDegrades: if the probe errors (exit >1 —
// e.g. an aiwf binary too old for --fast), the health segment degrades to no
// glyph rather than rendering a spurious or broken indicator.
func TestStatusline_M0193_AC2_ProbeErrorDegrades(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	tr := writeTranscript(t)

	out := runStatuslineHealth(t, repo, t.TempDir(), statuslineStdin(tr, repo), 2)
	if strings.Contains(out, healthGlyph) {
		t.Errorf("AC-2: a probe error (exit 2) must degrade to no glyph\n out: %q", out)
	}
}

// TestStatusline_M0193_AC2_CacheServedWithinTTL: the verdict is cached. A first
// render with errors (exit 1) caches "warn"; a second render within the TTL —
// even with the probe now reporting clean (exit 0) — still shows ⚠, proving the
// hot path served the cache and did not re-run the probe.
func TestStatusline_M0193_AC2_CacheServedWithinTTL(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	tr := writeTranscript(t)
	cacheDir := t.TempDir()

	first := runStatuslineHealth(t, repo, cacheDir, statuslineStdin(tr, repo), 1)
	if !strings.HasPrefix(first, healthGlyph) {
		t.Fatalf("AC-2 cache: first render with errors must show %q\n out: %q", healthGlyph, first)
	}
	second := runStatuslineHealth(t, repo, cacheDir, statuslineStdin(tr, repo), 0)
	if !strings.HasPrefix(second, healthGlyph) {
		t.Errorf("AC-2 cache: second render within TTL must serve the cached \"warn\" verdict (probe not re-run)\n out: %q", second)
	}
}
