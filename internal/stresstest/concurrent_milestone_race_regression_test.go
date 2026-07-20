package stresstest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// concurrent_milestone_race_regression_test.go — M-0258/AC-3: proves
// AC-2's oracle (classifyMilestoneRaceOutcomes) actually catches a
// reintroduced G-0335-shaped regression, not merely that it stays
// quiet on a healthy binary. Builds a disposable, patched copy of this
// module's source in an isolated `git worktree`, with BOTH the
// milestone-cancel open-AC guard AND internal/check/acs.go's
// milestone-cancelled-incomplete-acs check-rule backstop removed — so
// a violation this test observes can only come from
// ConcurrentMilestoneRaceScenario's own commit-order oracle, isolating
// that the new AC-2 oracle provides real protection independent of
// the pre-existing check-rule doing the work. Never touches this
// worktree's own tracked source; the patched copy lives entirely
// under a temp-dir worktree torn down in t.Cleanup.

// milestoneCancelGuardAnchor is internal/verb/cancel_guards.go's
// milestoneACsCascadeGuard — the exact G-0335 guard (D-0004) this test
// removes from the isolated copy. M-0270/AC-3 unified Cancel's and
// Promote's previously-separate milestone-open-AC checks onto this one
// shared function; anchoring on the function signature (rather than
// its interior body, the pre-M-0270 shape) survives a future refactor
// of the guard's own internals the same way checkRuleAnchor below
// already does for its sibling check-rule anchor. Built via "\t"/"\n"
// escapes rather than a raw string literal so the anchor's tab-vs-
// space bytes don't depend on how this source file itself happens to
// be indented; strings.Count(content, this) must equal 1 in
// cancel_guards.go, or patchFileExactlyOnce refuses to patch (see its
// own doc comment).
const milestoneCancelGuardAnchor = "func milestoneACsCascadeGuard(e *entity.Entity, newStatus string, buildErr func(openACs []string) error) error {\n"

// milestoneCancelGuardReplacement stubs milestoneACsCascadeGuard to
// always return nil — both Cancel and Promote's milestone-open-AC
// refusal (they share this one guard as of M-0270/AC-3) stop firing
// entirely.
const milestoneCancelGuardReplacement = milestoneCancelGuardAnchor +
	"\treturn nil // AC-3 regression probe (M-0258): open-AC cascade guard deliberately stubbed out\n"

// checkRuleAnchor is internal/check/acs.go's
// milestoneCancelledIncompleteACs function signature line — the
// check-rule backstop for the same G-0335 shape, reported under the
// milestone-cancelled-incomplete-acs finding code. Anchoring on the
// signature line alone (rather than the full function body) sidesteps
// needing to reproduce every interior line's exact whitespace; the
// replacement below appends an unconditional early return, leaving
// the original body intact-but-unreachable underneath it (legal Go: a
// variable referenced anywhere in a function body, reachable or not,
// still counts as "used").
const checkRuleAnchor = "func milestoneCancelledIncompleteACs(t *tree.Tree) []Finding {\n"

// checkRuleReplacement stubs milestoneCancelledIncompleteACs to
// always return no findings.
const checkRuleReplacement = checkRuleAnchor +
	"\treturn nil // AC-3 regression probe (M-0258): check-rule backstop deliberately stubbed out\n"

// patchExactlyOnce returns content with old replaced by new, but only
// when old occurs in content EXACTLY once. Split out of
// patchFileExactlyOnce as a pure function so both of its failure
// modes — the anchor missing entirely, and the anchor matching more
// than once — are directly unit-testable (TestPatchExactlyOnce)
// without any filesystem or git machinery: the sanity check this
// AC's own design calls for, proving a future refactor of the patched
// source can't silently make this regression-probe patch a no-op (or
// land on the wrong spot) without this test noticing.
func patchExactlyOnce(content, old, newText string) (string, error) {
	switch n := strings.Count(content, old); n {
	case 0:
		return "", fmt.Errorf("patch anchor not found (want exactly 1 occurrence, got 0): %q", old)
	case 1:
		return strings.Replace(content, old, newText, 1), nil
	default:
		return "", fmt.Errorf("patch anchor is ambiguous (want exactly 1 occurrence, got %d): %q", n, old)
	}
}

// patchFileExactlyOnce reads path, replaces old with newText via
// patchExactlyOnce (failing loudly if old isn't found exactly once),
// and writes the result back to path with its original permissions.
func patchFileExactlyOnce(t *testing.T, path, old, newText string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil { //coverage:ignore defensive: reading a file this test's own addDetachedGitWorktree just checked out from a real commit has no realistic failure mode
		t.Fatalf("reading %s: %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil { //coverage:ignore defensive: stat immediately following a successful ReadFile of the same path has no realistic failure mode
		t.Fatalf("stat %s: %v", path, err)
	}
	patched, err := patchExactlyOnce(string(raw), old, newText)
	if err != nil {
		t.Fatalf("patching %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(patched), info.Mode()); err != nil { //coverage:ignore defensive: writing back to a file this test just read from, in a disposable temp-dir worktree, has no realistic failure mode
		t.Fatalf("writing %s: %v", path, err)
	}
}

// removeMilestoneCancelOpenACGuard patches moduleRoot's copy of
// internal/verb/cancel_guards.go so neither Cancel nor Promote refuses
// a milestone carrying an open AC — the G-0335 shape.
func removeMilestoneCancelOpenACGuard(t *testing.T, moduleRoot string) {
	t.Helper()
	patchFileExactlyOnce(t, filepath.Join(moduleRoot, "internal", "verb", "cancel_guards.go"), milestoneCancelGuardAnchor, milestoneCancelGuardReplacement)
}

// stubMilestoneCancelledIncompleteACsCheckRule patches moduleRoot's
// copy of internal/check/acs.go so the milestone-cancelled-incomplete-
// acs post-hoc check-rule backstop reports nothing, isolating this
// test's regression signal to AC-2's own oracle.
func stubMilestoneCancelledIncompleteACsCheckRule(t *testing.T, moduleRoot string) {
	t.Helper()
	patchFileExactlyOnce(t, filepath.Join(moduleRoot, "internal", "check", "acs.go"), checkRuleAnchor, checkRuleReplacement)
}

// gitCaptureOutput runs git with args in dir and returns its trimmed
// stdout, failing the test on error. Distinct from this package's own
// runGit (gitrepo.go), which discards stdout — both `git rev-parse
// HEAD` (this test's own currentSHA) and the post-run `git status
// --short` (this test's own "did I touch the real worktree" check)
// need the output, not just success/failure.
func gitCaptureOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec // args are fixed literals this test controls, not attacker-controlled input
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil { //coverage:ignore defensive: git rev-parse HEAD / git status --short against a real, already-initialized repo this test itself is running inside has no realistic failure mode
		t.Fatalf("git %v in %s: %v", args, dir, err)
	}
	return strings.TrimSpace(string(out))
}

// addDetachedGitWorktree creates a disposable, detached-HEAD `git
// worktree` checked out at sha, rooted at a fresh t.TempDir(), and
// registers its teardown so git's own worktree registry never
// accumulates a stale entry once the temp dir is reaped. Can't run
// `git worktree add <dir> HEAD` here — in this test's caller, HEAD
// resolves to the milestone branch repoDir itself already has checked
// out, and git refuses to check out the same branch into two
// worktrees at once. Passing the explicit sha with --detach sidesteps
// that.
func addDetachedGitWorktree(t *testing.T, repoDir, sha string) string {
	t.Helper()
	dir := t.TempDir()
	if err := runGit(repoDir, "worktree", "add", "--detach", dir, sha); err != nil {
		t.Fatalf("adding a detached worktree at %s for %s: %v", dir, sha, err)
	}
	t.Cleanup(func() {
		if err := runGit(repoDir, "worktree", "remove", "--force", dir); err != nil { //coverage:ignore defensive: removing a worktree this test itself just added, still present and untouched by any other process, has no realistic failure mode
			t.Errorf("removing worktree %s: %v", dir, err)
		}
	})
	return dir
}

// TestConcurrentMilestoneRaceScenario_RealBinary_DetectsAReintroducedG0335Regression
// is AC-3: with the milestone-cancel open-AC guard AND its check-rule
// backstop both removed from a disposable, isolated copy of this
// module, ConcurrentMilestoneRaceScenario's run must fail — reporting
// at least one violation — across a 30-attempt repeat, mirroring
// G-0410's own repeat-N-times empirical methodology against the
// pre-fix G-0335 binary.
//
// Deliberately NOT t.Parallel(): this test does a full second `go
// build` from a patched source tree plus 30 real multi-subprocess
// race attempts — heavier than this package's other real-subprocess
// tests — and running it alongside them risks skewing the very
// race-timing behavior it measures.
func TestConcurrentMilestoneRaceScenario_RealBinary_DetectsAReintroducedG0335Regression(t *testing.T) {
	skipIfUnsupported(t)

	thisWorktreeDir, err := filepath.Abs(repoRootRelative)
	if err != nil { //coverage:ignore defensive: filepath.Abs on a constant relative path with a resolvable working directory has no realistic failure mode
		t.Fatalf("resolving this worktree's module root: %v", err)
	}
	// statusBefore is captured before any patching/building/racing
	// happens, so the end-of-test comparison isolates changes THIS
	// TEST caused from whatever pre-existing dirty state (e.g. this
	// very test file, still uncommitted) already sat in the tree —
	// the "did I touch the real worktree" check needs a before/after
	// diff, not a bare "is status empty" assertion.
	statusBefore := gitCaptureOutput(t, thisWorktreeDir, "status", "--short")
	headSHA := gitCaptureOutput(t, thisWorktreeDir, "rev-parse", "HEAD")

	regressedRoot := addDetachedGitWorktree(t, thisWorktreeDir, headSHA)
	removeMilestoneCancelOpenACGuard(t, regressedRoot)
	stubMilestoneCancelledIncompleteACsCheckRule(t, regressedRoot)

	regressedBin, err := BuildBinary(context.Background(), regressedRoot, t.TempDir())
	if err != nil {
		t.Fatalf("building the regressed binary: %v", err)
	}

	const n = 8
	const attempts = 30
	seeds := make([]int64, attempts)
	for i := range seeds {
		seeds[i] = int64(i + 1)
	}
	newScenario := func(seed int64) Scenario {
		return NewConcurrentMilestoneRaceScenario(regressedBin, n, seed)
	}

	rw := newReportWriter(&countingWriter{})
	results, err := RunRepeated(newScenario, t.TempDir(), attempts, seedSequence(seeds...), rw, "", nil)
	if err != nil {
		t.Fatalf("RunRepeated against the regressed binary: %v", err)
	}

	detected := 0
	for _, r := range results {
		if !r.Passed {
			detected++
		}
	}
	t.Logf("regression detected in %d/%d attempts against the reintroduced G-0335-shaped regression", detected, attempts)
	if detected == 0 {
		t.Fatalf("expected at least 1/%d attempts against the regressed binary to report a violation, got 0 — AC-2's oracle failed to catch the reintroduced regression", attempts)
	}

	if statusAfter := gitCaptureOutput(t, thisWorktreeDir, "status", "--short"); statusAfter != statusBefore {
		t.Fatalf("this worktree's own tracked source was modified by this test — status before:\n%s\nstatus after:\n%s", statusBefore, statusAfter)
	}
}

// TestPatchExactlyOnce pins patchExactlyOnce's three outcomes — the
// anchor missing entirely, matching exactly once, and matching more
// than once — proving the sanity check fails loudly rather than
// silently patching the wrong spot (or nothing at all).
func TestPatchExactlyOnce(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		old     string
		newText string
		want    string
		wantErr bool
	}{
		{
			name:    "anchor missing entirely errors",
			content: "alpha\nbeta\ngamma\n",
			old:     "delta",
			newText: "epsilon",
			wantErr: true,
		},
		{
			name:    "anchor matching exactly once replaces cleanly",
			content: "alpha\nbeta\ngamma\n",
			old:     "beta",
			newText: "BETA",
			want:    "alpha\nBETA\ngamma\n",
			wantErr: false,
		},
		{
			name:    "anchor matching more than once errors, leaving content untouched",
			content: "alpha\nbeta\nalpha\n",
			old:     "alpha",
			newText: "ALPHA",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := patchExactlyOnce(tc.content, tc.old, tc.newText)
			if (err != nil) != tc.wantErr {
				t.Fatalf("patchExactlyOnce error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("patchExactlyOnce = %q, want %q", got, tc.want)
			}
		})
	}
}
