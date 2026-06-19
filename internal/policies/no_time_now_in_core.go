package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
)

// coreMinTier is the lowest layer tier (per layerTier) considered the
// "domain core" for time-determinism purposes. Tier 0 (cmd) and tier 1
// (cli + adapters) are the edge: they legitimately acquire the wall
// clock and pass it inward. Tier 2 and below (verb, check, render,
// entity, gitops, …) are the core and must receive time as data.
const coreMinTier = 2

// timeNowSelectors are the time package functions that read the ambient
// wall clock. time.Now mints the current instant; time.Since and
// time.Until both read it relative to an argument. A core package that
// calls any of them depends on a non-deterministic input.
var timeNowSelectors = map[string]bool{
	"Now":   true,
	"Since": true,
	"Until": true,
}

// noTimeNowExempt names core packages allowed to read the ambient clock,
// with the rationale beside each (same pattern as the atomic-write
// chokepoint). These are operational/perf uses — not logical time that
// enters reported or persisted state — so injecting a clock would be
// wrong or pointless rather than an improvement.
var noTimeNowExempt = map[string]string{
	"internal/repolock":   "OS lock-acquisition timeout loop (deadline/After/Sleep) genuinely needs real wall-clock; a flock deadline cannot be driven by an injected fake clock",
	"internal/htmlrender": "Render measures its own wall-clock duration for the ElapsedMs metric; monotonic perf timing that never enters the rendered output (render determinism is independently pinned by the byte-identical test)",
}

// PolicyNoTimeNowInCore forbids ambient wall-clock reads (time.Now,
// time.Since, time.Until) in the domain core, so the core's outputs stay
// a deterministic function of (planning state + given inputs). The rule
// is the architectural principle stated directly: the domain core is
// time-deterministic; the wall clock is acquired only at the edge and
// flows inward as data (G-0235's no-time-now item).
//
// Scope is defined by *layer*, not an ad-hoc package list: a package is
// in the core iff layerTier ranks it at coreMinTier (2) or below in
// altitude — verb and everything beneath it. The edge (cmd, cli + its
// adapters; tier 0–1) is out of scope: that is exactly where the real
// clock is read and injected downward. Reusing layerTier makes this
// future-proof — a new core package is covered automatically, with no
// second scope list to maintain.
//
// Two operational exemptions are allowlisted by name + rationale
// (noTimeNowExempt): repolock's lock-timeout loop and htmlrender's
// self-perf metric. These are real architectural exceptions — ambient
// time that is operational, not logical — and the high-visibility
// allowlist keeps the bar to add another exemption deliberate.
//
// Not in scope here: BuildStatus (internal/cli/status, tier 1) stamps a
// logical StatusReport.Date from time.Now. That is the one place ambient
// time leaks into *reported state*, but it sits at the edge so this
// policy does not flag it; it is corrected by injecting the clock at that
// call site rather than by widening this policy to the CLI layer.
//
// Blind spots match the sibling AST policies: an aliased time import
// (`t "time"`) evades the selector match. Comments and strings do not, by
// virtue of the AST walk.
func PolicyNoTimeNowInCore(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err //coverage:ignore WalkGoFiles errors only on a filesystem walk failure; not reachable with a valid tree root.
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		pkg := path.Dir(f.Path)
		tier, known := layerTier(pkg)
		if !known || tier < coreMinTier {
			// Edge (tier 0–1) or untiered (the layering policy already
			// forces untiered packages to be placed); not core scope.
			continue
		}
		if _, ok := noTimeNowExempt[pkg]; ok {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "time" || !timeNowSelectors[sel.Sel.Name] {
				return true
			}
			out = append(out, Violation{
				Policy: "no-time-now-in-core",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: pkg + " (core tier " + strconv.Itoa(tier) + ") calls time." + sel.Sel.Name +
					"; logical time must be injected from the edge (tier <= 1), not minted in the domain core" +
					" (if this is operational/perf timing, add a documented entry to noTimeNowExempt)",
			})
			return true
		})
	}
	return out, nil
}
