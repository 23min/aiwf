package check

// Real-tree and walker assertions for the shipped-surface CLAUDE.md
// citation rule. The scanner's shapes are covered by unit cases; these pin
// that the walker actually reaches the shipped bytes, that those bytes are
// clean, and that the rule stays inert where the authoring tree is absent.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestSkillClaudeMDSection_RealEmbeddedTreeIsClean runs the production
// walker over this repo's actual shipped surfaces. A citation into a named
// CLAUDE.md section resolves only here and dangles in every consumer repo,
// so the shipped tree must carry none.
func TestSkillClaudeMDSection_RealEmbeddedTreeIsClean(t *testing.T) {
	t.Parallel()
	got := skillClaudeMDSectionReference(&tree.Tree{Root: repoRootForTest(t)})
	for _, f := range got {
		t.Errorf("%s:%d: %s", f.Path, f.Line, f.Message)
	}
}

// TestSkillClaudeMDSection_WalkerFiresAndSkips is the vacuity guard for the
// test above: a clean real tree reads identically whether the rule works or
// never reaches a file. This plants a citation in a synthetic authoring tree
// and requires it to surface, alongside the files the walk must pass over.
func TestSkillClaudeMDSection_WalkerFiresAndSkips(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	base := filepath.Join(root, skillScanDirs[0], "aiwf-demo")
	if err := os.MkdirAll(filepath.Join(base, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(base, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("SKILL.md", "Per CLAUDE.md §\"Working with the user,\" gates are explicit.\n")
	write("clean.md", "Project-specific rules live in `CLAUDE.md`.\n")
	write("notes.txt", "Per CLAUDE.md §\"Ignored\" — not markdown.\n")
	write(filepath.Join("nested", "deep.md"), "See CLAUDE.md *Provenance model* for identity.\n")

	got := skillClaudeMDSectionReference(&tree.Tree{Root: root})
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2 (SKILL.md and nested/deep.md): %+v", len(got), got)
	}
	for _, f := range got {
		if f.Code != CodeSkillClaudeMDSection {
			t.Errorf("code = %q, want %q", f.Code, CodeSkillClaudeMDSection)
		}
		if filepath.Ext(f.Path) != ".md" {
			t.Errorf("walker scanned a non-markdown file: %s", f.Path)
		}
	}
}

// TestSkillClaudeMDSection_InertWithoutAuthoringTree pins the consumer-repo
// case: the authoring dirs do not exist there, so the rule contributes no
// findings rather than erroring on the missing paths.
func TestSkillClaudeMDSection_InertWithoutAuthoringTree(t *testing.T) {
	t.Parallel()
	if got := skillClaudeMDSectionReference(&tree.Tree{Root: t.TempDir()}); len(got) != 0 {
		t.Errorf("got %d findings in a tree with no authoring dirs, want 0: %+v", len(got), got)
	}
}
