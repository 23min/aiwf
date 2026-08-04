package cli

import (
	"os"
	"testing"
)

// Serial skip-list — tests in this package that must NOT call t.Parallel,
// one rationale line each. Everything else here is parallel-by-default.
//
//   - TestExecute_Version, TestExecute_VersionVerb, TestExecute_Help —
//     swap os.Stdout to capture Execute's output (captureExecuteOutput).
//   - TestExecute_ParentCommandsRejectIncompleteInvocations — swaps both
//     os.Stdout and os.Stderr (captureExecuteStreams).

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
