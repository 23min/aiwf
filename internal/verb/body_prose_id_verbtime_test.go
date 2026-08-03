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
