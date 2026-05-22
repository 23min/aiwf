package check

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// White-box mechanical evidence for M-0130/AC-5: the three subcode
// hint entries exist in hintTable, and each subcode is named in the
// aiwf-check SKILL.md row-table.
//
// Per the A2 sequencing decision (chosen before AC-2 emits any code),
// hints + SKILL.md rows land first so PolicyFindingCodesHaveHints and
// PolicyFindingCodesAreDiscoverable stay green when AC-2/3/4's
// per-subcode predicates land their Code+Subcode literals. The
// existing kernel policies will fire on AC-2/3/4 if these rows go
// missing — but those are future-tense. The tests below are
// present-tense mechanical evidence that AC-5's deliverable exists.

// expectedFSMHistorySubcodes is the closed-set list M-0130's three
// per-subcode ACs will emit. Shared between the hint test and the
// SKILL.md test so adding a fourth subcode in the future requires
// updating one constant.
var expectedFSMHistorySubcodes = []string{
	"fsm-history-consistent/illegal-transition",
	"fsm-history-consistent/forced-untrailered",
	"fsm-history-consistent/manual-edit",
}

// TestFSMHistoryHints_AllSubcodesPresent asserts every expected
// fsm-history-consistent subcode has a non-empty hint entry in
// hintTable. The hints are loaded ahead of AC-2/3/4's code emission so
// PolicyFindingCodesHaveHints (one-directional: fires on emitted codes
// lacking hints) stays green when the predicates land.
func TestFSMHistoryHints_AllSubcodesPresent(t *testing.T) {
	t.Parallel()
	for _, key := range expectedFSMHistorySubcodes {
		if hintTable[key] == "" {
			t.Errorf("hintTable[%q] is empty; expected a non-empty hint for the subcode (required by PolicyFindingCodesHaveHints when AC-2/3/4 emit the code literal)", key)
		}
	}
}

// TestFSMHistorySubcodes_InAiwfCheckSkillMD asserts every expected
// fsm-history-consistent subcode appears as a literal string in
// internal/skills/embedded/aiwf-check/SKILL.md, satisfying
// PolicyFindingCodesAreDiscoverable when AC-2/3/4 emit. The
// discoverability policy reads the full set of channels (skills,
// printHelp, CLAUDE.md, docs/pocv3/**); checking just the aiwf-check
// SKILL.md is the strictest pin (it's where finding-code
// documentation conventionally lives).
func TestFSMHistorySubcodes_InAiwfCheckSkillMD(t *testing.T) {
	t.Parallel()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller returned ok=false")
	}
	// thisFile = .../internal/check/fsm_history_hints_test.go
	// repo root = ../../
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	skillPath := filepath.Join(repoRoot, "internal", "skills", "embedded", "aiwf-check", "SKILL.md")
	body, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	content := string(body)
	for _, subcode := range expectedFSMHistorySubcodes {
		if !strings.Contains(content, subcode) {
			t.Errorf("SKILL.md %s does not mention %q; required by PolicyFindingCodesAreDiscoverable when AC-2/3/4 emit the code literal", skillPath, subcode)
		}
	}
}
