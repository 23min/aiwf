package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestImport_TwoBranchesNoCollision pins M-0269/AC-1 (G-0426): an
// `id: auto` manifest entry allocates through the same cross-branch
// view `aiwf add` uses, not just the working tree. Mirrors
// TestAdd_TwoBranchesNoCollision's fixture shape (see
// ac3_two_branch_alloc_test.go) but drives `aiwf import` instead of
// `aiwf add` — the seam under test is import.go's auto-allocation
// loop, which historically read only the working tree and the
// manifest's own reservations, never t.AllocationIDs() (trunk +
// local-ref + remote-ref ids).
func TestImport_TwoBranchesNoCollision(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustGit(t, root, "add", "-A")
	mustGit(t, root, "commit", "-q", "-m", "base")

	// Branch A forks from the base commit and imports the first auto
	// epic.
	mustGit(t, root, "checkout", "-b", "branchA")
	manifestA := writeManifest(t, root, `version: 1
entities:
  - kind: epic
    id: auto
    frontmatter: {title: "Alpha epic", status: proposed}
`)
	if rc := cli.Execute([]string{"import", "--root", root, "--actor", "human/test", manifestA}); rc != cliutil.ExitOK {
		t.Fatalf("import on branchA: rc=%d", rc)
	}
	gotA := soleEpicID(t, root)

	// Branch B forks from BEFORE A's import commit, so its working
	// tree does not contain A's epic. Without the cross-branch scan,
	// import's own hand-rolled allocator would hand back the same id.
	mustGit(t, root, "checkout", "-b", "branchB", "branchA~1")
	manifestB := writeManifest(t, root, `version: 1
entities:
  - kind: epic
    id: auto
    frontmatter: {title: "Bravo epic", status: proposed}
`)
	if rc := cli.Execute([]string{"import", "--root", root, "--actor", "human/test", manifestB}); rc != cliutil.ExitOK {
		t.Fatalf("import on branchB: rc=%d", rc)
	}
	gotB := soleEpicID(t, root)

	if gotB == gotA {
		t.Fatalf("collision: both branches allocated %s; import's auto-id path should have seen branch A's id via the local-refs scan", gotA)
	}
	if gotA != "E-0001" || gotB != "E-0002" {
		t.Errorf("ids = (A=%s, B=%s), want (E-0001, E-0002)", gotA, gotB)
	}
}

// soleEpicID globs the single epic directory in root's working tree
// and returns its id (e.g. "E-0002"), derived from the directory name.
func soleEpicID(t *testing.T, root string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-*"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob work/epics/E-*: matches=%v err=%v", matches, err)
	}
	parts := strings.SplitN(filepath.Base(matches[0]), "-", 3)
	if len(parts) < 2 {
		t.Fatalf("unexpected epic dirname %q", matches[0])
	}
	return parts[0] + "-" + parts[1]
}
