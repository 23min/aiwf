package cliutil

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// Serial tests (do NOT call t.Parallel):
//   - TestOutputFormat_EmitHelpers (outputformat_test.go) — redirects the
//     process-global os.Stdout/os.Stderr to capture envelope output.
//   - TestParseTestsFlag (verbhelpers_test.go) — its "malformed" subtest
//     redirects the process-global os.Stderr to /dev/null; parallel would
//     race any concurrent reader (e.g. RunStatuslineRemove's Fprintf paths).

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
