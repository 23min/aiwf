package verb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

const areaAiwfYAML = `hosts: [claude-code]
areas:
  members:
    - platform
    - billing
`

// areaTree writes the given entity files under a fresh tempdir and
// returns a tree rooted there. Each entity is a minimal epic carrying
// the supplied area tag, so RenameArea's readBody finds a real file.
func areaTree(t *testing.T, areas map[string]string) *tree.Tree {
	t.Helper()
	root := t.TempDir()
	var ents []*entity.Entity
	for id, area := range areas {
		e := &entity.Entity{
			ID:     id,
			Kind:   entity.KindEpic,
			Title:  "Epic " + id,
			Status: "proposed",
			Area:   area,
			Path:   filepath.Join("work", "epics", id+"-slug", "epic.md"),
		}
		content, err := entity.Serialize(e, []byte("\n## Goal\n"))
		if err != nil {
			t.Fatalf("serialize %s: %v", id, err)
		}
		full := filepath.Join(root, e.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatalf("write %s: %v", id, err)
		}
		ents = append(ents, e)
	}
	return &tree.Tree{Root: root, Entities: ents}
}

func mustReadAreaDoc(t *testing.T) *aiwfyaml.Doc {
	t.Helper()
	d, _, err := aiwfyaml.ReadBytes([]byte(areaAiwfYAML))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	return d
}

func TestRenameArea_RewritesMemberAndEntities(t *testing.T) {
	t.Parallel()
	tr := areaTree(t, map[string]string{
		"E-0001": "platform",
		"E-0002": "platform",
		"E-0003": "billing",
	})
	doc := mustReadAreaDoc(t)

	res, err := RenameArea(context.Background(), tr, doc,
		[]string{"platform", "billing"}, "", "platform", "infra", "human/test")
	if err != nil {
		t.Fatalf("RenameArea: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected a Plan")
	}

	// One aiwf.yaml write + one write per platform entity (2), not the
	// billing entity.
	if len(res.Plan.Ops) != 3 {
		t.Fatalf("ops = %d, want 3 (aiwf.yaml + E-0001 + E-0002)", len(res.Plan.Ops))
	}
	if res.Plan.Ops[0].Path != "aiwf.yaml" {
		t.Errorf("first op path = %q, want aiwf.yaml", res.Plan.Ops[0].Path)
	}
	if !strings.Contains(string(res.Plan.Ops[0].Content), "- infra") {
		t.Errorf("aiwf.yaml op missing renamed member:\n%s", res.Plan.Ops[0].Content)
	}
	for _, op := range res.Plan.Ops[1:] {
		if !strings.Contains(string(op.Content), "area: infra") {
			t.Errorf("entity op %s not retagged:\n%s", op.Path, op.Content)
		}
		if strings.Contains(op.Path, "E-0003") {
			t.Errorf("billing entity should not be rewritten: %s", op.Path)
		}
	}

	// Trailers: verb + two entity trailers (sorted) + actor.
	wantTrailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "rename-area"},
		{Key: gitops.TrailerEntity, Value: "E-0001"},
		{Key: gitops.TrailerEntity, Value: "E-0002"},
		{Key: gitops.TrailerActor, Value: "human/test"},
	}
	if len(res.Plan.Trailers) != len(wantTrailers) {
		t.Fatalf("trailers = %v, want %v", res.Plan.Trailers, wantTrailers)
	}
	for i, tr := range wantTrailers {
		if res.Plan.Trailers[i] != tr {
			t.Errorf("trailer[%d] = %v, want %v", i, res.Plan.Trailers[i], tr)
		}
	}
}

func TestRenameArea_NoReferencingEntities(t *testing.T) {
	t.Parallel()
	tr := areaTree(t, map[string]string{"E-0003": "billing"})
	doc := mustReadAreaDoc(t)

	res, err := RenameArea(context.Background(), tr, doc,
		[]string{"platform", "billing"}, "", "platform", "infra", "human/test")
	if err != nil {
		t.Fatalf("RenameArea: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected a Plan")
	}
	if len(res.Plan.Ops) != 1 {
		t.Fatalf("ops = %d, want 1 (aiwf.yaml only)", len(res.Plan.Ops))
	}
	// Only verb + actor trailers when nothing references the old area.
	if len(res.Plan.Trailers) != 2 {
		t.Fatalf("trailers = %v, want verb+actor only", res.Plan.Trailers)
	}
	if res.Plan.Trailers[0].Key != gitops.TrailerVerb || res.Plan.Trailers[1].Key != gitops.TrailerActor {
		t.Errorf("trailers = %v, want [verb, actor]", res.Plan.Trailers)
	}
}

func TestRenameArea_PreservesDefaultLabel(t *testing.T) {
	t.Parallel()
	tr := areaTree(t, map[string]string{"E-0001": "platform"})
	d, _, err := aiwfyaml.ReadBytes([]byte("areas:\n  members:\n    - platform\n    - billing\n  default: untagged\n"))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	res, err := RenameArea(context.Background(), tr, d,
		[]string{"platform", "billing"}, "untagged", "platform", "infra", "human/test")
	if err != nil {
		t.Fatalf("RenameArea: %v", err)
	}
	if !strings.Contains(string(res.Plan.Ops[0].Content), "default: untagged") {
		t.Errorf("default label dropped:\n%s", res.Plan.Ops[0].Content)
	}
}

func TestRenameArea_ValidationRefusals(t *testing.T) {
	t.Parallel()
	members := []string{"platform", "billing"}
	cases := []struct {
		name        string
		nilDoc      bool
		old, new    string
		wantInError string
	}{
		{name: "nil doc", nilDoc: true, old: "platform", new: "infra", wantInError: "aiwf.yaml"},
		{name: "empty old", old: "", new: "infra", wantInError: "non-empty"},
		{name: "empty new", old: "platform", new: "", wantInError: "non-empty"},
		{name: "identical", old: "platform", new: "platform", wantInError: "identical"},
		{name: "undeclared old", old: "nonsense", new: "infra", wantInError: "not a declared member"},
		{name: "new already declared", old: "platform", new: "billing", wantInError: "already a declared member"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := areaTree(t, map[string]string{"E-0001": "platform"})
			var doc *aiwfyaml.Doc
			if !tc.nilDoc {
				doc = mustReadAreaDoc(t)
			}
			res, err := RenameArea(context.Background(), tr, doc,
				members, "", tc.old, tc.new, "human/test")
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

// TestRenameArea_UndeclaredErrorNamesDeclaredSet pins the self-
// explaining-error requirement: the refusal names the declared set so
// the operator can correct the typo without consulting aiwf.yaml.
func TestRenameArea_UndeclaredErrorNamesDeclaredSet(t *testing.T) {
	t.Parallel()
	tr := areaTree(t, map[string]string{"E-0001": "platform"})
	_, err := RenameArea(context.Background(), tr, mustReadAreaDoc(t),
		[]string{"platform", "billing"}, "", "nope", "infra", "human/test")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"nope", "platform", "billing"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}

// TestRenameArea_DocWithoutAreasBlockErrors covers the SetAreas
// refusal seam: validation passes (members says the area is declared)
// but the loaded doc carries no areas block, so the splice errors and
// the verb surfaces it wrapped. In practice config and the doc read the
// same file so this divergence can't arise, but the guard is real code.
func TestRenameArea_DocWithoutAreasBlockErrors(t *testing.T) {
	t.Parallel()
	tr := areaTree(t, map[string]string{"E-0001": "platform"})
	d, _, err := aiwfyaml.ReadBytes([]byte("hosts: [claude-code]\n"))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	res, err := RenameArea(context.Background(), tr, d,
		[]string{"platform", "billing"}, "", "platform", "infra", "human/test")
	if err == nil {
		t.Fatalf("expected error, got Plan=%v", res)
	}
	if !strings.Contains(err.Error(), "updating aiwf.yaml") {
		t.Errorf("error %q should wrap the aiwf.yaml update failure", err.Error())
	}
}

// TestRenameArea_MissingEntityFileErrors covers the readBody error
// seam: an entity present in the tree but missing on disk (corruption /
// race) aborts the verb before any Plan is produced.
func TestRenameArea_MissingEntityFileErrors(t *testing.T) {
	t.Parallel()
	tr := areaTree(t, map[string]string{"E-0001": "platform"})
	if err := os.Remove(filepath.Join(tr.Root, tr.Entities[0].Path)); err != nil {
		t.Fatalf("remove entity file: %v", err)
	}
	res, err := RenameArea(context.Background(), tr, mustReadAreaDoc(t),
		[]string{"platform", "billing"}, "", "platform", "infra", "human/test")
	if err == nil {
		t.Fatalf("expected error for missing entity file, got Plan=%v", res)
	}
}

// TestDeclaredList pins the empty-set rendering of the error helper.
func TestDeclaredList(t *testing.T) {
	t.Parallel()
	if got := declaredList(nil); got != "(none)" {
		t.Errorf("declaredList(nil) = %q, want (none)", got)
	}
	if got := declaredList([]string{"a", "b"}); got != "a, b" {
		t.Errorf("declaredList = %q, want 'a, b'", got)
	}
}
