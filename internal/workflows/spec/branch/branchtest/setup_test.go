//go:build testpins

package branchtest

import (
	"os"
	"testing"
)

// TestMain wires the uniform GIT-identity environment per the
// CLAUDE.md *Test discipline* convention (M-0091 / M-0093/AC-2).
// branchtest carries the testpins tag, so this file is only
// compiled-in for `go test -tags testpins`; the
// test-setup-presence policy enforces presence under the same
// build conditions the tests are exercised in.
//
// Serial skip-list: no tests in this package legitimately need
// serial execution. Both TestPin_API and TestPins_API call
// t.Parallel() and don't touch shared state beyond the
// package-local registry, which is mutex-guarded by design.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
