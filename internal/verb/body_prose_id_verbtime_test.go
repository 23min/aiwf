package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/manifest"
	"github.com/23min/aiwf/internal/verb"
)

// G-0184 verb-time scan tests. The body-prose-id rule is enforced at
// verb time across every verb that ingests operator-supplied body
// content (add, edit-body, import, reallocate, rewidth). These tests
// pin that each verb refuses with body-prose-id findings instead of
// writing the bad content to disk; positive controls verify clean
// bodies still flow through.

// TestAdd_RefusesMalformedIDInBody pins the add --body-file verb-time
// gate: a body containing a malformed id-shaped token (`M-a`) produces
// findings and no Plan; no file is written.
func TestAdd_RefusesMalformedIDInBody(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	body := "## What's missing\n\nDepends on M-a and M-NNNN.\n\n## Why it matters\n\nMatters.\n"

	res, err := verb.Add(r.ctx, r.tree(), entity.KindGap, "Bad body gap", testActor, verb.AddOptions{
		BodyOverride: []byte(body),
	})
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan != nil {
		t.Errorf("expected no Plan; verb should have refused with findings")
	}
	if !findingsContainSubcode(res.Findings, check.CodeBodyProseID, "malformed-shape") {
		t.Errorf("expected body-prose-id/malformed-shape finding; got %+v", res.Findings)
	}
}

// TestAdd_AllowsCleanBody pins the positive control: a body with no
// id-shaped tokens (or only correctly-backticked ones) flows through.
func TestAdd_AllowsCleanBody(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	body := "## What's missing\n\nDescription of the gap with `M-NNNN` placeholder syntax in backticks.\n\n## Why it matters\n\nMatters.\n"

	res := r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Clean body gap", testActor, verb.AddOptions{
		BodyOverride: []byte(body),
	}))
	if res.Plan == nil {
		t.Errorf("expected Plan; clean body should succeed")
	}
}

// TestEditBody_Explicit_RefusesMalformedIDInBody pins the edit-body
// --body-file verb-time gate.
func TestEditBody_Explicit_RefusesMalformedIDInBody(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Carrier epic", testActor, verb.AddOptions{}))

	badBody := []byte("## Goal\n\nDepends on M-alpha.\n\n## Scope\n\nScope.\n\n## Out of scope\n\nOOS.\n")
	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", badBody, testActor, "")
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan != nil {
		t.Errorf("expected no Plan; edit-body should have refused")
	}
	if !findingsContainSubcode(res.Findings, check.CodeBodyProseID, "malformed-shape") {
		t.Errorf("expected body-prose-id/malformed-shape finding; got %+v", res.Findings)
	}
}

// TestEditBody_Bless_RefusesMalformedIDInBody pins the bless-mode
// (working-copy edit) verb-time gate. The test hand-edits the entity
// file with a malformed id, then invokes the bless flow (nil body).
func TestEditBody_Bless_RefusesMalformedIDInBody(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Carrier epic", testActor, verb.AddOptions{}))

	epicPath := filepath.Join(r.root, "work", "epics", "E-0001-carrier-epic", "epic.md")
	committed, err := os.ReadFile(epicPath)
	if err != nil {
		t.Fatal(err)
	}
	// Insert a malformed id into the body; keep frontmatter unchanged.
	tainted := strings.Replace(string(committed),
		"## Goal\n",
		"## Goal\n\nDepends on M-foo.\n",
		1)
	if writeErr := os.WriteFile(epicPath, []byte(tainted), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", nil, testActor, "")
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan != nil {
		t.Errorf("expected no Plan; bless mode should have refused")
	}
	if !findingsContainSubcode(res.Findings, check.CodeBodyProseID, "malformed-shape") {
		t.Errorf("expected body-prose-id/malformed-shape finding; got %+v", res.Findings)
	}
}

// TestRewidth_RefusesMalformedIDInRewrittenBody pins the negative
// path for rewidth's verb-time scan: a malformed token in the post-
// rewidth body produces a finding whose EntityID is the canonical id
// string (not the file path). Reviewer pass 2 caught a B2-residual bug
// where the pathToID lookup missed because the projected tree's
// Path field hadn't been widened; parsing the entity ID directly from
// the op's frontmatter fixes that and this test asserts the shape.
func TestRewidth_RefusesMalformedIDInRewrittenBody(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	epicDir := filepath.Join(r.root, "work", "epics", "E-22-narrow")
	if err := os.MkdirAll(epicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Narrow-id fixture with a malformed `M-alpha` token in the body.
	epicBody := "---\nid: E-22\ntitle: Narrow epic\nstatus: active\n---\n\n## Goal\n\nReferences M-alpha which is malformed.\n\n## Scope\n\nScope.\n\n## Out of scope\n\nOOS.\n"
	if err := os.WriteFile(filepath.Join(epicDir, "epic.md"), []byte(epicBody), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := verb.Rewidth(r.ctx, r.root, testActor)
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan != nil {
		t.Errorf("expected no Plan; rewidth should have refused on malformed body token")
	}
	if !findingsContainSubcode(res.Findings, check.CodeBodyProseID, "malformed-shape") {
		t.Fatalf("expected body-prose-id/malformed-shape finding; got %+v", res.Findings)
	}
	// B2 regression: EntityID must be the canonical id `E-0022`, NOT
	// the file path. Mis-fix in pass 2 would set EntityID = the
	// post-rename path because pathToID was keyed off the pre-rename
	// path; parsing directly from op.Content's frontmatter avoids
	// that mismatch.
	for i := range res.Findings {
		f := &res.Findings[i]
		if f.Code != check.CodeBodyProseID {
			continue
		}
		if f.EntityID != "E-0022" {
			t.Errorf("finding.EntityID = %q, want canonical id %q (not the file path)", f.EntityID, "E-0022")
		}
	}
}

// TestRewidth_AllowsNarrowToCanonicalBodyRefs is the regression pin
// for the B1 bug surfaced by the G-0184-verb-time reviewer pass: a
// narrow-on-disk fixture (`E-22` epic with `M-77` milestone) gets
// rewritten to canonical width during rewidth. The post-rewidth body
// references the canonical `M-0077` form, which must resolve against
// the verb-time scan's index. The index built via the naive path
// (entity.Canonicalize keyed on pre-rewidth IDs) would miss because
// Canonicalize passes below-grammar narrow ids through unchanged
// (`M-77` stays `M-77`, not `M-0077`); the index needs `padToCanonical`
// to widen for lookup. This test pins that the verb correctly produces
// a Plan against a narrow tree.
func TestRewidth_AllowsNarrowToCanonicalBodyRefs(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	// Seed: narrow epic + narrow milestone, each body references the
	// other in narrow form.
	epicDir := filepath.Join(r.root, "work", "epics", "E-22-narrow")
	if err := os.MkdirAll(epicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	epicBody := "---\nid: E-22\ntitle: Narrow epic\nstatus: active\n---\n\n## Goal\n\nDeals with M-77 and refs E-22 itself.\n\n## Scope\n\nScope.\n\n## Out of scope\n\nOOS.\n"
	if err := os.WriteFile(filepath.Join(epicDir, "epic.md"), []byte(epicBody), 0o644); err != nil {
		t.Fatal(err)
	}
	mBody := "---\nid: M-77\ntitle: Narrow milestone\nstatus: in_progress\nparent: E-22\ntdd: none\n---\n\n## Goal\n\nReferences E-22 and M-77 in prose.\n\n## Approach\n\nApproach.\n\n## Acceptance criteria\n\nCriteria.\n"
	if err := os.WriteFile(filepath.Join(epicDir, "M-77-narrow.md"), []byte(mBody), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := verb.Rewidth(r.ctx, r.root, testActor)
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Errorf("expected no error findings; got %+v", res.Findings)
	}
	if res.Plan == nil {
		t.Errorf("expected Plan; narrow→canonical rewidth should produce a plan")
	}
}

// TestImport_RefusesMalformedIDInManifestBody pins the import verb-
// time gate: a manifest entry whose body: field contains a malformed
// id-shaped token produces findings and no plans; no file is written.
// EntityID on the finding must be the manifest's id, not a path.
func TestImport_RefusesMalformedIDInManifestBody(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	src := `version: 1
entities:
  - kind: gap
    id: G-0001
    frontmatter: {title: "Bad body gap", status: open}
    body: |
      ## What's missing
      Depends on M-foo which is malformed.
      ## Why it matters
      It matters.
`
	m, err := manifest.Parse([]byte(src), "yaml")
	if err != nil {
		t.Fatalf("manifest parse: %v", err)
	}

	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if len(res.Plans) != 0 {
		t.Errorf("expected no Plans; import should have refused with findings")
	}
	if !findingsContainSubcode(res.Findings, check.CodeBodyProseID, "malformed-shape") {
		t.Fatalf("expected body-prose-id/malformed-shape finding; got %+v", res.Findings)
	}
	for i := range res.Findings {
		f := &res.Findings[i]
		if f.Code != check.CodeBodyProseID {
			continue
		}
		if f.EntityID != "G-0001" {
			t.Errorf("finding.EntityID = %q, want %q", f.EntityID, "G-0001")
		}
	}
}

// TestReallocate_RefusesMalformedIDInProseRewrite pins the reallocate
// verb-time gate. The verb's prose-rewrite step touches every entity
// whose body references the renumbered id; if one of those bodies
// also carries an unrelated malformed token, the verb-time scan must
// catch it in the planned-write bytes. EntityID on the finding must be
// the affected entity's id (parsed from op.Content), not the file path.
func TestReallocate_RefusesMalformedIDInProseRewrite(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Target", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Other", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	// Inject a malformed token into M-0002's body alongside the
	// reference to M-0001 that triggers the prose rewrite.
	m2Path := filepath.Join(r.root, "work", "epics", "E-0001-platform", "M-0002-other.md")
	raw, err := os.ReadFile(m2Path)
	if err != nil {
		t.Fatal(err)
	}
	tainted := string(raw) + "\nDepends on M-0001 and M-foo (malformed).\n"
	if writeErr := os.WriteFile(m2Path, []byte(tainted), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	res, err := verb.Reallocate(r.ctx, r.tree(), "M-0001", testActor)
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan != nil {
		t.Errorf("expected no Plan; reallocate should have refused on malformed body token")
	}
	if !findingsContainSubcode(res.Findings, check.CodeBodyProseID, "malformed-shape") {
		t.Fatalf("expected body-prose-id/malformed-shape finding; got %+v", res.Findings)
	}
	for i := range res.Findings {
		f := &res.Findings[i]
		if f.Code != check.CodeBodyProseID {
			continue
		}
		if f.EntityID != "M-0002" {
			t.Errorf("finding.EntityID = %q, want %q (parsed from op.Content frontmatter)", f.EntityID, "M-0002")
		}
	}
}

// findingsContainSubcode reports whether any finding has the given
// (code, subcode) pair. Helper for the assertion shape above.
func findingsContainSubcode(fs []check.Finding, code, subcode string) bool {
	for i := range fs {
		if fs[i].Code == code && fs[i].Subcode == subcode {
			return true
		}
	}
	return false
}
