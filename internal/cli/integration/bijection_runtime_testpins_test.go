//go:build testpins

package integration

import (
	"sort"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch/branchtest"
)

// TestZZZ_M0162_AC4_BijectionInvariant4_Runtime closes the
// runtime side of the M-0162/AC-4 bijection contract by enforcing
// invariant 4 — "no test function pins 2+ cells" — at the
// branchtest.Pins() registry granularity per the AC body line
// 245. The check fires AFTER the bulk of parallel scenario
// subtests have completed (TestZZZ_ prefix → lex-last serial
// wave; the bijection scan is safe because the registry is
// mutex-guarded per AC-2 and no Pin call is in flight from a
// non-parallel test once this lex-last test starts).
//
// Why this exists separately from the static AC-4 check in
// internal/policies/m0162_ac4_bijection_test.go: static AST
// cannot resolve t.Name() (the load-bearing per-call-site
// identifier at runtime). The reviewer of AC-4's initial closure
// (S2 finding) called out the silent-defer of invariant 4. This
// file delivers it.
//
// Sequencing note: this test does NOT call t.Parallel. By Go's
// test scheduling, t.Parallel-tagged subtests pause at their
// t.Parallel call and resume in the parallel wave AFTER all
// serial tests complete. A serial test named TestZZZ_* runs in
// the serial wave too, but late within it (lex-late). At the
// moment this body executes, every Pin call from Scenario
// matrices has already been recorded (RunScenarios's parent
// functions ran serially, dispatched subtests via t.Run+Parallel,
// and returned; subtests have queued their Pin calls — they
// don't fire until the parallel wave). So this lex-late serial
// test sees pins from RunScenarios-using parents but NOT from
// their not-yet-resumed subtests.
//
// To compensate, the registry is read AGAIN inside the TestMain
// epilogue at setup_test.go (the BijectionPostHook pattern; see
// that file). The TestZZZ_ test below is the eager defense; the
// TestMain hook is the comprehensive defense after the parallel
// wave drains.
func TestZZZ_M0162_AC4_BijectionInvariant4_Runtime(t *testing.T) {
	// No t.Parallel — runs in serial wave.
	violations := checkBijectionInvariant4(branchtest.Pins())
	if len(violations) > 0 {
		t.Errorf("M-0162/AC-4 invariant 4 (eager pass): %d test(s) pin 2+ cells\n%s", len(violations), formatInvariant4(violations))
	}
}

// invariant4Violation is a (testName, cells) entry — one test
// function (by t.Name()) pinning 2+ distinct cells.
type invariant4Violation struct {
	TestName string
	Cells    []string
}

// checkBijectionInvariant4 inverts the cell→[]testName map into
// testName→[]cell and reports any test pinning 2+ cells.
func checkBijectionInvariant4(pins map[string][]string) []invariant4Violation {
	testCells := make(map[string]map[string]bool)
	for cellID, tests := range pins {
		for _, name := range tests {
			if testCells[name] == nil {
				testCells[name] = make(map[string]bool)
			}
			testCells[name][cellID] = true
		}
	}
	var out []invariant4Violation
	for name, cellSet := range testCells {
		if len(cellSet) < 2 {
			continue
		}
		cells := make([]string, 0, len(cellSet))
		for c := range cellSet {
			cells = append(cells, c)
		}
		sort.Strings(cells)
		out = append(out, invariant4Violation{TestName: name, Cells: cells})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].TestName < out[j].TestName
	})
	return out
}

func formatInvariant4(v []invariant4Violation) string {
	var out string
	for _, vv := range v {
		out += "  " + vv.TestName + " pins:\n"
		for _, c := range vv.Cells {
			out += "    " + c + "\n"
		}
	}
	return out
}
