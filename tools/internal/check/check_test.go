package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
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
	for i, f := range fs {
		out[i] = f.Code
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
		// Contract missing format and artifact.
		&entity.Entity{ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "draft", Path: "d.md"},
		// Clean.
		&entity.Entity{ID: "G-001", Kind: entity.KindGap, Title: "Foo", Status: "open", Path: "e.md"},
	)
	got := frontmatterShape(tr)
	want := []string{
		"frontmatter-shape", // missing id (a.md)
		"frontmatter-shape", // bad id format (b.md)
		"frontmatter-shape", // milestone missing parent (c.md)
		"frontmatter-shape", // contract missing format (d.md)
		"frontmatter-shape", // contract missing artifact (d.md)
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

func TestContractArtifactExists(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "work", "contracts", "C-001-orders", "schema"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "contracts", "C-001-orders", "schema", "openapi.yaml"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{
			// Good.
			{
				ID: "C-001", Kind: entity.KindContract, Title: "OK", Status: "draft",
				Format: "openapi", Artifact: "schema/openapi.yaml",
				Path: "work/contracts/C-001-orders/contract.md",
			},
			// Path-escape via "..".
			{
				ID: "C-002", Kind: entity.KindContract, Title: "Bad", Status: "draft",
				Format: "openapi", Artifact: "../escape.yaml",
				Path: "work/contracts/C-002-bad/contract.md",
			},
			// Absolute path.
			{
				ID: "C-003", Kind: entity.KindContract, Title: "Abs", Status: "draft",
				Format: "openapi", Artifact: "/etc/passwd",
				Path: "work/contracts/C-003-abs/contract.md",
			},
			// Missing file.
			{
				ID: "C-004", Kind: entity.KindContract, Title: "Missing", Status: "draft",
				Format: "openapi", Artifact: "schema/missing.yaml",
				Path: "work/contracts/C-004-missing/contract.md",
			},
		},
	}

	got := contractArtifactExists(tr)
	if len(got) != 3 {
		t.Fatalf("findings = %d, want 3: %+v", len(got), got)
	}
	gotIDs := []string{got[0].EntityID, got[1].EntityID, got[2].EntityID}
	wantIDs := []string{"C-002", "C-003", "C-004"}
	if diff := cmp.Diff(wantIDs, gotIDs); diff != "" {
		t.Errorf("entity ids mismatch (-want +got):\n%s", diff)
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

// TestContractArtifactExists_DirectoryAtArtifactPath rejects a
// directory present where a regular file is expected (the Q1 schema
// declares `artifact` is a path to a file, not a folder).
func TestContractArtifactExists_DirectoryAtArtifactPath(t *testing.T) {
	root := t.TempDir()
	// Create a directory at the artifact path instead of a file.
	if err := os.MkdirAll(filepath.Join(root, "work", "contracts", "C-001-foo", "schema", "openapi.yaml"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{
			{
				ID: "C-001", Kind: entity.KindContract, Title: "Dir-as-artifact", Status: "draft",
				Format: "openapi", Artifact: "schema/openapi.yaml",
				Path: "work/contracts/C-001-foo/contract.md",
			},
		},
	}
	got := contractArtifactExists(tr)
	if len(got) != 1 || got[0].EntityID != "C-001" {
		t.Errorf("got %+v, want one finding for C-001", got)
	}
}
