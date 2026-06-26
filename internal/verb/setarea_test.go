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

// setAreaTree builds a tree with one tagged epic E-0001 (area=epicArea),
// one milestone M-0001 under it, plus any extra epics supplied. Each
// entity is written to disk so SetArea's readBody finds a real file.
func setAreaTree(t *testing.T, epicArea string, extra map[string]string) *tree.Tree {
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

	epic := &entity.Entity{
		ID:     "E-0001",
		Kind:   entity.KindEpic,
		Title:  "Epic one",
		Status: "proposed",
		Area:   epicArea,
		Path:   filepath.Join("work", "epics", "E-0001-slug", "epic.md"),
	}
	write(epic, "\n## Goal\n")

	ms := &entity.Entity{
		ID:     "M-0001",
		Kind:   entity.KindMilestone,
		Title:  "Milestone one",
		Status: "draft",
		Parent: "E-0001",
		Path:   filepath.Join("work", "epics", "E-0001-slug", "M-0001-slug.md"),
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "first", Status: "open"},
		},
	}
	write(ms, "\n## Goal\n")

	for id, area := range extra {
		e := &entity.Entity{
			ID:     id,
			Kind:   entity.KindEpic,
			Title:  "Epic " + id,
			Status: "proposed",
			Area:   area,
			Path:   filepath.Join("work", "epics", id+"-slug", "epic.md"),
		}
		write(e, "\n## Goal\n")
	}
	return &tree.Tree{Root: root, Entities: ents}
}

var setAreaMembers = []string{"platform", "billing"}

// TestSetArea_RewritesSingleEntity pins the Plan shape: exactly one
// OpWrite for the target, the right trailers, in both the untagged→tagged
// and tagged→retagged directions.
func TestSetArea_RewritesSingleEntity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		startArea string
		member    string
		wantArea  string
	}{
		{name: "untagged to tagged", startArea: "", member: "platform", wantArea: "area: platform"},
		{name: "tagged to retagged", startArea: "platform", member: "billing", wantArea: "area: billing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := setAreaTree(t, tc.startArea, nil)
			res, err := SetArea(context.Background(), tr, setAreaMembers, "E-0001", tc.member, false, "human/test")
			if err != nil {
				t.Fatalf("SetArea: %v", err)
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
			if !strings.Contains(op.Path, "E-0001") {
				t.Errorf("op path = %q, want the E-0001 file", op.Path)
			}
			if !strings.Contains(string(op.Content), tc.wantArea) {
				t.Errorf("op content missing %q:\n%s", tc.wantArea, op.Content)
			}

			wantTrailers := []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "set-area"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
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

// TestSetArea_ValidationRefusals exhausts the refusal paths: unknown id,
// undeclared member, empty-members (no areas block), and both no-op
// cases. Each returns an error, a nil result, and writes nothing.
func TestSetArea_ValidationRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		members     []string
		startArea   string
		id, member  string
		clear       bool
		wantInError string
	}{
		{name: "unknown id", members: setAreaMembers, id: "E-9999", member: "platform", wantInError: "unknown id"},
		{name: "undeclared member", members: setAreaMembers, id: "E-0001", member: "nonsense", wantInError: "not a declared member"},
		{name: "no areas block", members: nil, id: "E-0001", member: "platform", wantInError: "not a declared member"},
		{name: "no-op already tagged", members: setAreaMembers, startArea: "platform", id: "E-0001", member: "platform", wantInError: "already tagged"},
		{name: "no-op clear already untagged", members: setAreaMembers, startArea: "", id: "E-0001", clear: true, wantInError: "already untagged"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := setAreaTree(t, tc.startArea, nil)
			res, err := SetArea(context.Background(), tr, tc.members, tc.id, tc.member, tc.clear, "human/test")
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

// TestSetArea_UndeclaredErrorNamesDeclaredSet pins the self-explaining
// refusal: an undeclared member names the declared set so the operator
// can correct the typo without opening aiwf.yaml.
func TestSetArea_UndeclaredErrorNamesDeclaredSet(t *testing.T) {
	t.Parallel()
	tr := setAreaTree(t, "", nil)
	_, err := SetArea(context.Background(), tr, setAreaMembers, "E-0001", "nope", false, "human/test")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"nope", "platform", "billing"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}

// TestSetArea_MilestoneTargetNamesEpic pins the milestone/composite
// refusal message shape: it names the parent epic and gives the
// remediation command pointed at the epic, for both a bare milestone id
// and a composite AC id.
func TestSetArea_MilestoneTargetNamesEpic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		id       string
		member   string
		clear    bool
		wantHint string // the remediation arg placeholder in the message
	}{
		{name: "bare milestone with member", id: "M-0001", member: "platform", wantHint: "aiwf set-area E-0001 platform"},
		{name: "composite AC id with member", id: "M-0001/AC-1", member: "platform", wantHint: "aiwf set-area E-0001 platform"},
		{name: "bare milestone with clear", id: "M-0001", clear: true, wantHint: "aiwf set-area E-0001 --clear"},
		{name: "bare milestone no member", id: "M-0001", wantHint: "aiwf set-area E-0001 <member>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := setAreaTree(t, "platform", nil)
			res, err := SetArea(context.Background(), tr, setAreaMembers, tc.id, tc.member, tc.clear, "human/test")
			if err == nil {
				t.Fatalf("expected refusal, got Plan=%v", res)
			}
			for _, want := range []string{"E-0001", "parent epic", tc.wantHint} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q missing %q", err.Error(), want)
				}
			}
			if res != nil {
				t.Errorf("result should be nil on refusal, got %v", res)
			}
		})
	}
}

// TestSetArea_ClearMemberMutex pins the <member>+--clear mutex at the
// verb level (a secondary guard behind the CLI arity check).
func TestSetArea_ClearMemberMutex(t *testing.T) {
	t.Parallel()
	tr := setAreaTree(t, "platform", nil)
	res, err := SetArea(context.Background(), tr, setAreaMembers, "E-0001", "billing", true, "human/test")
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

// TestSetArea_ClearEmptiesArea pins that --clear produces one OpWrite
// whose serialized frontmatter omits the area key (omitempty drops it),
// and the subject + trailers reflect the clear.
func TestSetArea_ClearEmptiesArea(t *testing.T) {
	t.Parallel()
	tr := setAreaTree(t, "platform", nil)
	res, err := SetArea(context.Background(), tr, setAreaMembers, "E-0001", "", true, "human/test")
	if err != nil {
		t.Fatalf("SetArea --clear: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected a Plan")
	}
	if len(res.Plan.Ops) != 1 {
		t.Fatalf("ops = %d, want 1", len(res.Plan.Ops))
	}
	content := string(res.Plan.Ops[0].Content)
	if strings.Contains(content, "area:") {
		t.Errorf("--clear should drop the area key from frontmatter:\n%s", content)
	}
	if !strings.Contains(res.Plan.Subject, "--clear") {
		t.Errorf("subject = %q, want it to mention --clear", res.Plan.Subject)
	}
	wantTrailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "set-area"},
		{Key: gitops.TrailerEntity, Value: "E-0001"},
		{Key: gitops.TrailerActor, Value: "human/test"},
	}
	for i, want := range wantTrailers {
		if res.Plan.Trailers[i] != want {
			t.Errorf("trailer[%d] = %v, want %v", i, res.Plan.Trailers[i], want)
		}
	}
}

// TestSetArea_MissingEntityFileErrors covers the readBody error seam: an
// entity present in the tree but missing on disk aborts before any Plan.
func TestSetArea_MissingEntityFileErrors(t *testing.T) {
	t.Parallel()
	tr := setAreaTree(t, "", nil)
	if err := os.Remove(filepath.Join(tr.Root, tr.Entities[0].Path)); err != nil {
		t.Fatalf("remove entity file: %v", err)
	}
	res, err := SetArea(context.Background(), tr, setAreaMembers, "E-0001", "platform", false, "human/test")
	if err == nil {
		t.Fatalf("expected error for missing entity file, got Plan=%v", res)
	}
}
