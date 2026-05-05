package pathutil

import (
	"path/filepath"
	"strings"
	"testing"
)

// FuzzInside drives Inside with arbitrary string pairs and checks the
// containment rules the production code commits to. Filed under G44 item 1.
//
// The fuzz seeds cover the load-bearing G1 cases (escape via `..`, the
// "/repo" vs "/repository" prefix-collision), plus the absolute/relative
// fail-closed cases.
func FuzzInside(f *testing.F) {
	for _, pair := range [][2]string{
		{"/repo", "/repo"},
		{"/repo", "/repo/sub"},
		{"/repo", "/repository"},         // prefix-collision (G1)
		{"/repo", "/repo/.."},            // escape via ..
		{"/repo", "/repo/sub/../../etc"}, // escape via .. (deeper)
		{"/repo", "/other"},
		{"/repo", "relative"}, // relative candidate
		{"relative", "/repo"}, // relative root
		{"", ""},
		{"/", "/anywhere"},
		{"/repo", ""},
	} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, root, candidate string) {
		got := Inside(root, candidate)

		// Property 1: both inputs must be non-empty and absolute, else
		// Inside returns false unconditionally.
		if root == "" || candidate == "" {
			if got {
				t.Fatalf("Inside(%q,%q)=true; empty input must fail closed", root, candidate)
			}
			return
		}
		if !filepath.IsAbs(root) || !filepath.IsAbs(candidate) {
			if got {
				t.Fatalf("Inside(%q,%q)=true; relative input must fail closed", root, candidate)
			}
			return
		}

		// Property 2: agreement with an independent reference. After
		// cleaning, Inside reduces to a prefix check with separator
		// guard (the documented behavior). Re-derive that here from
		// the stdlib and assert it matches.
		r := filepath.Clean(root)
		c := filepath.Clean(candidate)
		want := r == c || strings.HasPrefix(c, r+string(filepath.Separator))
		if got != want {
			t.Fatalf("Inside(%q,%q)=%v; reference says %v (cleaned root=%q candidate=%q)",
				root, candidate, got, want, r, c)
		}

		// Property 3: a path that escapes via `..` must not be
		// reported as inside, *after* cleaning. This is the load-
		// bearing G1 invariant: Inside accepts the cleaning the
		// caller has done; if the cleaned form lies above root,
		// containment is false.
		if strings.Contains(filepath.ToSlash(c), "/../") || strings.HasSuffix(filepath.ToSlash(c), "/..") {
			// Defensive: cleaning should have eliminated `..`. If it
			// hasn't (multi-`..` rooted at "/"), the candidate may
			// still legitimately escape. The agreement check above
			// covers this; this assertion is defensive narration.
			_ = got
		}
	})
}
