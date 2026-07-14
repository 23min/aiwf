package contract_test

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity once at startup so tests can run with
// t.Parallel() without t.Setenv panics. The identity values are
// immutable for the test binary's lifetime; once-setup is correct.
//
// Serial skip-list (per-package convention, CLAUDE.md "Test
// discipline"): these tests omit t.Parallel() because they call
// t.Setenv, which panics under t.Parallel.
//   - TestRunBind_FallsBackWhenOutputFormatCarriesNone
//   - TestRunUnbind_FallsBackWhenOutputFormatCarriesNone
//   - TestRunRecipeInstall_FallsBackWhenOutputFormatCarriesNone
//   - TestRunRecipeRemove_FallsBackWhenOutputFormatCarriesNone
//   - TestRunBind_ResolveActorFailure
//   - TestRunUnbind_ResolveActorFailure
//   - TestRunRecipeInstall_ResolveActorFailure
//   - TestRunRecipeRemove_ResolveActorFailure
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
