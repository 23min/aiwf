package main

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// TestMain seeds the four GIT identity vars once at startup so test
// functions can call t.Parallel safely (os.Setenv not t.Setenv, which
// panics under parallel). See CLAUDE.md §"Test discipline". This
// package shells out to git (each scenario's own Setup) and to the
// aiwf binary under test, so HardenGitTestEnv also applies.
//
// Serial skip-list (CLAUDE.md §"Test discipline"): every runRun-driving
// test that reaches runRun's AIWF_LOG* os.Setenv call (past scenario
// and out-dir resolution, past the binary build) — TestRunRun_Succeeds,
// TestRunRun_LockKillScenario_BuildsLockHolderAndRuns,
// TestRunRun_ScenarioAll_RunsWholeCatalogIntoOneReport,
// TestRunRun_PrintsPreservedDirOnAFailingAttempt (run_test.go), and
// TestRun_RunCommand_Succeeds (main_test.go) — omits t.Parallel(). Each
// mutates the process-wide AIWF_LOG/AIWF_LOG_FORMAT/AIWF_LOG_FILE env
// (M-0249/AC-2) so every scenario subprocess it launches inherits
// diagnostic logging; two such tests running concurrently could
// otherwise have a later test's AIWF_LOG_FILE Setenv land mid-flight
// and misdirect an earlier test's still-running attempt into the
// wrong diagnostic log.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
