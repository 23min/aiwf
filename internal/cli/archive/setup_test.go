package archive_test

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// Serial tests (do NOT call t.Parallel):
//   - TestArchive_TextDryRun_ExactMoveListing, TestArchive_TextApply_ExactSubjectLine,
//     TestArchive_JSONApply_CarriesCommitSHA (archive_envelope_pin_test.go) —
//     capture os.Stdout via testutil.CaptureStdout, a process global.

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
