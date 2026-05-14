package policies

import (
	"context"
	"sync"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// sharedRepoTree returns the live-repo planning tree loaded once
// per test binary. Consumers MUST NOT mutate the returned *Tree —
// it is shared across parallel tests.
//
// Composes with TestMain: env setup happens in TestMain; tree
// memoization is a separate sync.Once because not every policy
// test needs the tree, and the load cost (~1.4 MB walk) should
// not be paid by tests that don't read it.
//
// Repo-root resolution uses runtime.Caller (via repoRoot), which is
// resilient to working-directory changes and matches the rest of the
// policy tests' convention. The fallback resolver `repoRootFromTest`
// (walks up via os.Getwd) is still available for the few tests that
// prefer it; sharedRepoTree itself uses repoRoot for determinism.
//
// Per M-0091/AC-4: AC consumers updated to call sharedRepoTree are
// TestPolicy_ThisRepoTreeIsClean, TestPolicy_ThisRepoDriftCheckClean,
// loadM080Spec, loadADR0007, and TestAiwfxWrapEpic_AC4_RitualsRepoSHARecordedAtWrap.
var (
	sharedTreeOnce     sync.Once
	sharedTreeRoot     string
	sharedTree         *tree.Tree
	sharedTreeLoadErrs []tree.LoadError
	sharedTreeErr      error
)

func sharedRepoTree(t *testing.T) (string, *tree.Tree) { // do not mutate
	t.Helper()
	sharedTreeOnce.Do(func() {
		// Use repoRoot (runtime.Caller-based) for determinism. This
		// helper is package-local and only callable from inside a
		// running test, so the t-bound resolution is acceptable.
		sharedTreeRoot = repoRoot(t)
		ctx := context.Background()
		sharedTree, sharedTreeLoadErrs, sharedTreeErr = tree.Load(ctx, sharedTreeRoot)
	})
	if sharedTreeErr != nil {
		t.Fatalf("sharedRepoTree: %v", sharedTreeErr)
	}
	return sharedTreeRoot, sharedTree
}

// sharedRepoTreeLoadErrs returns the non-fatal per-file load errors
// emitted during the one-shot tree.Load. Tests that need to assert
// the live repo loads cleanly (no malformed frontmatter, etc.) check
// this slice for emptiness.
func sharedRepoTreeLoadErrs(t *testing.T) []tree.LoadError { // do not mutate
	t.Helper()
	sharedRepoTree(t)
	return sharedTreeLoadErrs
}
