package show_test

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestBuildCompositeShowView_NotFound pins BuildCompositeShowView's
// two not-found branches: an unknown parent milestone, and a known
// parent whose ACs don't include the requested sub-id. Neither path
// reaches a git read, so a hand-built *tree.Tree (no real repo) is
// enough. Surfaced by the diff-scoped coverage gate (G-0067) after
// M-0269/AC-2 mechanically widened both returns to a third value —
// the branches themselves predate that change and had no test
// anywhere in the repo.
func TestBuildCompositeShowView_NotFound(t *testing.T) {
	t.Parallel()
	milestone := &entity.Entity{
		ID:   "M-0001",
		Kind: entity.KindMilestone,
		Path: "work/epics/E-0001-foo/M-0001-bar.md",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "first behavior"},
		},
	}
	tr := &tree.Tree{Entities: []*entity.Entity{milestone}}

	t.Run("parent not found", func(t *testing.T) {
		t.Parallel()
		_, ok, err := show.BuildCompositeShowView(context.Background(), "", tr, nil, "M-9999/AC-1", 5)
		if err != nil {
			t.Fatalf("BuildCompositeShowView: %v", err)
		}
		if ok {
			t.Error("ok = true, want false (parent milestone M-9999 doesn't exist)")
		}
	})

	t.Run("AC not found under real parent", func(t *testing.T) {
		t.Parallel()
		_, ok, err := show.BuildCompositeShowView(context.Background(), "", tr, nil, "M-0001/AC-999", 5)
		if err != nil {
			t.Fatalf("BuildCompositeShowView: %v", err)
		}
		if ok {
			t.Error("ok = true, want false (AC-999 doesn't exist under M-0001)")
		}
	})
}
