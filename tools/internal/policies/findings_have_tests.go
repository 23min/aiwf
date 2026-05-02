package policies

import (
	"strings"
)

// PolicyFindingCodesHaveTests asserts that every finding code
// declared as a `Code...` constant in the check package is
// referenced from at least one *_test.go file. The test reference
// is what proves the code's emission and shape are exercised; an
// orphan code is by definition untested.
//
// Scope: every constant in tools/internal/check/ whose value is a
// kebab-case finding-code string. The policy reads the same
// constant table the discoverability policy uses, then greps the
// test-file population for each code.
//
// We do NOT require BOTH a positive and a negative fixture (that
// would need test-body parsing); the policy stops at presence,
// which is enough to flag a wholly orphaned code.
func PolicyFindingCodesHaveTests(root string) ([]Violation, error) {
	allFiles, err := WalkGoFiles(root, false) // include tests
	if err != nil {
		return nil, err
	}
	prodFiles, err := WalkGoFiles(root, true) // production-only for constants
	if err != nil {
		return nil, err
	}
	consts := loadCheckCodeConstants(prodFiles)

	// Collect every test-file body as one big haystack.
	var testHaystack strings.Builder
	for _, f := range allFiles {
		if !strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		testHaystack.Write(f.Contents)
		testHaystack.WriteByte('\n')
	}
	haystack := testHaystack.String()

	var out []Violation
	for name, value := range consts {
		// Only consider finding-code-shaped values: lowercase, dashes,
		// at least one dash. Excludes things like Severity values.
		if !looksLikeFindingCode(value) {
			continue
		}
		// Test reference can be by constant name OR by the literal value.
		if strings.Contains(haystack, name) {
			continue
		}
		quoted := `"` + value + `"`
		if strings.Contains(haystack, quoted) {
			continue
		}
		out = append(out, Violation{
			Policy: "finding-codes-have-tests",
			File:   "tools/internal/check/",
			Detail: "finding code " + value +
				" (constant " + name + ") is not referenced by any *_test.go file",
		})
	}
	return out, nil
}

// looksLikeFindingCode returns true when s looks like a kebab-case
// finding code — lowercase letters, digits, and dashes, with at
// least one dash. Used to skip non-finding-code constants.
func looksLikeFindingCode(s string) bool {
	if len(s) < 3 {
		return false
	}
	hasDash := false
	for _, r := range s {
		switch {
		case r == '-':
			hasDash = true
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '/':
			// subcoded codes (refs-resolve/unresolved) are valid finding-code shape.
		default:
			return false
		}
	}
	return hasDash
}
