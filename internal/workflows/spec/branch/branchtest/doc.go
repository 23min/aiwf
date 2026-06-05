// Package branchtest is the test-only Pin registry used by the
// M-0162 cell-expansion E2E tests and the bijection meta-test.
// All exported symbols live under //go:build testpins so they are
// excluded from production builds; this file is the no-tag stub
// that lets `go test ./...` (without -tags testpins) walk the
// package as empty rather than failing on "build constraints
// exclude all Go files."
//
// See pin.go for the build-tagged API and usage convention.
package branchtest
