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
		if hits[0].Field != "areas.coverage_roots" {
			t.Errorf("Field = %q, want %q", hits[0].Field, "areas.coverage_roots")
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

	t.Run("AC-8: coverage_roots declared but no area has paths fires area-coverage-no-paths (not silent)", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-b") // unclaimed, but the path oracle is dormant
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "label-only", Paths: nil}},
			[]string{"projects"},
		)
		hits := findByCode(got, CodeAreaCoverageNoPaths)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 area-coverage-no-paths finding, got %d: %+v", len(hits), got)
		}
		if hits[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want %q", hits[0].Severity, SeverityWarning)
		}
		// It must NOT degenerate into a per-child area-unslotted storm.
		if u := findByCode(got, CodeAreaUnslotted); len(u) != 0 {
			t.Errorf("no-paths must not emit area-unslotted, got %+v", u)
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

	t.Run("dot-prefixed children (.git/.claude) are skipped under a '.' root", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, ".git")
		mkAreaDir(t, root, ".claude")
		mkAreaDir(t, root, "app-a")  // claimed
		mkAreaDir(t, root, "orphan") // unclaimed, a real (non-dot) project
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"app-a/**"}}},
			[]string{"."},
		)
		hits := findByCode(got, CodeAreaUnslotted)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 unslotted finding (orphan only; .git/.claude must be skipped), got %d: %+v", len(hits), got)
		}
		if hits[0].Path != "orphan" {
			t.Errorf("Path = %q, want %q (dot-prefixed dirs must be skipped)", hits[0].Path, "orphan")
		}
	})

	t.Run("AC-8: a non-existent coverage root fires area-coverage-root-missing", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir() // no `projects` dir created
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			[]string{"projects"},
		)
		hits := findByCode(got, CodeAreaCoverageRootMissing)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 area-coverage-root-missing finding, got %d: %+v", len(hits), got)
		}
		if !strings.Contains(hits[0].Message, "projects") {
			t.Errorf("message %q should name the dead root", hits[0].Message)
		}
		if hits[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want %q", hits[0].Severity, SeverityWarning)
		}
		if hits[0].Path != "projects" {
			t.Errorf("Path = %q, want %q", hits[0].Path, "projects")
		}
	})

	t.Run("AC-8: a coverage root that names a file fires area-coverage-root-missing", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "projects"), []byte("not a dir\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			[]string{"projects"},
		)
		if hits := findByCode(got, CodeAreaCoverageRootMissing); len(hits) != 1 {
			t.Fatalf("a file coverage root must fire area-coverage-root-missing, got %d: %+v", len(hits), got)
		}
	})

	t.Run("AC-6: an indeterminate stat error (root under a file) is skipped, never fails on IO", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		// `afile` is a file; the coverage root `afile/sub` makes os.Stat fail
		// with ENOTDIR — NOT fs.ErrNotExist — exercising the indeterminate
		// branch that is skipped silently rather than flagged.
		if err := os.WriteFile(filepath.Join(root, "afile"), []byte("x\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		got := AreaCoverage(&tree.Tree{Root: root},
			[]AreaPaths{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
			[]string{"afile/sub"},
		)
		if len(got) != 0 {
			t.Errorf("an indeterminate stat error must be skipped (no finding), got %+v", got)
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

// TestApplyAreaRequiredStrict_EscalatesCoverageFindings pins the M-0185 AC-5 +
// AC-8 severity contract: under areas.required all three coverage findings
// (area-unslotted, area-coverage-root-missing, area-coverage-no-paths) are
// bumped to error so the pre-push hook blocks them, mirroring the area-unknown
// / area-dead-glob / area-overlap escalation. The entity-body-empty control
// proves the bump stays scoped to the area codes.
func TestApplyAreaRequiredStrict_EscalatesCoverageFindings(t *testing.T) {
	t.Parallel()
	coverageCodes := []string{CodeAreaUnslotted, CodeAreaCoverageRootMissing, CodeAreaCoverageNoPaths}
	build := func() []Finding {
		fs := make([]Finding, 0, len(coverageCodes)+1)
		for _, c := range coverageCodes {
			fs = append(fs, Finding{Code: c, Severity: SeverityWarning})
		}
		return append(fs, Finding{Code: CodeEntityBodyEmpty, Severity: SeverityWarning})
	}

	t.Run("required=true bumps every coverage finding to error, control untouched", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, true)
		for _, f := range findings {
			if f.Code == CodeEntityBodyEmpty {
				if f.Severity != SeverityWarning {
					t.Errorf("entity-body-empty severity = %v, want warning unchanged", f.Severity)
				}
				continue
			}
			if f.Severity != SeverityError {
				t.Errorf("%s severity = %v, want error under required", f.Code, f.Severity)
			}
		}
	})

	t.Run("required=false leaves every coverage finding a warning", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, false)
		for _, f := range findings {
			if f.Severity != SeverityWarning {
				t.Errorf("%s severity = %v, want warning when required=false", f.Code, f.Severity)
			}
		}
	})
}
