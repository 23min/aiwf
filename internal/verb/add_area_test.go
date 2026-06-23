package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestAdd_Area_WritesFrontmatter pins M-0173/AC-1: AddOptions.Area lands
// as `area:` in the created entity's frontmatter, across the root kinds.
func TestAdd_Area_WritesFrontmatter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		kind entity.Kind
		glob string
	}{
		{"epic", entity.KindEpic, "work/epics/E-*/epic.md"},
		{"adr", entity.KindADR, "docs/adr/ADR-*.md"},
		{"gap", entity.KindGap, "work/gaps/G-*.md"},
		{"decision", entity.KindDecision, "work/decisions/D-*.md"},
		{"contract", entity.KindContract, "work/contracts/C-*/contract.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), tc.kind, tc.name+" entity", testActor, verb.AddOptions{
				Area: "platform",
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
			if !strings.Contains(string(fm), "area: platform") {
				t.Errorf("%s frontmatter missing `area: platform`:\n%s", tc.kind, fm)
			}
		})
	}
}

// TestAdd_Area_AbsentWritesNoField pins that omitting Area writes no
// `area:` key (omitempty), so existing untagged entities are unchanged.
func TestAdd_Area_AbsentWritesNoField(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Untagged", testActor, verb.AddOptions{}))
	content, err := os.ReadFile(filepath.Join(r.root, "work", "epics", "E-0001-untagged", "epic.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	fm, _, _ := entity.Split(content)
	if strings.Contains(string(fm), "area:") {
		t.Errorf("untagged epic should write no area key:\n%s", fm)
	}
}

// TestAdd_Area_RejectedForMilestone pins M-0173/AC-3: a milestone derives
// its area from its parent epic and never stores one, so a non-empty Area
// on kind=milestone is a usage-shaped error (no entity created).
func TestAdd_Area_RejectedForMilestone(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Parent", testActor, verb.AddOptions{}))
	_, err := verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{
		EpicID: "E-0001", TDD: "none", Area: "platform",
	})
	if err == nil {
		t.Fatal("expected error for --area on a milestone, got nil")
	}
	if !strings.Contains(err.Error(), "area") || !strings.Contains(err.Error(), "root") {
		t.Errorf("error %q should explain that --area is for root kinds only", err)
	}
}
