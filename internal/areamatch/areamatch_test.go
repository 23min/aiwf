package areamatch

import "testing"

// Match cases are sourced from the doublestar/v4 pattern grammar
// (https://github.com/bmatcuk/doublestar#patterns): a single '*' matches
// within one path segment, '**' matches across separators (zero or more
// segments), and a malformed character class yields an error. Globs and paths
// are '/'-separated and repo-relative.
func TestMatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		glob    string
		path    string
		want    bool
		wantErr bool
	}{
		{"literal exact match", "projects/app-a", "projects/app-a", true, false},
		{"literal mismatch", "projects/app-a", "projects/app-b", false, false},
		{"single star matches one segment", "projects/*", "projects/app-a", true, false},
		{"single star does not cross separator", "projects/*", "projects/app-a/sub", false, false},
		{"doublestar crosses separators", "projects/**", "projects/app-a/sub/deep", true, false},
		{"doublestar scoped to its prefix", "projects/**", "other/app-a", false, false},
		{"multi-segment with star matches", "a/b/*", "a/b/c", true, false},
		{"multi-segment mismatch on middle segment", "a/b/*", "a/x/c", false, false},
		{"malformed character class errors", "a[", "a", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Match(tc.glob, tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Match(%q, %q): want error, got nil", tc.glob, tc.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%q, %q): unexpected error: %v", tc.glob, tc.path, err)
			}
			if got != tc.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tc.glob, tc.path, got, tc.want)
			}
		})
	}
}
