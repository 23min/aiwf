//go:build testpins

package integration

import (
	"sort"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch/branchtest"
)

// TestZZZ_M0162_AC4_BijectionInvariant4_Runtime is the EAGER
// best-effort check for the M-0162/AC-4 bijection contract's
// invariant 4 — "no test function pins 2+ cells" — at the
// branchtest.Pins() registry granularity.
//
// Sequencing note (R3-T1 reviewer honesty correction): an earlier
// version of this docstring claimed lex-last ordering by virtue
// of the TestZZZ_ prefix. That premise is wrong. Go's test runner
// uses SOURCE-DECLARATION order for top-level tests, not
// alphabetical-by-name. This test runs whenever it appears in the
// build's file/decl order — not necessarily after other serial
// tests. Live sabotage during the AC-4 milestone audit confirmed
// this test did NOT catch a deliberate double-pin violation; the
// TestMain post-hook did.
//
// The load-bearing runtime defense for invariant 4 is the
// TestMain post-hook (bijection_posthook_testpins_test.go +
// setup_test.go's TestMain epilogue). The post-hook reads
// branchtest.Pins() AFTER m.Run() returns — at which point all
// serial AND parallel waves have completed and every Pin call has
// been recorded. This eager check exists as belt-and-braces for
// rapid-feedback: when violations exist at the moment this test
// runs, it surfaces them sooner than the end-of-suite hook.
//
// Why this exists separately from the static AC-4 check in
// internal/policies/m0162_ac4_bijection_test.go: static AST
// cannot resolve t.Name() (the load-bearing per-call-site
// identifier at runtime). The reviewer of AC-4's initial closure
// (S2 finding) called out the silent-defer of invariant 4. The
// runtime portion is what closes it; the post-hook is the
// comprehensive read site, this test is the eager peek.
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
