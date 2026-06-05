//go:build testpins

// Package branchtest provides the Pin registry used by AC-3's
// cell-expansion E2E tests and AC-4's bijection meta-test. The
// package and its symbols are compiled only when -tags testpins
// is set; production `go build` omits them entirely.
//
// Usage:
//
//	func TestX_AC3_Foo(t *testing.T) {
//	    branchtest.Pin("branch-cell-foo", t.Name())
//	    ...
//	}
//
// The bijection meta-test at
// internal/policies/branch_cell_bijection_test.go inspects the
// registry after every E2E test in the test-pins build completes.
//
// The build tag is the load-bearing exclusion mechanism — the
// `branchtest` sub-package marker is informational. CI runs and
// the Makefile's `test-pins` target carry `-tags testpins`; bare
// `go test ./...` without the tag silently skips the pin-calling
// tests and the bijection meta-test.
package branchtest

import "sync"

var (
	mu       sync.Mutex
	registry = map[string][]string{}
)

// Pin records that a test function exercises a specific
// branch.Rules() cell. Calls accumulate into a process-local
// registry inspected by the bijection meta-test at AC-4.
//
// Calls from tests inside `t.Run` should pass t.Name() so the
// subtest's full name (TestX/sub-row) appears in the registry.
//
// Safe to call concurrently from `t.Parallel()` subtests — guarded
// by a sync.Mutex.
func Pin(cellID, testName string) {
	mu.Lock()
	defer mu.Unlock()
	registry[cellID] = append(registry[cellID], testName)
}

// Pins returns a snapshot of accumulated pins. Used by the
// bijection meta-test at internal/policies/. The returned map is
// a deep copy — mutations by the caller do not affect the
// underlying registry.
//
// Empty maps are returned non-nil so test code can range without
// nil-check.
func Pins() map[string][]string {
	mu.Lock()
	defer mu.Unlock()
	out := make(map[string][]string, len(registry))
	for k, v := range registry {
		dup := make([]string, len(v))
		copy(dup, v)
		out[k] = dup
	}
	return out
}
