package check

import (
	"context"
	"os"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/testsupport"
)

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}

// mustHead runs check.WalkHeadCommits over a healthy fixture repo,
// asserting the Finding-1 fail-loud error is nil. These tests build
// readable repos, so a non-nil error is a fixture/regression bug; the
// error path itself is covered in internal/check
// (TestWalkHeadCommits_FailsLoudOnUnreadableHistory).
func mustHead(t *testing.T, ctx context.Context, root string) []check.HeadCommit {
	t.Helper()
	h, err := check.WalkHeadCommits(ctx, root)
	if err != nil {
		t.Fatalf("WalkHeadCommits over fixture %s: %v", root, err)
	}
	return h
}
