package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// writeRoadmapVariant creates a roadmap file with the given basename at
// dir. It fails the test on any write error.
func writeRoadmapVariant(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("# Roadmap\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// hasCode reports whether any finding carries the given code.
func hasCode(fs []Finding, code string) bool {
	for i := range fs {
		if fs[i].Code == code {
			return true
		}
	}
	return false
}

// findByCode returns every finding with the given code.
func findByCode(fs []Finding, code string) []Finding {
	var out []Finding
	for i := range fs {
		if fs[i].Code == code {
			out = append(out, fs[i])
		}
	}
	return out
}

// TestRoadmapCaseCollision exercises every branch of the rule: both
// variants present (fires once), a single canonical file (silent), a
// single lowercase file (silent), no roadmap at all (silent), and an
// empty / unreadable root (silent — checks never fail on IO).
func TestRoadmapCaseCollision(t *testing.T) {
	t.Parallel()

	t.Run("both variants present fires one warning", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeRoadmapVariant(t, dir, "ROADMAP.md")
		writeRoadmapVariant(t, dir, "roadmap.md")
		got := roadmapCaseCollision(&tree.Tree{Root: dir})
		hits := findByCode(got, "roadmap-case-collision")
		if len(hits) != 1 {
			t.Fatalf("want exactly 1 roadmap-case-collision finding, got %d: %+v", len(hits), got)
		}
		if hits[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want %q", hits[0].Severity, SeverityWarning)
		}
	})

	t.Run("only canonical ROADMAP.md is silent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeRoadmapVariant(t, dir, "ROADMAP.md")
		if got := roadmapCaseCollision(&tree.Tree{Root: dir}); len(got) != 0 {
			t.Errorf("want no findings, got %+v", got)
		}
	})

	t.Run("only lowercase roadmap.md is silent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeRoadmapVariant(t, dir, "roadmap.md")
		if got := roadmapCaseCollision(&tree.Tree{Root: dir}); len(got) != 0 {
			t.Errorf("want no findings, got %+v", got)
		}
	})

	t.Run("no roadmap file is silent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if got := roadmapCaseCollision(&tree.Tree{Root: dir}); len(got) != 0 {
			t.Errorf("want no findings, got %+v", got)
		}
	})

	t.Run("empty root path is silent", func(t *testing.T) {
		t.Parallel()
		if got := roadmapCaseCollision(&tree.Tree{Root: ""}); len(got) != 0 {
			t.Errorf("want no findings for empty root, got %+v", got)
		}
	})

	t.Run("unreadable root is silent", func(t *testing.T) {
		t.Parallel()
		missing := filepath.Join(t.TempDir(), "does-not-exist")
		if got := roadmapCaseCollision(&tree.Tree{Root: missing}); len(got) != 0 {
			t.Errorf("want no findings for unreadable root, got %+v", got)
		}
	})

	t.Run("a directory named roadmap.md is ignored", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// A directory whose name case-folds to the roadmap artifact must
		// not count as a variant — only regular files are artifacts.
		if err := os.Mkdir(filepath.Join(dir, "Roadmap.md"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeRoadmapVariant(t, dir, "ROADMAP.md")
		if got := roadmapCaseCollision(&tree.Tree{Root: dir}); len(got) != 0 {
			t.Errorf("a directory variant should not collide with one file, got %+v", got)
		}
	})
}

// TestRoadmapCaseCollision_ThroughRun confirms the rule is wired into
// Run (the seam), so a fixture tree with both variants surfaces the
// finding through the public check entry point, and a clean tree does
// not.
func TestRoadmapCaseCollision_ThroughRun(t *testing.T) {
	t.Parallel()

	t.Run("collision surfaces through Run", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeRoadmapVariant(t, dir, "ROADMAP.md")
		writeRoadmapVariant(t, dir, "roadmap.md")
		got := Run(&tree.Tree{Root: dir}, nil)
		if !hasCode(got, "roadmap-case-collision") {
			t.Errorf("Run did not surface roadmap-case-collision: %+v", got)
		}
	})

	t.Run("clean tree has no collision finding", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeRoadmapVariant(t, dir, "ROADMAP.md")
		got := Run(&tree.Tree{Root: dir}, nil)
		if hasCode(got, "roadmap-case-collision") {
			t.Errorf("Run reported roadmap-case-collision on a clean tree: %+v", got)
		}
	})
}
