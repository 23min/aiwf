package policies

import (
	"regexp"
	"strings"
)

// statusValuesPattern matches the closed-set status values that
// appear in entity FSM tables. Hardcoding them outside the entity
// package would cause silent drift when a kind's allowed-status
// set evolves. The constants live in tools/internal/entity/ and
// should be the only literal source.
//
// We list every kind's status set explicitly; substring matching
// keeps the policy understandable.
var statusLiteralValues = map[string]bool{
	// Epic
	"\"proposed\"":  true,
	"\"active\"":    true,
	"\"done\"":      true,
	"\"cancelled\"": true,
	// Milestone
	"\"draft\"":       true,
	"\"in_progress\"": true,
	// ADR / Decision
	"\"accepted\"":   true,
	"\"superseded\"": true,
	"\"rejected\"":   true,
	// Gap
	"\"open\"":      true,
	"\"addressed\"": true,
	"\"wontfix\"":   true,
	// Contract
	"\"deprecated\"": true,
	"\"retired\"":    true,
	// AC
	"\"met\"":      true,
	"\"deferred\"": true,
	// TDD phase
	"\"red\"":      true,
	"\"green\"":    true,
	"\"refactor\"": true,
	// Scope state
	"\"paused\"":  true,
	"\"ended\"":   true,
	"\"opened\"":  true,
	"\"resumed\"": true,
}

// statusContextPattern detects places where a status literal is
// used in a "compare-to-status" context — e.g. assigned to a
// `status: ...` frontmatter, compared to `e.Status`, or set as the
// value of a Trailer{Key: TrailerTo, ...}. Outside these contexts,
// the same literal might be unrelated prose.
//
// Heuristic: look for tokens like `Status:`, `e.Status ==`, or
// `Value:` near the literal. Unfortunately many false positives
// arise from log strings, body templates, and documentation
// snippets. So this policy is intentionally narrow — we flag only
// where literally `Status: "value"` or `Status == "value"` appears.
var statusContextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`Status:\s*"([a-z_]+)"`),
	regexp.MustCompile(`\.Status\s*==\s*"([a-z_]+)"`),
	regexp.MustCompile(`\.Status\s*!=\s*"([a-z_]+)"`),
	regexp.MustCompile(`TDDPhase:\s*"([a-z_]+)"`),
	regexp.MustCompile(`\.TDDPhase\s*==\s*"([a-z_]+)"`),
}

// PolicyClosedSetStatusViaConstants flags context-relevant string
// literals matching closed-set status / phase / state values when
// they appear outside the entity package. The intent is to push
// every site through entity-package constants (they don't all
// exist yet — when this policy fires, the resolution may be
// "introduce a constant in entity/" rather than just s/// the
// literal).
func PolicyClosedSetStatusViaConstants(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, f := range files {
		// entity package owns the constants; scope package owns
		// scope-state strings.
		if strings.HasPrefix(f.Path, "tools/internal/entity/") ||
			strings.HasPrefix(f.Path, "tools/internal/scope/") {
			continue
		}
		for _, pat := range statusContextPatterns {
			matches := pat.FindAllSubmatchIndex(f.Contents, -1)
			for _, m := range matches {
				val := string(f.Contents[m[2]:m[3]])
				lit := "\"" + val + "\""
				if !statusLiteralValues[lit] {
					continue
				}
				out = append(out, Violation{
					Policy: "closed-set-status-via-constants",
					File:   f.Path,
					Line:   LineOf(f.Contents, m[0]),
					Detail: "literal status value " + lit +
						" used in a Status / TDDPhase context; introduce or reuse an entity-package constant instead",
				})
			}
		}
	}
	return out, nil
}
