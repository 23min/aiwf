package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0162_AC4_M0158AC5KeywordSetFileAbsent pins the
// M-0162/AC-4 retirement claim: the M-0158/AC-5 keyword-set
// meta-coverage at `internal/policies/m0158_ac5_meta_coverage_test.go`
// is deleted in the same commit as the bijection meta-test
// (TestM0162_AC4_Bijection) lands. Catches a confused merge
// that re-introduces the file.
//
// Per AC-4 body §"M-0158/AC-5 retirement", with reviewer S4 honesty
// correction:
//
//   - The keyword-set ≥1-match invariant is subsumed by AC-4
//     invariant 1 (every cell has ≥1 Pin call site) for the
//     AC-3-era cell-expansion surface, tightened via invariant 3
//     (no cell has 2+ Pin call sites at distinct source positions).
//   - On the M-0158/M-0161-era named cells: the keyword-set used
//     fuzzy substring matching against test function names; AC-4
//     uses literal cell-ID matching. For named cells whose
//     canonical tests don't carry the literal `branch-cell-X`
//     string, the keyword-set covered them via fuzz match while
//     AC-4 allowlists them. This is an honest carve-out, not a
//     strict improvement on that surface.
//   - M-0158/AC-5's promoted-met status remains valid because the
//     bijection meta-test STRENGTHENS coverage on the AC-3-era
//     expansion surface (literal Pin matching, dynamic-prefix
//     materialization, invariant 4 at runtime) while preserving
//     equivalent-or-allowlist coverage on the named-cell surface.
//     Net: stronger overall, weaker in one localized way that the
//     allowlist documents.
//
// Sabotage-verifiable: re-add the file and this test fires.
func TestM0162_AC4_M0158AC5KeywordSetFileAbsent(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	path := filepath.Join(root, "internal", "policies", "m0158_ac5_meta_coverage_test.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("M-0162/AC-4: %s exists; the keyword-set meta-coverage was retired in favor of TestM0162_AC4_Bijection. See M-0162/AC-4 body §\"M-0158/AC-5 retirement\". If re-added intentionally, document why in the AC-4 reviewer notes.", path)
	}
}

// TestM0162_AC4_CITestpinsTagWired pins M-0162/AC-4 body
// §"Mechanical assertions" item 5: the CI workflow at
// .github/workflows/go.yml carries `-tags testpins` on its
// test step so the bijection sabotage subtests (which require
// the tag to populate branchtest.Pins()) actually run.
//
// Without the tag, the sabotage subtests silently skip — losing
// the AC-4 sabotage discrimination guarantee. The structural
// check pins the wiring.
//
// Sabotage-verifiable: drop `-tags testpins` from the workflow
// and this test fires.
func TestM0162_AC4_CITestpinsTagWired(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	path := filepath.Join(root, ".github", "workflows", "go.yml")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	// The test step is identifiable by `go test`. Required substring
	// guard: `-tags testpins` must appear on the same line as `go test`.
	src := string(contents)
	var testLines []string
	for _, line := range strings.Split(src, "\n") {
		if strings.Contains(line, "go test") && strings.Contains(line, "-coverprofile") {
			testLines = append(testLines, line)
		}
	}
	if len(testLines) == 0 {
		t.Fatalf("M-0162/AC-4: no `go test ... -coverprofile` step found in %s — CI workflow shape changed?", path)
	}
	for _, line := range testLines {
		if !strings.Contains(line, "-tags testpins") {
			t.Errorf("M-0162/AC-4: CI test step missing `-tags testpins`\n  line: %s\n  see AC-4 body §\"Mechanical assertions\" item 5", line)
		}
	}
}

// TestM0162_AC4_MetaCellsRegistered pins the 3 meta-cells per
// M-0162/AC-4 body §"Meta-cells registered." The meta-cells
// document the bijection's enforcement chokepoints in the
// catalog itself; each is pinned by a sabotage subtest in
// m0162_ac4_sabotage_testpins_test.go.
//
// Sabotage-verifiable: remove an entry from ac4MetaCells() and
// this test fires naming the missing cell.
func TestM0162_AC4_MetaCellsRegistered(t *testing.T) {
	t.Parallel()

	required := []string{
		"branch-cell-meta-bijection-enforced",
		"branch-cell-meta-pin-orphan-detected",
		"branch-cell-meta-cell-orphan-detected",
	}
	present := make(map[string]bool)
	for _, r := range branch.Rules() {
		present[r.ID] = true
	}
	for _, id := range required {
		if !present[id] {
			t.Errorf("M-0162/AC-4: meta-cell %q ABSENT from branch.Rules() — see rules_m0162_ac4.go::ac4MetaCells()", id)
		}
	}
}
