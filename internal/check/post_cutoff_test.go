package check

import (
	"context"
	"strings"
	"testing"
)

// post_cutoff_test.go — G-0218 Patch 2 walker tests. The rule's unit
// tests (trailer_verb_unknown_test.go) drive the rule with synthetic
// postCutoffSHAs maps directly; this file pins the gather walker's
// `git rev-list <cutoff>..HEAD` invocation against real fixture
// repos to catch protocol drift the synthetic-map tests can't see.

// TestWalkPostCutoffSHAs_NonGitRoot_ReturnsNil pins the cheapest
// fallback path: a directory with no `.git/` returns nil rather than
// erroring. This is the contract the gather layer relies on when
// `aiwf check` runs outside a git repo (`aiwf doctor` in a fresh
// tmpdir).
func TestWalkPostCutoffSHAs_NonGitRoot_ReturnsNil(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	got := WalkPostCutoffSHAs(context.Background(), tmp)
	if got != nil {
		t.Errorf("got = %+v, want nil (non-git root)", got)
	}
}

// TestWalkPostCutoffSHAs_EmptyRoot_ReturnsNil pins the empty-string-
// root fallback. The gather layer passes a resolved repo root so
// this should never fire in practice, but the contract documents
// the safe degrade.
func TestWalkPostCutoffSHAs_EmptyRoot_ReturnsNil(t *testing.T) {
	t.Parallel()
	got := WalkPostCutoffSHAs(context.Background(), "")
	if got != nil {
		t.Errorf("got = %+v, want nil (empty root)", got)
	}
}

// TestWalkPostCutoffSHAs_CutoffNotInHistory_ReturnsNil pins the
// fallback for the most common real-world failure mode: a clone
// where HookInstallSHA was never imported (a fork that diverged
// from main before Patch 1 landed; a shallow clone; an unrelated
// repo someone happened to run aiwf inside). `git rev-list
// <unknown-sha>..HEAD` errors; the walker swallows and returns
// nil, which the rule consumes as "no post-cutoff commits" → all
// warnings (the G-0150 baseline).
func TestWalkPostCutoffSHAs_CutoffNotInHistory_ReturnsNil(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	// One commit so HEAD exists; HookInstallSHA is genuinely absent
	// from this fresh repo's object DB.
	r.gitCommit("seed")
	got := WalkPostCutoffSHAs(context.Background(), r.root)
	if got != nil {
		t.Errorf("got = %+v, want nil (HookInstallSHA not in fresh-repo history)", got)
	}
}

// TestWalkPostCutoffSHAsFromCutoff_HappyPath pins the load-bearing
// behavior: when the cutoff SHA IS in HEAD's ancestry, the walker
// returns exactly the set of commits strictly newer than the cutoff
// (the cutoff itself is excluded — the hook-install commit cannot
// be policed by the hook it installs).
//
// Uses the parameterized inner helper because production
// HookInstallSHA points at a specific main-branch commit that
// fresh-fixture repos don't have; the inner helper takes any SHA.
func TestWalkPostCutoffSHAsFromCutoff_HappyPath(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	preCutoffSHA := r.gitCommit("pre-cutoff-1")
	cutoffSHA := r.gitCommit("the-cutoff-commit")
	postSHA1 := r.gitCommit("post-cutoff-1")
	postSHA2 := r.gitCommit("post-cutoff-2")

	got := walkPostCutoffSHAsFromCutoff(context.Background(), r.root, cutoffSHA)
	if got == nil {
		t.Fatal("got = nil, want non-empty set")
	}
	if len(got) != 2 {
		t.Errorf("got = %d SHAs (%+v), want 2 (post-1, post-2 — cutoff excluded)", len(got), got)
	}
	if !got[postSHA1] {
		t.Errorf("post-cutoff SHA %s missing from set %+v", postSHA1, got)
	}
	if !got[postSHA2] {
		t.Errorf("post-cutoff SHA %s missing from set %+v", postSHA2, got)
	}
	if got[cutoffSHA] {
		t.Errorf("cutoff SHA %s must NOT be in the post-cutoff set (range excludes it)", cutoffSHA)
	}
	if got[preCutoffSHA] {
		t.Errorf("pre-cutoff SHA %s must NOT be in the post-cutoff set", preCutoffSHA)
	}
}

// TestWalkPostCutoffSHAsFromCutoff_HeadEqualsCutoff_ReturnsNil pins
// the edge case where HEAD IS the cutoff commit (no descendants
// yet). `git rev-list <cutoff>..<cutoff>` produces no commits; the
// walker collapses an empty set to nil so the rule's nil-fallback
// contract holds.
func TestWalkPostCutoffSHAsFromCutoff_HeadEqualsCutoff_ReturnsNil(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	cutoffSHA := r.gitCommit("the-cutoff-commit")
	got := walkPostCutoffSHAsFromCutoff(context.Background(), r.root, cutoffSHA)
	if got != nil {
		t.Errorf("got = %+v, want nil (no commits descend from the cutoff)", got)
	}
}

// TestWalkPostCutoffSHAsFromCutoff_EmptyCutoff_ReturnsNil pins the
// degenerate-input safety: an empty cutoff string short-circuits
// before shelling to git, so a misconfigured caller that passes ""
// doesn't get a `git rev-list "..HEAD"` shape (which would
// enumerate every commit and silently flip every finding to error).
func TestWalkPostCutoffSHAsFromCutoff_EmptyCutoff_ReturnsNil(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.gitCommit("any-commit")
	got := walkPostCutoffSHAsFromCutoff(context.Background(), r.root, "")
	if got != nil {
		t.Errorf("got = %+v, want nil (empty cutoff must short-circuit)", got)
	}
}

// TestHookInstallSHA_IsFullLength is a structural assertion the
// constant carries the canonical full 40-char SHA form. `git rev-
// list <abbrev>..HEAD` would still work for short SHAs, but pinning
// the full form keeps audit-trail clarity (`aiwf history` and
// addressed_by_commit refs are full-SHA-tracked) and prevents the
// "did someone abbreviate this?" rabbit hole at a future review.
func TestHookInstallSHA_IsFullLength(t *testing.T) {
	t.Parallel()
	if len(HookInstallSHA) != 40 {
		t.Errorf("HookInstallSHA length = %d, want 40 (full SHA)", len(HookInstallSHA))
	}
	if strings.ContainsAny(HookInstallSHA, "GHIJKLMNOPQRSTUVWXYZghijklmnopqrstuvwxyz") {
		t.Errorf("HookInstallSHA = %q, want lowercase hex only", HookInstallSHA)
	}
}
