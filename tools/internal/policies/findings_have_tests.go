package policies

import (
	"strings"
)

// PolicyFindingCodesHaveTests asserts that every finding code emitted
// by the kernel — both named `Code...` constants in
// tools/internal/check/ and inline `Code: "..."` literals in
// Finding{} composite literals across check/ and contractcheck/ —
// is referenced from at least one *_test.go file. The test reference
// is what proves the code's emission and shape are exercised; an
// orphan code is by definition untested.
//
// The policy stops at presence (does not require BOTH a positive
// and a negative fixture; that would need test-body parsing). For
// inline-literal codes there is no constant name to grep for, so
// the only acceptable test reference is the quoted string value.
//
// Closes G26: extends the prior named-constant-only enumeration to
// cover the same code population that PolicyFindingCodesAreDiscoverable
// covers (see G21). The two policies now share `allCheckCodes` so
// drift between "what the docs cover" and "what the tests cover"
// is structurally impossible.
func PolicyFindingCodesHaveTests(root string) ([]Violation, error) {
	allFiles, err := WalkGoFiles(root, false) // include tests
	if err != nil {
		return nil, err
	}
	prodFiles, err := WalkGoFiles(root, true) // production-only for code enumeration
	if err != nil {
		return nil, err
	}
	consts := loadCheckCodeConstants(prodFiles)
	literals := loadCheckCodeLiterals(prodFiles)

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
	// Named-constant codes: tests can reference by constant name OR
	// by the literal value.
	for name, value := range consts {
		if !looksLikeFindingCode(value) {
			continue
		}
		if strings.Contains(haystack, name) {
			continue
		}
		if strings.Contains(haystack, `"`+value+`"`) {
			continue
		}
		out = append(out, Violation{
			Policy: "finding-codes-have-tests",
			File:   "tools/internal/check/",
			Detail: "finding code " + value +
				" (constant " + name + ") is not referenced by any *_test.go file",
		})
	}
	// Inline-literal codes: only the quoted value is an acceptable
	// reference (there is no constant name to fall back on).
	//
	// Composite "<code>/<subcode>" entries get a relaxation: a test
	// that asserts both `"<code>"` and `"<subcode>"` (anywhere in the
	// same test file body) counts. That matches how check tests
	// natively express subcoded findings — by checking
	// `f.Code == ... && f.Subcode == ...`. Without this relaxation
	// the policy would force tests to fabricate the slash form purely
	// to satisfy a grep, which is a worse signal.
	for value := range literals {
		// Skip anything also declared as a named constant — already
		// handled above.
		if hasConstantValue(consts, value) {
			continue
		}
		if strings.Contains(haystack, `"`+value+`"`) {
			continue
		}
		if isSubcoded(value) && hasSubcodePair(allFiles, value) {
			continue
		}
		out = append(out, Violation{
			Policy: "finding-codes-have-tests",
			File:   "tools/internal/check/",
			Detail: "finding code " + value +
				" (inline literal) is not referenced by any *_test.go file",
		})
	}
	return out, nil
}

// isSubcoded reports whether s has the "code/subcode" shape.
func isSubcoded(s string) bool {
	idx := strings.Index(s, "/")
	return idx > 0 && idx < len(s)-1
}

// hasSubcodePair returns true when at least one *_test.go file under
// tools/internal/check/ or tools/internal/contractcheck/ contains
// quoted literals for both the code half and the subcode half of
// "<code>/<subcode>". That's the canonical assertion shape for a
// subcoded finding (e.g. `f.Code == "no-cycles" && f.Subcode == "depends_on"`).
func hasSubcodePair(files []FileEntry, composite string) bool {
	idx := strings.Index(composite, "/")
	if idx <= 0 || idx >= len(composite)-1 {
		return false
	}
	codeLit := `"` + composite[:idx] + `"`
	subLit := `"` + composite[idx+1:] + `"`
	for _, f := range files {
		if !strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		if !strings.HasPrefix(f.Path, "tools/internal/check/") &&
			!strings.HasPrefix(f.Path, "tools/internal/contractcheck/") {
			continue
		}
		body := string(f.Contents)
		if strings.Contains(body, codeLit) && strings.Contains(body, subLit) {
			return true
		}
	}
	return false
}

// hasConstantValue reports whether any named constant in consts has
// the given string value. Used to dedupe the inline-literal pass
// against the named-constant pass.
func hasConstantValue(consts map[string]string, value string) bool {
	for _, v := range consts {
		if v == value {
			return true
		}
	}
	return false
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
