package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestTreeDiscipline_NoStrays_NoFindings: a clean tree produces
// zero findings, confirming the rule short-circuits when Strays is
// empty.
func TestTreeDiscipline_NoStrays_NoFindings(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Path: "work/epics/E-01-foo/M-001-bar.md"})
	got := TreeDiscipline(tr, nil, false)
	if len(got) != 0 {
		t.Errorf("TreeDiscipline on clean tree = %+v, want []", got)
	}
}

// TestTreeDiscipline_PlainStray_Warning: a stray file under work/
// surfaces as a single warning by default.
func TestTreeDiscipline_PlainStray_Warning(t *testing.T) {
	t.Parallel()
	tr := makeTree()
	tr.Strays = []string{"work/epics/E-01-foo/notes.md"}

	got := TreeDiscipline(tr, nil, false)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	f := got[0]
	if f.Code != "unexpected-tree-file" {
		t.Errorf("Code = %q, want unexpected-tree-file", f.Code)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q, want warning (default)", f.Severity)
	}
	if f.Path != "work/epics/E-01-foo/notes.md" {
		t.Errorf("Path = %q", f.Path)
	}
	if !strings.Contains(f.Message, "tree-shape changes") {
		t.Errorf("Message missing tree-shape guidance: %q", f.Message)
	}
}

// TestTreeDiscipline_Strict_Error: tree.strict=true upgrades the
// warning to an error so the pre-push hook blocks the push.
func TestTreeDiscipline_Strict_Error(t *testing.T) {
	t.Parallel()
	tr := makeTree()
	tr.Strays = []string{"work/gaps/scratch.md"}

	got := TreeDiscipline(tr, nil, true)
	if len(got) != 1 || got[0].Severity != SeverityError {
		t.Fatalf("strict mode should produce one error: %+v", got)
	}
}

// TestTreeDiscipline_AllowPaths_Glob: paths matching a configured
// glob are exempt. Uses filepath.Match — `?` is single-char,
// `*` does not cross slashes.
func TestTreeDiscipline_AllowPaths_Glob(t *testing.T) {
	t.Parallel()
	tr := makeTree()
	tr.Strays = []string{
		"work/epics/E-01-foo/notes.md",
		"work/gaps/scratch.md",
	}

	got := TreeDiscipline(tr, []string{"work/epics/*/notes.md"}, false)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1 (scratch.md only): %+v", len(got), got)
	}
	if got[0].Path != "work/gaps/scratch.md" {
		t.Errorf("wrong path remained: %q", got[0].Path)
	}
}

// TestTreeDiscipline_ContractDir_Auto_Exempt: files under a
// recognized contract's directory are auto-exempt — contracts
// legitimately carry schema/fixture artifacts alongside contract.md.
// A stray sibling of a contract dir (not inside it) still fires.
func TestTreeDiscipline_ContractDir_Auto_Exempt(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID:   "C-0001",
		Kind: entity.KindContract,
		Path: "work/contracts/C-001-foo/contract.md",
	})
	tr.Strays = []string{
		"work/contracts/C-001-foo/schema.cue",
		"work/contracts/C-001-foo/fixtures/valid/example.json",
		"work/contracts/loose-note.md", // not inside any contract subdir
	}

	got := TreeDiscipline(tr, nil, false)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1 (loose-note only): %+v", len(got), got)
	}
	if got[0].Path != "work/contracts/loose-note.md" {
		t.Errorf("expected loose-note finding, got %q", got[0].Path)
	}
}

// TestTreeDiscipline_LoaderSeam asserts the seam between tree.Load
// (which decides what counts as a stray) and TreeDiscipline (which
// classifies them). Without this, a refactor that broke Strays
// population would still pass the in-memory unit tests above. Per
// tools/CLAUDE.md "Test the seam, not just the layer."
func TestTreeDiscipline_LoaderSeam(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite := func(rel, body string) {
		t.Helper()
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// One real entity, one stray under work/, and one file under
	// docs/adr (which is permissive — should NOT become a stray).
	mustWrite("work/gaps/G-001-real.md", "---\nid: G-001\nkind: gap\nstatus: open\ntitle: real gap\n---\nbody\n")
	mustWrite("work/gaps/scratch.md", "not an entity\n")
	mustWrite("docs/adr/README.md", "permissive zone\n")

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	wantStray := "work/gaps/scratch.md"
	if len(tr.Strays) != 1 || tr.Strays[0] != wantStray {
		t.Fatalf("Strays = %+v, want [%q] (docs/adr should be permissive)", tr.Strays, wantStray)
	}

	got := TreeDiscipline(tr, nil, false)
	if len(got) != 1 || got[0].Path != wantStray {
		t.Fatalf("TreeDiscipline = %+v, want one finding for %q", got, wantStray)
	}
}
