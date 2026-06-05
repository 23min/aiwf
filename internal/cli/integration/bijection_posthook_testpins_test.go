//go:build testpins

package integration

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/workflows/spec/branch/branchtest"
)

// bijectionPostHook reads branchtest.Pins() and checks the
// runtime side of M-0162/AC-4's bijection contract. Wired into
// setup_test.go's TestMain epilogue under -tags testpins.
//
// At this point m.Run() has returned, meaning every test in the
// package (serial + parallel waves) has finished. Every Pin call
// that will ever record into the registry has done so. This is
// the load-bearing read site for invariant 4 per AC-2's contract
// ("read the accumulator after all tests have completed").
//
// Returns a non-empty failure message and a non-zero exit
// override when violations are found.
func bijectionPostHook() (failure string, overrideExit int) {
	pins := branchtest.Pins()
	violations := checkBijectionInvariant4(pins)
	if len(violations) == 0 {
		return "", 0
	}
	var b strings.Builder
	fmt.Fprintf(&b, "M-0162/AC-4 bijection post-hook: invariant 4 — %d test(s) pin 2+ cells\n", len(violations))
	b.WriteString(formatInvariant4(violations))
	return b.String(), 1
}
