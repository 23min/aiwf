package render

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveRoadmapName is a white-box unit test (in-package, so it can
// reach the unexported helper) that pins every branch of the
// case-reconciliation: a single canonical file, a single lowercase
// variant, no roadmap, two variants (the unreconcilable case), a
// same-cased directory that must be ignored, and the directory-read
// error fall-back.
func TestResolveRoadmapName(t *testing.T) {
	t.Parallel()

	t.Run("no roadmap file returns canonical", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if got := resolveRoadmapName(dir); got != "ROADMAP.md" {
			t.Errorf("resolveRoadmapName = %q, want ROADMAP.md", got)
		}
	})

	t.Run("only canonical returns canonical", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "ROADMAP.md"))
		if got := resolveRoadmapName(dir); got != "ROADMAP.md" {
			t.Errorf("resolveRoadmapName = %q, want ROADMAP.md", got)
		}
	})

	t.Run("only lowercase variant returns that variant", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "roadmap.md"))
		if got := resolveRoadmapName(dir); got != "roadmap.md" {
			t.Errorf("resolveRoadmapName = %q, want roadmap.md", got)
		}
	})

	t.Run("only mixed-case variant returns that variant", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "Roadmap.md"))
		if got := resolveRoadmapName(dir); got != "Roadmap.md" {
			t.Errorf("resolveRoadmapName = %q, want Roadmap.md", got)
		}
	})

	t.Run("two variants fall back to canonical", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "ROADMAP.md"))
		mustWriteFile(t, filepath.Join(dir, "roadmap.md"))
		// The unreconcilable case: with two variants the renderer cannot
		// pick, so it defaults to canonical and the roadmap-case-collision
		// finding flags the divergence.
		if got := resolveRoadmapName(dir); got != "ROADMAP.md" {
			t.Errorf("resolveRoadmapName = %q, want ROADMAP.md (collision fallback)", got)
		}
	})

	t.Run("directory variant is ignored", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// A directory whose name case-folds to the artifact must not be
		// treated as the roadmap file.
		if err := os.Mkdir(filepath.Join(dir, "roadmap.md"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if got := resolveRoadmapName(dir); got != "ROADMAP.md" {
			t.Errorf("resolveRoadmapName = %q, want ROADMAP.md (dir ignored)", got)
		}
	})

	t.Run("unreadable root falls back to canonical", func(t *testing.T) {
		t.Parallel()
		// A path that does not exist drives os.ReadDir to error; the
		// helper must fail soft to the canonical name.
		missing := filepath.Join(t.TempDir(), "no-such-dir")
		if got := resolveRoadmapName(missing); got != "ROADMAP.md" {
			t.Errorf("resolveRoadmapName = %q, want ROADMAP.md (read-error fallback)", got)
		}
	})
}

// mustWriteFile writes an empty roadmap stub at path, failing the test
// on error.
func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("# Roadmap\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
