package initrepo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsureGuidance_PropagatesError covers ensureGuidance's error-wrap
// branch: when MaterializeGuidance fails (here, `.claude` is a file so
// MkdirAll fails), ensureGuidance returns the wrapped error rather than
// swallowing it (M-0163/AC-3 branch coverage).
func TestEnsureGuidance_PropagatesError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".claude"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding .claude as a file: %v", err)
	}
	if _, err := ensureGuidance(root, false); err == nil {
		t.Error("expected ensureGuidance to propagate the materialize error, got nil")
	}
}

// TestInit_MaterializesGuidanceFragment drives the full init/update
// pipeline (the seam, not just skills.MaterializeGuidance) and asserts
// the guidance fragment is written, gitignored, and idempotent across a
// second refresh (M-0163/AC-3).
func TestInit_MaterializesGuidanceFragment(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	guidancePath := filepath.Join(root, ".claude", "aiwf-guidance.md")
	first, err := os.ReadFile(guidancePath)
	if err != nil {
		t.Fatalf("AC-3: guidance fragment not materialized: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("AC-3: materialized guidance fragment is empty")
	}

	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !strings.Contains(string(gi), ".claude/aiwf-guidance.md") {
		t.Errorf("AC-3: .gitignore missing guidance entry; got:\n%s", gi)
	}

	// Idempotent: a second refresh (the update path) rewrites identical bytes.
	if _, _, refreshErr := RefreshArtifacts(context.Background(), root, RefreshOptions{}); refreshErr != nil {
		t.Fatalf("RefreshArtifacts (update): %v", refreshErr)
	}
	second, err := os.ReadFile(guidancePath)
	if err != nil {
		t.Fatalf("reading guidance fragment after update: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("AC-3: guidance fragment not idempotent across update")
	}
}
