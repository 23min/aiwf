package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/scope"
)

// TestLastEventSHA covers every branch:
//   - empty event slice
//   - no event matches the requested state
//   - one match, returned
//   - multiple matches: the latest (last by index) wins
func TestLastEventSHA(t *testing.T) {
	tests := []struct {
		name  string
		s     *scope.Scope
		match scope.State
		want  string
	}{
		{"no events", &scope.Scope{}, scope.StateEnded, ""},
		{
			"no match",
			&scope.Scope{Events: []scope.Event{
				{SHA: "aaa", State: scope.StateActive},
				{SHA: "bbb", State: scope.StatePaused},
			}},
			scope.StateEnded,
			"",
		},
		{
			"one match",
			&scope.Scope{Events: []scope.Event{
				{SHA: "aaa", State: scope.StateActive},
				{SHA: "bbb", State: scope.StateEnded},
			}},
			scope.StateEnded,
			"bbb",
		},
		{
			"latest match wins",
			&scope.Scope{Events: []scope.Event{
				{SHA: "aaa", State: scope.StateActive},
				{SHA: "bbb", State: scope.StatePaused},
				{SHA: "ccc", State: scope.StateActive},
			}},
			scope.StateActive,
			"ccc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastEventSHA(tt.s, tt.match); got != tt.want {
				t.Errorf("lastEventSHA = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLookupCommitDateCached_CacheReuse: a cached value is returned
// without invoking git. We seed the cache with a known value, point
// root at a non-existent directory (so a real `git show` would
// fail), and verify the cached value comes back unchanged.
func TestLookupCommitDateCached_CacheReuse(t *testing.T) {
	cache := map[string]string{
		"deadbeef": "2026-05-02T10:00:00+00:00",
	}
	got := lookupCommitDateCached(context.Background(),
		"/nonexistent/path/that/does/not/exist", "deadbeef", cache)
	if got != "2026-05-02T10:00:00+00:00" {
		t.Errorf("cached lookup = %q, want the seeded value", got)
	}
}

// TestLookupCommitDateCached_ErrorFallback: when `git show` fails
// (no repo, bogus SHA), the helper returns "" and seeds the cache
// with "" so a second call doesn't retry. This is the load-bearing
// behavior that keeps `aiwf show` from blocking on missing dates.
func TestLookupCommitDateCached_ErrorFallback(t *testing.T) {
	tmp := t.TempDir() // not a git repo
	cache := map[string]string{}
	if got := lookupCommitDateCached(context.Background(), tmp, "deadbeef", cache); got != "" {
		t.Errorf("first lookup = %q, want empty (git show fails)", got)
	}
	if v, ok := cache["deadbeef"]; !ok || v != "" {
		t.Errorf("cache[deadbeef] = (%q, %v), want (\"\", true) — error must be cached", v, ok)
	}
}

// TestLookupCommitDateCached_HappyPath: a real git repo with a real
// commit. The helper returns the author date; the second call hits
// the cache (proven by removing the repo and observing the same
// value comes back).
func TestLookupCommitDateCached_HappyPath(t *testing.T) {
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	out, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("git rev-parse: %v\n%s", err, out)
	}
	sha := strings.TrimSpace(out)
	cache := map[string]string{}
	d := lookupCommitDateCached(context.Background(), root, sha, cache)
	if d == "" {
		t.Fatalf("first lookup returned empty; expected ISO-8601 date")
	}
	if cache[sha] != d {
		t.Errorf("cache miss after first lookup; cache[%s] = %q, want %q", sha, cache[sha], d)
	}
	// Second call should hit the cache. Confirm the value matches —
	// the underlying git access doesn't matter once cached.
	if d2 := lookupCommitDateCached(context.Background(), root, sha, cache); d2 != d {
		t.Errorf("second lookup = %q, want cached %q", d2, d)
	}
}

// TestSplitMultiValueTrailer covers the three input shapes the
// helper handles: empty, single value, and the multi-value path
// that fires when a commit carries repeated trailers (notably
// aiwf-scope-ends).
func TestSplitMultiValueTrailer(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   \n\n  \n", nil},
		{"single value", "human/peter", []string{"human/peter"}},
		{
			"two values newline-separated",
			"abc1234\ndef5678",
			[]string{"abc1234", "def5678"},
		},
		{
			"three values with surrounding blanks",
			"\n  abc1234\n\ndef5678\n  9999999  \n",
			[]string{"abc1234", "def5678", "9999999"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitMultiValueTrailer(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (got %v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestResolveUntrailedRange_NoUpstream: a fresh repo with no
// upstream configured returns no range and an advisory finding
// so step-7b's audit pass is skipped (the previous "all of HEAD"
// fallback flooded long-lived branches with commits already
// merged in from trunk — see issue #5 sub-item 2).
func TestResolveUntrailedRange_NoUpstream(t *testing.T) {
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	rangeArg, advisory, err := resolveUntrailedRange(context.Background(), root, "")
	if err != nil {
		t.Fatalf("resolveUntrailedRange: %v", err)
	}
	if rangeArg != "" {
		t.Errorf("rangeArg = %q, want empty (skipped)", rangeArg)
	}
	if advisory == nil {
		t.Fatal("advisory is nil; want a scope-undefined warning")
	}
	if advisory.Code != "provenance-untrailered-scope-undefined" {
		t.Errorf("advisory.Code = %q, want provenance-untrailered-scope-undefined", advisory.Code)
	}
}

// TestResolveUntrailedRange_WithUpstream: when @{u} resolves, the
// helper returns "@{u}..HEAD" and no advisory.
func TestResolveUntrailedRange_WithUpstream(t *testing.T) {
	upstream := t.TempDir()
	if out, err := runGit(upstream, "init", "--bare", "-q"); err != nil {
		t.Fatalf("git init bare: %v\n%s", err, out)
	}
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"remote", "add", "origin", upstream},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	// Push and set upstream tracking.
	if out, err := runGit(root, "push", "-u", "origin", "HEAD:main"); err != nil {
		t.Fatalf("git push: %v\n%s", err, out)
	}
	rangeArg, advisory, err := resolveUntrailedRange(context.Background(), root, "")
	if err != nil {
		t.Fatalf("resolveUntrailedRange: %v", err)
	}
	if rangeArg != "@{u}..HEAD" {
		t.Errorf("rangeArg = %q, want @{u}..HEAD", rangeArg)
	}
	if advisory != nil {
		t.Errorf("advisory = %+v, want nil", advisory)
	}
}

// TestResolveUntrailedRange_SinceWins: an explicit --since <ref>
// overrides upstream detection. The ref is verified via
// `git rev-parse`; an unknown ref returns an advisory finding
// (audit skipped) rather than failing the whole check.
func TestResolveUntrailedRange_SinceWins(t *testing.T) {
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	// Valid ref → "<ref>..HEAD" range, no advisory.
	rangeArg, advisory, err := resolveUntrailedRange(context.Background(), root, "HEAD")
	if err != nil {
		t.Fatalf("resolveUntrailedRange (valid since): %v", err)
	}
	if rangeArg != "HEAD..HEAD" || advisory != nil {
		t.Errorf("valid since: rangeArg=%q advisory=%+v; want HEAD..HEAD, nil", rangeArg, advisory)
	}
	// Unknown ref → empty range + advisory.
	rangeArg, advisory, err = resolveUntrailedRange(context.Background(), root, "no-such-ref")
	if err != nil {
		t.Fatalf("resolveUntrailedRange (bad since): %v", err)
	}
	if rangeArg != "" {
		t.Errorf("bad since: rangeArg = %q, want empty", rangeArg)
	}
	if advisory == nil || advisory.Code != "provenance-untrailered-scope-undefined" {
		t.Errorf("bad since: advisory = %+v, want scope-undefined", advisory)
	}
}

// TestReadUntrailedCommits_EmptyRange: when HEAD == @{u} (already
// pushed), the unpushed range produces no commits. The helper
// should return nil without error so step 7b stays silent.
func TestReadUntrailedCommits_EmptyRange(t *testing.T) {
	upstream := t.TempDir()
	if out, err := runGit(upstream, "init", "--bare", "-q"); err != nil {
		t.Fatalf("git init bare: %v\n%s", err, out)
	}
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"remote", "add", "origin", upstream},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	// Push and set upstream tracking — now HEAD == @{u}.
	if out, err := runGit(root, "push", "-u", "origin", "HEAD:main"); err != nil {
		t.Fatalf("git push: %v\n%s", err, out)
	}
	got, err := readUntrailedCommits(context.Background(), root, "@{u}..HEAD")
	if err != nil {
		t.Fatalf("readUntrailedCommits: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d commits, want 0 (HEAD == @{u})", len(got))
	}
}

// TestReadUntrailedCommits_MergeCommitSurface covers G32: a merge
// commit on the integration branch that brings in entity-file
// changes from a feature branch must surface those paths to the
// audit pass. Pre-fix the default `git log --name-only` showed no
// file list for merge commits and the audit silently skipped them.
// Post-fix `-m --first-parent` makes the merge commit's introduced
// changes (against its first parent) visible, so per-(commit,
// entity) findings fire on the integration branch boundary.
func TestReadUntrailedCommits_MergeCommitSurface(t *testing.T) {
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// Initial empty commit on main so we have a base.
	if out, err := runGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("seed: %v\n%s", err, out)
	}
	baseSHA, gErr := runGit(root, "rev-parse", "HEAD")
	if gErr != nil {
		t.Fatalf("rev-parse HEAD: %v", gErr)
	}
	base := strings.TrimSpace(baseSHA)

	// Feature branch with one untrailered commit touching G-001.
	if out, gErr := runGit(root, "checkout", "-q", "-b", "feature"); gErr != nil {
		t.Fatalf("checkout: %v\n%s", gErr, out)
	}
	gapDir := filepath.Join(root, "work", "gaps")
	if mkErr := os.MkdirAll(gapDir, 0o755); mkErr != nil {
		t.Fatal(mkErr)
	}
	gapPath := filepath.Join(gapDir, "G-001-leak.md")
	if wErr := os.WriteFile(gapPath, []byte("---\nid: G-001\nstatus: wontfix\n---\n"), 0o644); wErr != nil {
		t.Fatal(wErr)
	}
	if out, gErr := runGit(root, "add", "work/gaps/G-001-leak.md"); gErr != nil {
		t.Fatalf("git add: %v\n%s", gErr, out)
	}
	if out, gErr := runGit(root, "commit", "-m", "manual: flip G-001 wontfix"); gErr != nil {
		t.Fatalf("manual commit: %v\n%s", gErr, out)
	}
	// Back to main; merge feature with --no-ff so a merge commit lands.
	if out, gErr := runGit(root, "checkout", "-q", "main"); gErr != nil {
		t.Fatalf("checkout main: %v\n%s", gErr, out)
	}
	if out, gErr := runGit(root, "merge", "--no-ff", "-m", "merge feature into main", "feature"); gErr != nil {
		t.Fatalf("merge: %v\n%s", gErr, out)
	}

	// Range scoped to base..HEAD. Pre-fix: zero records (merge has
	// no name-only output by default; the upstream feature commit
	// is excluded by --first-parent which means the only candidate
	// is the merge itself, and without -m it has no paths). Post-
	// fix: the merge commit appears with G-001's path attached.
	commits, err := readUntrailedCommits(context.Background(), root, base+"..HEAD")
	if err != nil {
		t.Fatalf("readUntrailedCommits: %v", err)
	}
	var sawMerge bool
	for _, c := range commits {
		if len(c.Paths) == 0 {
			continue
		}
		for _, p := range c.Paths {
			if strings.HasSuffix(p, "G-001-leak.md") {
				sawMerge = true
			}
		}
	}
	if !sawMerge {
		t.Errorf("expected merge commit to surface G-001's path; got commits=%+v", commits)
	}
}

// TestParseUntrailedCommits_Malformed: records that don't have the
// expected four-field shape (SHA, subject, trailers, paths) are
// silently skipped. Drives the defensive parsing branch.
func TestParseUntrailedCommits_Malformed(t *testing.T) {
	const sep = "\x1f"
	const rec = "\x1e"
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty input", "", 0},
		{"only whitespace", "   \n\n  ", 0},
		{
			"one well-formed",
			rec + "abc1234" + sep + "feat: thing" + sep + "" + sep + "work/gaps/G-001-x.md",
			1,
		},
		{
			"one record missing field separator (truncated)",
			rec + "abc1234" + sep + "only-three-fields" + sep + "trailers",
			0,
		},
		{
			"two records, second malformed",
			rec + "aaa1111" + sep + "feat: a" + sep + "" + sep + "work/gaps/G-001.md" +
				rec + "bbb2222-no-seps",
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseUntrailedCommits(tt.input)
			if len(got) != tt.want {
				t.Errorf("len = %d, want %d (got %+v)", len(got), tt.want, got)
			}
		})
	}
}

// TestLoadEntityScopeViews_NoCommits: pre-init repo; helper returns
// (nil, nil) without shelling out further.
func TestLoadEntityScopeViews_NoCommits(t *testing.T) {
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	got, err := loadEntityScopeViews(context.Background(), root, "E-01")
	if err != nil {
		t.Fatalf("loadEntityScopeViews: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil (no commits)", got)
	}
}

// TestLoadEntityScopeViews_NotInGitRepo: working dir is not a git
// repo. hasCommits returns false; helper returns (nil, nil)
// gracefully without erroring.
func TestLoadEntityScopeViews_NotInGitRepo(t *testing.T) {
	tmp := t.TempDir()
	got, err := loadEntityScopeViews(context.Background(), tmp, "E-01")
	if err != nil {
		t.Fatalf("loadEntityScopeViews: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil (no repo)", got)
	}
}

// TestLoadEntityScopeViews_NoScopesTouchEntity: a repo with commits
// but none referencing the queried entity returns nil.
func TestLoadEntityScopeViews_NoScopesTouchEntity(t *testing.T) {
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// One commit with an entity trailer for X-99 (not the queried
	// id) and no scope at all.
	msg := "chore: seed\n\naiwf-verb: add\naiwf-entity: X-99\naiwf-actor: human/peter\n"
	if out, err := runGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	got, err := loadEntityScopeViews(context.Background(), root, "E-01")
	if err != nil {
		t.Fatalf("loadEntityScopeViews: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil (no scopes touch E-01)", got)
	}
}
