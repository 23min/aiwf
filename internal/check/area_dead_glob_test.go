package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// mkAreaDir creates a directory (and parents) under root. Fails on error.
func mkAreaDir(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
}

// TestAreaDeadGlob exercises every branch: a glob that locates nothing fires
// (warning), a glob that locates a real path is silent, the check is per-glob
// (one dead glob among several fires once for that glob), a paths-less member
// is inert, nil areas are inert, and an empty/unreadable root never fails on
// IO.
func TestAreaDeadGlob(t *testing.T) {
	t.Parallel()

	t.Run("glob matching no real path fires one warning naming area and glob", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		// app-a exists with a file under it; app-z does not exist at all.
		mkAreaDir(t, root, "projects/app-a")
		if err := os.WriteFile(filepath.Join(root, "projects", "app-a", "main.go"), []byte("package a\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		got := AreaDeadGlob(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "live", Paths: []string{"projects/app-a/**"}},
			{Name: "dead", Paths: []string{"projects/app-z/**"}},
		})
		hits := findByCode(got, CodeAreaDeadGlob)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 dead-glob finding, got %d: %+v", len(hits), got)
		}
		if hits[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want %q", hits[0].Severity, SeverityWarning)
		}
		for _, want := range []string{"dead", "projects/app-z/**"} {
			if !strings.Contains(hits[0].Message, want) {
				t.Errorf("message %q does not name %q", hits[0].Message, want)
			}
		}
	})

	t.Run("every glob locating a real path is silent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		mkAreaDir(t, root, "infra")
		got := AreaDeadGlob(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "app", Paths: []string{"projects/app-a/**"}},
			{Name: "infra", Paths: []string{"infra/**"}},
		})
		if len(got) != 0 {
			t.Errorf("want no findings, got %+v", got)
		}
	})

	t.Run("a literal (wildcard-free) glob that does not exist fires", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		got := AreaDeadGlob(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "lit-live", Paths: []string{"projects/app-a"}},
			{Name: "lit-dead", Paths: []string{"projects/app-q"}},
		})
		hits := findByCode(got, CodeAreaDeadGlob)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 dead-glob finding (the missing literal), got %d: %+v", len(hits), got)
		}
		if !strings.Contains(hits[0].Message, "lit-dead") {
			t.Errorf("message %q should name the dead area", hits[0].Message)
		}
	})

	t.Run("per-glob: one dead glob among several in one member fires once", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		got := AreaDeadGlob(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "multi", Paths: []string{"projects/app-a/**", "projects/ghost/**"}},
		})
		hits := findByCode(got, CodeAreaDeadGlob)
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 dead-glob finding (the ghost glob), got %d: %+v", len(hits), got)
		}
		if !strings.Contains(hits[0].Message, "projects/ghost/**") {
			t.Errorf("message %q should name the dead glob", hits[0].Message)
		}
	})

	t.Run("a paths-less member is inert", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		got := AreaDeadGlob(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "label-only", Paths: nil},
		})
		if len(got) != 0 {
			t.Errorf("a paths-less member must fire nothing, got %+v", got)
		}
	})

	t.Run("a malformed glob is skipped, never fires or crashes", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mkAreaDir(t, root, "projects/app-a")
		// A malformed character class makes MatchFS error; the check skips
		// it (malformed globs are a Tier-1 config-load concern) rather than
		// firing a spurious dead-glob or panicking.
		got := AreaDeadGlob(&tree.Tree{Root: root}, []AreaPaths{
			{Name: "bad", Paths: []string{"projects/["}},
		})
		if len(got) != 0 {
			t.Errorf("a malformed glob must be skipped, got %+v", got)
		}
	})

	t.Run("nil areas are inert", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if got := AreaDeadGlob(&tree.Tree{Root: root}, nil); len(got) != 0 {
			t.Errorf("nil areas must be inert, got %+v", got)
		}
	})

	t.Run("empty root is silent (never fails on IO)", func(t *testing.T) {
		t.Parallel()
		got := AreaDeadGlob(&tree.Tree{Root: ""}, []AreaPaths{{Name: "x", Paths: []string{"a/**"}}})
		if len(got) != 0 {
			t.Errorf("empty root must be silent, got %+v", got)
		}
	})

	t.Run("unreadable root is silent (never fails on IO)", func(t *testing.T) {
		t.Parallel()
		missing := filepath.Join(t.TempDir(), "does-not-exist")
		got := AreaDeadGlob(&tree.Tree{Root: missing}, []AreaPaths{{Name: "x", Paths: []string{"a/**"}}})
		if len(got) != 0 {
			t.Errorf("unreadable root must be silent, got %+v", got)
		}
	})
}

// TestApplyAreaRequiredStrict_EscalatesDeadGlob pins the M-0180/AC-2 severity
// contract: under areas.required the dead-glob warning is bumped to error so
// the pre-push hook blocks it, mirroring the area-unknown escalation. The
// entity-body-empty control proves the bump stays scoped to the area codes.
func TestApplyAreaRequiredStrict_EscalatesDeadGlob(t *testing.T) {
	t.Parallel()
	build := func() []Finding {
		return []Finding{
			{Code: CodeAreaDeadGlob, Severity: SeverityWarning},
			{Code: CodeEntityBodyEmpty, Severity: SeverityWarning},
		}
	}

	t.Run("required=true bumps dead-glob to error", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, true)
		for _, f := range findings {
			switch f.Code {
			case CodeAreaDeadGlob:
				if f.Severity != SeverityError {
					t.Errorf("dead-glob severity = %v, want error under required", f.Severity)
				}
			case CodeEntityBodyEmpty:
				if f.Severity != SeverityWarning {
					t.Errorf("entity-body-empty severity = %v, want warning unchanged", f.Severity)
				}
			}
		}
	})

	t.Run("required=false leaves dead-glob a warning", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, false)
		for _, f := range findings {
			if f.Code == CodeAreaDeadGlob && f.Severity != SeverityWarning {
				t.Errorf("dead-glob severity = %v, want warning when required=false", f.Severity)
			}
		}
	})
}
