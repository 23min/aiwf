package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// noopExemptVerbs is the explicit, reviewed allowlist of exported
// internal/verb/*.go entry points that have no same-state input to converge
// on, so requiring a Result.NoOp test of them would be requiring a test for
// a state that cannot arise. Each Reason names why the verb is
// by-design-additive or otherwise NoOp-less — not just "has no NoOp test."
// A new entry requires a reason of the same shape, reviewed like any other
// code change.
//
// The bar is "can a caller supply input that already equals current state?"
// If yes, the verb converges to a NoOp and belongs under the policy. If the
// verb only ever appends (a new entity, a new AC), there is no same state to
// detect.
//
// Every Reason below states measured behavior, not inferred behavior. The
// premise that makes the difference: aiwf does NOT reject an empty diff.
// Apply's guard (internal/verb/apply.go) refuses only a plan with ZERO file
// ops, and `git commit-tree` has no same-tree refusal, so a verb writing
// byte-identical content lands a real commit with an empty diffstat. An entry
// claiming "an identical input is an empty diff that gets rejected" is wrong
// unless the verb itself compares and refuses.
var noopExemptVerbs = []struct {
	Func   string
	Reason string
}{
	// Verified additive: a repeat allocates a new id rather than matching
	// existing state (measured — a second identical `add` produced E-0002, a
	// second identical `add ac` produced AC-2, a batch repeat produced AC-4..AC-5).
	{"Add", "purely additive: allocates a fresh id every call, so no input can already equal current state"},
	{"AddAC", "purely additive: appends a new AC; a duplicate title is a legitimate distinct AC, not a same-state input"},
	{"AddACBatch", "shares AddAC's rationale: appends N new ACs in one commit"},
	{"Reallocate", "renumbers to the next FREE id, computed rather than supplied, so the new id never equals the current one (measured: E-0001 renumbered to E-0003)"},

	// Verified to compare-and-refuse in the verb itself, so a same-state input
	// already writes nothing.
	{"ContractUnbind", "removes a binding: an absent binding is refused as a referential-integrity error, no commit (measured: exit 2, `no binding for <id>`)"},
	{"RecipeRemove", "shares ContractUnbind's rationale: removing an absent validator is a referential-integrity refusal (measured: exit 2, `validator not declared`)"},

	// OPEN entries. These are NOT by-design exemptions — each records a
	// deferred decision, and each is the reason this chokepoint reports green
	// with a known hole. Every one was measured, not assumed.
	{"PromoteACPhase", "OPEN, tracked in G-0458: same-phase input refuses via the TDD-phase FSM (measured: exit 1, no commit). Unlike the status case, the phase ladder is audit-bearing evidence and the verb carries a --tests payload, so convergence needs a deliberate metrics carve-out rather than a mechanical repeat — resolve by converting with that carve-out, or by rewriting this entry with a by-design reason"},
	{"AcknowledgeMistag", "OPEN, tracked in G-0459: an identical re-run appends a duplicate audit commit (measured). check.WalkAcknowledgedMistags already walks HEAD for these commits, so the dedup capability exists and is simply unused — the closest analogue to the guard acknowledge-illegal received"},
	{"PromoteAuditOnly", "OPEN, tracked in G-0459: an identical re-run appends a duplicate audit commit (measured). The verb's precondition is that the entity already sits at the target state, so a duplicate guard must key on an existing audit RECORD, not on entity state"},
	{"PromoteACPhaseAuditOnly", "OPEN, tracked in G-0459: shares PromoteAuditOnly's measured duplicate-record behavior"},
	{"CancelAuditOnly", "OPEN, tracked in G-0459: shares PromoteAuditOnly's measured duplicate-record behavior"},
	{"Authorize", "OPEN, tracked in G-0459 and G-0460: a repeat open exits 0 and appends a second commit (measured), leaving TWO simultaneously-active scopes on one entity with no check finding. Convergence may be the WRONG fix here — a second grant can be a legitimate new event — and a silent NoOp would mask the two-active-scopes defect, so G-0460 settles the invariant first"},
}

// PolicyVerbResultNoOpInvariant asserts that every exported
// internal/verb/*.go entry point — a function returning (*Result, error) —
// has at least one test under internal/verb/ that binds that verb's *Result
// and inspects the bound value's NoOp field, unless the verb appears on the
// reviewed allowlist above. The binding is what connects the two halves; see
// noopInspectedVerbs for why a looser relation credits verbs that have no
// NoOp coverage at all.
//
// The property it protects is the same-state convergence convention — a
// mutating verb handed input that already equals current state returns a NoOp
// at exit 0 rather than an error — stated in CLAUDE.md under "Designing a new
// verb" as the resolve-then-converge rules. ADR-0036 is the narrower authority
// cited there: it settles the FSM-transition case (promote / cancel) and
// explicitly disclaims field-mutation verbs, which are most of the entry points
// scanned here. So the convention, not the ADR, is what this enforces.
//
// This policy is the chokepoint: a new verb either carries a NoOp test or
// earns an allowlist entry stating why it cannot.
//
// Granularity is deliberately structural, not semantic: it verifies that some
// test binds the verb's *Result and inspects that value's NoOp field. It
// cannot prove the test drives genuinely same-state input, nor that the
// assertion expects a NoOp rather than its absence — that is what review and
// the AC's own tests are for. What it does catch is the failure mode that
// actually recurs: a verb with no same-state NoOp coverage at all.
func PolicyVerbResultNoOpInvariant(root string) ([]Violation, error) {
	// excludeTests=false: this policy needs the test bodies as much as the
	// production ones — the property spans both halves.
	files, err := WalkGoFiles(root, false)
	if err != nil {
		return nil, err
	}

	type entryPoint struct {
		name string
		file string
		line int
	}

	fset := token.NewFileSet()
	var entries []entryPoint
	entryNames := map[string]bool{}
	var testFuncs []*ast.FuncDecl

	// One AST pass collects both halves: the entry points (from production
	// files) and every test function declaration (analyzed below, once the
	// entry-point set is complete). The entry-point set is derived here
	// rather than hardcoded, so a newly-added verb is picked up with no
	// list to update.
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		isTest := strings.HasSuffix(f.Path, "_test.go")
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
				continue
			}
			if isTest {
				testFuncs = append(testFuncs, fn)
				continue
			}
			if isCapitalized(fn.Name.Name) && returnsResultAndError(fn.Type) {
				entries = append(entries, entryPoint{
					name: fn.Name.Name,
					file: f.Path,
					line: fset.Position(fn.Pos()).Line,
				})
				entryNames[fn.Name.Name] = true
			}
		}
	}

	if len(entries) == 0 {
		// No entry points found — internal/verb/ moved, or the signature
		// every verb shares changed. Either way an empty set produces an
		// empty violation list, so the policy would report green while
		// scanning nothing. Surface it as a self-policy violation instead,
		// the same fail-closed shape PolicyTrailerKeysViaConstants uses
		// when its constants go missing.
		return []Violation{{
			Policy: "verb-result-noop-invariant",
			File:   "internal/verb/",
			Detail: "no exported (*Result, error) entry points found under internal/verb/ — the tree moved or the verb signature changed, so this policy is scanning nothing and cannot vouch for same-state NoOp coverage; repoint it at the verbs' new location",
		}}, nil
	}

	covered := map[string]bool{}
	for _, fn := range testFuncs {
		for name := range noopInspectedVerbs(fn, entryNames) {
			covered[name] = true
		}
	}

	exempt := map[string]bool{}
	for _, e := range noopExemptVerbs {
		exempt[e.Func] = true
	}

	var out []Violation
	for _, e := range entries {
		if exempt[e.name] || covered[e.name] {
			continue
		}
		out = append(out, Violation{
			Policy: "verb-result-noop-invariant",
			File:   e.file,
			Line:   e.line,
			Detail: e.name + " has no test under internal/verb/ that drives it and asserts Result.NoOp, and is not on the reviewed allowlist in verb_result_noop_invariant.go — add a same-state test asserting the verb converges to a NoOp, or add an allowlist entry stating why it has no same-state input",
		})
	}
	return out, nil
}

// noopInspectedVerbs reports which verb entry points the test function fn both
// drives and inspects the Result.NoOp of. The two facts must be connected by
// dataflow: the identifier a call's *Result is bound to must be the same
// identifier whose NoOp (or NoOpMessage) field is referenced.
//
// Requiring that connection is what makes the credit mean something. Scanning
// the function's text for a call and a `.NoOp` independently credits a verb for
// three shapes that carry no NoOp evidence whatsoever:
//
//   - a `.NoOp` inside a comment or a format string — neither is a selector
//     expression, so neither is visible to this walk;
//   - a verb called only as fixture setup, in a test whose NoOp assertion is
//     about some other verb's Result. This is the common one, and it is not
//     addressed by scoping the scan to a single function: setup and assertion
//     live in the same function. Only the binding identifier separates them.
//   - a Result that is discarded (`_, err := verb.Foo(...)`), which cannot be
//     inspected at all.
//
// The analysis is intra-function and flow-insensitive: it ignores statement
// order and does not follow a Result through a helper call, so a value
// laundered through one (`res := mustNoOp(t, verb.Foo(...))`) earns no credit.
// Flow-insensitivity is also why an identifier bound to more than one entry
// point within one scope credits neither: without statement order there is no
// way to tell which binding was live at the inspection, and guessing would
// re-open the fixture-setup hole one rebound variable at a time. Scopes are
// independent, so sibling `t.Run` closures that each declare their own `res`
// are two names bound once, not one name bound twice, and both earn credit.
// Every unrecognized shape therefore under-credits and the policy fires,
// rather than passing silently.
//
// What it deliberately does not judge is the assertion's polarity: a test that
// asserts a Result is *not* a NoOp inspects the same field, and no structural
// signal distinguishes the two. Polarity is review's job.
func noopInspectedVerbs(fn *ast.FuncDecl, entryNames map[string]bool) map[string]bool {
	out := map[string]bool{}
	creditScope(fn.Body, nil, entryNames, out)
	return out
}

// creditScope walks one lexical scope — a function body or a function literal
// body — and records the verbs it credits into out, then recurses into the
// function literals nested directly inside it, passing its own bindings down so
// a closure that inspects a Result bound outside still earns credit.
//
// Scoping per function literal is what keeps sibling `t.Run` subtests
// independent. Each declares its own `res`; treating the whole test function as
// one namespace made those look like a single identifier bound to two verbs and
// refused credit for both, which is a false negative on an idiomatic table-free
// subtest pair.
func creditScope(body *ast.BlockStmt, inherited map[string]map[string]bool, entryNames, out map[string]bool) {
	// Each scope gets its own maps — the outer name->verbs map and the inner
	// verb sets both — so nothing a nested scope binds reaches back into this
	// scope or across into a sibling. Sharing the inner sets would not buy
	// precision: this scope's credit decision is made below, before the walk
	// recurses, so a child can never inform its parent. What sharing would buy
	// is a verdict that depends on which sibling closure the walk reaches
	// first, which is source-order dependence.
	bound := map[string]map[string]bool{}
	for name, verbs := range inherited {
		copied := make(map[string]bool, len(verbs))
		for verbName := range verbs {
			copied[verbName] = true
		}
		bound[name] = copied
	}
	inspected := map[string]bool{}
	// locallyBound records the names this scope has already declared with `:=`.
	// The first `:=` shadows whatever the name held and replaces it; a second
	// one in the same scope rebinds the same variable, so it accumulates and
	// the two-verb rule refuses credit. Replacing on every `:=` instead would
	// hand credit to the last verb named even when the function's only .NoOp
	// assertion is about an earlier one — the fixture-setup hole, reopened.
	locallyBound := map[string]bool{}
	var nested []*ast.FuncLit

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncLit:
			// Its own scope; collected after this one's bindings are complete.
			nested = append(nested, node)
			return false
		case *ast.AssignStmt:
			// An entry point returns (*Result, error), so a call to one is
			// the sole right-hand side and the *Result lands in Lhs[0].
			if len(node.Rhs) != 1 || len(node.Lhs) == 0 {
				return true
			}
			call, ok := node.Rhs[0].(*ast.CallExpr)
			if !ok {
				return true
			}
			verbName, ok := calledEntryPoint(call, entryNames)
			if !ok {
				return true
			}
			// A non-identifier target (`got["k"], err = …`) binds no name
			// this walk can follow. A blank one (`_, err := …`) needs no
			// guard of its own: binding `_` is inert, because crediting it
			// would take a `_.NoOp` reference and that cannot appear in
			// source that compiles.
			target, ok := node.Lhs[0].(*ast.Ident)
			if !ok {
				return true
			}
			if node.Tok == token.DEFINE && !locallyBound[target.Name] {
				// The first `:=` for this name declares a new variable, which
				// shadows any binding inherited from an enclosing scope. The
				// new variable unambiguously holds this verb's Result, so it
				// REPLACES rather than joins — otherwise a subtest declaring
				// its own `res` would look like the outer `res` rebound, and
				// the two-verb rule would disqualify both.
				locallyBound[target.Name] = true
				bound[target.Name] = map[string]bool{verbName: true}
				return true
			}
			// A rebinding of a name this scope already holds: an `=`, or a
			// second `:=` in the same scope. Without statement order this walk
			// cannot tell which binding was live at the inspection, so the name
			// accumulates and the two-verb rule refuses credit.
			if bound[target.Name] == nil {
				bound[target.Name] = map[string]bool{}
			}
			bound[target.Name][verbName] = true
		case *ast.SelectorExpr:
			if node.Sel.Name != "NoOp" && node.Sel.Name != "NoOpMessage" {
				return true
			}
			if recv, ok := node.X.(*ast.Ident); ok {
				inspected[recv.Name] = true
			}
		}
		return true
	})

	for name := range inspected {
		// Two verbs bound to one name within a scope credits neither — see
		// noopInspectedVerbs.
		if verbs := bound[name]; len(verbs) == 1 {
			for verbName := range verbs {
				out[verbName] = true
			}
		}
	}
	for _, lit := range nested {
		creditScope(lit.Body, bound, entryNames, out)
	}
}

// calledEntryPoint resolves a call expression to the verb entry point it
// invokes, recognizing the two spellings the tests use: the external-test form
// `verb.Promote(...)` (package verb_test) and the in-package form
// `Promote(...)`. A call qualified by anything other than the verb package
// (`harness.Promote(...)`) and a same-named local wrapper
// (`mustPromote(...)`) are not entry points.
func calledEntryPoint(call *ast.CallExpr, entryNames map[string]bool) (string, bool) {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		pkg, ok := fun.X.(*ast.Ident)
		if !ok || pkg.Name != "verb" || !entryNames[fun.Sel.Name] {
			return "", false
		}
		return fun.Sel.Name, true
	case *ast.Ident:
		if !entryNames[fun.Name] {
			return "", false
		}
		return fun.Name, true
	}
	return "", false
}
