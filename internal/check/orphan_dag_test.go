package check

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestParseCommitDAG_ParentMap asserts the `git rev-list --parents`
// output ("<commit> <parent>..." per line) parses into the SHA→parents
// map.
func TestParseCommitDAG_ParentMap(t *testing.T) {
	t.Parallel()
	// A linear chain c <- b <- a (a is root), plus a merge m with two
	// parents, mirroring `git rev-list --parents` shape (root has no
	// parents listed).
	out := "m b d\nc b\nb a\nd a\na\n"
	dag := parseCommitDAG(out)
	cases := map[string][]string{
		"m": {"b", "d"},
		"c": {"b"},
		"b": {"a"},
		"d": {"a"},
		"a": nil,
	}
	for sha, want := range cases {
		got := dag.parents[sha]
		if len(got) != len(want) {
			t.Errorf("parents[%q] = %v, want %v", sha, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("parents[%q][%d] = %q, want %q", sha, i, got[i], want[i])
			}
		}
	}
}

// TestCommitDAG_IsAncestor pins the in-memory ancestry against
// `git merge-base --is-ancestor` semantics: reflexive, follows parent
// edges transitively, and is false across diverged tips.
func TestCommitDAG_IsAncestor(t *testing.T) {
	t.Parallel()
	// Graph:  a <- b <- c   (main line)
	//          \
	//           <- d        (a fork off a, diverged from b/c)
	dag := parseCommitDAG("c b\nb a\nd a\na\n")
	tests := []struct {
		old, newer string
		want       bool
	}{
		{"a", "c", true},  // transitive ancestor
		{"b", "c", true},  // direct parent
		{"a", "b", true},  // direct parent
		{"c", "c", true},  // reflexive
		{"c", "a", false}, // descendant is not an ancestor
		{"b", "d", false}, // diverged fork: b not reachable from d
		{"c", "d", false}, // diverged fork
		{"a", "d", true},  // common root is an ancestor of the fork
		{"z", "c", false}, // unknown sha
	}
	for _, tc := range tests {
		if got := dag.isAncestor(tc.old, tc.newer); got != tc.want {
			t.Errorf("isAncestor(%q, %q) = %v, want %v", tc.old, tc.newer, got, tc.want)
		}
	}
}

// TestCommitDAG_IsAncestor_DiamondRevisits exercises the DFS's
// already-seen short-circuit: a convergence node reachable by two
// paths lands on the stack twice (the second push happens while it is
// still unseen), so it is popped once-marked and once-already-seen.
// The parents order (x before b) forces b — which also has x as a
// parent — to re-push the still-unseen x. A query for an absent sha
// forces the full traversal that hits the revisit.
func TestCommitDAG_IsAncestor_DiamondRevisits(t *testing.T) {
	t.Parallel()
	// a -> {x, b}; b -> {x}; x is the shared root (the diamond's tail).
	dag := parseCommitDAG("a x b\nb x\nx\n")
	if dag.isAncestor("zzz", "a") {
		t.Error("isAncestor(absent, a) = true, want false")
	}
	// And the real ancestry answers stay correct across the diamond.
	for _, tc := range []struct {
		old, newer string
		want       bool
	}{
		{"x", "a", true}, // shared root reachable both ways
		{"b", "a", true},
		{"x", "b", true},
		{"a", "x", false},
	} {
		if got := dag.isAncestor(tc.old, tc.newer); got != tc.want {
			t.Errorf("isAncestor(%q, %q) = %v, want %v", tc.old, tc.newer, got, tc.want)
		}
	}
}

// TestCommitDAG_FirstParentChain pins CommitDAG.FirstParentChain (M-0216
// AC-6) against `git rev-list --first-parent <tip>` semantics: it follows
// parents[0] to a root, returns nil for an empty tip, returns just the tip
// for an absent/root commit, and terminates on a (defensive) cycle via the
// seen-guard.
func TestCommitDAG_FirstParentChain(t *testing.T) {
	t.Parallel()
	// First-parent chain c -> b -> a; b also has a second parent (d) that
	// the first-parent walk must NOT follow.
	dag := parseCommitDAG("c b e\nb a d\na\nd\ne\n")
	eq := func(got, want []string) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range want {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	}
	if got := dag.FirstParentChain(""); got != nil {
		t.Errorf("FirstParentChain(\"\") = %v, want nil", got)
	}
	if got := dag.FirstParentChain("c"); !eq(got, []string{"c", "b", "a"}) {
		t.Errorf("FirstParentChain(c) = %v, want [c b a] (first-parent only)", got)
	}
	if got := dag.FirstParentChain("a"); !eq(got, []string{"a"}) {
		t.Errorf("FirstParentChain(a) = %v, want [a] (root)", got)
	}
	if got := dag.FirstParentChain("zzz"); !eq(got, []string{"zzz"}) {
		t.Errorf("FirstParentChain(zzz) = %v, want [zzz] (absent tip)", got)
	}
	// Defensive cycle a -> a: the seen-guard must terminate.
	cyc := parseCommitDAG("a a\n")
	if got := cyc.FirstParentChain("a"); !eq(got, []string{"a"}) {
		t.Errorf("FirstParentChain(a) on cyclic dag = %v, want [a] (seen-guard terminates)", got)
	}
}

// TestWalkOrphanedAICommits_DAGDetectsForcePushedOrphan is the seam test
// (M-0216 AC-1): it exercises WalkOrphanedAICommits end-to-end against a
// real git repo where an ai/ commit was force-pushed away (a
// non-fast-forward reflog move), proving the in-memory-DAG ancestry path
// detects the orphan exactly as the old per-pair `git merge-base
// --is-ancestor` did. The pre-existing reflog_walk_test.go cases only
// cover the pure RunOrphanedAICommits finding logic, not this git walk.
func TestWalkOrphanedAICommits_DAGDetectsForcePushedOrphan(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	base := r.gitCommit("base")

	// A ritual milestone branch; on it, an ai/ commit (the orphan target).
	r.run("git", "checkout", "-q", "-b", "milestone/M-0001-fixture")
	aiSHA := r.commitEntityWithTrailers("M-0001", entity.KindMilestone, "in_progress", "ai work", map[string]string{
		"aiwf-actor":  "ai/claude",
		"aiwf-entity": "M-0001",
	})

	// A sibling commit off base — NOT a descendant of the ai commit.
	r.run("git", "checkout", "-q", "-B", "sidebranch", base)
	sibling := r.gitCommit("sibling off base")

	// Force the milestone branch to the sibling: a non-fast-forward move,
	// so the ai commit is now orphaned in the reflog.
	r.run("git", "checkout", "-q", "milestone/M-0001-fixture")
	r.run("git", "reset", "--hard", sibling)

	dag, _ := BuildCommitDAG(context.Background(), r.root)
	orphans := WalkOrphanedAICommits(context.Background(), r.root, dag)
	var found bool
	for _, o := range orphans {
		if o.SHA == aiSHA && o.EntityID == "M-0001" && o.Actor == "ai/claude" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected orphan %s (ai/claude, M-0001) detected via DAG; got %+v", aiSHA, orphans)
	}
}

// TestIsAncestorViaGit_Fallback covers the merge-base fallback (M-0216
// AC-1) used when the in-memory DAG is unavailable. It must give the same
// ancestry answers as the DAG path, on a real repo.
func TestIsAncestorViaGit_Fallback(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	a := r.gitCommit("a")
	b := r.gitCommit("b") // child of a
	r.run("git", "checkout", "-q", "-B", "fork", a)
	d := r.gitCommit("d") // child of a, diverged from b
	ctx := context.Background()
	cases := []struct {
		old, newer string
		want       bool
	}{
		{a, b, true},  // direct ancestor
		{b, a, false}, // descendant is not an ancestor
		{a, a, true},  // reflexive
		{b, d, false}, // diverged fork
		{a, d, true},  // common root
	}
	for _, tc := range cases {
		if got := isAncestorViaGit(ctx, r.root, tc.old, tc.newer); got != tc.want {
			t.Errorf("isAncestorViaGit(%s, %s) = %v, want %v", tc.old[:7], tc.newer[:7], got, tc.want)
		}
	}
}
