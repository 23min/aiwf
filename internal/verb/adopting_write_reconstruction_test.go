package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// adopting_write_reconstruction_test.go covers the other half of the
// adoption exemption ADR-0038 grants an AdoptsWorkingCopy write
// (dirty_write_guard_test.go pins the working-copy-vs-HEAD half).
// Declaring the flag lets a write skip the ordinary dirty-path refusal
// only when its own content is itself nothing more than a legitimate
// re-serialization of the working copy it claims to adopt — reconstructed
// by parsing the working copy the same way the tree loader and every
// serializing verb do, normalizing it for its kind (entity.NormalizeForKind),
// and comparing fields. A write is refused when that reconstruction cannot
// be produced at all, and when it can be produced but disagrees with the
// write's own content.

// TestApply_AdoptingWriteRefusesFabricatedFrontmatter pins the
// vulnerability this reconstruction check exists to close: disk matches
// HEAD exactly (only the body is dirty), so the working-copy-vs-HEAD half
// of the guard has nothing to catch, and a write free to declare
// AdoptsWorkingCopy with arbitrary Content could otherwise commit any
// frontmatter it likes under whatever verb's trailer the plan carries.
func TestApply_AdoptingWriteRefusesFabricatedFrontmatter(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	path := dirtyEntity(t, r, "G-0001", "## Why it matters", "## Why it matters\n\nOPERATOR EDIT.\n")

	fabricated := []byte("---\nid: G-0001\ntitle: Some gap\nstatus: addressed\n---\n" +
		"## What's missing\n\nFabricated — never on disk.\n\n## Why it matters\n\nFabricated.\n")
	p := &verb.Plan{
		Subject:  "test adopting write carrying fabricated frontmatter",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops:      []verb.FileOp{{Type: verb.OpWrite, Path: path, Content: fabricated, AdoptsWorkingCopy: true}},
	}
	assertRefusedAndUncommitted(t, r, p, path)

	raw, readErr := os.ReadFile(filepath.Join(r.root, path)) //nolint:gosec // fixture path inside the test's own temp root
	if readErr != nil {
		t.Fatalf("reading %s: %v", path, readErr)
	}
	if !strings.Contains(string(raw), "OPERATOR EDIT.") {
		t.Errorf("the refusal destroyed the operator's edit; working copy:\n%s", raw)
	}
	if strings.Contains(string(raw), "Fabricated") {
		t.Errorf("the fabricated content landed on disk despite the refusal")
	}
}

// TestApply_AdoptingWriteRefusesOnAPathOutsideAnyEntityShape pins the
// PathKind fallback inside the reconstruction check. The write's content
// does not match the working copy verbatim (so the direct comparison
// cannot accept it — a bless-mode-shaped write must be caught some other
// way), and the path is not one the loader would ever recognize as an
// entity, so there is no tree-load pipeline to reconstruct through
// either — refused rather than waved through for want of a comparison.
func TestApply_AdoptingWriteRefusesOnAPathOutsideAnyEntityShape(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	const relPath = "notes.md"
	full := filepath.Join(r.root, relPath)
	if writeErr := os.WriteFile(full, []byte("---\nfoo: bar\n---\nOriginal notes.\n"), 0o600); writeErr != nil {
		t.Fatalf("writing %s: %v", relPath, writeErr)
	}
	commitFixture(t, r.root, "add a non-entity frontmatter-bearing file")

	// Dirty the body only — frontmatter stays identical to HEAD, so the
	// working-copy-vs-HEAD half of the guard has nothing to catch, and
	// the reconstruction check is what's under test.
	dirty := []byte("---\nfoo: bar\n---\nEdited notes.\n")
	if writeErr := os.WriteFile(full, dirty, 0o600); writeErr != nil {
		t.Fatalf("dirtying %s: %v", relPath, writeErr)
	}

	content := []byte("---\nfoo: baz\n---\nEdited notes.\n")
	p := &verb.Plan{
		Subject:  "test adopting write on a non-entity path",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops:      []verb.FileOp{{Type: verb.OpWrite, Path: relPath, Content: content, AdoptsWorkingCopy: true}},
	}
	assertRefusedAndUncommitted(t, r, p, relPath)
}

// TestApply_AdoptingWriteRefusesWhenTheWorkingCopyCarriesAFieldTheEntityModelRejects
// pins the entity.Parse fallback inside the reconstruction check. The
// working-copy-vs-HEAD comparison decodes into a generic map and tolerates
// any key; the reconstruction decodes through the typed Entity model
// (entity.Parse's strict KnownFields(true) decoder) and does not. The
// write's content omits the unknown field — the shape a re-serialization
// that dropped it would produce, so the direct comparison against the
// working copy (which still carries it) cannot accept it either — and a
// field neither side's committed history ever declared through a verb
// makes no legitimate reconstruction producible, so the write is refused
// rather than compared against nothing.
func TestApply_AdoptingWriteRefusesWhenTheWorkingCopyCarriesAFieldTheEntityModelRejects(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	e := r.tree().ByID("G-0001")
	full := filepath.Join(r.root, e.Path)
	raw, readErr := os.ReadFile(full) //nolint:gosec // fixture path inside the test's own temp root
	if readErr != nil {
		t.Fatalf("reading %s: %v", e.Path, readErr)
	}

	withUnknownField := strings.Replace(string(raw), "status: open\n", "status: open\nbogus_field: 1\n", 1)
	if withUnknownField == string(raw) {
		t.Fatalf("fixture did not inject bogus_field into\n%s", raw)
	}
	if writeErr := os.WriteFile(full, []byte(withUnknownField), 0o600); writeErr != nil {
		t.Fatalf("writing %s: %v", e.Path, writeErr)
	}
	commitFixture(t, r.root, "hand-commit a field the Entity model does not declare")

	dirtied := strings.Replace(withUnknownField, "## Why it matters", "## Why it matters\n\nOPERATOR EDIT.\n", 1)
	if writeErr := os.WriteFile(full, []byte(dirtied), 0o600); writeErr != nil {
		t.Fatalf("dirtying %s: %v", e.Path, writeErr)
	}

	// The most favourable content a re-serializing write could present:
	// everything the working copy carries, minus the field no Entity
	// declares.
	content := []byte(strings.Replace(dirtied, "bogus_field: 1\n", "", 1))

	relPath := filepath.ToSlash(e.Path)
	p := &verb.Plan{
		Subject:  "test adopting write over an unknown frontmatter field",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops:      []verb.FileOp{{Type: verb.OpWrite, Path: relPath, Content: content, AdoptsWorkingCopy: true}},
	}
	assertRefusedAndUncommitted(t, r, p, relPath)
}

// TestApply_ExplicitEditBodyOverNonCanonicalFrontmatterStillCommits is the
// Apply-level companion of
// TestEditBody_ExplicitIdenticalBodyOverNonCanonicalFrontmatter_StillCommits
// (editbody_same_state_noop_test.go), which stops at the produced Plan and
// so never exercises this guard. HEAD carries hand-ordered frontmatter;
// the write's content re-canonicalizes it. The reconstruction check must
// recognize that re-canonicalization as the same fields HEAD already
// declares, or every edit-body call that also re-canonicalizes would
// refuse itself.
func TestApply_ExplicitEditBodyOverNonCanonicalFrontmatterStillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	path, body := epicBodyOnDisk(t, r.root)
	canonical, readErr := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if readErr != nil {
		t.Fatalf("reading the canonical epic: %v", readErr)
	}

	noncanon := []byte("---\nstatus: proposed\ntitle: Foundations\nid: E-0001\n---\n" + string(body))
	if writeErr := os.WriteFile(path, noncanon, 0o600); writeErr != nil {
		t.Fatalf("writing non-canonical frontmatter: %v", writeErr)
	}
	commitFixture(t, r.root, "hand-ordered frontmatter")
	if writeErr := os.WriteFile(path, canonical, 0o600); writeErr != nil {
		t.Fatalf("restoring the canonical epic: %v", writeErr)
	}

	res, editErr := verb.EditBody(r.ctx, r.tree(), "E-0001", body, testActor, "")
	if editErr != nil {
		t.Fatalf("edit-body over a non-canonically-serialized HEAD: %v", editErr)
	}
	if res.NoOp {
		t.Fatal("res.NoOp = true, want false — the frontmatter still needs re-canonicalizing")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan re-canonicalizing the entity")
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply refused to re-canonicalize a HEAD-divergent-order frontmatter it already agrees with on fields: %v", applyErr)
	}
	committed := showHEADFile(t, r.root, "work/epics/E-0001-foundations/epic.md")
	if !strings.Contains(committed, "id: E-0001\ntitle: Foundations\nstatus: proposed\n") {
		t.Errorf("HEAD is not canonically ordered after the commit:\n%s", committed)
	}
}

// TestApply_ExplicitEditBodyOverStrayMilestoneAreaStillCommits pins that
// closing the smuggling hole does not lock a milestone carrying a
// pre-existing stray `area` key out of edit-body. No aiwf verb can write
// or clear a milestone's `area` (`aiwf set-area` refuses milestone
// targets; ADR-0038 rules out `--force` and a repair verb), so the field
// can only have arrived by hand-edit or import (G-0488) — tree.Load blanks
// it in the loaded model regardless of how it got there, and content built
// from that model necessarily omits it too.
//
// A naive fix comparing the write's content against HEAD's or disk's raw
// frontmatter would refuse forever on this shape: content, built from the
// loader's already-blanked model, never carries the stray field, while
// HEAD and disk — untouched by the operator — still do. The reconstruction
// check applies the identical normalization (entity.NormalizeForKind)
// before comparing, so it recognizes the field's absence as the same
// legitimate re-serialization every serializing verb already performs.
//
// The write-then-route flow — the operator edits the body directly on
// disk, then routes the same content through explicit mode — is what
// makes the file dirty without touching frontmatter, which is the shape
// that reaches this guard at all.
func TestApply_ExplicitEditBodyOverStrayMilestoneAreaStillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Alpha epic", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First milestone", testActor, verb.AddOptions{
		EpicID: "E-0001",
		TDD:    "none",
	}))

	e := r.tree().ByID("M-0001")
	full := filepath.Join(r.root, e.Path)
	raw, readErr := os.ReadFile(full) //nolint:gosec // fixture path inside the test's own temp root
	if readErr != nil {
		t.Fatalf("reading M-0001: %v", readErr)
	}
	withArea := strings.Replace(string(raw), "id: M-0001\n", "id: M-0001\narea: some-area\n", 1)
	if withArea == string(raw) {
		t.Fatalf("fixture did not inject area: into\n%s", raw)
	}
	if writeErr := os.WriteFile(full, []byte(withArea), 0o600); writeErr != nil {
		t.Fatalf("writing M-0001 with a stray area: %v", writeErr)
	}
	commitFixture(t, r.root, "hand-edit: inject a stray area on M-0001")

	if loaded := r.tree().ByID("M-0001"); loaded.Area != "" {
		t.Fatalf("fixture setup: loader did not blank M-0001's area; got %q", loaded.Area)
	}

	const wantedBody = "\n## Goal\n\nWritten to disk first, then routed through the verb.\n\n## Acceptance criteria\n"
	writeBodyOnDisk(t, full, wantedBody)

	res, editErr := verb.EditBody(r.ctx, r.tree(), "M-0001", []byte(wantedBody), testActor, "")
	if editErr != nil {
		t.Fatalf("edit-body over a stray-area milestone: %v", editErr)
	}
	if res.Plan == nil {
		t.Fatal("no plan produced")
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply refused a legitimate write-then-route edit on a stray-area milestone: %v", applyErr)
	}
	committed := showHEADFile(t, r.root, e.Path)
	if !strings.Contains(committed, "Written to disk first") {
		t.Errorf("the routed body is not in HEAD:\n%s", committed)
	}
}

// TestApply_BlessModeStillCommitsOverStrayMilestoneArea is bless mode's
// counterpart to TestApply_ExplicitEditBodyOverStrayMilestoneAreaStillCommits.
// Bless mode commits the working copy verbatim rather than through the
// loaded entity model, so it never normalizes the stray `area` away —
// content still carries it, matching disk directly. A reconstruction-only
// check (compare content against a normalized re-serialization of disk)
// would refuse this: the normalized reconstruction lacks `area`, the
// verbatim content still has it, and they would never agree. The direct
// comparison against disk — tried first, in adoptionPreservesFrontmatter —
// is what accepts a bless-mode write unconditionally instead.
func TestApply_BlessModeStillCommitsOverStrayMilestoneArea(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Alpha epic", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First milestone", testActor, verb.AddOptions{
		EpicID: "E-0001",
		TDD:    "none",
	}))

	e := r.tree().ByID("M-0001")
	full := filepath.Join(r.root, e.Path)
	raw, readErr := os.ReadFile(full) //nolint:gosec // fixture path inside the test's own temp root
	if readErr != nil {
		t.Fatalf("reading M-0001: %v", readErr)
	}
	withArea := strings.Replace(string(raw), "id: M-0001\n", "id: M-0001\narea: some-area\n", 1)
	if withArea == string(raw) {
		t.Fatalf("fixture did not inject area: into\n%s", raw)
	}
	if writeErr := os.WriteFile(full, []byte(withArea), 0o600); writeErr != nil {
		t.Fatalf("writing M-0001 with a stray area: %v", writeErr)
	}
	commitFixture(t, r.root, "hand-edit: inject a stray area on M-0001")

	dirtyEntity(t, r, "M-0001", "## Goal", "## Goal\n\nBLESSED BODY EDIT.\n")

	res, editErr := verb.EditBody(r.ctx, r.tree(), "M-0001", nil, testActor, "")
	if editErr != nil {
		t.Fatalf("edit-body bless mode over a stray-area milestone: %v", editErr)
	}
	if res.Plan == nil {
		t.Fatal("no plan produced")
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply refused a working-copy edit bless mode exists to commit: %v", applyErr)
	}
	committed := showHEADFile(t, r.root, e.Path)
	if !strings.Contains(committed, "BLESSED BODY EDIT.") {
		t.Errorf("the blessed edit is not in HEAD:\n%s", committed)
	}
	if !strings.Contains(committed, "area: some-area") {
		t.Errorf("bless mode must preserve the working copy's frontmatter verbatim, including the stray area:\n%s", committed)
	}
}
