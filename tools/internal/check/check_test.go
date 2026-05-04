package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/trunk"
)

// makeTree constructs a non-filesystem tree from inline entities.
// Entities without an explicit Path get a synthetic one so finding
// messages stay readable.
func makeTree(es ...*entity.Entity) *tree.Tree {
	for _, e := range es {
		if e.Path == "" {
			e.Path = "synthetic.md"
		}
	}
	return &tree.Tree{Entities: es}
}

// codes extracts just the codes from findings, preserving order.
func codes(fs []Finding) []string {
	out := make([]string, len(fs))
	for i := range fs {
		out[i] = fs[i].Code
	}
	return out
}

func TestIDsUnique(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Path: "a.md"},
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Path: "b.md"},
		&entity.Entity{ID: "M-002", Kind: entity.KindMilestone, Path: "c.md"},
	)
	got := idsUnique(tr)
	if len(got) != 1 {
		t.Fatalf("idsUnique findings = %d, want 1: %+v", len(got), got)
	}
	if got[0].EntityID != "M-001" || got[0].Path != "b.md" {
		t.Errorf("got %+v", got[0])
	}
}

func TestIDsUnique_TrunkCollision(t *testing.T) {
	// Working tree has G-035 at one path; trunk has G-035 at a different
	// path — the G37 case. The check must surface this as a finding so
	// the pre-push hook fails before the colliding push lands.
	tr := makeTree(
		&entity.Entity{ID: "G-035", Kind: entity.KindGap, Path: "work/gaps/G-035-local.md"},
	)
	tr.TrunkIDs = []trunk.ID{
		{Kind: entity.KindGap, ID: "G-035", Path: "work/gaps/G-035-trunk.md"},
	}
	got := idsUnique(tr)
	if len(got) != 1 {
		t.Fatalf("idsUnique findings = %d, want 1: %+v", len(got), got)
	}
	f := got[0]
	if f.EntityID != "G-035" {
		t.Errorf("EntityID = %q, want G-035", f.EntityID)
	}
	if f.Subcode != "trunk-collision" {
		t.Errorf("Subcode = %q, want trunk-collision", f.Subcode)
	}
	if !strings.Contains(f.Message, "work/gaps/G-035-local.md") {
		t.Errorf("message %q should name the local path", f.Message)
	}
	if !strings.Contains(f.Message, "work/gaps/G-035-trunk.md") {
		t.Errorf("message %q should name the trunk-side path", f.Message)
	}
}

func TestIDsUnique_TrunkSamePath_NoFinding(t *testing.T) {
	// The entity is already on trunk at the same path — that's the
	// normal post-merge state, not a collision. The check must stay
	// silent so every aiwf check doesn't drown in noise.
	tr := makeTree(
		&entity.Entity{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/G-001-foo.md"},
	)
	tr.TrunkIDs = []trunk.ID{
		{Kind: entity.KindGap, ID: "G-001", Path: "work/gaps/G-001-foo.md"},
	}
	got := idsUnique(tr)
	if len(got) != 0 {
		t.Errorf("expected no findings (same path on trunk and locally), got %+v", got)
	}
}

func TestIDsUnique_TrunkOnlyID_NoFinding(t *testing.T) {
	// Trunk has G-007; the working tree doesn't. That is not a
	// collision — the working tree just hasn't pulled, or has elected
	// not to carry that entity yet. The allocator's job is to avoid
	// re-using G-007; the check's job is only to catch overlaps.
	tr := makeTree(
		&entity.Entity{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/G-001-foo.md"},
	)
	tr.TrunkIDs = []trunk.ID{
		{Kind: entity.KindGap, ID: "G-007", Path: "work/gaps/G-007-trunk-only.md"},
	}
	got := idsUnique(tr)
	if len(got) != 0 {
		t.Errorf("expected no findings (trunk-only id is not a collision), got %+v", got)
	}
}

// TestCasePaths_ReportsCaseEquivalentEntities is the load-bearing
// test for G10: two entities whose paths differ only in case (e.g.
// E-01-foo vs E-01-Foo committed from a case-sensitive Linux dev
// box) collapse to a single path on a case-insensitive macOS
// reviewer's machine. casePaths catches this footgun before it
// silently surfaces as data loss on checkout.
func TestCasePaths_ReportsCaseEquivalentEntities(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-foo/epic.md"},
		&entity.Entity{ID: "E-02", Kind: entity.KindEpic, Path: "work/epics/E-01-Foo/epic.md"},
		&entity.Entity{ID: "E-03", Kind: entity.KindEpic, Path: "work/epics/E-03-bar/epic.md"},
	)
	got := casePaths(tr)
	if len(got) != 1 {
		t.Fatalf("casePaths findings = %d, want 1: %+v", len(got), got)
	}
	if got[0].Code != "case-paths" {
		t.Errorf("code = %q, want case-paths", got[0].Code)
	}
	// Message must name the colliding pair so the user can locate them.
	msg := got[0].Message
	if !strings.Contains(msg, "E-01-foo") || !strings.Contains(msg, "E-01-Foo") {
		t.Errorf("message should name both colliding paths; got %q", msg)
	}
}

// TestCasePaths_CleanTreeNoFindings: a tree with all-distinct paths
// produces no case-paths findings.
func TestCasePaths_CleanTreeNoFindings(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-foo/epic.md"},
		&entity.Entity{ID: "E-02", Kind: entity.KindEpic, Path: "work/epics/E-02-bar/epic.md"},
	)
	got := casePaths(tr)
	if len(got) != 0 {
		t.Errorf("clean tree should produce no case-paths findings; got %+v", got)
	}
}

// TestCasePaths_ThreeWayCollision: three entities all collapsing to
// the same case-insensitive path each generate a finding so the
// user sees every offender, not just one.
func TestCasePaths_ThreeWayCollision(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-foo/epic.md"},
		&entity.Entity{ID: "E-02", Kind: entity.KindEpic, Path: "work/epics/E-01-FOO/epic.md"},
		&entity.Entity{ID: "E-03", Kind: entity.KindEpic, Path: "work/epics/E-01-Foo/epic.md"},
	)
	got := casePaths(tr)
	if len(got) < 2 {
		t.Errorf("3-way collision should produce at least 2 findings; got %d", len(got))
	}
}

func TestStatusValid(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-01", Kind: entity.KindEpic, Status: "active"},            // ok
		&entity.Entity{ID: "E-02", Kind: entity.KindEpic, Status: "in_progress"},       // milestone-only status
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Status: "in_progress"}, // ok
		&entity.Entity{ID: "M-002", Kind: entity.KindMilestone, Status: "done"},        // ok
		&entity.Entity{ID: "G-001", Kind: entity.KindGap, Status: ""},                  // empty: skipped
	)
	got := statusValid(tr)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(got), got)
	}
	if got[0].EntityID != "E-02" {
		t.Errorf("got %+v", got[0])
	}
}

func TestFrontmatterShape(t *testing.T) {
	tr := makeTree(
		// Missing id.
		&entity.Entity{Kind: entity.KindEpic, Title: "Foo", Status: "active", Path: "a.md"},
		// Bad id format for kind.
		&entity.Entity{ID: "X-99", Kind: entity.KindEpic, Title: "Foo", Status: "active", Path: "b.md"},
		// Milestone missing parent.
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Title: "Foo", Status: "draft", Path: "c.md"},
		// Contract minimal — id, title, status only — is clean.
		&entity.Entity{ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "proposed", Path: "d.md"},
		// Clean gap.
		&entity.Entity{ID: "G-001", Kind: entity.KindGap, Title: "Foo", Status: "open", Path: "e.md"},
	)
	got := frontmatterShape(tr)
	want := []string{
		"frontmatter-shape", // missing id (a.md)
		"frontmatter-shape", // bad id format (b.md)
		"frontmatter-shape", // milestone missing parent (c.md)
	}
	if diff := cmp.Diff(want, codes(got)); diff != "" {
		t.Errorf("codes mismatch (-want +got):\n%s\nfindings: %+v", diff, got)
	}
}

func TestRefsResolve_Unresolved(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-01", Kind: entity.KindEpic},
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Parent: "E-99"}, // unresolved
	)
	got := refsResolve(tr)
	if len(got) != 1 || got[0].Subcode != "unresolved" {
		t.Errorf("got %+v", got)
	}
}

func TestRefsResolve_WrongKind(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "D-001", Kind: entity.KindDecision},
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Parent: "D-001"}, // parent must be epic
	)
	got := refsResolve(tr)
	if len(got) != 1 || got[0].Subcode != "wrong-kind" {
		t.Errorf("got %+v", got)
	}
}

func TestRefsResolve_StubResolvesReferences(t *testing.T) {
	// Regression for the wrap-epic cascade bug: when E-01's epic.md
	// fails to parse (e.g. an unknown frontmatter field rejected by
	// KnownFields(true)), every entity that references E-01 used to
	// surface a refs-resolve/unresolved finding on top of the load
	// error. The tree loader now registers a stub for E-01; refsResolve
	// must consult Stubs so the cascade is suppressed. The original
	// parse failure still appears as a load-error finding via Run().
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "M-001", Kind: entity.KindMilestone, Parent: "E-01", Path: "m1.md"},
			{ID: "M-002", Kind: entity.KindMilestone, Parent: "E-01", Path: "m2.md"},
			{ID: "G-001", Kind: entity.KindGap, DiscoveredIn: "E-01", Path: "g1.md"},
			{ID: "D-001", Kind: entity.KindDecision, RelatesTo: []string{"E-01"}, Path: "d1.md"},
		},
		Stubs: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-bad/epic.md"},
		},
	}
	got := refsResolve(tr)
	if len(got) != 0 {
		t.Errorf("expected no refs-resolve findings (stub should resolve), got: %+v", got)
	}
}

func TestRefsResolve_StubPreservesWrongKindCheck(t *testing.T) {
	// A stub still carries its kind (derived from path), so wrong-kind
	// findings on referrers must still fire when the link is to the
	// wrong kind. Here a milestone's parent points at a stubbed gap;
	// the wrong-kind finding must still be raised.
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "M-001", Kind: entity.KindMilestone, Parent: "G-001", Path: "m.md"},
		},
		Stubs: []*entity.Entity{
			{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/G-001.md"},
		},
	}
	got := refsResolve(tr)
	if len(got) != 1 || got[0].Subcode != "wrong-kind" {
		t.Errorf("expected one wrong-kind finding, got: %+v", got)
	}
}

func TestRefsResolve_RealEntityWinsOverStub(t *testing.T) {
	// If both a real entity and a stub claim the same id (shouldn't
	// happen in practice, but defensive), the real one is indexed
	// first and wins.
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindEpic, Path: "good.md"},
			{ID: "M-001", Kind: entity.KindMilestone, Parent: "E-01", Path: "m.md"},
		},
		Stubs: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindGap, Path: "stub.md"}, // wrong kind
		},
	}
	got := refsResolve(tr)
	if len(got) != 0 {
		t.Errorf("expected no findings (real entity wins), got: %+v", got)
	}
}

// TestRefsResolve_ProliminalCascadeRepro is the wild repro from the
// proliminal.net dogfooding repo, distilled to its essentials. E-01's
// epic.md had a `completed:` field added by the wrap-epic skill;
// KnownFields(true) rejected it; the entity dropped out of Entities;
// every entity that referenced E-01 (5 milestones via parent, 5 gaps
// via discovered_in, 2 decisions via relates_to) surfaced an
// unresolved-reference finding. Net: 13 push-blocking errors from one
// bad field. This test fails on the pre-fix code (12 cascade findings
// would appear) and passes once stubs short-circuit refs-resolve.
func TestRefsResolve_ProliminalCascadeRepro(t *testing.T) {
	entities := []*entity.Entity{}
	// 5 milestones, all parented to E-01.
	for i := 1; i <= 5; i++ {
		entities = append(entities, &entity.Entity{
			ID:     fmt.Sprintf("M-%03d", i),
			Kind:   entity.KindMilestone,
			Parent: "E-01",
			Path:   fmt.Sprintf("work/epics/E-01-foo/M-%03d.md", i),
		})
	}
	// 5 gaps, all discovered in E-01.
	for i := 1; i <= 5; i++ {
		entities = append(entities, &entity.Entity{
			ID:           fmt.Sprintf("G-%03d", i),
			Kind:         entity.KindGap,
			DiscoveredIn: "E-01",
			Path:         fmt.Sprintf("work/gaps/G-%03d.md", i),
		})
	}
	// 2 decisions, both related to E-01.
	for i := 1; i <= 2; i++ {
		entities = append(entities, &entity.Entity{
			ID:        fmt.Sprintf("D-%03d", i),
			Kind:      entity.KindDecision,
			RelatesTo: []string{"E-01"},
			Path:      fmt.Sprintf("work/decisions/D-%03d.md", i),
		})
	}
	tr := &tree.Tree{
		Entities: entities,
		Stubs: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-foo/epic.md"},
		},
	}
	got := refsResolve(tr)
	if len(got) != 0 {
		t.Errorf("expected no refs-resolve findings (cascade should be suppressed by stub); got %d:\n%+v", len(got), got)
	}
}

func TestIdsUnique_StubVsRealCollision(t *testing.T) {
	// User has two epic dirs both claiming id E-01; one parses, one
	// doesn't. ids-unique must still flag the duplicate; otherwise
	// the cascade-suppression fix would silently swallow a real
	// id-collision.
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-good/epic.md"},
		},
		Stubs: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-bad/epic.md"},
		},
	}
	got := idsUnique(tr)
	if len(got) != 1 {
		t.Fatalf("expected 1 ids-unique finding for stub-vs-real collision; got %+v", got)
	}
	if got[0].Path != "work/epics/E-01-bad/epic.md" {
		t.Errorf("finding should point at the colliding (stub) path; got %q", got[0].Path)
	}
}

func TestIdsUnique_StubVsStubCollision(t *testing.T) {
	tr := &tree.Tree{
		Stubs: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-a/epic.md"},
			{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-b/epic.md"},
		},
	}
	got := idsUnique(tr)
	if len(got) != 1 {
		t.Errorf("expected 1 ids-unique finding for stub-vs-stub collision; got %+v", got)
	}
}

func TestIdPathConsistent_Mismatch(t *testing.T) {
	tr := makeTree(&entity.Entity{
		ID:   "M-100",
		Kind: entity.KindMilestone,
		Path: "work/epics/E-01-foo/M-099-thing.md",
	})
	got := idPathConsistent(tr)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %+v", got)
	}
	if got[0].Code != "id-path-consistent" {
		t.Errorf("Code = %q, want id-path-consistent", got[0].Code)
	}
	if got[0].EntityID != "M-100" {
		t.Errorf("EntityID = %q, want M-100", got[0].EntityID)
	}
	if !strings.Contains(got[0].Message, "M-099") || !strings.Contains(got[0].Message, "M-100") {
		t.Errorf("Message should mention both ids; got %q", got[0].Message)
	}
}

func TestIdPathConsistent_Agrees(t *testing.T) {
	tr := makeTree(&entity.Entity{
		ID:   "E-01",
		Kind: entity.KindEpic,
		Path: "work/epics/E-01-platform/epic.md",
	})
	got := idPathConsistent(tr)
	if len(got) != 0 {
		t.Errorf("want 0 findings, got %+v", got)
	}
}

func TestIdPathConsistent_SkipsEntitiesWithoutPathID(t *testing.T) {
	// Path is something IDFromPath can't extract an id from.
	// Defensive: shouldn't happen post-loader, but the check
	// must not crash if it does.
	tr := makeTree(&entity.Entity{
		ID:   "E-01",
		Kind: entity.KindEpic,
		Path: "work/epics/no-id-here/epic.md",
	})
	got := idPathConsistent(tr)
	if len(got) != 0 {
		t.Errorf("want 0 findings (skip when path has no id), got %+v", got)
	}
}

func TestIdPathConsistent_StubsTriviallyMatch(t *testing.T) {
	// Stubs are constructed from the path-derived id, so they always
	// pass id-path-consistent. The check iterates only Entities,
	// not Stubs, so this confirms the stubs slice doesn't get a
	// spurious pass through some other code path.
	tr := &tree.Tree{
		Stubs: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindEpic, Path: "work/epics/E-01-stubbed/epic.md"},
		},
	}
	got := idPathConsistent(tr)
	if len(got) != 0 {
		t.Errorf("want 0 findings on stubs-only tree, got %+v", got)
	}
}

// TestSchemaMatchesForwardRefs pins the per-kind reference-field
// metadata in entity.SchemaForKind to what entity.ForwardRefs actually
// reads. If a future change adds, removes, or retypes a reference
// field in ForwardRefs without updating the schema (or vice versa),
// this test fails — preventing the published `aiwf schema` surface
// from drifting away from runtime enforcement.
func TestSchemaMatchesForwardRefs(t *testing.T) {
	for _, k := range entity.AllKinds() {
		t.Run(string(k), func(t *testing.T) {
			s, _ := entity.SchemaForKind(k)

			// Build a synthetic entity carrying a sentinel value in
			// every reference field declared in the schema. ForwardRefs
			// must surface every field with a value, with matching
			// allowed-kinds.
			e := &entity.Entity{Kind: k}
			expect := make(map[string]entity.RefField, len(s.References))
			for _, r := range s.References {
				expect[r.Name] = r
				switch r.Name {
				case "parent":
					e.Parent = "X-1"
				case "depends_on":
					e.DependsOn = []string{"X-1"}
				case "supersedes":
					e.Supersedes = []string{"X-1"}
				case "superseded_by":
					e.SupersededBy = "X-1"
				case "discovered_in":
					e.DiscoveredIn = "X-1"
				case "addressed_by":
					e.AddressedBy = []string{"X-1"}
				case "relates_to":
					e.RelatesTo = []string{"X-1"}
				case "linked_adrs":
					e.LinkedADRs = []string{"X-1"}
				default:
					t.Fatalf("schema declares unknown ref field %q on %s — TestSchemaMatchesForwardRefs needs an arm for it", r.Name, k)
				}
			}

			got := entity.ForwardRefs(e)
			gotByName := make(map[string][]entity.Kind, len(got))
			for _, r := range got {
				gotByName[r.Field] = r.AllowedKinds
			}

			// Every schema-declared field must show up.
			for name, want := range expect {
				gotAllowed, ok := gotByName[name]
				if !ok {
					t.Errorf("schema declares ref field %q on %s, but ForwardRefs didn't surface it", name, k)
					continue
				}
				if !sameKinds(gotAllowed, want.AllowedKinds) {
					t.Errorf("ref %q on %s: ForwardRefs allowed=%v, schema allowed=%v", name, k, gotAllowed, want.AllowedKinds)
				}
			}
			// And no surplus fields in ForwardRefs.
			for name := range gotByName {
				if _, ok := expect[name]; !ok {
					t.Errorf("ForwardRefs surfaces ref field %q on %s, but schema does not declare it", name, k)
				}
			}
		})
	}
}

// sameKinds compares two []entity.Kind slices ignoring order. nil and
// empty are equivalent (both mean "any kind allowed").
func sameKinds(a, b []entity.Kind) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[entity.Kind]int, len(a))
	for _, k := range a {
		seen[k]++
	}
	for _, k := range b {
		seen[k]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func TestRefsResolve_AnyKindFields(t *testing.T) {
	// addressed_by and relates_to permit any kind, so a gap addressed_by
	// a milestone or a decision relates_to a contract should resolve fine.
	tr := makeTree(
		&entity.Entity{ID: "E-01", Kind: entity.KindEpic},
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Parent: "E-01"},
		&entity.Entity{ID: "C-001", Kind: entity.KindContract},
		&entity.Entity{ID: "G-001", Kind: entity.KindGap, AddressedBy: []string{"M-001"}},
		&entity.Entity{ID: "D-001", Kind: entity.KindDecision, RelatesTo: []string{"C-001"}},
	)
	got := refsResolve(tr)
	if len(got) != 0 {
		t.Errorf("unexpected findings: %+v", got)
	}
}

func TestNoCycles_DependsOn(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, DependsOn: []string{"M-002"}, Path: "1.md"},
		&entity.Entity{ID: "M-002", Kind: entity.KindMilestone, DependsOn: []string{"M-003"}, Path: "2.md"},
		&entity.Entity{ID: "M-003", Kind: entity.KindMilestone, DependsOn: []string{"M-001"}, Path: "3.md"},
	)
	got := noCycles(tr)
	if len(got) != 3 {
		t.Fatalf("findings = %d, want 3: %+v", len(got), got)
	}
	for _, f := range got {
		if f.Code != "no-cycles" || f.Subcode != "depends_on" {
			t.Errorf("bad finding: %+v", f)
		}
	}
}

func TestNoCycles_ADRChain(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, SupersededBy: "ADR-0002", Path: "1.md"},
		&entity.Entity{ID: "ADR-0002", Kind: entity.KindADR, SupersededBy: "ADR-0001", Path: "2.md"},
	)
	got := noCycles(tr)
	if len(got) != 2 {
		t.Fatalf("findings = %d, want 2: %+v", len(got), got)
	}
	for _, f := range got {
		if f.Code != "no-cycles" || f.Subcode != "supersedes" {
			t.Errorf("bad finding: %+v", f)
		}
	}
}

func TestNoCycles_AcyclicIsClean(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, DependsOn: []string{"M-002"}},
		&entity.Entity{ID: "M-002", Kind: entity.KindMilestone, DependsOn: []string{"M-003"}},
		&entity.Entity{ID: "M-003", Kind: entity.KindMilestone},
	)
	got := noCycles(tr)
	if len(got) != 0 {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestNoCycles_SelfLoop(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, DependsOn: []string{"M-001"}, Path: "1.md"},
	)
	got := noCycles(tr)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
}

func TestTitlesNonempty(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-01", Kind: entity.KindEpic, Title: "good"},
		&entity.Entity{ID: "E-02", Kind: entity.KindEpic, Title: ""},
		&entity.Entity{ID: "E-03", Kind: entity.KindEpic, Title: "   "},
	)
	got := titlesNonempty(tr)
	if len(got) != 2 {
		t.Fatalf("findings = %d, want 2", len(got))
	}
	for _, f := range got {
		if f.Severity != SeverityWarning {
			t.Errorf("severity = %v, want warning", f.Severity)
		}
	}
}

func TestADRSupersessionMutual(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, SupersededBy: "ADR-0002"},
		// Mutual link missing — ADR-0002 does not list ADR-0001 in supersedes.
		&entity.Entity{ID: "ADR-0002", Kind: entity.KindADR, Supersedes: []string{}},
		// Properly mutual.
		&entity.Entity{ID: "ADR-0003", Kind: entity.KindADR, SupersededBy: "ADR-0004"},
		&entity.Entity{ID: "ADR-0004", Kind: entity.KindADR, Supersedes: []string{"ADR-0003"}},
	)
	got := adrSupersessionMutual(tr)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	if got[0].EntityID != "ADR-0001" {
		t.Errorf("got %+v", got[0])
	}
}

func TestGapResolvedHasResolver(t *testing.T) {
	tr := makeTree(
		// Open gap: no constraint.
		&entity.Entity{ID: "G-001", Kind: entity.KindGap, Status: "open"},
		// Wontfix: no constraint.
		&entity.Entity{ID: "G-002", Kind: entity.KindGap, Status: "wontfix"},
		// Addressed without resolver.
		&entity.Entity{ID: "G-003", Kind: entity.KindGap, Status: "addressed"},
		// Addressed with resolver.
		&entity.Entity{ID: "G-004", Kind: entity.KindGap, Status: "addressed", AddressedBy: []string{"M-001"}},
	)
	got := gapResolvedHasResolver(tr)
	if len(got) != 1 || got[0].EntityID != "G-003" {
		t.Errorf("got %+v", got)
	}
}

func TestRun_OrdersBySeverity(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-01", Kind: entity.KindEpic, Title: "", Status: "active"}, // titles-nonempty (warning)
		&entity.Entity{ID: "E-02", Kind: entity.KindEpic, Title: "Foo", Status: "wat"}, // status-valid (error)
	)
	got := Run(tr, nil)
	if len(got) < 2 {
		t.Fatalf("got %d findings, want at least 2", len(got))
	}
	// First finding should be the error.
	if got[0].Severity != SeverityError {
		t.Errorf("first finding severity = %v, want error: %+v", got[0].Severity, got[0])
	}
}

func TestRun_LoadErrorsAreFindings(t *testing.T) {
	tr := makeTree() // empty
	loadErrs := []tree.LoadError{
		{Path: "work/epics/E-01/epic.md", Err: errFake},
	}
	got := Run(tr, loadErrs)
	if len(got) != 1 {
		t.Fatalf("got %d, want 1: %+v", len(got), got)
	}
	if got[0].Code != "load-error" {
		t.Errorf("got %+v", got[0])
	}
}

func TestHasErrors(t *testing.T) {
	if HasErrors([]Finding{{Severity: SeverityWarning}}) {
		t.Error("HasErrors true on warning-only")
	}
	if !HasErrors([]Finding{{Severity: SeverityWarning}, {Severity: SeverityError}}) {
		t.Error("HasErrors false on mix")
	}
	if HasErrors(nil) {
		t.Error("HasErrors true on nil")
	}
}

// errFake is a sentinel for the load-error test.
var errFake = &fakeError{msg: "synthetic load error"}

type fakeError struct{ msg string }

func (e *fakeError) Error() string { return e.msg }

// --- Edge cases (items 8-12 from the test-coverage audit) ---

// TestIDsUnique_ThreeWayCollision verifies that every duplicate after
// the first surfaces as its own finding (so a 3-way collision yields
// 2 findings, not 1).
func TestIDsUnique_ThreeWayCollision(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Path: "a.md"},
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Path: "b.md"},
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Path: "c.md"},
	)
	got := idsUnique(tr)
	if len(got) != 2 {
		t.Fatalf("findings = %d, want 2: %+v", len(got), got)
	}
	gotPaths := []string{got[0].Path, got[1].Path}
	if gotPaths[0] != "b.md" || gotPaths[1] != "c.md" {
		t.Errorf("paths = %v, want [b.md c.md] (the second and third occurrences)", gotPaths)
	}
}

// TestNoCycles_DiamondIsAcyclic confirms that a DAG with two paths
// from the same source converging on the same target is not flagged
// as a cycle.
func TestNoCycles_DiamondIsAcyclic(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, DependsOn: []string{"M-002", "M-003"}},
		&entity.Entity{ID: "M-002", Kind: entity.KindMilestone, DependsOn: []string{"M-004"}},
		&entity.Entity{ID: "M-003", Kind: entity.KindMilestone, DependsOn: []string{"M-004"}},
		&entity.Entity{ID: "M-004", Kind: entity.KindMilestone},
	)
	got := noCycles(tr)
	if len(got) != 0 {
		t.Errorf("diamond DAG flagged as cyclic: %+v", got)
	}
}

// TestNoCycles_TwoDisjointCycles surfaces both cycles independently.
func TestNoCycles_TwoDisjointCycles(t *testing.T) {
	tr := makeTree(
		// Cycle A: M-001 <-> M-002
		&entity.Entity{ID: "M-001", Kind: entity.KindMilestone, DependsOn: []string{"M-002"}, Path: "1.md"},
		&entity.Entity{ID: "M-002", Kind: entity.KindMilestone, DependsOn: []string{"M-001"}, Path: "2.md"},
		// Cycle B: M-003 <-> M-004
		&entity.Entity{ID: "M-003", Kind: entity.KindMilestone, DependsOn: []string{"M-004"}, Path: "3.md"},
		&entity.Entity{ID: "M-004", Kind: entity.KindMilestone, DependsOn: []string{"M-003"}, Path: "4.md"},
	)
	got := noCycles(tr)
	if len(got) != 4 {
		t.Fatalf("findings = %d, want 4 (both cycles, both nodes): %+v", len(got), got)
	}
	seen := map[string]bool{}
	for _, f := range got {
		seen[f.EntityID] = true
	}
	for _, want := range []string{"M-001", "M-002", "M-003", "M-004"} {
		if !seen[want] {
			t.Errorf("cycle finding for %s missing", want)
		}
	}
}

// TestParse_TypeErrorsBecomeLoadErrors verifies that YAML type
// mismatches (a sequence where a string is expected, or vice versa)
// surface as parse failures and become load-error findings.
func TestParse_TypeErrorsBecomeLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			"parent as list",
			`---
id: M-001
title: Foo
status: draft
parent:
  - E-01
  - E-02
---
`,
		},
		{
			"depends_on as scalar",
			`---
id: M-001
title: Foo
status: draft
parent: E-01
depends_on: M-002
---
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entity.Parse("synthetic.md", []byte(tt.content))
			if err == nil {
				t.Error("expected parse error for type mismatch")
			}
		})
	}
}

// TestRun_PopulatesHintsAndLines exercises the post-processing pass:
// after every check has run, Run() should fill Line (1-based, derived
// from the field name) and Hint (from the code+subcode table) on each
// finding. We construct a real on-disk fixture so the line resolver
// has something to scan.
func TestRun_PopulatesHintsAndLines(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "work", "epics", "E-01-foo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Layout: parent on line 5, status on line 4. The line resolver
	// indexes the first occurrence of `<key>:` per file.
	body := "---\nid: M-001\ntitle: Bad parent\nstatus: draft\nparent: E-99\n---\n"
	mPath := filepath.Join(dir, "M-001-bad.md")
	if err := os.WriteFile(mPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Minimal epic so the milestone has a tree to live in (its parent
	// is intentionally pointing at E-99 to trigger the finding).
	epicBody := "---\nid: E-01\ntitle: Foo\nstatus: active\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte(epicBody), 0o644); err != nil {
		t.Fatal(err)
	}

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{
			{ID: "E-01", Kind: entity.KindEpic, Title: "Foo", Status: "active", Path: "work/epics/E-01-foo/epic.md"},
			{ID: "M-001", Kind: entity.KindMilestone, Title: "Bad parent", Status: "draft", Parent: "E-99", Path: "work/epics/E-01-foo/M-001-bad.md"},
		},
	}

	findings := Run(tr, nil)
	var refsFinding *Finding
	for i := range findings {
		if findings[i].Code == "refs-resolve" {
			refsFinding = &findings[i]
			break
		}
	}
	if refsFinding == nil {
		t.Fatalf("expected refs-resolve finding, got: %+v", findings)
	}
	if refsFinding.Line != 5 {
		t.Errorf("Line = %d, want 5 (the line of `parent:`)", refsFinding.Line)
	}
	if refsFinding.Hint == "" {
		t.Errorf("Hint should be populated for refs-resolve/unresolved")
	}
}

// TestRun_LineFallsBackToOne: when the field annotation doesn't match
// any line in the file (or the file can't be read), Line falls back to 1
// so editors still get a clickable file:line link.
func TestRun_LineFallsBackToOne(t *testing.T) {
	tr := makeTree(&entity.Entity{
		ID: "E-01", Kind: entity.KindEpic, Title: "Foo", Status: "bogus",
		Path: "synthetic-no-such-file.md",
	})
	findings := Run(tr, nil)
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}
	for _, f := range findings {
		if f.Path == "" {
			continue
		}
		if f.Line == 0 {
			t.Errorf("finding %s: Line=0, want 1 (fallback)", f.Code)
		}
	}
}

// TestSortFindings_Stable: when two findings tie on every sort key
// (severity, code, path), their input order must be preserved. This
// guarantees that callers who pre-order within a code group keep
// that order through the merge.
func TestSortFindings_Stable(t *testing.T) {
	in := []Finding{
		{Code: "x", Severity: SeverityError, Path: "a.md", EntityID: "first"},
		{Code: "x", Severity: SeverityError, Path: "a.md", EntityID: "second"},
		{Code: "x", Severity: SeverityError, Path: "a.md", EntityID: "third"},
	}
	SortFindings(in)
	if in[0].EntityID != "first" || in[1].EntityID != "second" || in[2].EntityID != "third" {
		t.Errorf("stable sort lost relative order: %+v", in)
	}
}

// TestSortFindings_ErrorsBeforeWarnings: error-severity findings
// always sort ahead of warnings, regardless of code.
func TestSortFindings_ErrorsBeforeWarnings(t *testing.T) {
	in := []Finding{
		{Code: "z-warn", Severity: SeverityWarning, Path: "a.md"},
		{Code: "a-err", Severity: SeverityError, Path: "z.md"},
	}
	SortFindings(in)
	if in[0].Severity != SeverityError {
		t.Errorf("first finding severity = %v, want error", in[0].Severity)
	}
}

// TestHintFor_KnownAndUnknown probes the public hint table.
func TestHintFor_KnownAndUnknown(t *testing.T) {
	if HintFor("refs-resolve", "unresolved") == "" {
		t.Errorf("known code+subcode should return a hint")
	}
	if HintFor("titles-nonempty", "") == "" {
		t.Errorf("known code (no subcode) should return a hint")
	}
	if HintFor("never-registered", "") != "" {
		t.Errorf("unknown code should return empty string")
	}
}
