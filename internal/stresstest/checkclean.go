package stresstest

import "fmt"

// checkclean.go — M-0257/AC-2: classifyAgainstBaseline generalizes
// verb_sequence.go's original classifyCheckFindings/
// verbSequenceExpectedWarnings pair (G-0410) into one
// baseline-parameterized helper every scenario's own curated map
// backs, so the loop judging `aiwf check` findings against a curated
// "expected noise" baseline lives in exactly one place rather than
// being reinvented per scenario. verbSequenceExpectedWarnings itself
// stays in verb_sequence.go (verb-sequence's own baseline, unchanged);
// classifyCheckFindings there is now a thin wrapper over this
// function. Every other scenario under M-0257/AC-1 defines its own
// baseline map, alongside its own existing scenario-specific
// assertion — never sharing one map across scenarios (different
// scenarios produce different incidental noise).

// classifyAgainstBaseline reports a violation for every finding whose
// severity is "error" — unconditionally; no baseline entry can excuse
// an error-severity finding, since a scenario's own curated baseline
// only ever documents expected WARNING noise — or whose code is not
// marked true in baseline. A finding that is both warning-severity and
// present in baseline is accepted, documented noise the caller's own
// baseline map explains.
func classifyAgainstBaseline(findings []verbEnvelopeFinding, baseline map[string]bool) []Violation {
	var violations []Violation
	for _, f := range findings {
		if f.Severity == "error" || !baseline[f.Code] {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"unexpected aiwf check finding: %s (%s)", f.Code, f.Severity,
			)})
		}
	}
	return violations
}
