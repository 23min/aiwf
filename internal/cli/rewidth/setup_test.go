package rewidth_test

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// Serial tests (do NOT call t.Parallel):
//   - TestRun_NoOpJSON, TestRun_DryRunJSON (rewidth_error_paths_test.go) —
//     both capture os.Stdout via testutil.CaptureStdout, a process global.
//   - TestRewidth_TextDryRun_ExactOperationsListing, TestRewidth_TextApply_ExactSubjectLine,
//     TestRewidth_JSONApply_CarriesCommitSHA, TestRewidth_JSONDryRun_NoCommitSHA
//     (rewidth_envelope_pin_test.go) — same reason.

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
