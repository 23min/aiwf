package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/manifest"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
	"github.com/23min/aiwf/internal/verb"
)

// G-0184 verb-time scan tests. The body-prose-id rule is enforced at
// verb time across every verb that ingests operator-supplied body
// content (add, edit-body, import, reallocate). These tests
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

// --- ADR-0041: the verb layer does not refuse a write for a
// cross-branch classification. Which refs carry a target is a fact
// about the repository, not a defect in the bytes being written, and
// the author cannot answer it by editing them — so it is enforced at
// the push boundary by `aiwf check`, and authoring stays open. ---

// decisionBody satisfies the born-complete body gate so these tests
// reach the reference gate they are about.
const decisionBody = "## Question\n\nWhat follows from the gap?\n\n## Decision\n\nCite it.\n\n## Reasoning\n\nThe structured reference is the surface under test.\n"

// seedUnpushedBranchHit puts G-0500 on a local branch that has not been
// pushed, in a repository that HAS a remote — the state that classifies
// cross-branch-local-only at error severity.
//
// Both halves are load-bearing. Without HasRemoteTrackingRefs the
// classification degrades to the non-blocking pending warning, which no
// verb ever refused, and every test below would pass without exercising
// the exclusion it exists to pin.
func seedUnpushedBranchHit(tr *tree.Tree) {
	tr.CrossBranchHits = []trunk.RefHit{
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-sibling.md", Ref: "refs/heads/sibling"},
	}
	tr.HasRemoteTrackingRefs = true
}

func TestAdd_AllowsBodyCitingAnIDOnAnUnpushedBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	tr := r.tree()
	seedUnpushedBranchHit(tr)
	body := "## What's missing\n\nDepends on G-0500, filed on a branch nobody has pushed.\n\n## Why it matters\n\nMatters.\n"

	res, err := verb.Add(r.ctx, tr, entity.KindGap, "Cites an unpushed branch", testActor, verb.AddOptions{
		BodyOverride: []byte(body),
	})
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan == nil {
		t.Errorf("expected a Plan; the reference is well-formed and its target exists, so authoring must not be refused — got findings %+v", res.Findings)
	}
}

// The companion control. Without it, deleting the whole verb-time gate
// would pass the test above, so this pins that the exclusion reaches
// the cross-branch classification and nothing else. The cross-branch
// view is populated with a DIFFERENT id, so the tier is demonstrably
// consulted and the fabricated one still misses it.
func TestAdd_StillRefusesAnUnresolvedIDWhileTheCrossBranchViewIsPopulated(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	tr := r.tree()
	seedUnpushedBranchHit(tr)
	body := "## What's missing\n\nDepends on G-0999, which no branch carries.\n\n## Why it matters\n\nMatters.\n"

	res, err := verb.Add(r.ctx, tr, entity.KindGap, "Cites a fabricated id", testActor, verb.AddOptions{
		BodyOverride: []byte(body),
	})
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan != nil {
		t.Errorf("expected no Plan; a fabricated id is a defect in the bytes being written and must still be refused")
	}
	if !findingsContainSubcode(res.Findings, check.CodeBodyProseID, "unresolved") {
		t.Errorf("expected body-prose-id/unresolved finding; got %+v", res.Findings)
	}
}

// The structured-reference half. A frontmatter reference reaches the
// gate through projectionFindings rather than the body-prose scan, and
// a newly-added entity's cross-branch reference is absent from the
// pre-change tree — so without the exclusion there it reads as a
// finding this verb introduced, and the add is refused.
func TestAdd_AllowsStructuredReferenceToAnIDOnAnUnpushedBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	tr := r.tree()
	seedUnpushedBranchHit(tr)

	res, err := verb.Add(r.ctx, tr, entity.KindDecision, "Relates to an unpushed gap", testActor, verb.AddOptions{
		RelatesTo:    []string{"G-0500"},
		BodyOverride: []byte(decisionBody),
	})
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan == nil {
		t.Errorf("expected a Plan; relates_to names an entity that exists, so authoring must not be refused — got findings %+v", res.Findings)
	}
}

// The companion control for the structured path: a frontmatter
// reference to an id no ref carries is still the verb's own defect.
func TestAdd_StillRefusesAStructuredReferenceToAFabricatedID(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	tr := r.tree()
	seedUnpushedBranchHit(tr)

	res, err := verb.Add(r.ctx, tr, entity.KindDecision, "Relates to nothing", testActor, verb.AddOptions{
		RelatesTo:    []string{"G-0999"},
		BodyOverride: []byte(decisionBody),
	})
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan != nil {
		t.Errorf("expected no Plan; a relates_to naming no entity anywhere must still be refused")
	}
	if !findingsContainSubcode(res.Findings, check.CodeRefsResolve, "unresolved") {
		t.Errorf("expected refs-resolve/unresolved finding; got %+v", res.Findings)
	}
}

func TestEditBody_Explicit_AllowsBodyCitingAnIDOnAnUnpushedBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Carrier epic", testActor, verb.AddOptions{}))

	tr := r.tree()
	seedUnpushedBranchHit(tr)
	body := []byte("## Goal\n\nDepends on G-0500, filed on a branch nobody has pushed.\n\n## Scope\n\nScope.\n\n## Out of scope\n\nOOS.\n")

	res, err := verb.EditBody(r.ctx, tr, "E-0001", body, testActor, "")
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if res.Plan == nil {
		t.Errorf("expected a Plan; edit-body must not be refused for a cross-branch classification — got findings %+v", res.Findings)
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

// TestReallocate_ArchivedBodyDoesNotBlockProseRewrite pins the scope of
// the reallocate verb-time gate.
//
// Reallocation rewrites cross-references wherever they sit, archive
// included, so an archived body enters the write set carrying prose the
// verb did not author. Archived entities are outside the convention the
// scan enforces (ADR-0004 §"Check shape rules" — the tree-walking rule
// skips them for the same reason) and their bodies are frozen, so a
// malformed token already sitting in one must not refuse the rewrite.
// Without the scope, `aiwf reallocate` is unusable in any tree whose
// archive predates the convention.
func TestReallocate_ArchivedBodyDoesNotBlockProseRewrite(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Referring gap", testActor,
		verb.AddOptions{BodyOverride: []byte(
			"## What's missing\n\nFollows from G-0001.\n\n" +
				"## Why it matters\n\nFixture prose.\n",
		)}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0002", testActor, "fixture", false))
	r.must(verb.Archive(r.ctx, r.root, testActor, ""))
	archived := r.tree().ByID("G-0002")
	if archived == nil || !entity.IsArchivedPath(archived.Path) {
		t.Fatalf("G-0002 not archived; got %+v", archived)
	}

	// The malformed token is planted after the archive move: every verb
	// that writes a body refuses it, which is the behaviour under test
	// everywhere except here. A frozen archive acquires one by predating
	// the rule, not by passing a gate.
	archivedPath := filepath.Join(r.root, archived.Path)
	raw, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := os.WriteFile(archivedPath, append(raw, "\nCluster G-\u03b1 owns it.\n"...), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	commitFixture(t, r.root, "fixture: pre-convention archive body")

	res, err := verb.Reallocate(r.ctx, r.tree(), "G-0001", testActor)
	if err != nil {
		t.Fatalf("verb error: %v", err)
	}
	if findingsContainSubcode(res.Findings, check.CodeBodyProseID, "malformed-shape") {
		t.Fatalf("archived body blocked the rewrite: %+v", res.Findings)
	}
	if res.Plan == nil {
		t.Fatalf("expected a Plan; reallocate refused: %+v", res.Findings)
	}
	// The exemption is only under test while the archived body is
	// actually in the write set. Without this, a change that stopped
	// rewriting archived cross-references would leave the test green and
	// the guard dead.
	var wroteArchived bool
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpWrite && filepath.ToSlash(op.Path) == filepath.ToSlash(archived.Path) {
			wroteArchived = true
		}
	}
	if !wroteArchived {
		t.Fatalf("archived body never entered the write set; the exemption is untested. ops = %+v", res.Plan.Ops)
	}
}
