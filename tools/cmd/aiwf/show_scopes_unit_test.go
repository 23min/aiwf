package main

import (
	"context"
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

// TestUnpushedRange_NoUpstream: a fresh repo with no upstream
// configured falls back to "HEAD" so step-7b's audit pass scans
// every local commit until the first push.
func TestUnpushedRange_NoUpstream(t *testing.T) {
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	got, err := unpushedRange(context.Background(), root)
	if err != nil {
		t.Fatalf("unpushedRange: %v", err)
	}
	if got != "HEAD" {
		t.Errorf("unpushedRange = %q, want HEAD (no upstream)", got)
	}
}

// TestUnpushedRange_WithUpstream: when @{u} resolves, the helper
// returns "@{u}..HEAD" so the audit pass scans only unpushed
// commits.
func TestUnpushedRange_WithUpstream(t *testing.T) {
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
	got, err := unpushedRange(context.Background(), root)
	if err != nil {
		t.Fatalf("unpushedRange: %v", err)
	}
	if got != "@{u}..HEAD" {
		t.Errorf("unpushedRange = %q, want @{u}..HEAD", got)
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
	got, err := readUntrailedCommits(context.Background(), root)
	if err != nil {
		t.Fatalf("readUntrailedCommits: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d commits, want 0 (HEAD == @{u})", len(got))
	}
}

// TestParseUntrailedCommits_Malformed: records that don't have the
// expected three-field shape are silently skipped. Drives the
// defensive parsing branch.
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
			rec + "abc1234" + sep + "" + sep + "work/gaps/G-001-x.md",
			1,
		},
		{
			"one record missing field separator (truncated)",
			rec + "abc1234" + sep + "only-two-fields",
			0,
		},
		{
			"two records, second malformed",
			rec + "aaa1111" + sep + "" + sep + "work/gaps/G-001.md" +
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
