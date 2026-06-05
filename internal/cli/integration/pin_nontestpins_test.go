//go:build !testpins

package integration

// pinCell is a no-op when the testpins tag is absent. The
// build-tagged sibling at pin_testpins_test.go forwards to
// branchtest.Pin so the registry only accumulates when the
// bijection meta-test (also testpins-tagged) is going to read
// it. Production builds and default `go test ./...` invocations
// take this path so the integration package does not depend on
// the branchtest sub-package's symbols.
func pinCell(cellID, testName string) {
	_ = cellID
	_ = testName
}
