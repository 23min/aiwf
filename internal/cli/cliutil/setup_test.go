package cliutil

import (
	"os"
	"testing"
)

// Serial tests (do NOT call t.Parallel):
//   - TestOutputFormat_EmitHelpers (outputformat_test.go) — redirects the
//     process-global os.Stdout/os.Stderr to capture envelope output.

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
