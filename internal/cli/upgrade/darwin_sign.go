package upgrade

import (
	"fmt"
	"os/exec"
	"runtime"
)

// signDarwinBinary ad-hoc-signs path with codesign(1) on macOS. No-op
// on non-Darwin platforms. The ad-hoc signature (`--sign -`) is what
// satisfies macOS Sonoma 14.8.x's syspolicyd, which segfaults parsing
// Mach-O code-signing data from unsigned binaries — same root cause
// as G-0128/G-0133 for test binaries (see scripts/sign-and-run.sh for
// the parallel wrapper used during go test).
//
// Returns the codesign error verbatim on Darwin if signing fails. The
// caller decides whether to fail or warn-and-continue. In the upgrade
// flow we warn-and-continue: the binary is already installed and
// runs; signing is a syspolicyd-crash mitigation that the operator
// can re-attempt manually if it fails the first time.
//
// G-0134.
func signDarwinBinary(path string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	return runCodesign(path)
}

// runCodesign shells out to codesign(1) for ad-hoc signing. Extracted
// from signDarwinBinary so the GOOS gate has a unit-testable shape
// (Linux returns nil at the gate; the Darwin-only branch is isolated
// in this function).
func runCodesign(path string) error {
	//coverage:ignore Darwin-only invocation; codesign(1) is macOS-only
	// and not exercisable from Linux CI. Mirrors the //coverage:ignore
	// exception class used for the TOCTOU branch in
	// internal/cli/doctor/binary_staleness.go (G-0176).
	cmd := exec.Command("codesign", "--sign", "-", "--force", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("codesign --sign - --force %s: %w: %s",
			path, err, string(out))
	}
	return nil
}
