package check

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestAreaOverlap exercises every branch: two areas whose globs claim a shared
// directory fire one warning naming both areas and the shared path; disjoint
// globs are silent; a nested glob (one area's claim a subset of another's)
// fires; fewer than two paths-carrying areas is inert; and an empty/unreadable
// root never fails on IO.
func TestAreaOverlap(t *testing.T) {
	t.Parallel()

	t.Run("two areas claiming a shared directory fire one warning", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/shared")
		got := AreaOverlap(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "left", Paths: []string{"projects/shared/**"}},
			{Name: "right", Paths: []string{"projects/shared/**"}},
		})
		hits := findByCode(got, CodeAreaOverlap)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 overlap finding, got %d: %+v", len(hits), got)
		}
		if hits[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want warning", hits[0].Severity)
		}
		for _, want := range []string{"left", "right", "projects/shared"} {
			if !strings.Contains(hits[0].Message, want) {
				t.Errorf("message %q does not name %q", hits[0].Message, want)
			}
		}
	})

	t.Run("disjoint globs are silent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		mkAreaDir(t, root, "projects/app-b")
		got := AreaOverlap(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "a", Paths: []string{"projects/app-a/**"}},
			{Name: "b", Paths: []string{"projects/app-b/**"}},
		})
		if len(got) != 0 {
			t.Errorf("disjoint areas must be silent, got %+v", got)
		}
	})

	t.Run("a nested claim (subset) fires overlap", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		// "everything" claims all of projects; "app" claims projects/app-a,
		// a subset — the two overlap on projects/app-a.
		got := AreaOverlap(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "everything", Paths: []string{"projects/**"}},
			{Name: "app", Paths: []string{"projects/app-a/**"}},
		})
		hits := findByCode(got, CodeAreaOverlap)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 overlap finding, got %d: %+v", len(hits), got)
		}
		if !strings.Contains(hits[0].Message, "projects/app-a") {
			t.Errorf("message %q should name the shared subtree", hits[0].Message)
		}
	})

	t.Run("three areas all sharing one dir fire one finding per pair", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/shared")
		got := AreaOverlap(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "a", Paths: []string{"projects/shared/**"}},
			{Name: "b", Paths: []string{"projects/shared/**"}},
			{Name: "c", Paths: []string{"projects/shared/**"}},
		})
		// pairs: a-b, a-c, b-c.
		if hits := findByCode(got, CodeAreaOverlap); len(hits) != 3 {
			t.Fatalf("want 3 pairwise overlap findings, got %d: %+v", len(hits), got)
		}
	})

	t.Run("fewer than two paths-carrying areas is inert", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		got := AreaOverlap(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "solo", Paths: []string{"projects/app-a/**"}},
			{Name: "label-only", Paths: nil},
		})
		if len(got) != 0 {
			t.Errorf("a single paths-carrying area cannot overlap, got %+v", got)
		}
	})

	t.Run("nil areas are inert", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if got := AreaOverlap(&tree.Tree{Root: root}, nil); len(got) != 0 {
			t.Errorf("nil areas must be inert, got %+v", got)
		}
	})

	t.Run("empty root is silent (never fails on IO)", func(t *testing.T) {
		t.Parallel()
		got := AreaOverlap(&tree.Tree{Root: ""}, []AreaPaths{
			{Name: "a", Paths: []string{"x/**"}},
			{Name: "b", Paths: []string{"x/**"}},
		})
		if len(got) != 0 {
			t.Errorf("empty root must be silent, got %+v", got)
		}
	})

	t.Run("unreadable root is silent (never fails on IO)", func(t *testing.T) {
		t.Parallel()
		missing := filepath.Join(t.TempDir(), "does-not-exist")
		got := AreaOverlap(&tree.Tree{Root: missing}, []AreaPaths{
			{Name: "a", Paths: []string{"x/**"}},
			{Name: "b", Paths: []string{"x/**"}},
		})
		if len(got) != 0 {
			t.Errorf("unreadable root must be silent, got %+v", got)
		}
	})

	t.Run("malformed globs are skipped, never fire or crash", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		// Malformed globs make MatchFS error; each is skipped (malformed is a
		// Tier-1 config-load concern), so the areas' match sets stay empty and
		// no overlap is reported.
		got := AreaOverlap(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "bad-a", Paths: []string{"projects/["}},
			{Name: "bad-b", Paths: []string{"projects/["}},
		})
		if len(got) != 0 {
			t.Errorf("malformed globs must be skipped, got %+v", got)
		}
	})
}

// TestApplyAreaRequiredStrict_EscalatesOverlap pins the M-0180/AC-3 severity
// contract: under areas.required the overlap warning is bumped to error so the
// pre-push hook blocks it, the same escalation dead-glob and area-unknown get.
func TestApplyAreaRequiredStrict_EscalatesOverlap(t *testing.T) {
	t.Parallel()
	build := func() []Finding {
		return []Finding{
			{Code: CodeAreaOverlap, Severity: SeverityWarning},
			{Code: CodeEntityBodyEmpty, Severity: SeverityWarning},
		}
	}

	t.Run("required=true bumps overlap to error", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, true)
		for _, f := range findings {
			switch f.Code {
			case CodeAreaOverlap:
				if f.Severity != SeverityError {
					t.Errorf("overlap severity = %v, want error under required", f.Severity)
				}
			case CodeEntityBodyEmpty:
				if f.Severity != SeverityWarning {
					t.Errorf("entity-body-empty severity = %v, want warning unchanged", f.Severity)
				}
			}
		}
	})

	t.Run("required=false leaves overlap a warning", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, false)
		for _, f := range findings {
			if f.Code == CodeAreaOverlap && f.Severity != SeverityWarning {
				t.Errorf("overlap severity = %v, want warning when required=false", f.Severity)
			}
		}
	})
}
