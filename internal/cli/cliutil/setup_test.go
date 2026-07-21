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
//   - TestTextIO_Wrappers (textio_test.go) — uses captureStdStreams.
//   - TestResolveLogger_StderrDestination_NeverClosesRealStderr
//     (resolvelogger_test.go) — uses captureStdStreams.
//   - TestAcquireRepoLock_JSONEnvelopeOnBusy, TestAcquireRepoLock_TextModeUnchanged
//     (lock_test.go) — use testutil.CaptureStdout / CaptureStderr.
//   - TestFinishVerbOutcome_DryRun_JSON, TestFinishVerbOutcome_DryRun_Text,
//     TestFinishVerbOutcome_MultiPlan_Apply, TestFinishVerbOutcome_ApplyError_MessageFormat,
//     TestFinishVerbOutcome_ApplySuccess_FindingsRenderInTextMode
//     (apply_outcome_test.go) — use captureStdStreams.

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
