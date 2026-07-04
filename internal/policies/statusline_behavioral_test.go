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

	"github.com/23min/aiwf/internal/skills"
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
// a hermetic git repo + a stubbed `gh`, strips ANSI, and asserts the rendered
// segments — token count, ahead/behind sync, repo/branch, and the CI glyph.
// The CI cases drive the G-0189 stale-after-push fix:
//   - AC-2: a run whose headSha differs from local HEAD must render the
//     stale-pending glyph (…), not the previous commit's verdict.
//   - AC-3: folding HEAD into the cache key must make a new commit invalidate
//     a cached verdict (otherwise the 45s TTL serves the pre-commit result).

// statuslineANSI strips ANSI SGR escape sequences from rendered output.
var statuslineANSI = regexp.MustCompile("\x1b\\[[0-9;]*m")

// statuslineScript materializes the embedded statusline script
// (`skills.StatuslineBytes`) to an executable temp file and returns its
// path. The behavioral harness execs the single source of truth — the
// embedded snapshot — rather than a materialized `.claude/statusline.sh`
// copy, which the repo no longer tracks. Each caller gets its own
// t.TempDir copy, so this is safe under t.Parallel.
func statuslineScript(t *testing.T) string {
	t.Helper()
	body := skills.StatuslineBytes()
	if len(body) == 0 {
		t.Fatal("skills.StatuslineBytes() returned empty — the go:embed directive is not wired or the source file is empty")
	}
	dest := filepath.Join(t.TempDir(), "statusline.sh")
	if err := os.WriteFile(dest, body, 0o755); err != nil {
		t.Fatalf("materializing statusline.sh: %v", err)
	}
	return dest
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

// statuslineStdin builds the session-context JSON Claude Code feeds on stdin,
// with a context_window object whose total_input_tokens renders "6k" — a safe
// default for tests that don't care about token rendering. Tests that do care
// build their own context_window-bearing stdin directly.
func statuslineStdin(repoDir string) string {
	return `{"model":{"display_name":"Opus 4.8 (1M context)"},` +
		`"context_window":{"total_input_tokens":6000,"context_window_size":200000},` +
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
	env = append(
		env,
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
// output reflects the fixture — token count from context_window, ahead/behind
// sync, repo + branch names, and a green CI glyph for a success run at HEAD.
func TestStatusline_M0191_AC1_RendersRealSegments(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	head := gitIn(t, repo, "rev-parse", "HEAD")

	out := runStatusline(t, repo, statuslineStdin(repo), ghRunJSON(head, "success", "completed"))

	checks := []struct{ want, why string }{
		{"6k", "token count from context_window.total_input_tokens (6000)"},
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

// --- G-0352: token/color source from stdin context_window, not the transcript
//
// Claude Code's stdin JSON carries a context_window object reflecting the
// *current* context. The statusline used to walk the transcript file for the
// last assistant usage block instead, which read stale for one render after
// /compact. These tests pin the replacement: context_window is the sole
// source (no transcript fallback — see G-0352), with each field degrading
// independently to 0 on absence/malformation, never a hard error.

// TestStatusline_G0352_TokensFromContextWindow proves the primary path: the
// token segment renders context_window.total_input_tokens (not a fixture
// value coincidentally matching it), and used_percentage drives the color.
// used_percentage is deliberately 85 with no context_window_size present: the
// size-fallback path (used when used_percentage is ignored) would compute
// pct=0 (green) here, so red is only reachable if used_percentage is actually
// read — a regression that dropped the used_percentage read would still pass
// a green-vs-green assertion, which is why this isn't e.g. 22.6.
func TestStatusline_G0352_TokensFromContextWindow(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	stdin := `{"model":{"display_name":"Opus 4.8"},` +
		`"context_window":{"total_input_tokens":45231,"used_percentage":85},` +
		`"workspace":{"current_dir":"` + repo + `"}}`

	raw := runStatuslineUsageRaw(t, repo, stdin)
	if !strings.Contains(raw, "\x1b[31m45k") {
		t.Errorf("G-0352: expected total_input_tokens 45231 to render red \"45k\" (used_percentage 85 >= 80; a green result would mean used_percentage was ignored)\n raw: %q", raw)
	}
}

// TestStatusline_G0352_PctFallsBackToContextWindowSize: when used_percentage
// is absent, the color derives from total_input_tokens/context_window_size
// (30000/200000 = 15% -> green) rather than defaulting to green regardless.
// The operand order matters: total_input_tokens=30000 and
// context_window_size=200000 are deliberately far apart (not e.g. 180000/
// 200000) so a swapped-operand bug (size*100/tokens = 666%) crosses out of
// the green bucket into red — a same-bucket fixture wouldn't catch that.
func TestStatusline_G0352_PctFallsBackToContextWindowSize(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	stdin := `{"model":{"display_name":"Opus 4.8"},` +
		`"context_window":{"total_input_tokens":30000,"context_window_size":200000},` +
		`"workspace":{"current_dir":"` + repo + `"}}`

	raw := runStatuslineUsageRaw(t, repo, stdin)
	if !strings.Contains(raw, "\x1b[32m30k") {
		t.Errorf("G-0352: absent used_percentage should derive pct from total_input_tokens/context_window_size (30000/200000 = 15%% -> green)\n raw: %q", raw)
	}
}

// TestStatusline_G0352_ColorThresholds pins the 50/80 color-bucket boundaries
// for the context_window-sourced pct — the ball/token color path. This is a
// structurally separate code path from usage_color() (which colors the
// rate-limit dots and is already boundary-tested by
// TestStatusline_G0310_UsageDotColorReflectsThreshold): the two share a
// numeric scale but not an implementation, so boundary coverage on one
// doesn't imply it on the other.
func TestStatusline_G0352_ColorThresholds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		pct  int
		want string
	}{
		{"49 -> green", 49, "\x1b[32m"},
		{"50 -> yellow (lower boundary)", 50, "\x1b[33m"},
		{"79 -> yellow", 79, "\x1b[33m"},
		{"80 -> red (lower boundary)", 80, "\x1b[31m"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			repo := newStatuslineRepo(t)
			stdin := `{"model":{"display_name":"Opus 4.8"},` +
				`"context_window":{"total_input_tokens":500,"used_percentage":` + strconv.Itoa(c.pct) + `},` +
				`"workspace":{"current_dir":"` + repo + `"}}`

			raw := runStatuslineUsageRaw(t, repo, stdin)
			want := c.want + "500"
			if !strings.Contains(raw, want) {
				t.Errorf("pct=%d: want color %q before the token text\n raw: %q", c.pct, c.want, raw)
			}
		})
	}
}

// TestStatusline_G0352_MissingContextWindowDegradesToZero: an absent
// context_window degrades to "0" tokens, green — never a hard error. Also
// proves there is no transcript fallback left: this stdin carries no
// transcript_path at all.
func TestStatusline_G0352_MissingContextWindowDegradesToZero(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	stdin := `{"model":{"display_name":"Opus 4.8"},"workspace":{"current_dir":"` + repo + `"}}`

	raw := runStatuslineUsageRaw(t, repo, stdin) // Fatals on non-zero exit -> proves no hard error
	if !strings.Contains(raw, "\x1b[32m0") {
		t.Errorf("G-0352: absent context_window must degrade to \"0\" tokens, green\n raw: %q", raw)
	}
}

// TestStatusline_G0352_MalformedContextWindowDegradesToZero: non-numeric
// total_input_tokens AND used_percentage must not crash the render; both
// degrade to their zero default independently. used_percentage is the
// string "abc" (not JSON null/absent — jq's `// empty` already collapses
// null to the same code path the missing-context_window test covers, so a
// null here wouldn't exercise numeric-malformation at all).
func TestStatusline_G0352_MalformedContextWindowDegradesToZero(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	stdin := `{"model":{"display_name":"Opus 4.8"},` +
		`"context_window":{"total_input_tokens":"not-a-number","used_percentage":"abc"},` +
		`"workspace":{"current_dir":"` + repo + `"}}`

	raw := runStatuslineUsageRaw(t, repo, stdin) // Fatals on non-zero exit -> proves no hard error
	if !strings.Contains(raw, "\x1b[32m0") {
		t.Errorf("G-0352: malformed context_window fields must degrade to \"0\" tokens, green\n raw: %q", raw)
	}
}

// TestStatusline_M0191_AC2_StaleCIShowsPending drives the G-0189 fix: when the
// latest CI run is for a different commit than local HEAD, its verdict does not
// apply to what is checked out. The statusline must render stale-pending "… ci"
// rather than the run's "✓".
func TestStatusline_M0191_AC2_StaleCIShowsPending(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	const otherSha = "0000000000000000000000000000000000000000"

	out := runStatusline(t, repo, statuslineStdin(repo), ghRunJSON(otherSha, "success", "completed"))

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
	cache := t.TempDir()

	headA := gitIn(t, repo, "rev-parse", "HEAD")
	out1 := runStatuslineCache(t, repo, cache, statuslineStdin(repo), ghRunJSON(headA, "success", "completed"))
	if !strings.Contains(out1, "✓ ci") {
		t.Fatalf("AC-3 precondition: run 1 should render \"✓ ci\" for a success run at HEAD\n got: %q", out1)
	}

	writeFile(t, filepath.Join(repo, "f2"), "2\n")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-m", "second")
	headB := gitIn(t, repo, "rev-parse", "HEAD")

	out2 := runStatuslineCache(t, repo, cache, statuslineStdin(repo), ghRunJSON(headB, "failure", "completed"))
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
	cache := t.TempDir()
	head := gitIn(t, repo, "rev-parse", "HEAD")

	out1 := runStatuslineCache(t, repo, cache, statuslineStdin(repo), ghRunJSON(head, "success", "completed"))
	if !strings.Contains(out1, "✓ ci") {
		t.Fatalf("precondition: run 1 should render \"✓ ci\"\n got: %q", out1)
	}
	// Same HEAD + shared cache, but gh now reports failure. Within the TTL the
	// cached ✓ must win (no re-fetch).
	out2 := runStatuslineCache(t, repo, cache, statuslineStdin(repo), ghRunJSON(head, "failure", "completed"))
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
	gitIn(t, repo, "checkout", "-b", "feature/x")
	head := gitIn(t, repo, "rev-parse", "HEAD")

	// feature/x has no runs (STUB_GH_EMPTY_BRANCH) -> fall back to main, which
	// has a success run. The "m:" prefix marks the proxy.
	out := runStatusline(t, repo, statuslineStdin(repo),
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

// TestStatusline_G0304_NonRitualRendersNoEpicHUD: on a non-ritual branch (main,
// patch, …) the epic HUD renders nothing — the session isn't in an epic, so the
// backlog belongs in `aiwf status`. Supersedes the M-0192 non-ritual in-flight
// list (G-0188's anti-blank rationale, now overridden).
func TestStatusline_G0304_NonRitualRendersNoEpicHUD(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t) // stays on main (non-ritual)

	// Several in-flight epics exist — none should appear in the HUD.
	writeEpicFixture(t, repo, "E-1001", "alpha", "active")
	writeEpicFixture(t, repo, "E-1002", "bravo", "proposed")
	writeEpicFixture(t, repo, "E-1003", "charlie", "active")
	writeEpicFixture(t, repo, "E-1004", "delta", "proposed")
	commitFixtures(t, repo)

	out := runStatusline(t, repo, statuslineStdin(repo))
	if hud := epicHUDSegment(out); hud != "" {
		t.Errorf("non-ritual branch must render no epic HUD; got %q", hud)
	}
	// No epic id should leak into the rendered line at all.
	for _, id := range []string{"E-1001", "E-1002", "E-1003", "E-1004"} {
		if strings.Contains(out, id) {
			t.Errorf("non-ritual line must not mention epic %s\n out: %q", id, out)
		}
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
		gitIn(t, repo, "checkout", "-b", "epic/E-1005-echo")

		hud := epicHUDSegment(runStatusline(t, repo, statuslineStdin(repo)))
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
		gitIn(t, repo, "checkout", "-b", "milestone/M-2001-work")

		hud := epicHUDSegment(runStatusline(t, repo, statuslineStdin(repo)))
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
	gitIn(t, repo, "checkout", "-b", "epic/E-9999-ghost") // no epic.md for E-9999

	hud := epicHUDSegment(runStatusline(t, repo, statuslineStdin(repo)))
	if !strings.Contains(hud, "? E-9999") {
		t.Errorf("missing epic.md should fall back to \"? E-9999\" (unknown glyph)\n hud: %q", hud)
	}
}

// --- E-0055 / G-0305: statusline installation-health stoplight --------------
//
// The statusline reads .claude/health.*.json (one per producer), unions the
// findings, and prefixes a four-state stoplight at the maximum severity: 🔴
// error, 🟡 warn, 🟢 a health file present with no warn/error, ⚪ an aiwf repo
// with no health file yet. It runs no check on the render path — producers
// write the files out of band. Resolves to the main checkout so one file serves
// every worktree. Supersedes the M-0193 cached-`--fast`-probe glyph (ADR-0026).

// writeHealthFixture writes a producer health file at
// <repo>/.claude/health.<source>.json with the given raw JSON body.
func writeHealthFixture(t *testing.T, repo, source, body string) {
	t.Helper()
	dir := filepath.Join(repo, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "health."+source+".json"), body)
}

// healthJSON builds a minimal producer health file body carrying one finding.
func healthJSON(source, severity, message string) string {
	return `{"generated_at":"x","findings":[{"source":"` + source +
		`","severity":"` + severity + `","message":"` + message + `"}]}`
}

// Health stoplight glyphs as they lead the rendered line, with the script's raw
// ANSI color codes. After ANSI-strip warn and error are identical (both ▲) and
// healthy and unknown are identical (both ●), so the tests assert on the raw
// colored prefix.
const (
	glyphError   = "\x1b[31m▲\x1b[0m " // red triangle
	glyphWarn    = "\x1b[33m▲\x1b[0m " // yellow triangle
	glyphHealthy = "\x1b[32m●\x1b[0m " // green dot
	glyphUnknown = "\x1b[90m●\x1b[0m " // gray dot
)

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

// TestStatusline_HealthStoplight_Error: an error-severity finding in a producer
// health file leads the line with the red stoplight.
func TestStatusline_HealthStoplight_Error(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	writeHealthFixture(t, repo, "aiwf", healthJSON("aiwf", "error", "aiwf.yaml not found"))

	out := runStatuslineUsageRaw(t, repo, statuslineStdin(repo))
	if !strings.HasPrefix(out, glyphError) {
		t.Errorf("an error finding must lead with the red ▲\n out: %q", out)
	}
}

// TestStatusline_HealthStoplight_Warn: a warn finding (and no error) leads with
// the yellow stoplight.
func TestStatusline_HealthStoplight_Warn(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	writeHealthFixture(t, repo, "aiwf", healthJSON("aiwf", "warn", "hook not aiwf-managed"))

	out := runStatuslineUsageRaw(t, repo, statuslineStdin(repo))
	if !strings.HasPrefix(out, glyphWarn) {
		t.Errorf("a warn finding must lead with the yellow ▲\n out: %q", out)
	}
}

// TestStatusline_HealthStoplight_HealthyGreen: a health file present with no
// warn/error (empty findings) is healthy — the green stoplight.
func TestStatusline_HealthStoplight_HealthyGreen(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	writeHealthFixture(t, repo, "aiwf", `{"generated_at":"x","findings":[]}`)

	out := runStatuslineUsageRaw(t, repo, statuslineStdin(repo))
	if !strings.HasPrefix(out, glyphHealthy) {
		t.Errorf("a present, clean health file must lead with the green ●\n out: %q", out)
	}
}

// TestStatusline_HealthStoplight_NoFileGray: an aiwf repo with no health file
// yet is unknown — the gray stoplight.
func TestStatusline_HealthStoplight_NoFileGray(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t) // aiwf.yaml present, no health file

	out := runStatuslineUsageRaw(t, repo, statuslineStdin(repo))
	if !strings.HasPrefix(out, glyphUnknown) {
		t.Errorf("an aiwf repo with no health file must lead with the gray ●\n out: %q", out)
	}
}

// TestStatusline_HealthStoplight_UnionMaxSeverity: findings union across
// producers — a second producer reporting error wins over a clean aiwf file.
func TestStatusline_HealthStoplight_UnionMaxSeverity(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	writeHealthFixture(t, repo, "aiwf", `{"generated_at":"x","findings":[]}`)
	writeHealthFixture(t, repo, "dotfiles", healthJSON("dotfiles", "error", "boom"))

	out := runStatuslineUsageRaw(t, repo, statuslineStdin(repo))
	if !strings.HasPrefix(out, glyphError) {
		t.Errorf("the union must take the max severity (error) across producers\n out: %q", out)
	}
}

// TestStatusline_HealthStoplight_CorruptDegrades: a corrupt producer file
// contributes no severity match; a valid sibling still drives the glyph.
func TestStatusline_HealthStoplight_CorruptDegrades(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	writeHealthFixture(t, repo, "dotfiles", "{ this is not json")
	writeHealthFixture(t, repo, "aiwf", healthJSON("aiwf", "warn", "advisory"))

	out := runStatuslineUsageRaw(t, repo, statuslineStdin(repo))
	if !strings.HasPrefix(out, glyphWarn) {
		t.Errorf("a corrupt sibling must not suppress the valid file's yellow ▲\n out: %q", out)
	}
}

// TestStatusline_HealthStoplight_AllCorruptGray: when every health file is
// present but unparseable, the state is unknown (gray) — not a false green
// (ADR-0026: "no health file present, or none parse").
func TestStatusline_HealthStoplight_AllCorruptGray(t *testing.T) {
	t.Parallel()
	repo := newHealthRepo(t)
	writeHealthFixture(t, repo, "aiwf", "{ this is not json")
	writeHealthFixture(t, repo, "dotfiles", "also not json")

	out := runStatuslineUsageRaw(t, repo, statuslineStdin(repo))
	if !strings.HasPrefix(out, glyphUnknown) {
		t.Errorf("an all-corrupt health file set must lead with the gray ● (unknown), not a false green\n out: %q", out)
	}
}

// --- G-0303: CI glyph aggregates across all workflows for HEAD --------------
//
// `gh run list` returns one entry per workflow run; the CI segment must reduce
// them to the WORST state for the checked-out commit, so a failed workflow
// (e.g. `go`) is never masked by a passing sibling (link-check, …) that happens
// to be the most-recent run. Pre-fix the segment sampled only `.[0]`.

// ghMultiRunJSON builds a STUB_GH_JSON array with several workflow runs for one
// headSha; each run is a {conclusion, status} pair.
func ghMultiRunJSON(headSha string, runs ...[2]string) string {
	parts := make([]string, 0, len(runs))
	for _, r := range runs {
		parts = append(parts, `{"headSha":"`+headSha+`","conclusion":"`+r[0]+`","status":"`+r[1]+`"}`)
	}
	return "STUB_GH_JSON=[" + strings.Join(parts, ",") + "]"
}

// ciSegment returns the " · "-delimited segment carrying the CI glyph (the one
// ending in the "ci" label), so assertions scope to it rather than the whole
// line.
func ciSegment(out string) string {
	for seg := range strings.SplitSeq(out, " · ") {
		if strings.HasSuffix(strings.TrimSpace(seg), "ci") {
			return strings.TrimSpace(seg)
		}
	}
	return ""
}

// TestStatusline_G0303_FailedWorkflowMaskedByPassingSibling is the core fix:
// the latest run (.[0]) passed, but another workflow for the same commit
// failed — the glyph must be ✗, not ✓.
func TestStatusline_G0303_FailedWorkflowMaskedByPassingSibling(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	head := gitIn(t, repo, "rev-parse", "HEAD")
	stub := ghMultiRunJSON(head, [2]string{"success", "completed"}, [2]string{"failure", "completed"})

	seg := ciSegment(runStatusline(t, repo, statuslineStdin(repo), stub))
	if !strings.Contains(seg, "✗") {
		t.Errorf("a failed workflow must render ✗ even when the latest run passed\n ci segment: %q", seg)
	}
	if strings.Contains(seg, "✓") {
		t.Errorf("must not render ✓ when a sibling workflow failed\n ci segment: %q", seg)
	}
}

// TestStatusline_G0303_AllWorkflowsSucceed: when every run for the commit
// succeeded, the glyph is ✓ (no regression of the green path).
func TestStatusline_G0303_AllWorkflowsSucceed(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	head := gitIn(t, repo, "rev-parse", "HEAD")
	stub := ghMultiRunJSON(head, [2]string{"success", "completed"}, [2]string{"success", "completed"})

	seg := ciSegment(runStatusline(t, repo, statuslineStdin(repo), stub))
	if !strings.Contains(seg, "✓") {
		t.Errorf("all-success runs must render ✓\n ci segment: %q", seg)
	}
	if strings.Contains(seg, "✗") {
		t.Errorf("all-success runs must not render ✗\n ci segment: %q", seg)
	}
}

// TestStatusline_G0303_InProgressAmongSuccess: a still-running workflow (with no
// failures) renders the pending glyph →.
func TestStatusline_G0303_InProgressAmongSuccess(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	head := gitIn(t, repo, "rev-parse", "HEAD")
	stub := ghMultiRunJSON(head, [2]string{"success", "completed"}, [2]string{"", "in_progress"})

	seg := ciSegment(runStatusline(t, repo, statuslineStdin(repo), stub))
	if !strings.Contains(seg, "→") {
		t.Errorf("an in-progress workflow (no failures) must render →\n ci segment: %q", seg)
	}
}

// TestStatusline_G0303_StaleWhenNoRunForHead: runs exist only for a different
// commit — the glyph stays the stale … (the aggregation preserves the
// HEAD-staleness guard, G-0189).
func TestStatusline_G0303_StaleWhenNoRunForHead(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	stub := ghMultiRunJSON("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", [2]string{"failure", "completed"})

	seg := ciSegment(runStatusline(t, repo, statuslineStdin(repo), stub))
	if !strings.Contains(seg, "…") {
		t.Errorf("a run for a different commit must render the stale glyph …\n ci segment: %q", seg)
	}
}

// TestStatusline_G0303_SuccessWithSkippedSibling: a benign skipped sibling (a
// path-filtered or if:-gated workflow) must not demote a passing commit to ? —
// it still renders ✓. Pins the any-success (not all-success) green arm.
func TestStatusline_G0303_SuccessWithSkippedSibling(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	head := gitIn(t, repo, "rev-parse", "HEAD")
	stub := ghMultiRunJSON(head, [2]string{"success", "completed"}, [2]string{"skipped", "completed"})

	seg := ciSegment(runStatusline(t, repo, statuslineStdin(repo), stub))
	if !strings.Contains(seg, "✓") {
		t.Errorf("a skipped sibling must not demote a passing commit; want ✓\n ci segment: %q", seg)
	}
	if strings.Contains(seg, "?") {
		t.Errorf("success + skipped must render ✓, not ?\n ci segment: %q", seg)
	}
}

// --- G-0304: patch branches show the session's gap --------------------------
//
// A `patch/G-NNNN-*` branch is a wf-patch fixing a gap; the HUD shows that gap
// with its status glyph/color, the same session-entity treatment epics get on a
// ritual branch.

// statuslineHUDEntry matches the session-entity HUD's "▸ <glyph> <id>" shape
// (epic / milestone / gap), distinguishing it from the head segment's
// "▸ <tokens>" (where ▸ precedes the token count, not a status glyph).
var statuslineHUDEntry = regexp.MustCompile(`▸ [→○✓✗?] [EMG]-\d`)

// hudSegment returns the " · "-delimited segment carrying the session-entity HUD
// (epic or gap). Scopes assertions to it.
func hudSegment(out string) string {
	for seg := range strings.SplitSeq(out, " · ") {
		if statuslineHUDEntry.MatchString(seg) {
			return strings.TrimSpace(seg)
		}
	}
	return ""
}

// writeGapFixture scaffolds work/gaps/<id>-<slug>.md with the status.
func writeGapFixture(t *testing.T, repo, id, slug, status string) {
	t.Helper()
	dir := filepath.Join(repo, "work", "gaps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, id+"-"+slug+".md"),
		"---\nid: "+id+"\ntitle: "+slug+"\nstatus: "+status+"\n---\n## Problem\n\nfixture\n")
}

// TestStatusline_G0304_PatchBranchShowsGap: on a patch/G-NNNN-* branch the HUD
// renders the gap with its status glyph (open → ○) and color.
func TestStatusline_G0304_PatchBranchShowsGap(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	writeGapFixture(t, repo, "G-0500", "fix-thing", "open")
	commitFixtures(t, repo)
	gitIn(t, repo, "checkout", "-b", "patch/G-0500-fix-thing")

	hud := hudSegment(runStatusline(t, repo, statuslineStdin(repo)))
	if !strings.Contains(hud, "G-0500") {
		t.Errorf("patch branch must show its gap G-0500\n hud: %q", hud)
	}
	if !strings.Contains(hud, "○") {
		t.Errorf("an open gap should render the ○ glyph\n hud: %q", hud)
	}
}

// TestStatusline_G0304_PatchBranchAddressedGapGlyph: the gap's status drives the
// glyph — an addressed gap renders ✓, distinct from an open one.
func TestStatusline_G0304_PatchBranchAddressedGapGlyph(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	writeGapFixture(t, repo, "G-0500", "fix-thing", "addressed")
	commitFixtures(t, repo)
	gitIn(t, repo, "checkout", "-b", "patch/G-0500-fix-thing")

	hud := hudSegment(runStatusline(t, repo, statuslineStdin(repo)))
	if !strings.Contains(hud, "✓ G-0500") {
		t.Errorf("an addressed gap should render \"✓ G-0500\"\n hud: %q", hud)
	}
}

// TestStatusline_G0304_PatchBranchMissingGapFileFallsBack: a patch branch whose
// gap file is absent (stale/mistyped) still shows the id with the ? unknown
// glyph rather than breaking under set -u.
func TestStatusline_G0304_PatchBranchMissingGapFileFallsBack(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	gitIn(t, repo, "checkout", "-b", "patch/G-9999-ghost") // no gap file for G-9999

	hud := hudSegment(runStatusline(t, repo, statuslineStdin(repo)))
	if !strings.Contains(hud, "? G-9999") {
		t.Errorf("missing gap file should fall back to \"? G-9999\"\n hud: %q", hud)
	}
}

// TestStatusline_G0304_RepoNameIsMainRepoNotWorktreeDir: in a linked worktree
// the repo segment shows the MAIN repo's name (shared .git's parent), not the
// worktree directory's basename — so an entity-id-named worktree
// (.../worktrees/G-0500) doesn't render its id as the repo name.
func TestStatusline_G0304_RepoNameIsMainRepoNotWorktreeDir(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)               // main repo basename: "myrepo"
	wt := filepath.Join(t.TempDir(), "G-0500") // worktree dir literally "G-0500"
	gitIn(t, repo, "worktree", "add", "-b", "patch/G-0500-x", wt)

	out := runStatusline(t, wt, statuslineStdin(wt))
	mainSeen := false
	for seg := range strings.SplitSeq(out, " · ") {
		s := strings.TrimSpace(seg)
		if s == "myrepo" {
			mainSeen = true
		}
		if s == "G-0500" {
			t.Errorf("repo segment must be the main repo, not the worktree dir \"G-0500\"\n out: %q", out)
		}
	}
	if !mainSeen {
		t.Errorf("expected the main repo name \"myrepo\" as a segment\n out: %q", out)
	}
}

// --- G-0310: subscription-usage dots (weekly seven_day / 5-hour five_hour) ---
//
// The statusline reads rate_limits.{seven_day,five_hour}.used_percentage from
// its stdin JSON (the figures /usage shows; Pro/Max only, present after the
// first API response, each window independently optional) and renders one
// colored dot + label per present window — green/yellow/red on the same scale
// as the context ball. An absent window renders nothing.

// statuslineStdinRates is statuslineStdin plus a rate_limits block. A negative
// percentage omits that window (models an absent field).
func statuslineStdinRates(repoDir string, sevenDay, fiveHour float64) string {
	var w []string
	if fiveHour >= 0 {
		w = append(w, `"five_hour":{"used_percentage":`+strconv.FormatFloat(fiveHour, 'f', -1, 64)+`,"resets_at":1738425600}`)
	}
	if sevenDay >= 0 {
		w = append(w, `"seven_day":{"used_percentage":`+strconv.FormatFloat(sevenDay, 'f', -1, 64)+`,"resets_at":1738857600}`)
	}
	rl := ""
	if len(w) > 0 {
		rl = `,"rate_limits":{` + strings.Join(w, ",") + `}`
	}
	return `{"model":{"display_name":"Opus 4.8 (1M context)"},` +
		`"context_window":{"total_input_tokens":6000,"context_window_size":200000},` +
		`"workspace":{"current_dir":"` + repoDir + `"},` +
		`"effort":{"level":"xhigh"}` + rl + `}`
}

// runStatuslineUsageRaw runs the script and returns RAW (non-ANSI-stripped)
// stdout — the usage-dot color is the behavioral signal under test. Mirrors
// runStatuslineCache's env setup (fresh gh stub, isolated cache dir).
func runStatuslineUsageRaw(t *testing.T, repoDir, stdinJSON string) string {
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
	env = append(
		env,
		"PATH="+ghDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"AIWF_STATUSLINE_CACHE_DIR="+t.TempDir(),
	)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("statusline.sh: %v\nstderr: %s", err, stderr.String())
	}
	return stdout.String()
}

func TestStatusline_G0310_UsageDotsRenderWhenPresent(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	out := runStatusline(t, repo, statuslineStdinRates(repo, 90, 20))
	if !strings.Contains(out, "● 7d") {
		t.Errorf("weekly usage dot ● 7d missing\n out: %q", out)
	}
	if !strings.Contains(out, "● 5h") {
		t.Errorf("5-hour usage dot ● 5h missing\n out: %q", out)
	}
}

func TestStatusline_G0310_UsageDotsAbsentWhenNoRateLimits(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	// plain stdin: no rate_limits block (non-subscriber / pre-first-API-response)
	out := runStatusline(t, repo, statuslineStdin(repo))
	if strings.Contains(out, "7d") || strings.Contains(out, "5h") {
		t.Errorf("no rate_limits -> no usage dots, got\n out: %q", out)
	}
}

func TestStatusline_G0310_OneWindowAbsentRendersOnlyTheOther(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	// weekly present, five_hour omitted (negative) — windows are independently optional
	out := runStatusline(t, repo, statuslineStdinRates(repo, 90, -1))
	if !strings.Contains(out, "● 7d") {
		t.Errorf("weekly present -> ● 7d expected\n out: %q", out)
	}
	if strings.Contains(out, "5h") {
		t.Errorf("five_hour absent -> no ● 5h\n out: %q", out)
	}
}

func TestStatusline_G0310_UsageDotColorReflectsThreshold(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	const red, green, yellow, reset = "\x1b[31m", "\x1b[32m", "\x1b[33m", "\x1b[0m"
	cases := []struct {
		name               string
		sevenDay, fiveHour float64
		wantWeekly         string // color escape expected immediately before ● 7d
	}{
		{"weekly red at 90", 90, 20, red},
		{"weekly green at 20", 20, 20, green},
		{"weekly yellow at 65", 65, 20, yellow},
		// 79.9 proves truncation (not rounding): truncates to 79 -> yellow; a
		// round would give 80 -> red. Also the only case whose JSON carries a
		// decimal point, exercising the `${1%%.*}` strip.
		{"weekly 79.9 truncates to 79 -> yellow", 79.9, 20, yellow},
		{"weekly boundary 80 -> red", 80, 20, red},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			raw := runStatuslineUsageRaw(t, repo, statuslineStdinRates(repo, c.sevenDay, c.fiveHour))
			want := c.wantWeekly + "●" + reset + " 7d"
			if !strings.Contains(raw, want) {
				t.Errorf("weekly dot color: want %q in raw output\n raw: %q", want, raw)
			}
		})
	}
}

func TestStatusline_G0310_NonNumericUsageRendersNothing(t *testing.T) {
	t.Parallel()
	repo := newStatuslineRepo(t)
	// rate_limits present but used_percentage is non-numeric: the defensive guard
	// (`*[!0-9.]*`) must render no dot and emit no shell error. The float64-typed
	// statuslineStdinRates can't express this, so craft the stdin directly.
	stdin := `{"model":{"display_name":"Opus 4.8 (1M context)"},` +
		`"workspace":{"current_dir":"` + repo + `"},` +
		`"effort":{"level":"xhigh"},` +
		`"rate_limits":{"seven_day":{"used_percentage":"oops"},"five_hour":{"used_percentage":"n/a"}}}`
	out := runStatusline(t, repo, stdin)
	if strings.Contains(out, "7d") || strings.Contains(out, "5h") {
		t.Errorf("non-numeric used_percentage -> no usage dots, got\n out: %q", out)
	}
}
