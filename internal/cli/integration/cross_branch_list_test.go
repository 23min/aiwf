package integration

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/cli/list"
	"github.com/23min/aiwf/internal/tree"
)

// findRow returns the row for id, or nil.
func findRow(rows []list.ListSummary, id string) *list.ListSummary {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

// TestBuildListRows_CrossBranchResolvesAndLabels_M0260AC1AC2 — a gap
// minted on a sibling branch, absent from main, participates in a
// kind-filtered listing (AC-1) labeled distinctly via CrossBranchRef
// (AC-2), with its real content resolved.
func TestBuildListRows_CrossBranchResolvesAndLabels_M0260AC1AC2(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	gBody := "---\nid: G-0100\ntitle: Sibling Gap\nstatus: open\n---\n\n## Problem\n\ndescribed.\n"
	writeAndCommit(t, root, "work/gaps/G-0100-sibling.md", gBody, "sibling: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	rows := list.BuildListRows(ctx, tr, "gap", "", "", "", false)
	row := findRow(rows, "G-0100")
	if row == nil {
		t.Fatalf("rows = %+v, want a row for cross-branch-known G-0100", rows)
	}
	if row.CrossBranchRef != "refs/heads/sibling" {
		t.Errorf("CrossBranchRef = %q, want refs/heads/sibling", row.CrossBranchRef)
	}
	if row.Title != "Sibling Gap" {
		t.Errorf("Title = %q, want resolved content %q", row.Title, "Sibling Gap")
	}
	if row.Status != "open" {
		t.Errorf("Status = %q, want open", row.Status)
	}
}

// setupCollidingSiblings mints gap G-0100 with divergent content on
// two sibling branches, neither merged into main. Returns the repo
// root.
func setupCollidingSiblings(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling-a"); err != nil {
		t.Fatalf("checkout sibling-a: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-collide.md",
		"---\nid: G-0100\ntitle: Version A\nstatus: open\n---\n\n## Problem\n\nA.\n",
		"sibling-a: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling-b"); err != nil {
		t.Fatalf("checkout sibling-b: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-collide.md",
		"---\nid: G-0100\ntitle: Version B\nstatus: addressed\n---\n\n## Problem\n\nB, differently.\n",
		"sibling-b: mint G-0100 independently")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}
	return root
}

// TestBuildListRows_CrossBranchCollision_KindOnlyQuery_M0260AC3 — a
// kind-only (or unfiltered) query surfaces a collision row rather than
// silently hiding the ambiguity; no side's content is picked.
func TestBuildListRows_CrossBranchCollision_KindOnlyQuery_M0260AC3(t *testing.T) {
	root := setupCollidingSiblings(t)
	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	rows := list.BuildListRows(ctx, tr, "gap", "", "", "", false)
	row := findRow(rows, "G-0100")
	if row == nil {
		t.Fatalf("rows = %+v, want a collision row for G-0100 in a kind-only query", rows)
	}
	if !row.CrossBranchCollision {
		t.Errorf("row = %+v, want CrossBranchCollision: true", row)
	}
	if row.Title != "" || row.Status != "" {
		t.Errorf("row = %+v, want empty Title/Status — must not pick a side", row)
	}
	if len(row.CrossBranchRefs) != 2 {
		t.Errorf("CrossBranchRefs = %v, want 2 candidate refs", row.CrossBranchRefs)
	}
}

// TestBuildListRows_CrossBranchCollision_StatusFilterExcludes_M0260AC3
// — a --status filter can't honestly evaluate a collision row (its
// real status is exactly what's in dispute), so the row is excluded
// rather than risk a false-positive match.
func TestBuildListRows_CrossBranchCollision_StatusFilterExcludes_M0260AC3(t *testing.T) {
	root := setupCollidingSiblings(t)
	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	rows := list.BuildListRows(ctx, tr, "gap", "open", "", "", false)
	if row := findRow(rows, "G-0100"); row != nil {
		t.Errorf("rows = %+v, want the collision row excluded once --status is set", rows)
	}
}

// TestBuildListRows_CrossBranchCollision_ParentFilterExcludes_M0260AC3
// mirrors the status case for --parent.
func TestBuildListRows_CrossBranchCollision_ParentFilterExcludes_M0260AC3(t *testing.T) {
	root := setupCollidingSiblings(t)
	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	rows := list.BuildListRows(ctx, tr, "gap", "", "E-0001", "", false)
	if row := findRow(rows, "G-0100"); row != nil {
		t.Errorf("rows = %+v, want the collision row excluded once --parent is set", rows)
	}
}

// TestBuildListRows_CrossBranchCollision_ArchivedNeverExcludes —
// --archived only controls default suppression of terminal-status
// entities; it must never hide an unresolved collision, since we
// cannot even confirm it is terminal.
func TestBuildListRows_CrossBranchCollision_ArchivedNeverExcludes(t *testing.T) {
	root := setupCollidingSiblings(t)
	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	rows := list.BuildListRows(ctx, tr, "gap", "", "", "", true)
	if row := findRow(rows, "G-0100"); row == nil {
		t.Errorf("rows = %+v, want the collision row still present with --archived", rows)
	}
}

// TestBuildListRows_CrossBranchResolved_RespectsStatusFilter — a
// resolved (non-collision) row's real status is known, so it
// participates in --status filtering exactly like a local row.
func TestBuildListRows_CrossBranchResolved_RespectsStatusFilter(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")
	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-sibling.md",
		"---\nid: G-0100\ntitle: Sibling Gap\nstatus: addressed\n---\n\n## Problem\n\ndescribed.\n",
		"sibling: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	if row := findRow(list.BuildListRows(ctx, tr, "gap", "open", "", "", false), "G-0100"); row != nil {
		t.Errorf("rows = %+v, want G-0100 excluded — its real status (addressed) doesn't match --status=open", row)
	}
	if row := findRow(list.BuildListRows(ctx, tr, "gap", "addressed", "", "", true), "G-0100"); row == nil {
		t.Error("want G-0100 included when --status=addressed matches its real resolved status")
	}
}

// TestBuildListRows_LocalEntityTakesPrecedenceOverCrossBranchShadow —
// an id present in the local working tree must never be shadowed or
// duplicated by its own cross-branch hit (the current branch's own
// ref is itself one of the scanned local refs).
func TestBuildListRows_LocalEntityTakesPrecedenceOverCrossBranchShadow(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "work/gaps/G-0100-local.md",
		"---\nid: G-0100\ntitle: Local Gap\nstatus: open\n---\n\n## Problem\n\nlocal.\n",
		"seed: local G-0100")

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	rows := list.BuildListRows(ctx, tr, "gap", "", "", "", false)
	var count int
	for _, r := range rows {
		if r.ID == "G-0100" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("G-0100 appears %d times in rows = %+v, want exactly 1", count, rows)
	}
	row := findRow(rows, "G-0100")
	if row.CrossBranchRef != "" || row.CrossBranchCollision {
		t.Errorf("row = %+v, want an ordinary local row (no cross-branch marker)", row)
	}
}

// TestBuildListRows_CrossBranchKindMismatch_Excluded — a cross-branch
// hit whose kind doesn't match --kind is excluded, the same as any
// local row would be.
func TestBuildListRows_CrossBranchKindMismatch_Excluded(t *testing.T) {
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

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	if row := findRow(list.BuildListRows(ctx, tr, "milestone", "", "", "", false), "G-0100"); row != nil {
		t.Errorf("rows = %+v, want the cross-branch gap excluded from a --kind milestone query", row)
	}
}

// TestBuildListRows_CrossBranchNoKindFilter_StillIncluded — a
// cross-branch row participates even when --kind is unset, as long as
// some other filter keeps the call out of the no-args counts path.
func TestBuildListRows_CrossBranchNoKindFilter_StillIncluded(t *testing.T) {
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

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	if row := findRow(list.BuildListRows(ctx, tr, "", "", "", "", true), "G-0100"); row == nil {
		t.Error("want the cross-branch gap included with no --kind filter (archived=true keeps this off the no-args counts path)")
	}
}

// TestBuildListRows_CrossBranchCollision_AreaFilterExcludes mirrors
// the status/parent cases for --area.
func TestBuildListRows_CrossBranchCollision_AreaFilterExcludes(t *testing.T) {
	root := setupCollidingSiblings(t)
	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	rows := list.BuildListRows(ctx, tr, "gap", "", "", "platform", false)
	if row := findRow(rows, "G-0100"); row != nil {
		t.Errorf("rows = %+v, want the collision row excluded once --area is set", rows)
	}
}

// TestBuildListRows_CrossBranchResolved_DefaultArchivedExcludesTerminalStatus
// — a resolved row's real (terminal) status excludes it by default,
// the same as a local row, isolated from the --status filter branch
// by leaving --status unset.
func TestBuildListRows_CrossBranchResolved_DefaultArchivedExcludesTerminalStatus(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")
	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-sibling.md",
		"---\nid: G-0100\ntitle: Sibling Gap\nstatus: addressed\n---\n\n## Problem\n\ndescribed.\n",
		"sibling: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	if row := findRow(list.BuildListRows(ctx, tr, "gap", "", "", "", false), "G-0100"); row != nil {
		t.Errorf("rows = %+v, want the terminal-status resolved row excluded by default (no --status filter involved)", row)
	}
	if row := findRow(list.BuildListRows(ctx, tr, "gap", "", "", "", true), "G-0100"); row == nil {
		t.Error("want the terminal-status resolved row included with --archived")
	}
}

// TestBuildListRows_CrossBranchResolved_RespectsParentFilter — a
// resolved row's real parent is known, so --parent filters it exactly
// like a local row.
func TestBuildListRows_CrossBranchResolved_RespectsParentFilter(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "work/epics/E-0001-foo/epic.md",
		"---\nid: E-0001\ntitle: Foo\nstatus: active\n---\n\n## Goal\n\ng\n", "seed: epic E-0001")
	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	writeAndCommit(t, root, "work/epics/E-0001-foo/M-0100-sibling.md",
		"---\nid: M-0100\ntitle: Sibling Milestone\nstatus: draft\nparent: E-0001\ntdd: none\n---\n\n## Goal\n\ng\n",
		"sibling: mint M-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	if row := findRow(list.BuildListRows(ctx, tr, "milestone", "", "E-0099", "", false), "M-0100"); row != nil {
		t.Errorf("rows = %+v, want M-0100 excluded — its real parent (E-0001) doesn't match --parent=E-0099", row)
	}
	if row := findRow(list.BuildListRows(ctx, tr, "milestone", "", "E-0001", "", false), "M-0100"); row == nil {
		t.Error("want M-0100 included when --parent=E-0001 matches its real resolved parent")
	}
}

// TestBuildListRows_CrossBranchResolved_RespectsAreaFilter — a
// resolved row's real area is known, so --area filters it exactly
// like a local row.
func TestBuildListRows_CrossBranchResolved_RespectsAreaFilter(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")
	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-sibling.md",
		"---\nid: G-0100\ntitle: Sibling Gap\nstatus: open\narea: platform\n---\n\n## Problem\n\ndescribed.\n",
		"sibling: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	if row := findRow(list.BuildListRows(ctx, tr, "gap", "", "", "billing", false), "G-0100"); row != nil {
		t.Errorf("rows = %+v, want G-0100 excluded — its real area (platform) doesn't match --area=billing", row)
	}
	if row := findRow(list.BuildListRows(ctx, tr, "gap", "", "", "platform", false), "G-0100"); row == nil {
		t.Error("want G-0100 included when --area=platform matches its real resolved area")
	}
}

// TestBuildListRows_CrossBranchResolved_MalformedContentDegradesGracefully
// — a cross-branch id whose content fails to parse (malformed
// frontmatter) is simply omitted from the row set, not a crash or a
// hard error surfaced through aiwf list.
func TestBuildListRows_CrossBranchResolved_MalformedContentDegradesGracefully(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")
	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-sibling.md", "not valid frontmatter at all\n", "sibling: malformed G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	rows := list.BuildListRows(ctx, tr, "gap", "", "", "", true)
	if row := findRow(rows, "G-0100"); row != nil {
		t.Errorf("rows = %+v, want the malformed cross-branch entity omitted, not surfaced", row)
	}
}
