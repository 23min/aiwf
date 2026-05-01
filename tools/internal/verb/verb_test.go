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
// invocations can use multi-value passing: r.must(verb.Add(context.Background(), ...)).
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
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

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
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache warmup", testActor, verb.AddOptions{EpicID: "E-01"}))

	wantPath := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-001-cache-warmup.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("milestone missing: %v", err)
	}

	m := r.tree().ByID("M-001")
	if m == nil || m.Parent != "E-01" {
		t.Errorf("M-001 = %+v", m)
	}
}

// TestAdd_GapDiscoveredInStubbedEntity is the verb-projection
// belt-and-braces for G14: when a referrer is added that points at an
// entity whose source file failed to parse, the stub mechanism must
// keep the verb from being blocked by a "newly introduced" unresolved
// reference. Pre-fix this test would fail with a refs-resolve finding
// on the new gap; post-fix the stub resolves the reference and the
// verb succeeds. Confirms that Stubs propagate from loaded tree
// through projectAdd's shallow copy into the projection check.
func TestAdd_GapDiscoveredInStubbedEntity(t *testing.T) {
	r := newRunner(t)

	// Drop a corrupt epic.md directly — the wrap-epic skill bug
	// shape: an unknown frontmatter field rejected by KnownFields.
	corrupt := []byte(`---
id: E-01
title: Platform
status: active
completed: 2026-04-30
---
`)
	if err := os.MkdirAll(filepath.Join(r.root, "work/epics/E-01-platform"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(r.root, "work/epics/E-01-platform/epic.md"), corrupt, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Load the tree directly — r.tree() would fail-fast on loadErrs.
	tr, loadErrs, err := tree.Load(r.ctx, r.root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	if len(loadErrs) != 1 {
		t.Fatalf("expected 1 loadErr; got %+v", loadErrs)
	}
	if len(tr.Stubs) != 1 || tr.Stubs[0].ID != "E-01" {
		t.Fatalf("expected stub for E-01; got %+v", tr.Stubs)
	}

	// Add a gap that references the stubbed E-01. Pre-fix, the
	// projection check would surface a refs-resolve/unresolved on the
	// new gap and the verb would fail. Post-fix the stub resolves it.
	res, err := verb.Add(r.ctx, tr, entity.KindGap, "Flaky", testActor, verb.AddOptions{DiscoveredIn: "E-01"})
	if err != nil {
		t.Fatalf("verb.Add: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("expected no error findings (stub should resolve discovered_in); got: %+v", res.Findings)
	}
	if res.Plan == nil {
		t.Fatal("no plan produced")
	}
}

func TestAdd_MilestoneRequiresEpic(t *testing.T) {
	r := newRunner(t)
	_, err := verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Orphan", testActor, verb.AddOptions{})
	if err == nil || !strings.Contains(err.Error(), "--epic") {
		t.Errorf("expected --epic error, got %v", err)
	}
}

func TestAdd_AllocatesSequentially(t *testing.T) {
	r := newRunner(t)
	for i := 0; i < 3; i++ {
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Epic", testActor, verb.AddOptions{}))
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
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "", false))

	if e := r.tree().ByID("E-01"); e == nil || e.Status != "active" {
		t.Errorf("E-01 = %+v", e)
	}
}

func TestPromote_RejectsBadTransition(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	_, err := verb.Promote(r.ctx, r.tree(), "E-01", "done", testActor, "", false)
	if err == nil || !strings.Contains(err.Error(), "cannot transition") {
		t.Errorf("expected illegal-transition error, got %v", err)
	}
}

func TestCancel_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Cancel(r.ctx, r.tree(), "E-01", testActor, "", false))

	if e := r.tree().ByID("E-01"); e == nil || e.Status != "cancelled" {
		t.Errorf("E-01 = %+v", e)
	}
}

// TestCancel_WithReason: --reason prose lands in the commit body
// between the subject and the trailers, queryable via `git show`.
func TestCancel_WithReason(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Cancel(r.ctx, r.tree(), "E-01", testActor, "scope folded into E-02", false))

	body, err := gitops.HeadBody(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadBody: %v", err)
	}
	if !strings.Contains(body, "scope folded into E-02") {
		t.Errorf("body missing reason text: %q", body)
	}
}

// TestPromote_WithReason mirrors TestCancel_WithReason for promote.
func TestPromote_WithReason(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "kicking off after the planning review", false))

	body, err := gitops.HeadBody(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadBody: %v", err)
	}
	if !strings.Contains(body, "kicking off after the planning review") {
		t.Errorf("body missing reason text: %q", body)
	}
}

func TestRename_FilePath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache warmup", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Rename(r.ctx, r.tree(), "M-001", "warm-the-cache", testActor))

	wantPath := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-001-warm-the-cache.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("renamed milestone missing: %v", err)
	}
}

func TestRename_DirectoryKind(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Old name", testActor, verb.AddOptions{}))
	r.must(verb.Rename(r.ctx, r.tree(), "E-01", "new-name", testActor))

	wantPath := filepath.Join(r.root, "work", "epics", "E-01-new-name", "epic.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("renamed epic missing: %v", err)
	}
}

func TestReallocate_RewritesReferences(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Depends on first", testActor, verb.AddOptions{EpicID: "E-01"}))

	// Hand-edit M-002 to depend on M-001.
	m2Path := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-002-depends-on-first.md")
	content, _ := os.ReadFile(m2Path)
	updated := strings.Replace(string(content), "parent: E-01", "parent: E-01\ndepends_on:\n  - M-001", 1)
	if err := os.WriteFile(m2Path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	r.must(verb.Reallocate(r.ctx, r.tree(), "M-001", testActor))

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

// TestAdd_NonASCIITitle_SurfacesSlugWarning is the load-bearing test
// for G8: an `aiwf add` with a non-ASCII title (`Café`) succeeds,
// produces the expected ASCII slug, AND surfaces a warning naming
// the dropped characters and the resulting slug. The user is no
// longer surprised when they later try to rename to a related slug.
func TestAdd_NonASCIITitle_SurfacesSlugWarning(t *testing.T) {
	r := newRunner(t)
	res, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, "Café au Lait", testActor, verb.AddOptions{})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan; warning should not block")
	}
	if check.HasErrors(res.Findings) {
		t.Errorf("expected only warning, got errors: %+v", res.Findings)
	}
	hasWarning := false
	for _, f := range res.Findings {
		if f.Code == "slug-dropped-chars" {
			hasWarning = true
			if !strings.Contains(f.Message, `"é"`) {
				t.Errorf("warning should name the dropped char 'é'; got %q", f.Message)
			}
			if !strings.Contains(f.Message, `"caf-au-lait"`) {
				t.Errorf("warning should name the resulting slug; got %q", f.Message)
			}
			if f.EntityID != "E-01" {
				t.Errorf("warning entity id = %q, want E-01", f.EntityID)
			}
		}
	}
	if !hasWarning {
		t.Errorf("expected slug-dropped-chars warning, got %+v", res.Findings)
	}
	// Apply the plan and confirm the entity actually lands.
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}
	if _, err := os.Stat(filepath.Join(r.root, "work", "epics", "E-01-caf-au-lait", "epic.md")); err != nil {
		t.Errorf("entity not created at expected path: %v", err)
	}
}

// TestAdd_PureASCIITitle_NoWarning: a normal English title gets no
// slug warning (regression check that we don't flag everything).
func TestAdd_PureASCIITitle_NoWarning(t *testing.T) {
	r := newRunner(t)
	res, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, "Cache Warmup", testActor, verb.AddOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Findings {
		if f.Code == "slug-dropped-chars" {
			t.Errorf("ASCII-only title should not produce slug warning; got %+v", f)
		}
	}
}

// TestRename_NonASCIINewSlug_SurfacesSlugWarning: same shape for
// rename — when the user passes a slug containing non-ASCII chars,
// they see what was dropped.
func TestRename_NonASCIINewSlug_SurfacesSlugWarning(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	res, err := verb.Rename(r.ctx, r.tree(), "E-01", "Café-Bar", testActor)
	if err != nil {
		t.Fatal(err)
	}
	hasWarning := false
	for _, f := range res.Findings {
		if f.Code == "slug-dropped-chars" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Errorf("expected slug-dropped-chars warning on rename; got %+v", res.Findings)
	}
}

// TestReallocate_RewritesProseReferences is the load-bearing test
// for G5: when reallocate renumbers an entity, prose mentions of
// the old id in other entities' bodies are rewritten to the new id
// in the same commit. No "fix it yourself" warnings.
func TestReallocate_RewritesProseReferences(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Mention test", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Mentions M-001 in prose", testActor, verb.AddOptions{EpicID: "E-01"}))

	m2Path := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-002-mentions-m-001-in-prose.md")
	if err := os.WriteFile(m2Path, []byte(`---
id: M-002
title: Mentions M-001 in prose
status: draft
parent: E-01
---

This depends on M-001 (mentioned in prose).
M-001 again, and a longer id M-0010 that must NOT match.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := verb.Reallocate(r.ctx, r.tree(), "M-001", testActor)
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan")
	}
	for _, f := range res.Findings {
		if f.Code == "reallocate-body-reference" {
			t.Errorf("body-reference warning should be gone now (we rewrite); got %+v", f)
		}
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}

	// Read M-002's body after reallocate; old id should be gone, new id present.
	got, err := os.ReadFile(m2Path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(got)
	if strings.Contains(body, "depends on M-001 (mentioned in prose).") {
		t.Errorf("body still mentions old id M-001:\n%s", body)
	}
	if !strings.Contains(body, "depends on M-003 (mentioned in prose).") {
		t.Errorf("body should mention new id M-003:\n%s", body)
	}
	// The longer id M-0010 must remain untouched (word boundary).
	if !strings.Contains(body, "M-0010 that must NOT match") {
		t.Errorf("M-0010 should be left alone; word-boundary regex required:\n%s", body)
	}
}

// TestReallocate_RewritesProseAcrossMultipleEntities: multiple
// other entities each mentioning the old id all get rewritten in
// one commit.
func TestReallocate_RewritesProseAcrossMultipleEntities(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Target", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Other A", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Other B", testActor, verb.AddOptions{EpicID: "E-01"}))

	for _, name := range []string{"M-002-other-a.md", "M-003-other-b.md"} {
		p := filepath.Join(r.root, "work", "epics", "E-01-platform", name)
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		updated := string(raw) + "\nReferences M-001 in prose.\n"
		if err := os.WriteFile(p, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	res, err := verb.Reallocate(r.ctx, r.tree(), "M-001", testActor)
	if err != nil {
		t.Fatal(err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}

	for _, name := range []string{"M-002-other-a.md", "M-003-other-b.md"} {
		body, err := os.ReadFile(filepath.Join(r.root, "work", "epics", "E-01-platform", name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "References M-001 in prose.") {
			t.Errorf("%s still mentions M-001:\n%s", name, body)
		}
		if !strings.Contains(string(body), "References M-004 in prose.") {
			t.Errorf("%s should reference M-004:\n%s", name, body)
		}
	}
}

// TestReallocate_RewritesSelfReferenceInTargetBody: the entity
// being reallocated may mention itself in its own body. That
// self-reference must update too.
func TestReallocate_RewritesSelfReferenceInTargetBody(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Target", testActor, verb.AddOptions{EpicID: "E-01"}))

	targetPath := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-001-target.md")
	raw, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(raw) + "\nThis is M-001 (self-reference in body).\n"
	if werr := os.WriteFile(targetPath, []byte(updated), 0o644); werr != nil {
		t.Fatal(werr)
	}

	res, err := verb.Reallocate(r.ctx, r.tree(), "M-001", testActor)
	if err != nil {
		t.Fatal(err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}

	// M-001 was renumbered to M-002 (next free).
	newPath := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-002-target.md")
	body, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("post-reallocate read: %v", err)
	}
	if strings.Contains(string(body), "This is M-001") {
		t.Errorf("self-reference to M-001 should have been rewritten:\n%s", body)
	}
	if !strings.Contains(string(body), "This is M-002") {
		t.Errorf("self-reference should now read M-002:\n%s", body)
	}
}

func TestAddContract_Minimal(t *testing.T) {
	r := newRunner(t)

	r.must(verb.Add(r.ctx, r.tree(), entity.KindContract, "Orders API", testActor, verb.AddOptions{}))

	contractDir := filepath.Join(r.root, "work", "contracts", "C-001-orders-api")
	if _, err := os.Stat(filepath.Join(contractDir, "contract.md")); err != nil {
		t.Fatal(err)
	}

	c := r.tree().ByID("C-001")
	if c == nil {
		t.Fatal("C-001 not found after add")
	}
	if c.Status != "proposed" {
		t.Errorf("status = %q, want %q (initial contract status)", c.Status, "proposed")
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

// --- Edge cases (items 1-7 from the test-coverage audit) ---

// TestReallocate_ByPath_DisambiguatesCollision exercises the
// merge-collision recovery flow: two entities share an id (impossible
// to reach via aiwf verbs alone, but realistic after `git merge`
// merges two branches that each independently allocated M-001 with
// different slugs). `aiwf reallocate <path>` picks the entity at that
// specific path and renumbers it.
func TestReallocate_ByPath_DisambiguatesCollision(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Original", testActor, verb.AddOptions{EpicID: "E-01"}))

	// Manually create a colliding M-001 with a different slug — the
	// shape a merge from a parallel branch would land in. Stage and
	// commit so git considers it tracked (real merges produce tracked
	// files; bare working-tree files would fail the eventual git mv).
	collidingPath := filepath.Join(r.root, "work", "epics", "E-01-platform", "M-001-from-other-branch.md")
	if err := os.WriteFile(collidingPath, []byte(`---
id: M-001
title: From other branch
status: draft
parent: E-01
---
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, "work/epics/E-01-platform/M-001-from-other-branch.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(r.ctx, r.root, "simulate merge of colliding M-001", "", nil); err != nil {
		t.Fatal(err)
	}

	// Resolving "M-001" by id is now ambiguous — t.ByID returns the
	// first one, which is the original. Resolve by path instead.
	collidingRel := "work/epics/E-01-platform/M-001-from-other-branch.md"
	res, err := verb.Reallocate(r.ctx, r.tree(), collidingRel, testActor)
	if err != nil {
		t.Fatalf("reallocate by path: %v", err)
	}
	if res.Plan == nil || check.HasErrors(res.Findings) {
		t.Fatalf("unexpected: %+v", res)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}

	tr := r.tree()
	// Original M-001 still present; the colliding one became M-002.
	if e := tr.ByID("M-001"); e == nil || e.Title != "Original" {
		t.Errorf("M-001 = %+v, want the original", e)
	}
	if e := tr.ByID("M-002"); e == nil || e.Title != "From other branch" {
		t.Errorf("M-002 = %+v, want the renumbered colliding entity", e)
	}

	// Tree validates clean.
	if findings := check.Run(tr, nil); check.HasErrors(findings) {
		t.Errorf("post-reallocate tree has errors: %+v", findings)
	}

	// Trailer carries both new and prior id.
	trailers, _ := gitops.HeadTrailers(r.ctx, r.root)
	mustHaveTrailer(t, trailers, "aiwf-entity", "M-002")
	mustHaveTrailer(t, trailers, "aiwf-prior-entity", "M-001")
}

// TestReallocate_Contract exercises the directory-rename flow:
// reallocate a contract (which lives in a directory) and verify that
// the dir moved and the contract.md's frontmatter id was rewritten.
func TestReallocate_Contract(t *testing.T) {
	r := newRunner(t)

	r.must(verb.Add(r.ctx, r.tree(), entity.KindContract, "Orders API", testActor, verb.AddOptions{}))

	// Trigger reallocate (any reason — we're testing the directory move).
	r.must(verb.Reallocate(r.ctx, r.tree(), "C-001", testActor))

	// New contract dir holds contract.md.
	newDir := filepath.Join(r.root, "work", "contracts", "C-002-orders-api")
	if _, err := os.Stat(filepath.Join(newDir, "contract.md")); err != nil {
		t.Fatalf("contract.md missing in new dir: %v", err)
	}

	// Old dir is gone.
	oldDir := filepath.Join(r.root, "work", "contracts", "C-001-orders-api")
	if _, err := os.Stat(oldDir); err == nil {
		t.Errorf("old contract dir still present at %s", oldDir)
	}

	// Frontmatter id is rewritten.
	c := r.tree().ByID("C-002")
	if c == nil {
		t.Fatal("C-002 not found")
	}
	if c.Title != "Orders API" {
		t.Errorf("C-002 title = %q, want %q", c.Title, "Orders API")
	}

	// Tree validates clean.
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-reallocate tree has errors: %+v", findings)
	}
}

// TestReallocate_EpicWithMilestoneInside is the regression for a bug
// where reallocating an epic correctly rewrote the inner milestone's
// `parent` field but wrote the milestone's file at its pre-move path,
// which recreated the old epic directory and produced both
// ids-unique and refs-resolve errors. The milestone should land
// inside the new epic dir with the rewritten parent.
func TestReallocate_EpicWithMilestoneInside(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache layer", testActor, verb.AddOptions{EpicID: "E-01"}))

	r.must(verb.Reallocate(r.ctx, r.tree(), "E-01", testActor))

	// New epic dir holds both epic.md and the milestone, parented to E-02.
	newDir := filepath.Join(r.root, "work", "epics", "E-02-platform")
	if _, err := os.Stat(filepath.Join(newDir, "epic.md")); err != nil {
		t.Errorf("epic.md missing in new dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, "M-001-cache-layer.md")); err != nil {
		t.Errorf("milestone missing in new dir: %v", err)
	}

	// Old dir is gone (no resurrection of the source).
	oldDir := filepath.Join(r.root, "work", "epics", "E-01-platform")
	if _, err := os.Stat(oldDir); err == nil {
		t.Errorf("old epic dir resurrected at %s", oldDir)
	}

	// Tree validates clean: no ids-unique, no unresolved parent refs.
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-reallocate tree has errors: %+v", findings)
	}

	m1 := r.tree().ByID("M-001")
	if m1 == nil {
		t.Fatal("M-001 missing after reallocate")
	}
	if m1.Parent != "E-02" {
		t.Errorf("M-001.parent = %q, want %q", m1.Parent, "E-02")
	}
}

// TestPromote_ForceSkipsFSM lets a normally-illegal transition through
// when force=true is set. Without force, proposed → done would be
// rejected by the FSM (proposed only goes to active or cancelled). With
// force, the verb writes the new status and produces a clean plan.
func TestPromote_ForceSkipsFSM(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))

	// Sanity: without force, proposed → done is illegal.
	if _, err := verb.Promote(r.ctx, r.tree(), "E-01", "done", testActor, "", false); err == nil {
		t.Fatal("expected illegal-transition error without force")
	}
	// With force, the same transition lands.
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "done", testActor, "the rare emergency", true))
	if e := r.tree().ByID("E-01"); e == nil || e.Status != "done" {
		t.Errorf("E-01 = %+v after forced promote", e)
	}
}

// TestPromote_ForceStillFailsCoherence: --force relaxes only the
// transition-legality rule. A move to a status outside the kind's
// closed set is still caught — by the projection's status-valid
// finding, not by ValidateTransition.
func TestPromote_ForceStillFailsCoherence(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))

	// Force does not allow promoting to a non-kind status.
	result, err := verb.Promote(r.ctx, r.tree(), "E-01", "in_progress", testActor, "tried to skip the FSM", true)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result with findings")
	}
	if !check.HasErrors(result.Findings) {
		t.Errorf("expected status-valid finding, got %+v", result.Findings)
	}
	foundStatusValid := false
	for _, f := range result.Findings {
		if f.Code == "status-valid" {
			foundStatusValid = true
			break
		}
	}
	if !foundStatusValid {
		t.Errorf("expected a status-valid finding among %+v", result.Findings)
	}
}

// TestPromote_ForceEmitsTrailer: a forced promote produces an
// `aiwf-force: <reason>` trailer alongside the standard ones, so
// `aiwf history` can render forced events distinctly.
func TestPromote_ForceEmitsTrailer(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "done", testActor, "the rare emergency", true))

	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var found gitops.Trailer
	for _, tr := range trailers {
		if tr.Key == "aiwf-force" {
			found = tr
			break
		}
	}
	if found.Key == "" {
		t.Fatalf("aiwf-force trailer missing; got %+v", trailers)
	}
	if found.Value != "the rare emergency" {
		t.Errorf("aiwf-force value = %q, want %q", found.Value, "the rare emergency")
	}
}

// TestPromote_NoForceNoTrailer: a normal (non-forced) promote must NOT
// emit `aiwf-force`. Backwards-compat guard for the `aiwf history`
// renderer which distinguishes forced events.
func TestPromote_NoForceNoTrailer(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "kicking off", false))

	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	for _, tr := range trailers {
		if tr.Key == "aiwf-force" {
			t.Errorf("non-forced promote emitted aiwf-force trailer: %+v", tr)
		}
	}
}

// TestCancel_ForceEmitsTrailer covers the cancel path. Cancel has no
// FSM rule to relax (any non-target status → target is permitted), so
// force here is purely an audit signal — but the trailer still lands.
func TestCancel_ForceEmitsTrailer(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Cancel(r.ctx, r.tree(), "E-01", testActor, "policy violation", true))

	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var force gitops.Trailer
	for _, tr := range trailers {
		if tr.Key == "aiwf-force" {
			force = tr
		}
	}
	if force.Key == "" {
		t.Fatalf("aiwf-force trailer missing on forced cancel; got %+v", trailers)
	}
	if force.Value != "policy violation" {
		t.Errorf("aiwf-force value = %q, want %q", force.Value, "policy violation")
	}
}

// TestPromote_ForceTrailerTrimsReason confirms the trailer value is
// the trimmed form of the reason (leading/trailing whitespace removed).
// The body itself is rendered verbatim by gitops.CommitMessage which
// also trims; this test pins the trailer side specifically.
func TestPromote_ForceTrailerTrimsReason(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "done", testActor, "  whitespace around it  ", true))

	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	for _, tr := range trailers {
		if tr.Key == "aiwf-force" {
			if tr.Value != "whitespace around it" {
				t.Errorf("aiwf-force value = %q, want trimmed %q", tr.Value, "whitespace around it")
			}
			return
		}
	}
	t.Fatal("aiwf-force trailer missing")
}

// TestPromote_NonExistentID returns a Go error before any disk work.
func TestPromote_NonExistentID(t *testing.T) {
	r := newRunner(t)
	_, err := verb.Promote(r.ctx, r.tree(), "E-99", "active", testActor, "", false)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestCancel_NonExistentID covers the same path for cancel.
func TestCancel_NonExistentID(t *testing.T) {
	r := newRunner(t)
	_, err := verb.Cancel(r.ctx, r.tree(), "M-99", testActor, "", false)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestRename_NonExistentID covers the same path for rename.
func TestRename_NonExistentID(t *testing.T) {
	r := newRunner(t)
	_, err := verb.Rename(r.ctx, r.tree(), "E-99", "new-slug", testActor)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestReallocate_NonExistentTarget covers the same path for reallocate.
func TestReallocate_NonExistentTarget(t *testing.T) {
	r := newRunner(t)
	_, err := verb.Reallocate(r.ctx, r.tree(), "X-99", testActor)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestCancel_AlreadyTerminal returns an error rather than producing a
// no-op commit.
func TestCancel_AlreadyTerminal(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed twice", testActor, verb.AddOptions{}))
	r.must(verb.Cancel(r.ctx, r.tree(), "E-01", testActor, "", false))

	_, err := verb.Cancel(r.ctx, r.tree(), "E-01", testActor, "", false)
	if err == nil || !strings.Contains(err.Error(), "already") {
		t.Errorf("expected 'already cancelled' error, got %v", err)
	}
}

// TestRename_SameSlug returns an error rather than producing a no-op
// commit.
func TestRename_SameSlug(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Same name", testActor, verb.AddOptions{}))
	_, err := verb.Rename(r.ctx, r.tree(), "E-01", "same-name", testActor)
	if err == nil || !strings.Contains(err.Error(), "matches the current slug") {
		t.Errorf("expected same-slug error, got %v", err)
	}
}

// TestAdd_GapWithDiscoveredIn confirms the --discovered-in flag wires
// through to the gap's frontmatter and resolves correctly.
func TestAdd_GapWithDiscoveredIn(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Need a thing", testActor, verb.AddOptions{DiscoveredIn: "M-001"}))

	g := r.tree().ByID("G-001")
	if g == nil || g.DiscoveredIn != "M-001" {
		t.Errorf("G-001 = %+v, want discovered_in: M-001", g)
	}
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("tree errors: %+v", findings)
	}
}

// --- Pre-existing-error isolation (item 15) ---

// TestAdd_PreExistingErrorDoesNotBlockUnrelatedVerb verifies the
// behavior change from item 15: a verb's projection only blocks on
// findings *introduced* by the change, not on errors that already
// existed in the loaded tree. Lets users incrementally fix a partially
// broken tree without first having to clean up unrelated breakage.
func TestAdd_PreExistingErrorDoesNotBlockUnrelatedVerb(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))

	// Hand-edit a broken state into the tree: a gap whose
	// discovered_in points to a non-existent milestone. This is a
	// pre-existing refs-resolve/unresolved error.
	gapPath := filepath.Join(r.root, "work", "gaps", "G-001-broken.md")
	if err := os.MkdirAll(filepath.Dir(gapPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gapPath, []byte(`---
id: G-001
title: Broken gap
status: open
discovered_in: M-999
---
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Confirm the tree is in fact broken before we try anything.
	pre := check.Run(r.tree(), nil)
	if !check.HasErrors(pre) {
		t.Fatal("setup invalid: expected pre-existing error before testing")
	}

	// Add an unrelated epic — the verb should not block on the gap's
	// pre-existing error.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Second epic", testActor, verb.AddOptions{}))

	// New entity exists.
	if e := r.tree().ByID("E-02"); e == nil {
		t.Error("E-02 was not created — pre-existing error blocked the verb")
	}
	// Pre-existing error still exists (verbs don't fix unrelated state).
	if !check.HasErrors(check.Run(r.tree(), nil)) {
		t.Error("pre-existing error disappeared somehow — fixture drift?")
	}
}

// TestAdd_DecisionWithRelatesTo confirms the --relates-to flag wires
// through to the decision's relates_to list and resolves correctly.
func TestAdd_DecisionWithRelatesTo(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindDecision, "Pin the order", testActor, verb.AddOptions{
		RelatesTo: []string{"E-01", "M-001"},
	}))

	d := r.tree().ByID("D-001")
	if d == nil || len(d.RelatesTo) != 2 {
		t.Fatalf("D-001 = %+v, want relates_to: [E-01 M-001]", d)
	}
	if d.RelatesTo[0] != "E-01" || d.RelatesTo[1] != "M-001" {
		t.Errorf("relates_to = %v", d.RelatesTo)
	}
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("tree errors: %+v", findings)
	}
}
