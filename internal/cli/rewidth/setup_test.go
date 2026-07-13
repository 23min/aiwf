package rewidth_test

import (
	"os"
	"testing"
)

// Serial tests (do NOT call t.Parallel):
//   - TestRun_NoOpJSON, TestRun_DryRunJSON (rewidth_error_paths_test.go) —
//     both capture os.Stdout via testutil.CaptureStdout, a process global.

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
