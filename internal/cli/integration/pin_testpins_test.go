//go:build testpins

package integration

import "github.com/23min/aiwf/internal/workflows/spec/branch/branchtest"

// pinCell forwards to branchtest.Pin under -tags testpins so the
// Scenarios framework accumulates per-subtest pins into the
// process-local registry for AC-4's bijection meta-test. Without
// the tag, the negated-tag stub at pin_nontestpins_test.go is a
// no-op so production builds and the default `go test ./...`
// invocation do not depend on the test-only branchtest package.
func pinCell(cellID, testName string) {
	branchtest.Pin(cellID, testName)
}
