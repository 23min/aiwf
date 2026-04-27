package verb_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

const testActor = "human/test"

// runner bundles the per-test context (testing.T, ctx, root) so verb
// invocations can use multi-value passing: r.must(verb.Add(...)).
type runner struct {
	t    *testing.T
	ctx  context.Context
	root string
}

func newRunner(t *testing.T) *runner {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	root := t.TempDir()
	if err := gitops.Init(context.Background(), root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return &runner{t: t, ctx: context.Background(), root: root}
}

// must asserts the verb returned no Go error and no error-severity
// findings, then applies the plan. Returns the result so warnings can
// be inspected.
func (r *runner) must(res *verb.Result, err error) *verb.Result {
	r.t.Helper()
	if err != nil {
		r.t.Fatalf("verb error: %v", err)
	}
	if check.HasErrors(res.Findings) {
		r.t.Fatalf("unexpected findings: %+v", res.Findings)
	}
	if res.Plan == nil {
		r.t.Fatal("no plan produced")
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		r.t.Fatalf("apply: %v", applyErr)
	}
	return res
}

// tree reloads the on-disk tree.
func (r *runner) tree() *tree.Tree {
	r.t.Helper()
	tr, loadErrs, err := tree.Load(r.ctx, r.root)
	if err != nil {
		r.t.Fatalf("tree.Load: %v", err)
	}
	if len(loadErrs) != 0 {
		r.t.Fatalf("loadErrs: %+v", loadErrs)
	}
	return tr
}

func TestAdd_Epic_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	wantPath := filepath.Join(r.root, "work", "epics", "E-01-foundations", "epic.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("epic.md missing: %v", err)
	}

	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-add tree has errors: %+v", findings)
	}

	subj, err := gitops.HeadSubject(r.ctx, r.root)
	if err != nil || subj != `aiwf add epic E-01 "Foundations"` {
		t.Errorf("subject = %q (err %v)", subj, err)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-verb", "add")
	mustHaveTrailer(t, tr, "aiwf-entity", "E-01")
	mustHaveTrailer(t, tr, "aiwf-actor", testActor)
}

func TestAdd_MilestoneUnderEpic(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.tree(), entity.KindMilestone, "Cache warmup", testActor, verb.AddOptions{EpicID: "E-01"}))

	wantPath := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-001-cache-warmup.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("milestone missing: %v", err)
	}

	m := r.tree().ByID("M-001")
	if m == nil || m.Parent != "E-01" {
		t.Errorf("M-001 = %+v", m)
	}
}

func TestAdd_MilestoneRequiresEpic(t *testing.T) {
	r := newRunner(t)
	_, err := verb.Add(r.tree(), entity.KindMilestone, "Orphan", testActor, verb.AddOptions{})
	if err == nil || !strings.Contains(err.Error(), "--epic") {
		t.Errorf("expected --epic error, got %v", err)
	}
}

func TestAdd_AllocatesSequentially(t *testing.T) {
	r := newRunner(t)
	for i := 0; i < 3; i++ {
		r.must(verb.Add(r.tree(), entity.KindEpic, "Epic", testActor, verb.AddOptions{}))
	}
	want := map[string]bool{"E-01": true, "E-02": true, "E-03": true}
	for _, e := range r.tree().Entities {
		if !want[e.ID] {
			t.Errorf("unexpected id %q", e.ID)
		}
	}
}

func TestPromote_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.tree(), "E-01", "active", testActor))

	if e := r.tree().ByID("E-01"); e == nil || e.Status != "active" {
		t.Errorf("E-01 = %+v", e)
	}
}

func TestPromote_RejectsBadTransition(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	_, err := verb.Promote(r.tree(), "E-01", "done", testActor)
	if err == nil || !strings.Contains(err.Error(), "cannot transition") {
		t.Errorf("expected illegal-transition error, got %v", err)
	}
}

func TestCancel_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Cancel(r.tree(), "E-01", testActor))

	if e := r.tree().ByID("E-01"); e == nil || e.Status != "cancelled" {
		t.Errorf("E-01 = %+v", e)
	}
}

func TestRename_FilePath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.tree(), entity.KindMilestone, "Cache warmup", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Rename(r.tree(), "M-001", "warm-the-cache", testActor))

	wantPath := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-001-warm-the-cache.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("renamed milestone missing: %v", err)
	}
}

func TestRename_DirectoryKind(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Old name", testActor, verb.AddOptions{}))
	r.must(verb.Rename(r.tree(), "E-01", "new-name", testActor))

	wantPath := filepath.Join(r.root, "work", "epics", "E-01-new-name", "epic.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("renamed epic missing: %v", err)
	}
}

func TestReallocate_RewritesReferences(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.tree(), entity.KindMilestone, "Depends on first", testActor, verb.AddOptions{EpicID: "E-01"}))

	// Hand-edit M-002 to depend on M-001.
	m2Path := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-002-depends-on-first.md")
	content, _ := os.ReadFile(m2Path)
	updated := strings.Replace(string(content), "parent: E-01", "parent: E-01\ndepends_on:\n  - M-001", 1)
	if err := os.WriteFile(m2Path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	r.must(verb.Reallocate(r.tree(), "M-001", testActor))

	tr := r.tree()
	if e := tr.ByID("M-001"); e != nil {
		t.Errorf("M-001 still present after reallocate: %+v", e)
	}
	if e := tr.ByID("M-003"); e == nil {
		t.Fatal("M-003 (renumber target) missing")
	}

	m002 := tr.ByID("M-002")
	if m002 == nil || len(m002.DependsOn) != 1 || m002.DependsOn[0] != "M-003" {
		t.Errorf("M-002.depends_on = %+v, want [M-003]", m002)
	}

	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, trailers, "aiwf-verb", "reallocate")
	mustHaveTrailer(t, trailers, "aiwf-entity", "M-003")
	mustHaveTrailer(t, trailers, "aiwf-prior-entity", "M-001")
}

func TestReallocate_BodyProseSurfacesAsWarning(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.tree(), entity.KindMilestone, "Mention test", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.tree(), entity.KindMilestone, "Mentions M-001 in prose", testActor, verb.AddOptions{EpicID: "E-01"}))

	m2Path := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-002-mentions-m-001-in-prose.md")
	if err := os.WriteFile(m2Path, []byte(`---
id: M-002
title: Mentions M-001 in prose
status: draft
parent: E-01
---

This depends on M-001 (mentioned in prose).
`), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := verb.Reallocate(r.tree(), "M-001", testActor)
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan; reallocate should succeed alongside warnings")
	}
	if check.HasErrors(res.Findings) {
		t.Errorf("expected only warnings, got errors: %+v", res.Findings)
	}
	hasBodyWarning := false
	for _, f := range res.Findings {
		if f.Code == "reallocate-body-reference" {
			hasBodyWarning = true
			break
		}
	}
	if !hasBodyWarning {
		t.Errorf("expected reallocate-body-reference warning, got %+v", res.Findings)
	}
}

func TestVerb_FailingProjectionLeavesNoCommit(t *testing.T) {
	r := newRunner(t)

	// Missing --artifact-source for contract — verb returns Go error
	// before any file or commit lands.
	_, err := verb.Add(r.tree(), entity.KindContract, "API", testActor, verb.AddOptions{Format: "openapi"})
	if err == nil {
		t.Fatal("expected error for missing --artifact-source")
	}
	if _, err := gitops.HeadSubject(r.ctx, r.root); err == nil {
		t.Errorf("expected no commits in fresh repo, but got HEAD")
	}
}

func TestAddContract_WithArtifact(t *testing.T) {
	r := newRunner(t)

	srcDir := filepath.Join(r.root, "tmp")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(srcDir, "openapi.yaml")
	if err := os.WriteFile(srcPath, []byte("openapi: 3.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r.must(verb.Add(r.tree(), entity.KindContract, "Orders API", testActor, verb.AddOptions{
		Format:         "openapi",
		ArtifactSource: srcPath,
	}))

	contractDir := filepath.Join(r.root, "work", "contracts", "C-001-orders-api")
	if _, err := os.Stat(filepath.Join(contractDir, "contract.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(contractDir, "schema", "openapi.yaml")); err != nil {
		t.Fatalf("artifact missing in schema/: %v", err)
	}

	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-add contract tree has errors: %+v", findings)
	}
}

func mustHaveTrailer(t *testing.T, trailers []gitops.Trailer, key, value string) {
	t.Helper()
	for _, tr := range trailers {
		if tr.Key == key && tr.Value == value {
			return
		}
	}
	t.Errorf("trailer %s=%q missing from %+v", key, value, trailers)
}
