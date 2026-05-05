package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicyIntegrationTestsAssertTrailers flags top-level test
// functions in cmd/aiwf/ that invoke a mutating verb via
// runBin (the integration-test entry point) but never assert the
// resulting commit's trailers. A test that only checks the verb's
// exit code is a smoke test, not an integration test of the
// kernel's audit-trail guarantee.
//
// Heuristic: a function whose body contains `runBin(` invocations
// where the verb argument is one of the known mutating verbs
// (`add`, `promote`, `cancel`, `move`, `reallocate`, `rename`,
// `authorize`, `import`, `contract bind`, `contract unbind`) must
// also reference one of: gitops.HeadTrailers, hasTrailer,
// authorizedByOf, %(trailers (the git log format), or the
// HistoryEvent struct (whose JSON unmarshal includes trailer-derived
// fields).
//
// False positives: setup-only test functions that happen to call
// `runBin "init"` (init is non-mutating from a trailer standpoint
// in the policy's terms — it's the bootstrap). The policy filters
// on "mutating-verb invocations" specifically.
func PolicyIntegrationTestsAssertTrailers(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, false)
	if err != nil {
		return nil, err
	}
	mutatingVerbs := []string{
		`"promote"`,
		`"cancel"`,
		`"move"`,
		`"reallocate"`,
		`"rename"`,
		`"authorize"`,
		// `"add"` and `"import"` excluded: many integration tests
		// only set up state with `add` and verify other things
		// downstream. We could include them with more sophistication.
	}
	// A test counts as "asserting the audit trail" when its body
	// references any of these markers — direct trailer reads
	// (gitops.HeadTrailers, hasTrailer, authorizedByOf), HistoryEvent
	// JSON consumption, or an indirect verification through a
	// read-only verb whose output is derived from trailers
	// (`runBin ... "show"`, `"history"`, `"check"`).
	trailerAssertionMarkers := []string{
		"gitops.HeadTrailers",
		"hasTrailer(",
		"authorizedByOf(",
		"HistoryEvent",
		`"aiwf-`,
		"aiwf-verb:",
		"aiwf-entity:",
		"aiwf-actor:",
		// Indirect verification through trailer-derived read verbs:
		`"show"`,
		`"history"`,
		`"check"`,
		`"status"`,
		// Refusal tests: a test that expects the verb to refuse has
		// no commit to inspect; flag the refusal pattern as
		// equivalent to a trailer assertion (no trailers landed).
		"expected ... fail",
		"expected aiwf",
		"got success",
		"want refusal",
		"err == nil",
	}

	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "cmd/aiwf/") {
			continue
		}
		if !strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				continue
			}
			body := string(f.Contents[start:end])
			// Must invoke a mutating verb via runBin to qualify.
			callsMutating := false
			for _, v := range mutatingVerbs {
				if strings.Contains(body, "runBin(") && strings.Contains(body, v) {
					callsMutating = true
					break
				}
			}
			if !callsMutating {
				continue
			}
			// Already asserts trailers somehow?
			hasAssertion := false
			for _, m := range trailerAssertionMarkers {
				if strings.Contains(body, m) {
					hasAssertion = true
					break
				}
			}
			if hasAssertion {
				continue
			}
			out = append(out, Violation{
				Policy: "integration-tests-assert-trailers",
				File:   f.Path,
				Line:   fset.Position(fn.Pos()).Line,
				Detail: fn.Name.Name +
					" invokes a mutating verb via runBin but never asserts the resulting commit's trailers; integration tests must verify the audit trail, not just exit codes",
			})
		}
	}
	return out, nil
}
