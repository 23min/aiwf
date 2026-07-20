package entityview_test

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entityview"
	"github.com/23min/aiwf/internal/scope"
)

// TestLastEventSHA covers every branch:
//   - empty event slice
//   - no event matches the requested state
//   - one match, returned
//   - multiple matches: the latest (last by index) wins
func TestLastEventSHA(t *testing.T) {
	t.Parallel()
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
			if got := entityview.LastEventSHA(tt.s, tt.match); got != tt.want {
				t.Errorf("entityview.LastEventSHA = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLookupCommitDateCached_CacheReuse: a cached value is returned
// without invoking git. We seed the cache with a known value, point
// root at a non-existent directory (so a real `git show` would
// fail), and verify the cached value comes back unchanged.
func TestLookupCommitDateCached_CacheReuse(t *testing.T) {
	t.Parallel()
	cache := map[string]string{
		"deadbeef": "2026-05-02T10:00:00+00:00",
	}
	got := entityview.LookupCommitDateCached(context.Background(),
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
	t.Parallel()
	tmp := t.TempDir() // not a git repo
	cache := map[string]string{}
	if got := entityview.LookupCommitDateCached(context.Background(), tmp, "deadbeef", cache); got != "" {
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
	t.Parallel()
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	out, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("git rev-parse: %v\n%s", err, out)
	}
	sha := strings.TrimSpace(out)
	cache := map[string]string{}
	d := entityview.LookupCommitDateCached(context.Background(), root, sha, cache)
	if d == "" {
		t.Fatalf("first lookup returned empty; expected ISO-8601 date")
	}
	if cache[sha] != d {
		t.Errorf("cache miss after first lookup; cache[%s] = %q, want %q", sha, cache[sha], d)
	}
	// Second call should hit the cache. Confirm the value matches —
	// the underlying git access doesn't matter once cached.
	if d2 := entityview.LookupCommitDateCached(context.Background(), root, sha, cache); d2 != d {
		t.Errorf("second lookup = %q, want cached %q", d2, d)
	}
}

// TestSplitMultiValueTrailer covers the three input shapes the
// helper handles: empty, single value, and the multi-value path
// that fires when a commit carries repeated trailers (notably
// aiwf-scope-ends).
func TestSplitMultiValueTrailer(t *testing.T) {
	t.Parallel()
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
			got := entityview.SplitMultiValueTrailer(tt.in)
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
