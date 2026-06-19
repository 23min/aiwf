package policies

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the
// test binary, so once-setup is correct.
//
// Policies tests run against the live repo tree (memoized via
// sharedRepoTree, see shared_tree_test.go per M-0091/AC-4) and a
// handful of fixtures under testdata/. Some tests shell out to git
// (e.g. log queries for the design-doc anchor and SHA-recording
// audits); a stable identity keeps those reproducible.
//
// Serial tests (must not call t.Parallel — they use t.Setenv, which
// panics under t.Parallel):
//   - TestPolicyBranchCoverageAudit_Env — sets AIWF_COVERAGE_PROFILE /
//     _BASE to drive the env-fed coverage-audit entry point.
//   - TestPolicyFiringFixturePresence_Env — sets AIWF_COVERAGE_PROFILE
//     to drive the env-fed firing-fixture-presence entry point.
//
// Every other Test* function reads-only against the shared *Tree (do
// not mutate) or uses t.TempDir for fixture work.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	// Harden the git test env: scrub ambient locator vars a parent git
	// hook passes down (GIT_DIR/GIT_INDEX_FILE/...) and disable git
	// auto-gc. Tests under this package shell out to `git init` in a
	// t.TempDir (e.g. M-0124's per-cell positive driver via
	// cellcoverage); without this, inherited locator vars steer those
	// into the parent repo's gitdir/index (G-0250) and background
	// auto-gc races fixtures under load (G-0251). Factored to
	// internal/testsupport and enforced for every exec-bearing
	// internal/* package by PolicyGitTestEnvHardened.
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
