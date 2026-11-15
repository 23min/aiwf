package check

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/gitops"
)

// placesNothing is a branch oracle that places no commit. Walking with it
// reports every non-fast-forward candidate, which shows a fixture really
// produced the candidate a test then expects the real oracle to judge.
type placesNothing struct{}

func (placesNothing) FirstParentBranches(string) []string { return nil }
func (placesNothing) OracleErrors() []check.OracleErr     { return nil }
func (placesNothing) BranchOfSHA(string) string           { return "" }

// rewoundAICommit builds a repo whose epic branch carried one ai/ commit
// and was then rewound to main's fork point: a non-fast-forward reflog
// move whose previous tip is that commit. place then runs on main, given
// the commit's SHA, to leave the commit wherever the test needs it.
func rewoundAICommit(t *testing.T, place func(root, aiSHA string)) (root, aiSHA string) {
	t.Helper()
	root = t.TempDir()
	if err := gitops.Init(context.Background(), root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	gitRun(t, root, "commit", "-q", "--allow-empty", "-m", "baseline")
	base := gitOutput(t, root, "rev-parse", "HEAD")
	gitRun(t, root, "checkout", "-q", "-b", "epic/E-0001-engine")
	gitRun(t, root, "commit", "-q", "--allow-empty", "-m", "ai work\n\naiwf-entity: E-0001\naiwf-actor: ai/claude\n")
	aiSHA = gitOutput(t, root, "rev-parse", "HEAD")
	gitRun(t, root, "checkout", "-q", "main")
	gitRun(t, root, "branch", "-f", "epic/E-0001-engine", base)
	place(root, aiSHA)
	return root, aiSHA
}

// walkOrphans runs the orphan walk over root with the real
// isolation-escape oracle, or with placesNothing when withIndex is false.
func walkOrphans(t *testing.T, root string, withIndex bool) []check.OrphanedAICommit {
	t.Helper()
	ctx := context.Background()
	dag := mustDAG(ctx, root)
	var oracle check.BranchOracle = placesNothing{}
	if withIndex {
		o, err := newGitBranchOracle(ctx, root, dag, "main")
		if err != nil {
			t.Fatalf("newGitBranchOracle: %v", err)
		}
		oracle = o
	}
	return check.WalkOrphanedAICommits(ctx, root, dag, "main", oracle)
}

func orphanReported(orphans []check.OrphanedAICommit, sha string) bool {
	for _, o := range orphans {
		if o.SHA == sha {
			return true
		}
	}
	return false
}

// TestWalkOrphanedAICommits_CommitTheEscapeIndexPlacesIsNotOrphaned: a
// rewound ai/ commit that an indexed branch holds on its first-parent
// line is not orphaned, since isolation-escape can still place it there.
func TestWalkOrphanedAICommits_CommitTheEscapeIndexPlacesIsNotOrphaned(t *testing.T) {
	t.Parallel()
	cases := map[string]func(root, aiSHA string){
		"its own branch moved back onto it and past it": func(root, aiSHA string) {
			gitRun(t, root, "checkout", "-q", "epic/E-0001-engine")
			gitRun(t, root, "reset", "-q", "--hard", aiSHA)
			gitRun(t, root, "commit", "-q", "--allow-empty", "-m", "later work")
			gitRun(t, root, "checkout", "-q", "main")
		},
		"trunk took it by fast-forward": func(root, aiSHA string) {
			gitRun(t, root, "merge", "-q", "--ff-only", aiSHA)
		},
		"another ritual branch holds it": func(root, aiSHA string) {
			gitRun(t, root, "branch", "epic/E-0002-other", aiSHA)
		},
	}
	for name, place := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root, aiSHA := rewoundAICommit(t, place)
			if !orphanReported(walkOrphans(t, root, false), aiSHA) {
				t.Fatalf("walking with an index that places nothing did not report %s: the fixture or the walk lost the candidate", aiSHA)
			}
			if got := walkOrphans(t, root, true); orphanReported(got, aiSHA) {
				t.Errorf("commit %s is on an indexed branch's first-parent line yet reported orphaned: %+v", aiSHA, got)
			}
		})
	}
}

// TestWalkOrphanedAICommits_CommitOnlyAMergeHoldsIsOrphaned: a rewound
// ai/ commit that trunk holds only as a merge's second parent is
// orphaned. isolation-escape places commits by first-parent line, so it
// cannot place this one.
func TestWalkOrphanedAICommits_CommitOnlyAMergeHoldsIsOrphaned(t *testing.T) {
	t.Parallel()
	root, aiSHA := rewoundAICommit(t, func(root, aiSHA string) {
		gitRun(t, root, "merge", "-q", "--no-ff", "-m", "merge epic", aiSHA)
	})
	gitRun(t, root, "merge-base", "--is-ancestor", aiSHA, "main")
	if got := walkOrphans(t, root, true); !orphanReported(got, aiSHA) {
		t.Errorf("commit %s is reachable from main only through a merge, yet not reported: %+v", aiSHA, got)
	}
}
