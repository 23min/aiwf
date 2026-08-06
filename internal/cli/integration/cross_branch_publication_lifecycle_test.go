package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// ADR-0041's Validation section: a cross-branch reference is classified
// by whether the branch carrying its target has been published, and the
// classification is recomputed from live refs rather than tracked, so it
// must follow the branch in both directions. This drives the whole
// lifecycle against a real repo and a real bare remote — mint on an
// unpushed branch, push, unpublish, delete — and asserts the verdict at
// each step.
//
// Both reference surfaces ride the same fixture because ADR-0041 applies
// to them symmetrically: a decision's `relates_to` (structured, the
// refs-resolve rule) and a gap's body prose (the body-prose-id rule)
// both point at the same cross-branch id, so each phase asserts the pair
// moved together. A change classifying one and not the other fails here.

// crossBranchVerdict is what the two reference rules concluded about one
// id in one run: the subcode and severity each rule reported for it.
type crossBranchVerdict struct {
	refsSubcode     string
	refsSeverity    check.Severity
	proseSubcode    string
	proseSeverity   check.Severity
	refsMessage     string
	proseMessage    string
	refsFindings    int
	proseFindings   int
	unrelatedErrors int
}

// verdictFor loads root through the same loader `aiwf check` uses — the
// one that builds the cross-branch view — and reports what the two
// reference rules concluded about id. Findings about anything else are
// counted but not inspected: the fixture's own entities carry whatever
// unrelated findings they carry, and this test is about one id.
func verdictFor(t *testing.T, root, id string) crossBranchVerdict {
	t.Helper()
	tr, loadErrs, err := cliutil.LoadTreeWithTrunk(context.Background(), root)
	if err != nil {
		t.Fatalf("LoadTreeWithTrunk: %v", err)
	}
	var v crossBranchVerdict
	findings := check.Run(tr, loadErrs)
	for i := range findings {
		f := &findings[i]
		if !strings.Contains(f.Message, id) {
			continue
		}
		switch f.Code {
		case check.CodeRefsResolve:
			v.refsFindings++
			v.refsSubcode, v.refsSeverity, v.refsMessage = f.Subcode, f.Severity, f.Message
		case check.CodeBodyProseID:
			v.proseFindings++
			v.proseSubcode, v.proseSeverity, v.proseMessage = f.Subcode, f.Severity, f.Message
		default:
			if f.Severity == check.SeverityError {
				v.unrelatedErrors++
			}
		}
	}
	return v
}

// assertVerdict fails unless both rules reported exactly one finding for
// the id, carrying the wanted subcode and severity.
func assertVerdict(t *testing.T, phase string, got crossBranchVerdict, wantSubcode string, wantSeverity check.Severity) {
	t.Helper()
	if got.refsFindings != 1 {
		t.Errorf("%s: refs-resolve findings for the target = %d, want exactly 1", phase, got.refsFindings)
	}
	if got.proseFindings != 1 {
		t.Errorf("%s: body-prose-id findings for the target = %d, want exactly 1", phase, got.proseFindings)
	}
	if got.refsSubcode != wantSubcode {
		t.Errorf("%s: refs-resolve subcode = %q, want %q (message: %s)", phase, got.refsSubcode, wantSubcode, got.refsMessage)
	}
	if got.proseSubcode != wantSubcode {
		t.Errorf("%s: body-prose-id subcode = %q, want %q (message: %s)", phase, got.proseSubcode, wantSubcode, got.proseMessage)
	}
	if got.refsSeverity != wantSeverity {
		t.Errorf("%s: refs-resolve severity = %q, want %q", phase, got.refsSeverity, wantSeverity)
	}
	if got.proseSeverity != wantSeverity {
		t.Errorf("%s: body-prose-id severity = %q, want %q", phase, got.proseSeverity, wantSeverity)
	}
	// The fixture's other entities are well-formed, so any error from a
	// rule other than the two under test means the fixture drifted and
	// the phase is measuring something it did not intend to.
	if got.unrelatedErrors != 0 {
		t.Errorf("%s: %d error-severity findings from rules other than the two under test, want 0", phase, got.unrelatedErrors)
	}
}

// TestCrossBranchClassification_FollowsBranchPublication_ADR0041 walks
// the four states ADR-0041 names, in order, against one working copy.
func TestCrossBranchClassification_FollowsBranchPublication_ADR0041(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	// A bare remote, so `refs/remotes/*` is a real namespace here rather
	// than a shape the fixture asserts about in the abstract.
	bare := filepath.Join(t.TempDir(), "origin.git")
	if err := osExec(t, root, "git", "init", "--bare", "-q", "-b", "main", bare); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}

	// Two references to G-0100 on mainline, one per rule.
	writeAndCommit(t, root, "work/decisions/D-0001-cite-the-sibling-gap.md",
		"---\nid: D-0001\ntitle: Cite the sibling gap\nstatus: accepted\nrelates_to:\n    - G-0100\n---\n"+
			"## Question\n\nWhat does the sibling gap imply here?\n\n"+
			"## Decision\n\nCite it and move on.\n\n"+
			"## Reasoning\n\nThe structured reference is the surface under test.\n",
		"mint D-0001 referencing G-0100")
	writeAndCommit(t, root, "work/gaps/G-0001-prose-cites-the-sibling-gap.md",
		"---\nid: G-0001\ntitle: Prose cites the sibling gap\nstatus: open\n---\n"+
			"## What's missing\n\nThe body prose cites G-0100, which lives elsewhere.\n\n"+
			"## Why it matters\n\nThe prose reference is the second surface under test.\n",
		"mint G-0001 citing G-0100 in prose")

	if err := osExec(t, root, "git", "remote", "add", "origin", bare); err != nil {
		t.Fatalf("git remote add: %v", err)
	}
	// Mainline must be published before anything else: the configured
	// trunk ref defaults to refs/remotes/origin/main, and trunk.Read
	// hard-errors on a repo that has tracking refs but no resolvable
	// trunk.
	if err := osExec(t, root, "git", "push", "-q", "-u", "origin", "main"); err != nil {
		t.Fatalf("git push main: %v", err)
	}

	// The target is minted on a branch that exists only here.
	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-minted-on-the-sibling-branch.md",
		"---\nid: G-0100\ntitle: Minted on the sibling branch\nstatus: open\n---\n"+
			"## What's missing\n\nThis entity exists on one branch.\n\n"+
			"## Why it matters\n\nIt is the target both references point at.\n",
		"sibling: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	// Phase 1 — the branch is unpushed, so the references resolve here
	// and on no other machine.
	got := verdictFor(t, root, "G-0100")
	assertVerdict(t, "unpushed", got, "cross-branch-local-only", check.SeverityError)
	if !strings.Contains(got.refsMessage, "refs/heads/sibling") {
		t.Errorf("unpushed: refs-resolve message = %q, want it to name the unpublished ref", got.refsMessage)
	}

	// Phase 2 — publishing the branch de-escalates it. Nothing tracks
	// the earlier verdict; the next run simply reads a ref that now
	// exists under refs/remotes/.
	if err := osExec(t, root, "git", "push", "-q", "origin", "sibling"); err != nil {
		t.Fatalf("git push sibling: %v", err)
	}
	assertVerdict(t, "published", verdictFor(t, root, "G-0100"), "cross-branch-pending", check.SeverityWarning)

	// Phase 3 — deleting the remote branch re-escalates it. The local
	// branch still carries the id, so this is local-only again rather
	// than unresolved, which is the distinction the classification
	// exists to draw.
	if err := osExec(t, root, "git", "push", "-q", "origin", "--delete", "sibling"); err != nil {
		t.Fatalf("git push --delete sibling: %v", err)
	}
	assertVerdict(t, "unpublished", verdictFor(t, root, "G-0100"), "cross-branch-local-only", check.SeverityError)

	// Phase 4 — with the branch gone entirely the id is reachable from
	// no ref at all, and falls back to unresolved.
	if err := osExec(t, root, "git", "branch", "-q", "-D", "sibling"); err != nil {
		t.Fatalf("git branch -D sibling: %v", err)
	}
	assertVerdict(t, "deleted", verdictFor(t, root, "G-0100"), "unresolved", check.SeverityError)
}

// TestCrossBranchLocalOnly_BlocksTheCheckButNotAuthoring_ADR0041 drives
// the split ADR-0041 settles through the CLI, where an operator meets
// it: with a reference resolvable only from an unpushed branch, `aiwf
// add` writes the entity and `aiwf check` refuses to pass — so the tree
// is stopped at the push boundary and nowhere earlier.
//
// It runs through cli.Execute rather than the verb package because the
// exit status is the thing the pre-push hook reads, and a severity that
// never reaches it would block nothing.
func TestCrossBranchLocalOnly_BlocksTheCheckButNotAuthoring_ADR0041(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	bare := filepath.Join(t.TempDir(), "origin.git")
	if err := osExec(t, root, "git", "init", "--bare", "-q", "-b", "main", bare); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")
	if err := osExec(t, root, "git", "remote", "add", "origin", bare); err != nil {
		t.Fatalf("git remote add: %v", err)
	}
	if err := osExec(t, root, "git", "push", "-q", "-u", "origin", "main"); err != nil {
		t.Fatalf("git push main: %v", err)
	}

	// G-0100 is minted on a branch that is never pushed.
	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	writeAndCommit(t, root, "work/gaps/G-0100-minted-on-the-sibling-branch.md",
		"---\nid: G-0100\ntitle: Minted on the sibling branch\nstatus: open\n---\n"+
			"## What's missing\n\nThis entity exists on one branch.\n\n"+
			"## Why it matters\n\nIt is the target the citation points at.\n",
		"sibling: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	bodyPath := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyPath, []byte(
		"## What's missing\n\nDepends on G-0100, filed on a branch nobody has pushed.\n\n"+
			"## Why it matters\n\nAuthoring must not be refused for it.\n"), 0o644); err != nil {
		t.Fatalf("write body: %v", err)
	}

	if rc := cli.Execute([]string{
		"add", "gap", "--root=" + root, "--title", "Cites the sibling gap", "--body-file", bodyPath,
	}); rc != cliutil.ExitOK {
		t.Fatalf("aiwf add rc = %d, want ExitOK (%d) — the reference is well-formed and its target exists, so authoring must not be refused", rc, cliutil.ExitOK)
	}

	if rc := cli.Execute([]string{"check", "--root=" + root}); rc != cliutil.ExitFindings {
		t.Errorf("aiwf check rc = %d, want ExitFindings (%d) — the pre-push hook reads this status, and the tree references an id no other machine can resolve", rc, cliutil.ExitFindings)
	}

	// Publishing the branch is the remedy the finding names, and it
	// clears the check without any edit to the tree.
	if err := osExec(t, root, "git", "push", "-q", "origin", "sibling"); err != nil {
		t.Fatalf("git push sibling: %v", err)
	}
	if rc := cli.Execute([]string{"check", "--root=" + root}); rc != cliutil.ExitOK {
		t.Errorf("aiwf check rc = %d after publishing the branch, want ExitOK (%d) — same bytes, and the remedy the hint names must clear it", rc, cliutil.ExitOK)
	}
}
