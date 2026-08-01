package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// claimGuardCall names the claim-side comparison both scoped guards
// funnel through. Targeting the shared implementation rather than either
// wrapper is what makes guardClaim and guardClaimConfig count the same:
// the two differ only in whether a path absent from HEAD is exempt, which
// is the scope's business and not this policy's.
const claimGuardCall = "guardClaimPaths"

// verbFunc is one package-level function declared under internal/verb,
// carried with the same-package names it calls and enough of its
// declaration site for a finding to name it.
type verbFunc struct {
	File  string
	Line  int
	Calls map[string]bool
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
		if entry.Scope != claimScopeTargetEntity && entry.Scope != claimScopeConfigFile {
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
		if reachesCall(entry.Func, claimGuardCall, graph, map[string]bool{}) {
			continue
		}
		out = append(out, Violation{
			Policy: "claim-guard-presence",
			File:   fn.File,
			Line:   fn.Line,
			Detail: fmt.Sprintf("%s records claim scope %q but never reaches %s, directly or via a same-package helper — the scope names what its convergence reads, and a verb that converges without comparing it answers from bytes no verb wrote; call guardClaim (or guardClaimConfig for an aiwf.yaml claim) in the prelude, or record the scope the route actually takes (ADR-0038)",
				entry.Func, entry.Scope, claimGuardCall),
		})
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
		if entry.Scope != claimScopeSweepDeciders && entry.Scope != claimScopeNone {
			continue
		}
		fn, found := funcs[entry.Func]
		if !found || !reachesCall(entry.Func, claimGuardCall, graph, map[string]bool{}) {
			continue
		}
		out = append(out, Violation{
			Policy: "claim-guard-presence",
			File:   fn.File,
			Line:   fn.Line,
			Detail: fmt.Sprintf("%s is recorded exempt with scope %q but now reaches %s — the exemption's whole content is that no per-verb comparison is wired, and one is; record the scope the guard is comparing, or drop the call",
				entry.Func, entry.Scope, claimGuardCall),
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
// What it does not prove: that a scope is the right one for its claim, or
// that the guard is passed the right paths. The paths are an expression
// this scan does not evaluate — that judgement is the review's, the same
// bound both sibling ledgers state about themselves.
func PolicyClaimGuardPresence(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil { //coverage:ignore defensive: WalkGoFiles errors only when the scan root is unreadable, which every other policy in this package would fail on first
		return nil, err
	}

	fset := token.NewFileSet()
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
				File:  f.Path,
				Line:  fset.Position(fn.Pos()).Line,
				Calls: calledIdents(fn),
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

	out := checkClaimGuardPresence(noOpClaimScopes, funcs)
	out = append(out, checkDormantClaimExemptions(noOpClaimScopes, funcs)...)
	return append(out, validateExemptClaimReasons(noOpClaimScopes)...), nil
}
