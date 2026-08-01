package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// claimGuardWrappers pairs a recorded claim scope with the guard wrapper
// a verb carrying that scope must reach.
//
// The pairing is the point. Both wrappers delegate to guardClaimPaths,
// differing only in its exemptAbsentFromHEAD argument — and that
// argument is what the scope decides. guardClaim's own doc states that
// exempting a path absent from HEAD for an entity claim "would let the
// verb read a status HEAD contradicts and answer 'already set; nothing
// to change'", the reproduction the guard exists to refuse. So requiring
// the wrapper rather than the shared implementation is what keeps a
// direct guardClaimPaths(..., true, ...) call from satisfying an
// entity-scoped row while defeating it.
var claimGuardWrappers = []struct{ Scope, Wrapper string }{
	{claimScopeTargetEntity, "guardClaim"},
	{claimScopeConfigFile, "guardClaimConfig"},
}

// claimGuardWrapperFor returns the wrapper a scope requires, and whether
// the scope requires one at all.
func claimGuardWrapperFor(scope string) (string, bool) {
	for _, w := range claimGuardWrappers {
		if w.Scope == scope {
			return w.Wrapper, true
		}
	}
	return "", false
}

// guardWrapperNames is the wrapper set, for scans that ask "any guard".
func guardWrapperNames() map[string]bool {
	out := make(map[string]bool, len(claimGuardWrappers))
	for _, w := range claimGuardWrappers {
		out[w.Wrapper] = true
	}
	return out
}

// verbFunc is one package-level function declared under internal/verb,
// carried with the same-package names it calls and enough of its
// declaration site for a finding to name it.
//
// GuardLine and NoOpLine carry what a call-graph edge cannot: where the
// guard sits relative to the convergence it protects. Both are 0 when
// the function has no such site, and GuardLine is 0 for a guard reached
// only through a helper, which is the shape the placement check cannot
// measure.
type verbFunc struct {
	File      string
	Line      int
	Calls     map[string]bool
	GuardLine int
	NoOpLine  int
}

// graphOf projects the scanned functions onto the call graph the walk
// consumes.
func graphOf(funcs map[string]verbFunc) callGraph {
	out := make(callGraph, len(funcs))
	for name, fn := range funcs {
		out[name] = fn.Calls
	}
	return out
}

// checkClaimGuardPresence asserts that every ledger row whose scope names
// a file — target-entity or aiwf-yaml — belongs to a verb that actually
// consults it.
//
// This is the half the ledger cannot carry. A scope is a name in a list;
// the paths a guard receives are an expression, and a verb recording
// `target-entity` while calling nothing compiles, runs, and reads as
// covered on both surfaces.
//
// A row whose scope is outside the closed set falls through both this
// check and the dormancy one, so a malformed ledger is reported once, by
// PolicyNoOpClaimScope, which owns the ledger's shape.
func checkClaimGuardPresence(ledger []claimScope, funcs map[string]verbFunc) []Violation {
	graph := graphOf(funcs)
	var out []Violation
	for _, entry := range ledger {
		wrapper, guarded := claimGuardWrapperFor(entry.Scope)
		if !guarded {
			continue
		}
		fn, found := funcs[entry.Func]
		if !found {
			out = append(out, Violation{
				Policy: "claim-guard-presence",
				File:   "internal/policies/noop_claim_scope.go",
				Detail: fmt.Sprintf("%s records claim scope %q, but no package-level function by that name was found under internal/verb — the row is either stale, which PolicyNoOpClaimScope reports on its own terms, or it names a shape this scan does not see, such as a method; either way this policy cannot vouch for it",
					entry.Func, entry.Scope),
			})
			continue
		}
		if !reachesCall(entry.Func, wrapper, graph, map[string]bool{}) {
			out = append(out, Violation{
				Policy: "claim-guard-presence",
				File:   fn.File,
				Line:   fn.Line,
				Detail: fmt.Sprintf("%s records claim scope %q but never reaches %s, directly or via a same-package helper — the scope names what its convergence reads, and a verb that converges without comparing it answers from bytes no verb wrote; call %s in the prelude, or record the scope the route actually takes (ADR-0038)",
					entry.Func, entry.Scope, wrapper, wrapper),
			})
			continue
		}
		// Placement, which the call-graph edge cannot carry. ADR-0038
		// settles that the guard runs in the prelude: a guard below the
		// convergence never executes on the input it exists to refuse,
		// because the verb has already answered "already set; nothing to
		// change" from the disputed bytes and returned.
		if fn.GuardLine > 0 && fn.NoOpLine > 0 && fn.GuardLine > fn.NoOpLine {
			out = append(out, Violation{
				Policy: "claim-guard-presence",
				File:   fn.File,
				Line:   fn.GuardLine,
				Detail: fmt.Sprintf("%s calls %s at line %d, below the Result{NoOp: true} it protects at line %d — the converging return fires first, so the guard never sees the input it exists to refuse; move the call into the prelude, before the same-state comparison (ADR-0038)",
					entry.Func, wrapper, fn.GuardLine, fn.NoOpLine),
			})
		}
	}
	return out
}

// checkDormantClaimExemptions asserts the other direction: a row recorded
// exempt still has nothing wired.
//
// An exemption states a premise — that no per-verb comparison exists, or
// that the comparison happens per candidate inside the verb's own planner
// — and a premise that has stopped holding still reads as a reviewed
// decision. That is the shape an exemption outliving its condition takes.
func checkDormantClaimExemptions(ledger []claimScope, funcs map[string]verbFunc) []Violation {
	graph := graphOf(funcs)
	var out []Violation
	for _, entry := range ledger {
		if _, guarded := claimGuardWrapperFor(entry.Scope); guarded {
			continue
		}
		// A name absent from funcs is absent from the graph too, so
		// reachesCall already answers false for it; the staleness that
		// causes is PolicyNoOpClaimScope's finding, not this one's.
		reached := ""
		for _, w := range claimGuardWrappers {
			if reachesCall(entry.Func, w.Wrapper, graph, map[string]bool{}) {
				reached = w.Wrapper
				break
			}
		}
		if reached == "" {
			continue
		}
		fn := funcs[entry.Func]
		out = append(out, Violation{
			Policy: "claim-guard-presence",
			File:   fn.File,
			Line:   fn.Line,
			Detail: fmt.Sprintf("%s is recorded exempt with scope %q but now reaches %s — the exemption's whole content is that no per-verb comparison is wired, and one is; record the scope the guard is comparing, or drop the call",
				entry.Func, entry.Scope, reached),
		})
	}
	return out
}

// validateExemptClaimReasons asserts that each exempt row carries its own
// specific reason.
//
// The guarded rows are checked against a guard call. An exempt row has no
// such anchor — the reason is the entire content of the entry, so a
// sentence shared with a sibling, or one that only restates the route's
// name, leaves nothing a reader can check the exemption against.
func validateExemptClaimReasons(ledger []claimScope) []Violation {
	seen := map[string]string{}
	var out []Violation
	for _, entry := range ledger {
		if entry.Scope != claimScopeSweepDeciders && entry.Scope != claimScopeNone {
			continue
		}
		reason := strings.TrimSpace(entry.Reason)
		if prior, dup := seen[reason]; dup {
			out = append(out, Violation{
				Policy: "claim-guard-presence",
				File:   "internal/policies/noop_claim_scope.go",
				Detail: fmt.Sprintf("%s and %s record the same exemption reason — an exempt route carries its own specific reason, not a category label shared across routes, because the reason is the only thing distinguishing a reasoned exemption from an omission",
					prior, entry.Func),
			})
		}
		seen[reason] = entry.Func
		if strings.EqualFold(reason, entry.Func) {
			out = append(out, Violation{
				Policy: "claim-guard-presence",
				File:   "internal/policies/noop_claim_scope.go",
				Detail: fmt.Sprintf("%s records an exemption reason that restates its own name — say what makes the claim uncontradictable, or what the comparison is scoped to instead",
					entry.Func),
			})
		}
	}
	return out
}

// PolicyClaimGuardPresence asserts that the claim-side guard (ADR-0038)
// is wired wherever the same-state ledger says a claim rests on a file,
// and absent wherever that ledger records an exemption.
//
// Pins M-0285/AC-1 and AC-2. It is the presence half of
// PolicyNoOpClaimScope, which records what each converging verb's claim
// is about but states in its own doc comment that it cannot see whether a
// guard is wired to that scope.
//
// Three bounds a reader should hold:
//
//   - Exported and unexported package-level functions are alike in scope,
//     which is what puts the four composite-id routes inside it. The
//     exported-only reach of PolicyVerbResultNoOpInvariant is a property
//     of that policy, not of the model.
//   - A converging method is the shape the scan does not see. Such a row
//     is reported as unvouchable rather than passed over, so the bound
//     costs a finding rather than a silent gap.
//   - Its subject is the converging routes, because those are the ones
//     the commit-side guard structurally cannot reach: a verb that
//     converges returns before a plan exists, so Apply never runs. A
//     route that instead refuses on the strength of working-copy bytes
//     without producing ops has the same shape and no ledger row, so it
//     is outside this policy too.
//
// What it does not prove, in the register both sibling ledgers use about
// themselves:
//
//   - That a scope is the right one for its claim. Recording an exemption
//     instead of wiring a guard clears every check here, so an exemption
//     is a reviewed decision rather than a correction, and the one-word
//     edit that makes it is what review is looking for.
//   - That the guard is passed the right paths. guardClaim(ctx, root, id)
//     with no paths at all compares nothing and satisfies this policy;
//     the paths are an expression the scan does not evaluate.
//   - That a call it counts executes. An edge is a name in call position,
//     so a call in a branch no input selects, or to a local that shadows
//     the wrapper, reads as reaching. The placement check narrows this
//     where a guard is called directly; a guard reached only through a
//     helper carries no position here and is not placement-checked.
//
// Those are review's, not CI's.
func PolicyClaimGuardPresence(root string) ([]Violation, error) {
	return claimGuardPresence(root, noOpClaimScopes)
}

// claimGuardPresence is the policy with its ledger injected, so a test
// can drive the whole composition — scan, then all four checks — rather
// than each check in isolation. A suite that only reaches the helpers
// cannot tell a wired policy from one whose scan output goes nowhere.
func claimGuardPresence(root string, ledger []claimScope) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil { //coverage:ignore defensive: WalkGoFiles errors only when the scan root is unreadable, which every other policy in this package would fail on first
		return nil, err
	}

	fset := token.NewFileSet()
	wrappers := guardWrapperNames()
	funcs := map[string]verbFunc{}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil { //coverage:ignore defensive: the tree under scan compiles, so a parse failure needs a file edited mid-run
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
				continue
			}
			funcs[fn.Name.Name] = verbFunc{
				File:      f.Path,
				Line:      fset.Position(fn.Pos()).Line,
				Calls:     calledIdents(fn),
				GuardLine: firstCallLine(fset, fn, wrappers),
				NoOpLine:  firstLineOf(fset, noOpResultPositions(fn)),
			}
		}
	}

	// Fail closed. An empty scan would otherwise satisfy every row by
	// vacuity and report green while checking nothing.
	if len(funcs) == 0 {
		return []Violation{{
			Policy: "claim-guard-presence",
			File:   "internal/verb/",
			Detail: "no package-level functions found under internal/verb/ — the policy is scanning nothing, which is not the same as every scoped row consulting its guard; repoint it at the verbs' new location",
		}}, nil
	}

	out := checkGuardAnchorsExist(funcs)
	out = append(out, checkClaimGuardPresence(ledger, funcs)...)
	out = append(out, checkDormantClaimExemptions(ledger, funcs)...)
	return append(out, validateExemptClaimReasons(ledger)...), nil
}

// checkGuardAnchorsExist reports a wrapper this policy hangs on that the
// scan no longer finds.
//
// Every other finding here is phrased against a wrapper name. Renaming
// or inlining one would otherwise produce a violation per guarded row,
// each blaming a verb that did nothing, instead of one naming the anchor
// that moved. PolicyCommitConstructionSingleSeam asserts its own
// cross-package anchor the same way.
func checkGuardAnchorsExist(funcs map[string]verbFunc) []Violation {
	var out []Violation
	for _, w := range claimGuardWrappers {
		if _, ok := funcs[w.Wrapper]; ok {
			continue
		}
		out = append(out, Violation{
			Policy: "claim-guard-presence",
			File:   "internal/verb/claimguard.go",
			Detail: fmt.Sprintf("%s is not declared under internal/verb — every %s finding is phrased against that name, so this policy cannot answer anything until the anchor is repointed (it is the wrapper the %q scope requires)",
				w.Wrapper, "claim-guard-presence", w.Scope),
		})
	}
	return out
}

// firstCallLine returns the line of the earliest call to any name in
// want within fn's body, or 0 when fn calls none of them.
func firstCallLine(fset *token.FileSet, fn *ast.FuncDecl, want map[string]bool) int {
	best := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || !want[ident.Name] {
			return true
		}
		if line := fset.Position(call.Pos()).Line; best == 0 || line < best {
			best = line
		}
		return true
	})
	return best
}

// firstLineOf returns the earliest line among positions, or 0 for none.
func firstLineOf(fset *token.FileSet, positions []token.Pos) int {
	best := 0
	for _, p := range positions {
		if line := fset.Position(p).Line; best == 0 || line < best {
			best = line
		}
	}
	return best
}
