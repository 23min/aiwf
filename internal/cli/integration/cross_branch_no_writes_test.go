package integration

import (
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// gitOutput runs a git command in workdir and returns trimmed stdout,
// failing the test on error.
func gitOutput(t *testing.T, workdir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

// repoSnapshot captures the working-tree/index/refs state a read-only
// verb must never disturb (M-0260/AC-4): the current commit, every
// ref and the SHA it resolves to, the porcelain status (clean/dirty),
// and a content hash of the on-disk index file.
type repoSnapshot struct {
	head            string
	refs            string
	statusPorcelain string
	indexHash       string
}

func snapshotRepo(t *testing.T, root string) repoSnapshot {
	t.Helper()
	indexPath := filepath.Join(root, ".git", "index")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read .git/index: %v", err)
	}
	return repoSnapshot{
		head:            gitOutput(t, root, "rev-parse", "HEAD"),
		refs:            gitOutput(t, root, "show-ref"),
		statusPorcelain: gitOutput(t, root, "status", "--porcelain"),
		indexHash:       string(sha256.New().Sum(indexBytes)),
	}
}

// TestShowAndList_CrossBranchResolution_NoWorkingTreeIndexOrRefWrites
// — M-0260/AC-4: resolving cross-branch content via `aiwf show`/`aiwf
// list` (both the resolved and the collision path) must never write
// to the working tree, the index, or any ref. Snapshots bracket only
// the verb invocations themselves, not the fixture setup.
func TestShowAndList_CrossBranchResolution_NoWorkingTreeIndexOrRefWrites(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-sibling.md",
		"---\nid: G-0100\ntitle: Sibling Gap\nstatus: open\n---\n\n## Problem\n\ndescribed.\n",
		"sibling: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling-b"); err != nil {
		t.Fatalf("checkout sibling-b: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0200-collide.md",
		"---\nid: G-0200\ntitle: Version B\nstatus: open\n---\n\n## Problem\n\nB.\n",
		"sibling-b: mint G-0200")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}
	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling-c"); err != nil {
		t.Fatalf("checkout sibling-c: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0200-collide.md",
		"---\nid: G-0200\ntitle: Version C\nstatus: open\n---\n\n## Problem\n\nC, differently.\n",
		"sibling-c: mint G-0200 independently")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	before := snapshotRepo(t, root)

	if rc := cli.Execute([]string{"show", "--root", root, "G-0100"}); rc != cliutil.ExitOK {
		t.Fatalf("aiwf show G-0100 (resolved): rc = %d", rc)
	}
	if rc := cli.Execute([]string{"show", "--root", root, "G-0200"}); rc != cliutil.ExitOK {
		t.Fatalf("aiwf show G-0200 (collision): rc = %d", rc)
	}
	if rc := cli.Execute([]string{"list", "--root", root, "--kind", "gap"}); rc != cliutil.ExitOK {
		t.Fatalf("aiwf list --kind gap: rc = %d", rc)
	}

	after := snapshotRepo(t, root)

	if before.head != after.head {
		t.Errorf("HEAD changed: %q -> %q", before.head, after.head)
	}
	if before.refs != after.refs {
		t.Errorf("refs changed:\nbefore:\n%s\nafter:\n%s", before.refs, after.refs)
	}
	if after.statusPorcelain != "" {
		t.Errorf("working tree dirty after cross-branch resolution: %q", after.statusPorcelain)
	}
	if before.statusPorcelain != after.statusPorcelain {
		t.Errorf("status --porcelain changed: %q -> %q", before.statusPorcelain, after.statusPorcelain)
	}
	if before.indexHash != after.indexHash {
		t.Error("index content hash changed — a cross-branch read must never write the index")
	}
}
