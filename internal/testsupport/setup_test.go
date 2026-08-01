package testsupport

import (
	"os"
	"testing"
)

// TestMain satisfies the internal/* test-discipline convention (every
// test-bearing package carries a setup_test.go with a TestMain — see
// CLAUDE.md *Test discipline*, M-0093/AC-2).
//
// It hardens its own git test env even though nothing here runs git:
// execfile_test.go execs the stand-ins it writes, and the chokepoint
// (policies.PolicyGitTestEnvHardened) keys on spawning a subprocess at
// all rather than on spawning git specifically. TestHardenGitTestEnv
// seeds the locator vars it asserts on itself, so it is unaffected by
// the environment this call leaves behind.
//
// Git identity vars are not seeded: no test here commits.
//
// Serial tests:
//   - TestHardenGitTestEnv (gitenv_test.go) mutates process env
//     (t.Setenv + raw os.Setenv of GIT_CONFIG_*).
//   - TestWriteExecutable_TakesForkLock (execfile_test.go) holds
//     syscall.ForkLock for writing, which blocks every fork in the
//     process for as long as it is held.
//
// The other execfile_test.go tests are parallel.
func TestMain(m *testing.M) {
	HardenGitTestEnv()
	os.Exit(m.Run())
}
