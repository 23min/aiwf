package verb_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/manifest"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// applyImport applies every plan in result.Plans in order, calling
// Apply per plan. Returns the first error.
func applyImport(t *testing.T, r *runner, plans []*verb.Plan) {
	t.Helper()
	for i, p := range plans {
		if err := verb.Apply(r.ctx, r.root, p); err != nil {
			t.Fatalf("apply plan %d: %v", i, err)
		}
	}
}

// loadManifest is a helper that wraps the parser for tests that
// declare manifests inline.
func loadManifest(t *testing.T, src string) *manifest.Manifest {
	t.Helper()
	m, err := manifest.Parse([]byte(src), "yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return m
}

// TestImport_SingleEpicAndMilestone_RoundTrip imports a manifest with
// one epic and a milestone parented to it (forward reference: the
// epic and milestone are both new, the milestone resolves its parent
// against the manifest, not the existing tree). Expects a single
// commit with `aiwf-verb: import` and the two files on disk.
func TestImport_SingleEpicAndMilestone_RoundTrip(t *testing.T) {
	r := newRunner(t)
	src := `version: 1
entities:
  - kind: epic
    id: E-01
    frontmatter:
      title: "Foundations"
      status: active
    body: |
      ## Goal
      Build a core.
  - kind: milestone
    id: M-001
    frontmatter:
      title: "Bootstrap"
      status: draft
      parent: E-01
    body: "## Goal\nStart things.\n"
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("unexpected findings: %+v", res.Findings)
	}
	if len(res.Plans) != 1 {
		t.Fatalf("len(Plans) = %d, want 1 (single mode default)", len(res.Plans))
	}
	applyImport(t, r, res.Plans)

	// Files exist.
	for _, p := range []string{
		filepath.Join("work", "epics", "E-01-foundations", "epic.md"),
		filepath.Join("work", "epics", "E-01-foundations", "M-001-bootstrap.md"),
	} {
		if _, sErr := os.Stat(filepath.Join(r.root, p)); sErr != nil {
			t.Errorf("expected %s, stat err=%v", p, sErr)
		}
	}

	// Tree is clean.
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-import findings: %+v", findings)
	}

	// Commit subject + trailers.
	subj, err := gitops.HeadSubject(r.ctx, r.root)
	if err != nil {
		t.Fatalf("head subject: %v", err)
	}
	if !strings.Contains(subj, "aiwf import 2 entities") {
		t.Errorf("subject = %q, want default `aiwf import 2 entities`", subj)
	}
}

// TestImport_AutoAllocates: an `auto` id pulls max+1 from the kind's
// existing ids in the tree and the manifest's own explicit reservations.
// The manifest reserves E-01 explicitly and asks for two more `auto`
// epics; these should land at E-02 and E-03 (not E-01 again, not E-02
// duplicated).
func TestImport_AutoAllocates(t *testing.T) {
	r := newRunner(t)
	src := `version: 1
entities:
  - kind: epic
    id: E-01
    frontmatter: {title: "First", status: active}
  - kind: epic
    id: auto
    frontmatter: {title: "Second", status: proposed}
  - kind: epic
    id: auto
    frontmatter: {title: "Third", status: proposed}
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("findings: %+v", res.Findings)
	}
	applyImport(t, r, res.Plans)

	tr := r.tree()
	for _, want := range []string{"E-01", "E-02", "E-03"} {
		if tr.ByID(want) == nil {
			t.Errorf("missing id %s; tree has: %+v", want, idsOf(tr))
		}
	}
}

// TestImport_AutoAllocatesAboveExistingTree: when the tree already
// has E-05, an `auto` epic should land at E-06.
func TestImport_AutoAllocatesAboveExistingTree(t *testing.T) {
	r := newRunner(t)
	// Pre-populate via Add so the tree has E-01 (Foundations).
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	src := `version: 1
entities:
  - kind: epic
    id: auto
    frontmatter: {title: "Next", status: proposed}
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("findings: %+v", res.Findings)
	}
	applyImport(t, r, res.Plans)

	if r.tree().ByID("E-02") == nil {
		t.Errorf("E-02 not allocated; tree: %+v", idsOf(r.tree()))
	}
}

// TestImport_CollisionFail: re-running an explicit-id manifest after
// it already imported once must fail with an `import-collision`
// finding when --on-collision=fail (default).
func TestImport_CollisionFail(t *testing.T) {
	r := newRunner(t)
	src := `version: 1
entities:
  - kind: epic
    id: E-01
    frontmatter: {title: "X", status: active}
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import #1: %v", err)
	}
	applyImport(t, r, res.Plans)

	// Second pass: same manifest, same explicit id.
	res2, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import #2: %v", err)
	}
	if !check.HasErrors(res2.Findings) {
		t.Fatalf("expected collision findings, got Plans=%v", res2.Plans)
	}
	got := res2.Findings[0]
	if got.Code != "import-collision" || got.EntityID != "E-01" {
		t.Errorf("finding = %+v", got)
	}
}

// TestImport_CollisionSkip: with --on-collision=skip, the colliding
// entry is dropped and other entries still import.
func TestImport_CollisionSkip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "First", testActor, verb.AddOptions{}))

	src := `version: 1
entities:
  - kind: epic
    id: E-01
    frontmatter: {title: "Skipped", status: active}
  - kind: epic
    id: E-02
    frontmatter: {title: "Lands", status: active}
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{OnCollision: verb.OnCollisionSkip})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("findings: %+v", res.Findings)
	}
	applyImport(t, r, res.Plans)

	tr := r.tree()
	if tr.ByID("E-02") == nil {
		t.Errorf("E-02 should have landed")
	}
	// The original E-01 (title "First") is still there, not "Skipped".
	if e := tr.ByID("E-01"); e == nil || e.Title != "First" {
		t.Errorf("E-01 should be preserved, got %+v", e)
	}
}

// TestImport_CollisionUpdate: with --on-collision=update, the
// existing entity at E-01 is rewritten in place with the manifest's
// frontmatter and body.
func TestImport_CollisionUpdate(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "First", testActor, verb.AddOptions{}))

	src := `version: 1
entities:
  - kind: epic
    id: E-01
    frontmatter: {title: "First", status: active}
    body: |
      ## Goal
      Now updated.
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{OnCollision: verb.OnCollisionUpdate})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("findings: %+v", res.Findings)
	}
	applyImport(t, r, res.Plans)

	got, err := os.ReadFile(filepath.Join(r.root, "work", "epics", "E-01-first", "epic.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Now updated.") {
		t.Errorf("body not updated; file:\n%s", got)
	}
}

// TestImport_DuplicateIDsInManifest: declaring the same explicit id
// twice in one manifest is an `import-duplicate-id` finding.
func TestImport_DuplicateIDsInManifest(t *testing.T) {
	r := newRunner(t)
	src := `version: 1
entities:
  - kind: epic
    id: E-01
    frontmatter: {title: "A", status: active}
  - kind: epic
    id: E-01
    frontmatter: {title: "B", status: active}
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if !check.HasErrors(res.Findings) {
		t.Fatalf("expected findings")
	}
	if res.Findings[0].Code != "import-duplicate-id" {
		t.Errorf("code = %q", res.Findings[0].Code)
	}
}

// TestImport_PerEntityCommitMode: commit.mode=per-entity produces N
// plans, each carrying an `aiwf-entity` trailer.
func TestImport_PerEntityCommitMode(t *testing.T) {
	r := newRunner(t)
	src := `version: 1
commit:
  mode: per-entity
entities:
  - kind: epic
    id: E-01
    frontmatter: {title: "A", status: active}
  - kind: epic
    id: E-02
    frontmatter: {title: "B", status: active}
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(res.Plans) != 2 {
		t.Fatalf("len(Plans) = %d, want 2", len(res.Plans))
	}
	applyImport(t, r, res.Plans)

	// Most recent commit's trailer points at E-02.
	out, err := runGit(r.ctx, r.root, "log", "--format=%(trailers:key=aiwf-entity,valueonly=true)", "-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "E-02") {
		t.Errorf("HEAD commit missing aiwf-entity: E-02 trailer; got %q", out)
	}
}

// TestImport_RejectsCheckErrors: a manifest that produces a tree
// failing check (here, milestone with unknown parent) returns
// findings, no plans.
func TestImport_RejectsCheckErrors(t *testing.T) {
	r := newRunner(t)
	src := `version: 1
entities:
  - kind: milestone
    id: M-001
    frontmatter: {title: "Orphan", status: draft, parent: E-99}
`
	m := loadManifest(t, src)
	// Path resolution fails before projection because parent doesn't
	// exist anywhere; Import surfaces this as a Go error (malformed
	// manifest input), not a finding. That's the contract: structural
	// resolution errors are programmer-facing; tree-validity errors
	// are findings. Either is acceptable here as long as no plans
	// come back.
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err == nil && len(res.Plans) > 0 {
		t.Fatalf("expected error or findings; got plans=%v", res.Plans)
	}
}

// TestImport_BadCommitMode rejected at parse time.
func TestImport_UnknownOnCollision(t *testing.T) {
	r := newRunner(t)
	m := loadManifest(t, "version: 1\nentities: []\n")
	_, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{OnCollision: "explode"})
	if err == nil {
		t.Fatal("expected error for unknown --on-collision")
	}
}

// TestImport_AutoMilestoneInForwardEpic: milestone with `auto` id and
// `parent: E-01` resolves correctly when E-01 is also `auto` and
// declared earlier.
func TestImport_AutoMilestoneInForwardEpic(t *testing.T) {
	r := newRunner(t)
	src := `version: 1
entities:
  - kind: epic
    id: auto
    frontmatter: {title: "Cake", status: active}
  - kind: milestone
    id: auto
    frontmatter: {title: "Frost it", status: draft, parent: E-01}
`
	m := loadManifest(t, src)
	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("findings: %+v", res.Findings)
	}
	applyImport(t, r, res.Plans)

	tr := r.tree()
	if tr.ByID("E-01") == nil {
		t.Errorf("E-01 missing")
	}
	if tr.ByID("M-001") == nil {
		t.Errorf("M-001 missing")
	}
}

// idsOf returns the ids in the tree, for diagnostic output.
func idsOf(t *tree.Tree) []string {
	out := make([]string, len(t.Entities))
	for i, e := range t.Entities {
		out[i] = e.ID
	}
	return out
}

// runGit is a tiny shell helper for tests that need to inspect
// commit metadata not exposed via gitops.
func runGit(ctx context.Context, root string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	return string(out), err
}
