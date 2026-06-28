package verb_test

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// TestAcknowledgeMistag pins M-0181/AC-6's verb: the per-entity sovereign ack
// records the right trailers on an empty commit, refuses an empty reason, a
// non-human actor, and an unknown entity, and rolls a composite AC id up to its
// milestone.
func TestAcknowledgeMistag(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{Entities: []*entity.Entity{
		{ID: "G-0001", Kind: entity.KindGap, Path: "work/gaps/G-0001-x.md", Area: "app-a"},
	}}

	t.Run("happy path records the per-entity ack trailers on an empty commit", func(t *testing.T) {
		t.Parallel()
		res, err := verb.AcknowledgeMistag(context.Background(), tr, "G-0001", "human/peter", "cross-cutting by design")
		if err != nil {
			t.Fatalf("AcknowledgeMistag: %v", err)
		}
		if !res.Plan.AllowEmpty {
			t.Error("AllowEmpty = false, want true (acknowledge commits are empty)")
		}
		mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerVerb, "acknowledge-mistag")
		mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerEntity, "G-0001")
		mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerActor, "human/peter")
	})

	t.Run("empty reason refused", func(t *testing.T) {
		t.Parallel()
		if _, err := verb.AcknowledgeMistag(context.Background(), tr, "G-0001", "human/peter", "   "); err == nil {
			t.Error("want error for empty reason")
		}
	})

	t.Run("non-human actor refused", func(t *testing.T) {
		t.Parallel()
		if _, err := verb.AcknowledgeMistag(context.Background(), tr, "G-0001", "ai/claude", "real reason"); err == nil {
			t.Error("want error for non-human actor")
		}
	})

	t.Run("unknown entity refused", func(t *testing.T) {
		t.Parallel()
		if _, err := verb.AcknowledgeMistag(context.Background(), tr, "G-9999", "human/peter", "real reason"); err == nil {
			t.Error("want error for an id that resolves to no entity")
		}
	})

	t.Run("composite AC id rolls up to its milestone", func(t *testing.T) {
		t.Parallel()
		trM := &tree.Tree{Entities: []*entity.Entity{
			{ID: "M-0001", Kind: entity.KindMilestone, Path: "work/epics/E-0001-x/M-0001-y.md"},
		}}
		res, err := verb.AcknowledgeMistag(context.Background(), trM, "M-0001/AC-2", "human/peter", "real reason")
		if err != nil {
			t.Fatalf("composite ack: %v", err)
		}
		mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerEntity, "M-0001")
	})
}
