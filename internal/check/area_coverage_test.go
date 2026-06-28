package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestAreaCoverage exercises the covering law (M-0185): within a declared
// coverage root, an immediate child directory claimed by no area's glob fires
// area-unslotted; a fully-slotted root is silent. Plus the inert guards
// (AC-4), the single-level / IO-safe enumeration (AC-6), and the areamatch
// reuse (AC-3 — a `**` glob claims the bare project dir).
func TestAreaCoverage(t *testing.T) {
	t.Parallel()

	t.Run("unclaimed child fires one warning naming the dir and root", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		mkAreaDir(t, root, "projects/app-b") // claimed by no area
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			[]string{"projects"},
		)
		hits := findByCode(got, CodeAreaUnslotted)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 unslotted finding, got %d: %+v", len(hits), got)
		}
		if hits[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want %q", hits[0].Severity, SeverityWarning)
		}
		for _, want := range []string{"projects/app-b", "coverage root \"projects\""} {
			if !strings.Contains(hits[0].Message, want) {
				t.Errorf("message %q does not contain %q", hits[0].Message, want)
			}
		}
		if hits[0].Path != "projects/app-b" {
			t.Errorf("Path = %q, want %q", hits[0].Path, "projects/app-b")
		}
	})

	t.Run("fully-slotted root is silent (** glob claims the bare project dir)", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		mkAreaDir(t, root, "projects/app-b")
		// Whole-project `**` globs; areamatch.Match claims the bare project dir
		// (projects/app-a) via projects/app-a/** — proving the SSOT reuse.
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{
				{Name: "app-a", Paths: []string{"projects/app-a/**"}},
				{Name: "app-b", Paths: []string{"projects/app-b/**"}},
			},
			[]string{"projects"},
		)
		if len(got) != 0 {
			t.Errorf("a fully-slotted root must be silent, got %+v", got)
		}
	})

	t.Run("multiple unclaimed children each fire", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		mkAreaDir(t, root, "projects/app-b")
		mkAreaDir(t, root, "projects/app-c")
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			[]string{"projects"},
		)
		if hits := findByCode(got, CodeAreaUnslotted); len(hits) != 2 {
			t.Fatalf("want 2 unslotted findings (app-b, app-c), got %d: %+v", len(hits), got)
		}
	})

	t.Run("multiple roots: a literal-claimed root stays silent, an unclaimed sibling root fires", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "apps/web")
		mkAreaDir(t, root, "services/api") // unclaimed
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "web", Paths: []string{"apps/web/**"}}},
			[]string{"apps", "services"},
		)
		hits := findByCode(got, CodeAreaUnslotted)
		if len(hits) != 1 {
			t.Fatalf("want 1 unslotted finding (services/api), got %d: %+v", len(hits), got)
		}
		if !strings.Contains(hits[0].Message, "services/api") {
			t.Errorf("message %q should name services/api", hits[0].Message)
		}
	})

	t.Run("AC-4: no coverage root declared is inert", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-b") // unclaimed, but no root declared
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			nil,
		)
		if len(got) != 0 {
			t.Errorf("no coverage root must be inert, got %+v", got)
		}
	})

	t.Run("AC-4: no area declares paths is inert", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-b") // unclaimed
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "label-only", Paths: nil}},
			[]string{"projects"},
		)
		if len(got) != 0 {
			t.Errorf("a paths-less areas block must keep coverage inert, got %+v", got)
		}
	})

	t.Run("AC-6: single-level — a grandchild dir is never flagged", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a/sub") // sub is a grandchild of the root
		// The area claims the project dir itself (literal), NOT its subtree, so
		// projects/app-a/sub would be unslotted IF the check recursed. It must
		// not — only immediate children of the root are enumerated.
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a"}}},
			[]string{"projects"},
		)
		if len(got) != 0 {
			t.Errorf("single-level enumeration must not flag a grandchild, got %+v", got)
		}
	})

	t.Run("AC-6: a non-directory child is skipped", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		if err := os.WriteFile(filepath.Join(root, "projects", "notes.txt"), []byte("x\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			[]string{"projects"},
		)
		if len(got) != 0 {
			t.Errorf("a file child must be skipped (only dirs are projects), got %+v", got)
		}
	})

	t.Run("AC-6: a missing coverage root is silent (never fails on IO)", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir() // no `projects` dir created
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			[]string{"projects"},
		)
		if len(got) != 0 {
			t.Errorf("a missing coverage root must be silent, got %+v", got)
		}
	})

	t.Run("AC-6: empty root is silent (never fails on IO)", func(t *testing.T) {
		t.Parallel()
		got := AreaCoverage(&tree.Tree{Root: ""},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			[]string{"projects"},
		)
		if len(got) != 0 {
			t.Errorf("empty root must be silent, got %+v", got)
		}
	})

	t.Run("a malformed glob is indeterminate — the child is skipped, never fires or crashes", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		// A malformed character class makes areamatch.Match error; claimedByAnyArea
		// returns the error and AreaCoverage skips the child rather than firing a
		// false unslotted finding (malformed globs are a Tier-1 config-load
		// concern that cannot reach here in production).
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "bad", Paths: []string{"projects/["}}},
			[]string{"projects"},
		)
		if len(got) != 0 {
			t.Errorf("a malformed glob must leave the child unflagged, got %+v", got)
		}
	})
}

// TestApplyAreaRequiredStrict_EscalatesUnslotted pins the M-0185/AC-5 severity
// contract: under areas.required the area-unslotted warning is bumped to error
// so the pre-push hook blocks it, mirroring the area-unknown / area-dead-glob /
// area-overlap escalation. The entity-body-empty control proves the bump stays
// scoped to the area codes.
func TestApplyAreaRequiredStrict_EscalatesUnslotted(t *testing.T) {
	t.Parallel()
	build := func() []Finding {
		return []Finding{
			{Code: CodeAreaUnslotted, Severity: SeverityWarning},
			{Code: CodeEntityBodyEmpty, Severity: SeverityWarning},
		}
	}

	t.Run("required=true bumps unslotted to error", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, true)
		for _, f := range findings {
			switch f.Code {
			case CodeAreaUnslotted:
				if f.Severity != SeverityError {
					t.Errorf("unslotted severity = %v, want error under required", f.Severity)
				}
			case CodeEntityBodyEmpty:
				if f.Severity != SeverityWarning {
					t.Errorf("entity-body-empty severity = %v, want warning unchanged", f.Severity)
				}
			}
		}
	})

	t.Run("required=false leaves unslotted a warning", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, false)
		for _, f := range findings {
			if f.Code == CodeAreaUnslotted && f.Severity != SeverityWarning {
				t.Errorf("unslotted severity = %v, want warning when required=false", f.Severity)
			}
		}
	})
}
