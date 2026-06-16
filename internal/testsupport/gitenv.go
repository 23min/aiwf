// Package testsupport holds helpers shared across the module's test
// suites. It carries no production call sites — only *_test.go files
// import it — so it stays out of the production binary's build graph.
package testsupport

import (
	"os"
	"strconv"
)

// gitLocatorEnvVars are the git "locator" environment variables that a
// parent git process — most importantly a git hook (pre-commit,
// pre-push) — exports to the processes it spawns. They override
// cwd-based repository discovery, so a test that shells out to git in a
// t.TempDir() silently operates against the *parent* repo's
// gitdir/index/object-store instead of its isolated fixture whenever
// any of these is set in the ambient environment. Under parallel test
// execution that means many fixtures racing on one shared index /
// object DB / lockfile — the G-0250 flake ("invalid object / Error
// building trees", "index.lock exists", "directory not empty").
var gitLocatorEnvVars = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_COMMON_DIR",
	"GIT_INDEX_FILE",
	"GIT_OBJECT_DIRECTORY",
}

// gitTestConfig is forced for every git invocation in a test process,
// injected via the GIT_CONFIG_COUNT / GIT_CONFIG_KEY_n /
// GIT_CONFIG_VALUE_n mechanism (git >= 2.31). Disabling auto-gc — and
// in particular its background detach — stops a detached `git gc`,
// which git spawns after commits once the loose-object threshold is
// crossed, from racing (a) the fixture's own subsequent git commands
// (it repacks/prunes a loose object another command expects, yielding
// "invalid object / Error building trees") and (b) t.TempDir's
// RemoveAll (it is still writing .git/objects, yielding "directory not
// empty"). Both surface only under concurrent load, which is why the
// flake "passes isolated". See G-0251.
var gitTestConfig = [][2]string{
	{"gc.auto", "0"},
	{"gc.autoDetach", "false"},
}

// HardenGitTestEnv prepares the process environment so test fixtures
// that shell out to git are insulated from the invoking context and
// from concurrency hazards. Call it once from a package's TestMain —
// alongside the GIT identity seeding — before m.Run(). It:
//
//   - Unsets the git locator vars (GIT_DIR/GIT_INDEX_FILE/...) a parent
//     git hook exports, which would otherwise steer fixture git
//     commands into the parent repo's gitdir/index (G-0250).
//   - Forces gc.auto=0 / gc.autoDetach=false for every child git via
//     GIT_CONFIG_COUNT, so background auto-gc cannot race fixture
//     commits or TempDir cleanup under load (G-0251).
//
// It is safe to call when the locator vars are already unset
// (os.Unsetenv on an absent key is a no-op) and safe to call from any
// package whether or not it shells out to git. os.Setenv/Unsetenv (not
// t.Setenv) because TestMain has no *testing.T and the changes must
// apply process-wide for the test binary's lifetime — the same reason
// the identity vars use os.Setenv (t.Setenv panics under t.Parallel).
// The enforcement chokepoint (policies.PolicyGitTestEnvHardened)
// requires this call in every exec-bearing internal/* package's
// TestMain.
func HardenGitTestEnv() {
	scrubGitLocatorEnv()
	disableGitAutoGC()
}

// scrubGitLocatorEnv unsets the git locator environment variables for
// the current process. See G-0250.
func scrubGitLocatorEnv() {
	for _, v := range gitLocatorEnvVars {
		// os.Unsetenv only errors on a malformed key (one containing
		// '='); these literals are well-formed, so the error is inert.
		_ = os.Unsetenv(v)
	}
}

// disableGitAutoGC forces the gitTestConfig pairs onto every child git
// via the GIT_CONFIG_COUNT env mechanism. See G-0251.
func disableGitAutoGC() {
	set := func(k, v string) {
		// os.Setenv only errors on a malformed key; these are well-formed.
		_ = os.Setenv(k, v)
	}
	set("GIT_CONFIG_COUNT", strconv.Itoa(len(gitTestConfig)))
	for i, kv := range gitTestConfig {
		set("GIT_CONFIG_KEY_"+strconv.Itoa(i), kv[0])
		set("GIT_CONFIG_VALUE_"+strconv.Itoa(i), kv[1])
	}
}
