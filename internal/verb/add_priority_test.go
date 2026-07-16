package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestAdd_Priority_WritesFrontmatter pins AC-3: AddOptions.Priority lands
// as `priority:` in the created entity's frontmatter, on both kinds that
// carry a priority (gap, decision).
func TestAdd_Priority_WritesFrontmatter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		kind entity.Kind
		glob string
	}{
		{"gap", entity.KindGap, "work/gaps/G-*.md"},
		{"decision", entity.KindDecision, "work/decisions/D-*.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), tc.kind, tc.name+" entity", testActor, verb.AddOptions{
				Priority:     "urgent",
				BodyOverride: bornCompleteFixtureBody(tc.kind),
			}))
			matches, err := filepath.Glob(filepath.Join(r.root, tc.glob))
			if err != nil || len(matches) != 1 {
				t.Fatalf("glob %s: matches=%v err=%v", tc.glob, matches, err)
			}
			content, err := os.ReadFile(matches[0])
			if err != nil {
				t.Fatalf("read %s: %v", matches[0], err)
			}
			fm, _, ok := entity.Split(content)
			if !ok {
				t.Fatalf("%s: no frontmatter:\n%s", tc.kind, content)
			}
			if !strings.Contains(string(fm), "priority: urgent") {
				t.Errorf("%s frontmatter missing `priority: urgent`:\n%s", tc.kind, fm)
			}
		})
	}
}

// TestAdd_Priority_AbsentWritesNoField pins that omitting Priority writes
// no `priority:` key (omitempty), so existing gaps/decisions are unchanged.
func TestAdd_Priority_AbsentWritesNoField(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Unprioritized", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	matches, err := filepath.Glob(filepath.Join(r.root, "work", "gaps", "G-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob: matches=%v err=%v", matches, err)
	}
	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	fm, _, _ := entity.Split(content)
	if strings.Contains(string(fm), "priority:") {
		t.Errorf("unprioritized gap should write no priority key:\n%s", fm)
	}
}

// TestAdd_Priority_RejectedForNonCarryingKind pins AC-3's kind gate,
// mirroring TestAdd_Area_RejectedForMilestone: --priority on a kind that
// doesn't carry one (epic, milestone, ADR, contract) is a usage-shaped
// error and creates nothing.
func TestAdd_Priority_RejectedForNonCarryingKind(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	_, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, "Epic one", testActor, verb.AddOptions{
		Priority: "urgent",
	})
	if err == nil {
		t.Fatal("expected error for --priority on an epic, got nil")
	}
	if !strings.Contains(err.Error(), "priority") || !strings.Contains(err.Error(), "gap and decision") {
		t.Errorf("error %q should explain that --priority is only for gap and decision", err)
	}
}

// TestAdd_Priority_RejectedOutOfRange pins that Add routes the level
// check through the same entity.IsAllowedPriorityLevel SSOT predicate
// set-priority uses — no parallel value check.
func TestAdd_Priority_RejectedOutOfRange(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	_, err := verb.Add(r.ctx, r.tree(), entity.KindGap, "Gap one", testActor, verb.AddOptions{
		Priority:     "critical",
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	})
	if err == nil {
		t.Fatal("expected error for out-of-range --priority, got nil")
	}
	for _, want := range []string{"critical", "urgent", "high", "medium", "low"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}
