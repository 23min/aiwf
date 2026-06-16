package testsupport

import (
	"os"
	"testing"
)

// TestMain satisfies the internal/* test-discipline convention (every
// test-bearing package carries a setup_test.go with a TestMain — see
// CLAUDE.md *Test discipline*, M-0093/AC-2). This package does not
// shell out to git, so it does not seed git identity vars or call
// HardenGitTestEnv — it is the home of that helper, not a consumer.
//
// Serial tests: TestHardenGitTestEnv (gitenv_test.go) mutates process
// env (t.Setenv + raw os.Setenv of GIT_CONFIG_*); it does not call
// t.Parallel and is the package's only test.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
