package areamatch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
)

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

// TestMatchFS exercises the filesystem-walk variant against a real temp tree:
// a glob with live matches returns non-empty, a dead glob returns empty, a
// wildcard-free literal returns the located path, and a malformed glob errors.
func TestMatchFS(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "projects", "app-a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "projects", "app-a", "main.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	fsys := os.DirFS(root)

	t.Run("live glob returns non-empty", func(t *testing.T) {
		t.Parallel()
		got, err := MatchFS(fsys, "projects/app-a/**")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) == 0 {
			t.Errorf("want non-empty matches for a live glob, got none")
		}
	})

	t.Run("dead glob returns empty", func(t *testing.T) {
		t.Parallel()
		got, err := MatchFS(fsys, "projects/ghost/**")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want empty for a dead glob, got %v", got)
		}
	})

	t.Run("wildcard-free literal returns the located path", func(t *testing.T) {
		t.Parallel()
		got, err := MatchFS(fsys, "projects/app-a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0] != "projects/app-a" {
			t.Errorf("want [projects/app-a], got %v", got)
		}
	})

	t.Run("malformed glob errors", func(t *testing.T) {
		t.Parallel()
		if _, err := MatchFS(fsys, "a["); err == nil {
			t.Errorf("want error for malformed glob, got nil")
		}
	})
}

// TestMatchesAny exercises the early-terminating boolean-any variant: a live
// glob is true, a dead glob is false, a wildcard-free literal that exists is
// true, and a malformed glob errors.
func TestMatchesAny(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "projects", "app-a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "projects", "app-a", "main.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	fsys := os.DirFS(root)

	cases := []struct {
		name    string
		glob    string
		want    bool
		wantErr bool
	}{
		{"live glob", "projects/app-a/**", true, false},
		{"dead glob", "projects/ghost/**", false, false},
		{"wildcard-free literal that exists", "projects/app-a", true, false},
		{"wildcard-free literal that is absent", "projects/app-q", false, false},
		{"malformed glob", "a[", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := MatchesAny(fsys, tc.glob)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("MatchesAny(%q): want error, got nil", tc.glob)
				}
				return
			}
			if err != nil {
				t.Fatalf("MatchesAny(%q): unexpected error: %v", tc.glob, err)
			}
			if got != tc.want {
				t.Errorf("MatchesAny(%q) = %v, want %v", tc.glob, got, tc.want)
			}
		})
	}
}

// TestValidate pins the Tier-1 syntax gate: well-formed globs (literal, '*',
// '**', alternation) pass; malformed ones (unterminated class / brace) error,
// and the error wraps doublestar.ErrBadPattern so callers can classify it.
func TestValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		glob    string
		wantErr bool
	}{
		{"literal", "projects/app-a", false},
		{"single star", "projects/*", false},
		{"doublestar", "projects/**", false},
		{"alternation", "projects/{app-a,app-b}", false},
		{"unterminated class", "projects/[app", true},
		{"unterminated brace", "projects/{a,b", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := Validate(tc.glob)
			if tc.wantErr && err == nil {
				t.Errorf("Validate(%q): want error, got nil", tc.glob)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Validate(%q): unexpected error: %v", tc.glob, err)
			}
		})
	}

	t.Run("malformed error wraps ErrBadPattern", func(t *testing.T) {
		t.Parallel()
		err := Validate("a[")
		if !errors.Is(err, doublestar.ErrBadPattern) {
			t.Errorf("Validate error = %v, want it to wrap doublestar.ErrBadPattern", err)
		}
	})
}
