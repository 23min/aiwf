package verb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// setPriorityTree builds a tree with one gap G-0001 (priority=gapPriority),
// one decision D-0001 (priority unset), and one epic E-0001 (a kind that
// never carries a priority) — enough surface to exercise both the
// gap/decision write path and the non-carrying-kind refusal. Each entity is
// written to disk so SetPriority's readBody finds a real file.
func setPriorityTree(t *testing.T, gapPriority string) *tree.Tree {
	t.Helper()
	root := t.TempDir()
	var ents []*entity.Entity

	write := func(e *entity.Entity, body string) {
		content, err := entity.Serialize(e, []byte(body))
		if err != nil {
			t.Fatalf("serialize %s: %v", e.ID, err)
		}
		full := filepath.Join(root, e.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatalf("write %s: %v", e.ID, err)
		}
		ents = append(ents, e)
	}

	gap := &entity.Entity{
		ID:       "G-0001",
		Kind:     entity.KindGap,
		Title:    "Gap one",
		Status:   "open",
		Priority: gapPriority,
		Path:     filepath.Join("work", "gaps", "G-0001-slug.md"),
	}
	write(gap, "\n## Problem\n")

	decision := &entity.Entity{
		ID:     "D-0001",
		Kind:   entity.KindDecision,
		Title:  "Decision one",
		Status: "proposed",
		Path:   filepath.Join("work", "decisions", "D-0001-slug.md"),
	}
	write(decision, "\n## Decision\n")

	epic := &entity.Entity{
		ID:     "E-0001",
		Kind:   entity.KindEpic,
		Title:  "Epic one",
		Status: "proposed",
		Path:   filepath.Join("work", "epics", "E-0001-slug", "epic.md"),
	}
	write(epic, "\n## Goal\n")

	return &tree.Tree{Root: root, Entities: ents}
}

// TestSetPriority_RewritesSingleEntity pins the Plan shape: exactly one
// OpWrite for the target, the right trailers, in both the
// unset->set and set->reset directions.
func TestSetPriority_RewritesSingleEntity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		startPri     string
		level        string
		wantPriority string
	}{
		{name: "unset to urgent", startPri: "", level: "urgent", wantPriority: "priority: urgent"},
		{name: "high to low", startPri: "high", level: "low", wantPriority: "priority: low"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := setPriorityTree(t, tc.startPri)
			res, err := SetPriority(context.Background(), tr, "G-0001", tc.level, false, "human/test")
			if err != nil {
				t.Fatalf("SetPriority: %v", err)
			}
			if res.Plan == nil {
				t.Fatal("expected a Plan")
			}
			if len(res.Plan.Ops) != 1 {
				t.Fatalf("ops = %d, want 1 (the target entity only)", len(res.Plan.Ops))
			}
			op := res.Plan.Ops[0]
			if op.Type != OpWrite {
				t.Errorf("op type = %v, want OpWrite", op.Type)
			}
			if !strings.Contains(op.Path, "G-0001") {
				t.Errorf("op path = %q, want the G-0001 file", op.Path)
			}
			if !strings.Contains(string(op.Content), tc.wantPriority) {
				t.Errorf("op content missing %q:\n%s", tc.wantPriority, op.Content)
			}

			wantTrailers := []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "set-priority"},
				{Key: gitops.TrailerEntity, Value: "G-0001"},
				{Key: gitops.TrailerActor, Value: "human/test"},
			}
			if len(res.Plan.Trailers) != len(wantTrailers) {
				t.Fatalf("trailers = %v, want %v", res.Plan.Trailers, wantTrailers)
			}
			for i, want := range wantTrailers {
				if res.Plan.Trailers[i] != want {
					t.Errorf("trailer[%d] = %v, want %v", i, res.Plan.Trailers[i], want)
				}
			}
		})
	}
}

// TestSetPriority_WritesDecision confirms the decision kind (the second
// CarriesOwnPriority member) is also a legal write target.
func TestSetPriority_WritesDecision(t *testing.T) {
	t.Parallel()
	tr := setPriorityTree(t, "")
	res, err := SetPriority(context.Background(), tr, "D-0001", "medium", false, "human/test")
	if err != nil {
		t.Fatalf("SetPriority on decision: %v", err)
	}
	if res.Plan == nil || len(res.Plan.Ops) != 1 {
		t.Fatalf("expected exactly one OpWrite, got %+v", res.Plan)
	}
	if !strings.Contains(string(res.Plan.Ops[0].Content), "priority: medium") {
		t.Errorf("expected `priority: medium` in frontmatter:\n%s", res.Plan.Ops[0].Content)
	}
}

// TestSetPriority_ValidationRefusals exhausts the refusal paths: unknown
// id, a non-gap/decision target, an out-of-range level, and both no-op
// cases. Each returns an error, a nil result, and writes nothing.
func TestSetPriority_ValidationRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		startPri    string
		id, level   string
		clear       bool
		wantInError string
	}{
		{name: "unknown id", id: "G-9999", level: "urgent", wantInError: "unknown id"},
		{name: "non-gap/decision target", id: "E-0001", level: "urgent", wantInError: "does not carry a priority"},
		{name: "out-of-range level", id: "G-0001", level: "critical", wantInError: "not a recognized priority level"},
		{name: "no-op already set", startPri: "high", id: "G-0001", level: "high", wantInError: "already set to"},
		{name: "no-op clear already unset", startPri: "", id: "G-0001", clear: true, wantInError: "already unset"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := setPriorityTree(t, tc.startPri)
			res, err := SetPriority(context.Background(), tr, tc.id, tc.level, tc.clear, "human/test")
			if err == nil {
				t.Fatalf("expected error, got Plan=%v", res)
			}
			if !strings.Contains(err.Error(), tc.wantInError) {
				t.Errorf("error %q missing %q", err.Error(), tc.wantInError)
			}
			if res != nil {
				t.Errorf("result should be nil on validation failure, got %v", res)
			}
		})
	}
}

// TestSetPriority_OutOfRangeErrorNamesAllowedSet pins the self-explaining
// refusal: an out-of-range level names the allowed set so the operator can
// correct the typo without opening documentation.
func TestSetPriority_OutOfRangeErrorNamesAllowedSet(t *testing.T) {
	t.Parallel()
	tr := setPriorityTree(t, "")
	_, err := SetPriority(context.Background(), tr, "G-0001", "critical", false, "human/test")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"critical", "urgent", "high", "medium", "low"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}

// TestSetPriority_ClearLevelMutex pins the <level>+--clear mutex at the
// verb level (a secondary guard behind the CLI arity check).
func TestSetPriority_ClearLevelMutex(t *testing.T) {
	t.Parallel()
	tr := setPriorityTree(t, "high")
	res, err := SetPriority(context.Background(), tr, "G-0001", "low", true, "human/test")
	if err == nil {
		t.Fatalf("expected mutex refusal, got Plan=%v", res)
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error %q should explain the mutex", err.Error())
	}
	if res != nil {
		t.Errorf("result should be nil on refusal, got %v", res)
	}
}

// TestSetPriority_ClearEmptiesPriority pins that --clear produces one
// OpWrite whose serialized frontmatter omits the priority key (omitempty
// drops it), and the subject + trailers reflect the clear.
func TestSetPriority_ClearEmptiesPriority(t *testing.T) {
	t.Parallel()
	tr := setPriorityTree(t, "high")
	res, err := SetPriority(context.Background(), tr, "G-0001", "", true, "human/test")
	if err != nil {
		t.Fatalf("SetPriority --clear: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected a Plan")
	}
	if len(res.Plan.Ops) != 1 {
		t.Fatalf("ops = %d, want 1", len(res.Plan.Ops))
	}
	content := string(res.Plan.Ops[0].Content)
	if strings.Contains(content, "priority:") {
		t.Errorf("--clear should drop the priority key from frontmatter:\n%s", content)
	}
	if !strings.Contains(res.Plan.Subject, "--clear") {
		t.Errorf("subject = %q, want it to mention --clear", res.Plan.Subject)
	}
	wantTrailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "set-priority"},
		{Key: gitops.TrailerEntity, Value: "G-0001"},
		{Key: gitops.TrailerActor, Value: "human/test"},
	}
	for i, want := range wantTrailers {
		if res.Plan.Trailers[i] != want {
			t.Errorf("trailer[%d] = %v, want %v", i, res.Plan.Trailers[i], want)
		}
	}
}

// TestSetPriority_MissingEntityFileErrors covers the readBody error seam:
// an entity present in the tree but missing on disk aborts before any
// Plan.
func TestSetPriority_MissingEntityFileErrors(t *testing.T) {
	t.Parallel()
	tr := setPriorityTree(t, "")
	if err := os.Remove(filepath.Join(tr.Root, tr.Entities[0].Path)); err != nil {
		t.Fatalf("remove entity file: %v", err)
	}
	res, err := SetPriority(context.Background(), tr, "G-0001", "urgent", false, "human/test")
	if err == nil {
		t.Fatalf("expected error for missing entity file, got Plan=%v", res)
	}
}
