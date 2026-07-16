package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/tree"
)

// writeAndCommit writes content at root/rel (creating parent dirs) and
// commits it. Shared by the cross-branch show/list fixtures below.
func writeAndCommit(t *testing.T, root, rel, content, msg string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	if err := osExec(t, root, "git", "add", rel); err != nil {
		t.Fatalf("git add %s: %v", rel, err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", msg); err != nil {
		t.Fatalf("git commit (%s): %v", msg, err)
	}
}

// TestBuildShowView_CrossBranchResolvesAndLabelsContent_M0260AC1AC2 —
// a milestone minted on a sibling branch, absent from the checked-out
// branch entirely, resolves live via BlobReader (AC-1) and renders
// visibly labeled as cross-branch, distinct from a local entity
// (AC-2).
func TestBuildShowView_CrossBranchResolvesAndLabelsContent_M0260AC1AC2(t *testing.T) {
	root := setupCLITestRepo(t)
	// A commit on main so the repo has a HEAD before branching.
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	mBody := "---\nid: M-0100\ntitle: Sibling Milestone\nstatus: draft\nparent: E-0001\ntdd: none\n---\n\n## Goal\n\nDo the sibling thing.\n"
	writeAndCommit(t, root, "work/epics/E-0001-foo/M-0100-sibling-milestone.md", mBody, "sibling: mint M-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	if tr.ByID("M-0100") != nil {
		t.Fatal("M-0100 must be absent from the local (main) tree for this fixture")
	}

	view, ok := show.BuildShowView(ctx, root, tr, nil, "M-0100", 5)
	if !ok {
		t.Fatal("BuildShowView: not found, want cross-branch resolution")
	}
	if view.CrossBranch == nil {
		t.Fatal("CrossBranch = nil, want populated (AC-2 labeling)")
	}
	if view.CrossBranch.Collision {
		t.Error("CrossBranch.Collision = true, want false (single ref, no divergence)")
	}
	if view.CrossBranch.Ref != "refs/heads/sibling" {
		t.Errorf("CrossBranch.Ref = %q, want refs/heads/sibling", view.CrossBranch.Ref)
	}
	if view.Title != "Sibling Milestone" {
		t.Errorf("Title = %q, want %q (resolved live via BlobReader)", view.Title, "Sibling Milestone")
	}
	if view.Status != "draft" {
		t.Errorf("Status = %q, want draft", view.Status)
	}
	if got := view.Body["goal"]; !strings.Contains(got, "Do the sibling thing") {
		t.Errorf("Body[goal] = %q, want it to contain the sibling body text", got)
	}

	// Text rendering distinctly labels the row (AC-2's "distinct
	// rendering mode" evidence bar).
	out := string(testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"show", "--root", root, "M-0100"}); rc != cliutil.ExitOK {
			t.Fatalf("aiwf show M-0100: rc = %d", rc)
		}
	}))
	if !strings.Contains(out, "cross-branch") {
		t.Errorf("rendered text = %q, want a visible cross-branch label", out)
	}
}

// TestBuildShowView_CrossBranchCollision_DeclinesToRender_M0260AC3 —
// the same id minted independently (divergent content) on two
// sibling branches must not have either side's content rendered as
// canonical; aiwf show surfaces the ambiguity and names both refs
// instead.
func TestBuildShowView_CrossBranchCollision_DeclinesToRender_M0260AC3(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling-a"); err != nil {
		t.Fatalf("checkout sibling-a: %v", err)
	}
	gBodyA := "---\nid: G-0100\ntitle: Version A\nstatus: open\n---\n\n## Problem\n\nDescribed from side A.\n"
	writeAndCommit(t, root, "work/gaps/G-0100-collide.md", gBodyA, "sibling-a: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling-b"); err != nil {
		t.Fatalf("checkout sibling-b: %v", err)
	}
	gBodyB := "---\nid: G-0100\ntitle: Version B\nstatus: open\n---\n\n## Problem\n\nDescribed from side B, differently.\n"
	writeAndCommit(t, root, "work/gaps/G-0100-collide.md", gBodyB, "sibling-b: mint G-0100 independently")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	view, ok := show.BuildShowView(ctx, root, tr, nil, "G-0100", 5)
	if !ok {
		t.Fatal("BuildShowView: not found, want a collision view")
	}
	if view.CrossBranch == nil || !view.CrossBranch.Collision {
		t.Fatalf("CrossBranch = %+v, want Collision: true", view.CrossBranch)
	}
	if view.Title != "" {
		t.Errorf("Title = %q, want empty — a collision must not render either side's content as canonical", view.Title)
	}
	if view.Status != "" {
		t.Errorf("Status = %q, want empty", view.Status)
	}
	if len(view.Body) != 0 {
		t.Errorf("Body = %+v, want empty — AC-3 declines to render body content", view.Body)
	}
	refs := view.CrossBranch.Refs
	if len(refs) != 2 {
		t.Fatalf("Refs = %v, want exactly 2 candidate refs", refs)
	}
	wantRefs := map[string]bool{"refs/heads/sibling-a": true, "refs/heads/sibling-b": true}
	for _, r := range refs {
		if !wantRefs[r] {
			t.Errorf("unexpected ref %q in Refs = %v", r, refs)
		}
	}

	out := string(testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"show", "--root", root, "G-0100"}); rc != cliutil.ExitOK {
			t.Fatalf("aiwf show G-0100: rc = %d", rc)
		}
	}))
	if !strings.Contains(out, "collision") {
		t.Errorf("rendered text = %q, want a visible collision label", out)
	}
	if strings.Contains(out, "Version A") || strings.Contains(out, "Version B") {
		t.Errorf("rendered text = %q, must not leak either side's disputed content", out)
	}
}

// TestBuildShowView_UnknownEverywhere_StillNotFound is a regression
// guard: an id absent from the local tree AND every cross-branch ref
// must still read as an ordinary "not found," not a false match.
func TestBuildShowView_UnknownEverywhere_StillNotFound(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	if _, ok := show.BuildShowView(ctx, root, tr, nil, "G-9999", 5); ok {
		t.Error("BuildShowView(G-9999) = ok, want not-found (id exists nowhere)")
	}
}

// TestBuildShowView_CrossBranchResolved_MilestoneWithACs — a
// cross-branch-resolved milestone carries its ACs slice and per-AC
// body descriptions, the same as a locally-resolved milestone show.
func TestBuildShowView_CrossBranchResolved_MilestoneWithACs(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "work/epics/E-0001-foo/epic.md",
		"---\nid: E-0001\ntitle: Foo\nstatus: active\n---\n\n## Goal\n\ng\n", "seed: epic E-0001")

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	mBody := "---\nid: M-0100\ntitle: Sibling Milestone\nstatus: draft\nparent: E-0001\ntdd: none\n" +
		"acs:\n  - id: AC-1\n    title: First AC\n    status: open\n---\n\n## Goal\n\ng\n\n" +
		"## Acceptance criteria\n\n### AC-1 — First AC\n\nDescription of the first AC.\n"
	writeAndCommit(t, root, "work/epics/E-0001-foo/M-0100-sibling.md", mBody, "sibling: mint M-0100 with ACs")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	view, ok := show.BuildShowView(ctx, root, tr, nil, "M-0100", 5)
	if !ok {
		t.Fatal("BuildShowView: not found, want cross-branch resolution")
	}
	if len(view.ACs) != 1 {
		t.Fatalf("ACs = %+v, want exactly 1", view.ACs)
	}
	if view.ACs[0].ID != "AC-1" || view.ACs[0].Title != "First AC" {
		t.Errorf("ACs[0] = %+v, want id AC-1 / title %q", view.ACs[0], "First AC")
	}
	if !strings.Contains(view.ACs[0].Description, "Description of the first AC") {
		t.Errorf("ACs[0].Description = %q, want it to contain the AC's body prose", view.ACs[0].Description)
	}
}

// TestBuildShowView_CrossBranchResolved_MalformedContentNotFound — a
// cross-branch id whose content fails to parse (malformed
// frontmatter) reads as an ordinary "not found," not a crash or a
// hard error.
func TestBuildShowView_CrossBranchResolved_MalformedContentNotFound(t *testing.T) {
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
	if _, ok := show.BuildShowView(ctx, root, tr, nil, "G-0100", 5); ok {
		t.Error("BuildShowView(G-0100): ok = true, want not-found for malformed cross-branch content")
	}
}
