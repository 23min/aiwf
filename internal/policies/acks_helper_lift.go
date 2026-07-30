package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// ackedSHAsConsumers is the closed set of exported rules the CLI gather
// layer must call with the single gather-computed ackedSHAs map. It is
// the policy's copy of the list internal/check/acks.go enumerates in
// WalkAcknowledgedSHAs' doc comment, and class 4f fails when the two
// disagree — so a rule that starts reading ackedSHAs without being added
// here and documented there cannot land silently.
var ackedSHAsConsumers = []string{
	"FSMHistoryConsistent",
	"RunIsolationEscape",
	"RunTrailerVerbUnknown",
	"RunIDRenameUntrailered",
	"RunOrphanedAICommits",
	"RunPromoteOnWrongBranch",
}

// ackedSHAsBodyConsumers is the set whose bodies class 4d requires to
// reference the map. It is every name in ackedSHAsConsumers plus the two
// leaf predicates at the end of FSMHistoryConsistent's forwarding chain,
// which are where that chain's per-observation lookup actually happens.
// FSMHistoryConsistent is here despite not indexing the map itself: 4d
// accepts forwarding it to a helper as a consuming reference, which is the
// shape that rule uses.
var ackedSHAsBodyConsumers = []string{
	"FSMHistoryConsistent",
	"RunIsolationEscape",
	"RunTrailerVerbUnknown",
	"RunIDRenameUntrailered",
	"RunOrphanedAICommits",
	"RunPromoteOnWrongBranch",
	"illegalTransitionFindings",
	"forcedUntraileredFindings",
}

// PolicyAcksHelperLift pins M-0159/AC-3's structural claim that the
// retroactive-acknowledgment SHA walker lives at a single canonical
// location — internal/check/acks.go — is exposed under a single
// canonical name (WalkAcknowledgedSHAs, exported because the CLI
// gather layer in internal/cli/check/ consumes it across the
// package boundary), is called from a sanctioned site exactly ONCE,
// and the resulting ackedSHAs value flows to every consumer named
// in ackedSHAsConsumers through identifier provenance — each call
// site's argument identifier
// must trace either to the local WalkAcknowledgedSHAs assignment
// or to a function parameter named ackedSHAs (parameter
// pass-through).
//
// The AC's load-bearing language: "walkAcknowledgedSHAs lifted to
// internal/check/acks.go; consumed by fsm-history-consistent,
// isolation-escape, and trailer-verb-unknown rules through a single
// ackedSHAs map[string]bool parameter populated by the CLI gather
// layer." Both halves of the claim — structural (file location,
// identifier presence, no-duplicate, no-recompute) and
// architectural (single-compute, consumer wiring with traced
// provenance) — are policed here as one chokepoint.
//
// The signature half (the consuming rules' surfaces accept
// ackedSHAs map[string]bool) is policed by sibling behavioral
// unit tests in
// internal/check/{isolation_escape,trailer_verb_unknown,fsm_history_consistent}_ack_test.go
// (M-0159) and internal/check/id_rename_untrailered_test.go
// (M-0160/AC-4) which exercise the new signatures directly and
// fail with compile errors if the lift hasn't happened.
//
// The violation classes are grouped 1, 2, 3a-3c, 4a-4f:
//
//  1. internal/check/acks.go does not exist OR exists but does not
//     declare WalkAcknowledgedSHAs as a top-level FuncDecl. Without
//     this the lift never landed and the consumers cannot reach
//     the helper as a package-shared symbol.
//
//  2. internal/check/fsm_history_consistent.go still declares
//     walkAcknowledgedSHAs (lowercased — the pre-lift name) at the
//     top-level. The lift must MOVE the helper, not duplicate it.
//
//     3a. Zero calls to WalkAcknowledgedSHAs found at any sanctioned
//     production site. The gather layer never computes acks.
//
//     3b. Multiple calls in internal/cli/check/ non-test files. The
//     gather computes redundantly — violates the "single ackedSHAs
//     ... populated by the CLI gather layer" wording.
//
//     3c. Any call to WalkAcknowledgedSHAs (bare identifier, same
//     package) in internal/check/ non-test files EXCEPT acks.go
//     itself. A rule recomputing the set internally defeats the
//     single-compute claim regardless of whether the rule also
//     accepts ackedSHAs as a parameter. Closes the "swap to the
//     lifted symbol but keep computing internally" sabotage.
//
//     4a. A consumer named in ackedSHAsConsumers is not called from
//     internal/cli/check/ at all. The wiring is incomplete.
//
//     4b. A consumer call site does not receive an `ackedSHAs`
//     identifier as one of its arguments. The convention-driven
//     identifier name is the AC's seam contract.
//
//     4c. A consumer call site receives an `ackedSHAs` identifier
//     BUT the enclosing function provides no provenance for it:
//     the identifier is neither a parameter of the enclosing
//     function NOR the LHS of an assignment whose RHS calls
//     check.WalkAcknowledgedSHAs. The identifier is fabricated
//     (zero-value var declaration, free identifier, etc.); the
//     gather-layer single-compute does not actually flow into
//     this consumer. Closes the "uninitialized identifier of the
//     right name" sabotage.
//
//     4d. A consumer's function body does not reference
//     `ackedSHAs` in a consuming context — either an IndexExpr
//     `ackedSHAs[X]` (the per-SHA lookup pattern the rules
//     actually use) OR a CallExpr argument (the forward-to-helper
//     pattern, e.g., FSMHistoryConsistent forwards to
//     fsmHistoryConsistentWithDeps which then performs the
//     lookup). A green-phase regression that adds the parameter
//     to the signature but ignores it in the body — or silences
//     it via `_ = ackedSHAs` — would otherwise pass classes 1-4c.
//     The gather-layer's value never reaches the rule's silencing
//     logic; AC-3/AC-4's behavioral promise breaks silently.
//     Closes the "consumer ignores parameter" sabotage at the
//     policy layer (the behavioral tests also catch it; this is
//     the structural backstop).
//
//     Consumers covered: ackedSHAsBodyConsumers — the exported
//     rules plus the two FSMHistoryConsistent-internal predicate
//     helpers that perform the per-observation check
//     (illegalTransitionFindings, forcedUntraileredFindings).
//
//     4e. A call to one of the leaf predicate helpers
//     (illegalTransitionFindings or forcedUntraileredFindings)
//     does NOT pass an `ackedSHAs` identifier as the ackedSHAs
//     argument — the call site passes `nil`, a CompositeLit
//     (`map[string]bool{}`), a CallExpr, or any other non-Ident
//     shape. The body-level class 4d guarantees the predicate
//     READS its parameter; 4e guarantees the FORWARDER actually
//     PASSES the gather-layer value through. Without 4e, the
//     sabotage `forcedUntraileredFindings(observations, nil)` at
//     the call site in fsmHistoryConsistentWithDeves leaves
//     class 4d satisfied (body still has IndexExpr `ackedSHAs[X]`,
//     just reading from the nil-map's always-false return) and
//     the silencing is mechanically broken. The behavioral tests
//     would catch this; class 4e closes the gap at the policy
//     layer too. The convention-driven match (must be an
//     *ast.Ident named "ackedSHAs") mirrors class 4b's gather-
//     side seam-contract on the exported consumers.
//
// The policy is intentionally narrow — file locations, symbol
// names, call shape, identifier provenance at known paths. A
// future refactor that legitimately moves the helper or renames
// the convention requires updating this policy in the same commit;
// that visibility is the chokepoint.
func PolicyAcksHelperLift(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, false)
	if err != nil {
		return nil, err
	}

	var (
		acksFile          *FileEntry
		fsmHistoryFile    *FileEntry
		cliCheckProdFiles []*FileEntry
		checkInternalProd []*FileEntry
		hasCliCheck       bool
	)
	for i := range files {
		f := &files[i]
		switch f.Path {
		case "internal/check/acks.go":
			acksFile = f
		case "internal/check/fsm_history_consistent.go":
			fsmHistoryFile = f
		}
		isTest := strings.HasSuffix(f.Path, "_test.go")
		switch {
		case strings.HasPrefix(f.Path, "internal/cli/check/") && !isTest:
			cliCheckProdFiles = append(cliCheckProdFiles, f)
			hasCliCheck = true
		case strings.HasPrefix(f.Path, "internal/check/") && !isTest && f.Path != "internal/check/acks.go":
			checkInternalProd = append(checkInternalProd, f)
		}
	}

	var out []Violation

	// (1) acks.go must exist and declare WalkAcknowledgedSHAs.
	if acksFile == nil {
		out = append(out, Violation{
			Policy: "acks-helper-lift",
			File:   "internal/check/acks.go",
			Detail: "M-0159/AC-3 requires the retroactive-acknowledgment SHA walker to live at internal/check/acks.go (lifted from fsm_history_consistent.go); file is missing",
		})
	} else {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, acksFile.AbsPath, acksFile.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
		}
		found := false
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if fn.Name.Name == "WalkAcknowledgedSHAs" {
				found = true
				break
			}
		}
		if !found {
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   "internal/check/acks.go",
				Detail: "M-0159/AC-3 requires acks.go to declare WalkAcknowledgedSHAs as a top-level exported function (the CLI gather layer in internal/cli/check/ consumes it across the package boundary)",
			})
		}
	}

	// (2) fsm_history_consistent.go must NOT still declare the
	// pre-lift walkAcknowledgedSHAs at the top level. The lift
	// is a move, not a copy.
	if fsmHistoryFile != nil {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, fsmHistoryFile.AbsPath, fsmHistoryFile.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if fn.Name.Name == "walkAcknowledgedSHAs" || fn.Name.Name == "WalkAcknowledgedSHAs" {
				out = append(out, Violation{
					Policy: "acks-helper-lift",
					File:   "internal/check/fsm_history_consistent.go",
					Line:   fset.Position(fn.Pos()).Line,
					Detail: "M-0159/AC-3 lifts the SHA walker to internal/check/acks.go; this declaration is a leftover from the pre-lift location and defeats the AC's single-helper guarantee — delete it",
				})
				break
			}
		}
	}

	// (3) + (4) Gather-layer single-compute + consumer wiring.
	if !hasCliCheck {
		out = append(out, Violation{
			Policy: "acks-helper-lift",
			File:   "internal/cli/check/",
			Detail: "M-0159/AC-3 expects the CLI gather layer at internal/cli/check/ but the directory was not found in the walk; tree shape unexpected",
		})
		return out, nil
	}

	type callSite struct {
		File string
		Line int
	}

	// 3c: scan internal/check/ non-test files (except acks.go) for
	// any call to WalkAcknowledgedSHAs (bare identifier — same
	// package). Each call is a rule-internal recompute that defeats
	// the single-compute claim.
	for _, f := range checkInternalProd {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if id.Name != "WalkAcknowledgedSHAs" {
				return true
			}
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "M-0159/AC-3 forbids rule-internal recompute of WalkAcknowledgedSHAs (call must come from the CLI gather layer ONCE so the value flows in through the ackedSHAs parameter); this call recomputes the set and defeats the single-compute claim",
			})
			return true
		})
	}

	// 3a/3b + 4*: scan internal/cli/check/ non-test files.
	var walkCallSites []callSite
	consumerCalledAt := map[string]callSite{}
	consumerHits := map[string][]consumerHit{}
	for _, name := range ackedSHAsConsumers {
		consumerHits[name] = nil
	}

	for _, f := range cliCheckProdFiles {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
		}
		// Pre-scan: count WalkAcknowledgedSHAs calls (selector form;
		// cross-package call). Record call sites for 3a/3b diagnostic.
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if pkg.Name == "check" && sel.Sel.Name == "WalkAcknowledgedSHAs" {
				walkCallSites = append(walkCallSites, callSite{
					File: f.Path,
					Line: fset.Position(call.Pos()).Line,
				})
			}
			return true
		})

		// FuncDecl-scoped pass for 4*: each FuncDecl is the
		// provenance unit. For every consumer call inside it that
		// passes `ackedSHAs`, the same FuncDecl must declare
		// `ackedSHAs` as a parameter OR assign it from a
		// WalkAcknowledgedSHAs call.
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			declaresAckedAsParam := false
			if fn.Type != nil && fn.Type.Params != nil {
				for _, field := range fn.Type.Params.List {
					for _, name := range field.Names {
						if name.Name == "ackedSHAs" {
							declaresAckedAsParam = true
						}
					}
				}
			}
			assignsAckedFromWalk := false
			// rhsCallsWalk reports whether any expression in rhs
			// contains a CallExpr to check.WalkAcknowledgedSHAs.
			// Shared helper between the AssignStmt path
			// (`ackedSHAs := ...` / `ackedSHAs = ...`) and the
			// GenDecl-with-initializer path (`var ackedSHAs = ...`),
			// both of which are idiomatic Go shapes a green-phase
			// might use to bind the gather result to the local
			// identifier. Without GenDecl support the policy fires
			// false 4c violations on the var-form.
			rhsCallsWalk := func(rhs []ast.Expr) bool {
				for _, expr := range rhs {
					hit := false
					ast.Inspect(expr, func(m ast.Node) bool {
						call, ok := m.(*ast.CallExpr)
						if !ok {
							return true
						}
						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}
						pkg, ok := sel.X.(*ast.Ident)
						if !ok {
							return true
						}
						if pkg.Name == "check" && sel.Sel.Name == "WalkAcknowledgedSHAs" {
							hit = true
							return false
						}
						return true
					})
					if hit {
						return true
					}
				}
				return false
			}
			if fn.Body != nil {
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					switch s := n.(type) {
					case *ast.AssignStmt:
						// AssignStmt path: `ackedSHAs := <expr>`
						// or `ackedSHAs = <expr>` or
						// `ackedSHAs, err := <expr>`.
						assignedLocally := false
						for _, lhs := range s.Lhs {
							if id, ok := lhs.(*ast.Ident); ok && id.Name == "ackedSHAs" {
								assignedLocally = true
							}
						}
						if assignedLocally && rhsCallsWalk(s.Rhs) {
							assignsAckedFromWalk = true
						}
					case *ast.DeclStmt:
						// GenDecl-with-initializer path:
						// `var ackedSHAs = check.WalkAcknowledgedSHAs(...)`
						// or `var ackedSHAs map[string]bool = check.WalkAcknowledgedSHAs(...)`.
						// `var ackedSHAs map[string]bool` alone
						// (no initializer) is NOT provenance —
						// that's the fabricated-identifier
						// sabotage case the policy must keep
						// catching.
						gd, ok := s.Decl.(*ast.GenDecl)
						if !ok || gd.Tok != token.VAR {
							return true
						}
						for _, spec := range gd.Specs {
							vs, ok := spec.(*ast.ValueSpec)
							if !ok {
								continue
							}
							if len(vs.Values) == 0 {
								continue // declaration only — fabricated path
							}
							assignedLocally := false
							for _, name := range vs.Names {
								if name.Name == "ackedSHAs" {
									assignedLocally = true
								}
							}
							if assignedLocally && rhsCallsWalk(vs.Values) {
								assignsAckedFromWalk = true
							}
						}
					}
					return true
				})
			}
			hasProvenance := declaresAckedAsParam || assignsAckedFromWalk

			// Now walk the body for consumer calls.
			if fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if pkg.Name != "check" {
					return true
				}
				name := sel.Sel.Name
				if _, tracked := consumerHits[name]; !tracked {
					return true
				}
				cs := callSite{
					File: f.Path,
					Line: fset.Position(call.Pos()).Line,
				}
				if _, already := consumerCalledAt[name]; !already {
					consumerCalledAt[name] = cs
				}
				passesAcked := false
				for _, arg := range call.Args {
					if id, ok := arg.(*ast.Ident); ok && id.Name == "ackedSHAs" {
						passesAcked = true
						break
					}
				}
				if !passesAcked {
					consumerHits[name] = append(consumerHits[name], consumerHit{
						file:          cs.File,
						line:          cs.Line,
						call:          call,
						hasProvenance: false,
					})
					return true
				}
				consumerHits[name] = append(consumerHits[name], consumerHit{
					file:          cs.File,
					line:          cs.Line,
					call:          call,
					hasProvenance: hasProvenance,
				})
				return true
			})
		}
	}

	// (3a/3b) WalkAcknowledgedSHAs call cardinality at the CLI
	// gather layer.
	switch len(walkCallSites) {
	case 0:
		out = append(out, Violation{
			Policy: "acks-helper-lift",
			File:   "internal/cli/check/",
			Detail: "M-0159/AC-3 requires the CLI gather layer to call check.WalkAcknowledgedSHAs exactly once; found zero call sites — the gather never computes ackedSHAs and every consuming rule has nothing to consume",
		})
	case 1:
		// happy path
	default:
		for _, cs := range walkCallSites {
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   cs.File,
				Line:   cs.Line,
				Detail: "M-0159/AC-3 requires the CLI gather layer to call check.WalkAcknowledgedSHAs exactly once (single-compute claim); this is one of multiple call sites — consolidate",
			})
		}
	}

	// (4a/4b/4c) Each consumer must (a) be called from the gather
	// layer, (b) receive an ackedSHAs arg, (c) have provenance for
	// that arg within the enclosing function.
	for _, name := range ackedSHAsConsumers {
		hits := consumerHits[name]
		if len(hits) == 0 {
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   "internal/cli/check/",
				Detail: "M-0159/AC-3 (extended at M-0160/AC-4) requires the CLI gather layer to call check." + name + " with ackedSHAs; no call site for this consumer was found in internal/cli/check/ — the AC's consumer wiring is incomplete",
			})
			continue
		}
		// If ANY hit has the ackedSHAs arg AND provenance, the
		// consumer is wired. The AC permits multiple call sites
		// (e.g., a recursive helper) as long as the property
		// holds at one. Track per-site violations otherwise.
		var anyWired bool
		var firstNoArg *consumerHit
		var firstNoProvenance *consumerHit
		for i := range hits {
			h := &hits[i]
			switch {
			case !h.hasProvenance && !passesAckedAtHit(h):
				if firstNoArg == nil {
					firstNoArg = h
				}
			case !h.hasProvenance:
				if firstNoProvenance == nil {
					firstNoProvenance = h
				}
			default:
				anyWired = true
			}
		}
		if anyWired {
			continue
		}
		switch {
		case firstNoProvenance != nil:
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   firstNoProvenance.file,
				Line:   firstNoProvenance.line,
				Detail: "M-0159/AC-3: check." + name + " receives an `ackedSHAs` identifier here but the enclosing function provides no provenance for it (no parameter named `ackedSHAs`, no assignment from check.WalkAcknowledgedSHAs); the identifier is fabricated and the gather-layer single-compute does not actually flow into this consumer",
			})
		case firstNoArg != nil:
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   firstNoArg.file,
				Line:   firstNoArg.line,
				Detail: "M-0159/AC-3 requires check." + name + " to receive ackedSHAs as one of its arguments (convention-driven match: an *ast.Ident named 'ackedSHAs'); this call site does not pass it — the gather-layer single-compute does not flow into this consumer",
			})
		}
	}

	// (4d) Each named consumer's function body must actually USE
	// the ackedSHAs parameter. Closes the N1 sabotage class
	// surfaced in AC-3 GREEN-phase dual review: a regression that
	// adds the parameter to the signature but ignores it in the
	// body — or silences "unused" via `_ = ackedSHAs` — would pass
	// classes 1-4c. Two consuming contexts are accepted:
	//
	//   - IndexExpr: `ackedSHAs[X]` — the per-SHA lookup pattern
	//     the rules use directly (RunIsolationEscape and
	//     RunTrailerVerbUnknown).
	//
	//   - CallExpr argument: `helper(..., ackedSHAs, ...)` — the
	//     forward-to-helper pattern (FSMHistoryConsistent forwards
	//     to fsmHistoryConsistentWithDeps which then performs the
	//     lookup via illegalTransitionFindings).
	//
	// Both shapes are present in the green-phase implementation;
	// either alone satisfies the policy. The check scans the
	// internal/check/ production files (non-test) for the
	// ackedSHAsBodyConsumers FuncDecls and asserts the body has at
	// least one consuming reference. A FuncDecl whose body is missing
	// (interface method, nil body) is skipped — every consumer has a
	// concrete body, so a nil body would be an unrelated regression
	// already caught elsewhere.
	consumerFiles := map[string]*FileEntry{}
	for _, f := range checkInternalProd {
		consumerFiles[f.Path] = f
	}
	// fsm_history_consistent.go IS in checkInternalProd; the other
	// consumers' files are also there. Walk the slice.
	type bodyHit struct {
		name string
		file string
		line int
	}
	// ackedSHAsBodyConsumers: the exported rules plus the two internal
	// predicate helpers that perform the per-observation per-SHA lookup
	// at the leaf of FSMHistoryConsistent's call chain. Anchoring
	// the lookup at the predicates (not just the top-level
	// public surface) closes the "fsmHistoryConsistentWithDeps
	// drops the value before reaching the predicate" sabotage.
	consumerBodySeen := map[string]bool{}
	for _, name := range ackedSHAsBodyConsumers {
		consumerBodySeen[name] = false
	}
	consumerBodyDecl := map[string]bodyHit{}
	for _, f := range checkInternalProd {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			name := fn.Name.Name
			if _, tracked := consumerBodySeen[name]; !tracked {
				continue
			}
			consumerBodyDecl[name] = bodyHit{
				name: name,
				file: f.Path,
				line: fset.Position(fn.Pos()).Line,
			}
			if fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.IndexExpr:
					// `ackedSHAs[<expr>]` — the per-SHA lookup
					// pattern.
					if id, ok := x.X.(*ast.Ident); ok && id.Name == "ackedSHAs" {
						consumerBodySeen[name] = true
						return false
					}
				case *ast.CallExpr:
					// `helper(..., ackedSHAs, ...)` — forward
					// pattern.
					for _, arg := range x.Args {
						if id, ok := arg.(*ast.Ident); ok && id.Name == "ackedSHAs" {
							consumerBodySeen[name] = true
							return false
						}
					}
				}
				return true
			})
		}
	}
	for _, name := range ackedSHAsBodyConsumers {
		if consumerBodySeen[name] {
			continue
		}
		hit, declared := consumerBodyDecl[name]
		if !declared {
			// The consumer doesn't have a FuncDecl in
			// internal/check/. Either renamed or missing —
			// class (4a) already flags the exported
			// surfaces from the gather side; for the two
			// internal predicate helpers a missing FuncDecl
			// is unusual but not a separate AC-policed
			// concern, so don't duplicate.
			continue
		}
		out = append(out, Violation{
			Policy: "acks-helper-lift",
			File:   hit.file,
			Line:   hit.line,
			Detail: "M-0159/AC-3/AC-4: " + name + " has the `ackedSHAs` parameter on its signature but the body never reads it through a consuming pattern (no IndexExpr `ackedSHAs[X]`, no CallExpr passing it as an argument); the gather-layer-computed value is dropped on the floor and the rule's silencing logic is unreachable — close the sabotage by adding the per-SHA lookup or forwarding the parameter to the function that performs the lookup",
		})
	}

	// (4e) For each call to one of the leaf predicate helpers
	// (illegalTransitionFindings, forcedUntraileredFindings) in
	// internal/check/ non-test files, the ackedSHAs arg position
	// must be an *ast.Ident named "ackedSHAs". A nil literal, a
	// CompositeLit `map[string]bool{}`, a CallExpr, or any
	// non-Ident shape fires. Closes the "call-site drops the
	// parameter" sabotage at the forwarder seam (the body-level
	// 4d guarantees the predicate reads its parameter; this
	// guarantees the forwarder actually passes a real one).
	predicateArgPositions := map[string]int{
		"illegalTransitionFindings": 1, // (observations, ackedSHAs)
		"forcedUntraileredFindings": 1, // (observations, ackedSHAs)
	}
	for _, f := range checkInternalProd {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			argPos, tracked := predicateArgPositions[id.Name]
			if !tracked {
				return true
			}
			if argPos >= len(call.Args) {
				return true
			}
			arg := call.Args[argPos]
			argIdent, isIdent := arg.(*ast.Ident)
			if isIdent && argIdent.Name == "ackedSHAs" {
				return true
			}
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "M-0159/AC-4: call to " + id.Name + " at this site does not pass `ackedSHAs` as the ackedSHAs argument — the forwarder dropped the gather-layer-computed value before reaching the predicate, breaking silencing while leaving class 4d (body reads the parameter) spuriously satisfied (the predicate now reads from a nil/empty map and silently fails to silence)",
			})
			return true
		})
	}

	// (4f) The consumer roster is stated in three places — the two vars
	// above and WalkAcknowledgedSHAs' doc comment — and a reader trusts
	// the prose. Assert all three name the same set, and that a rule which
	// either reads the map or takes it as a parameter is in the list whose
	// classes cover that shape. Without it a new consumer can land while
	// every roster keeps describing the old set.
	listViolations, lerr := policeConsumerListAgreement(acksFile, checkInternalProd)
	if lerr != nil {
		return nil, lerr
	}
	out = append(out, listViolations...)

	// G-0239: extend the same single-compute / one-consumer / no-rule-
	// internal-recompute contract to WalkAcknowledgedSHAEntities — the
	// per-(SHA, entity) ack walker added by G-0231 item 3 and consumed
	// by the provenance-untrailered-entity-commit rule.
	entViolations, eerr := policeEntitiesWalkerSingleCompute(acksFile, cliCheckProdFiles, checkInternalProd)
	if eerr != nil {
		return nil, eerr
	}
	out = append(out, entViolations...)

	return out, nil
}

// policeConsumerListAgreement implements class 4f: the ackedSHAs consumer
// set is enumerated in three places — ackedSHAsConsumers,
// ackedSHAsBodyConsumers, and WalkAcknowledgedSHAs' doc comment in
// internal/check/acks.go — and they must not drift apart. Three directions,
// each closing a different way a new consumer lands unpoliced:
//
// Direction 1 — the doc comment names exactly the union of the two vars, no
// more and no fewer. Set equality, on whole identifier tokens: a doc that
// omits a policed consumer understates the set, and one that names a
// function no longer wired overstates it. This direction cannot judge what
// the surrounding prose *asserts* about those names — only that the roster
// matches. Prose that lists the right names while claiming the opposite
// about them is a review concern, not a mechanical one.
//
// Direction 2 — every function in internal/check/ whose body indexes
// `ackedSHAs[...]` is in ackedSHAsBodyConsumers, so a rule that starts
// reading the map cannot sit behind a doc that still describes the old set.
//
// Direction 3 — every exported function in internal/check/ whose signature
// carries an `ackedSHAs map[string]bool` parameter is in
// ackedSHAsConsumers. Direction 2 alone forces only the body-consumer list,
// which leaves the forwarder shape unpoliced: an exported rule that threads
// the map to a leaf predicate without indexing it satisfies direction 2 the
// moment the *leaf* is listed, while the rule's own gather-layer wiring
// (classes 4a-4c) is never checked. Keying on the signature rather than the
// body also catches a rule that reads the map through a struct field or
// under a different local name.
func policeConsumerListAgreement(acksFile *FileEntry, checkInternalProd []*FileEntry) ([]Violation, error) {
	var out []Violation
	if acksFile == nil {
		return nil, nil // class 1 already reports the missing file.
	}

	// Direction 1: WalkAcknowledgedSHAs' doc comment names exactly the
	// union of the two policed sets. Scoped to that function's own doc, so
	// a mention elsewhere in acks.go does not satisfy the requirement.
	fset := token.NewFileSet()
	astFile, perr := parser.ParseFile(fset, acksFile.AbsPath, acksFile.Contents, parser.ParseComments)
	if perr != nil {
		return nil, fmt.Errorf("parsing %s: %w", acksFile.Path, perr)
	}
	var walkerDoc string
	for _, decl := range astFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "WalkAcknowledgedSHAs" {
			continue
		}
		if fn.Doc != nil {
			walkerDoc = fn.Doc.Text()
		}
		break
	}
	documented := map[string]bool{}
	for _, name := range ackedSHAsConsumers {
		documented[name] = true
	}
	for _, name := range ackedSHAsBodyConsumers {
		documented[name] = true
	}
	// Whole-token match: a substring hit would let a four-character prefix,
	// or a longer identifier that merely contains the name, stand in for it.
	docTokens := map[string]bool{}
	for _, tok := range identifierTokenPattern.FindAllString(walkerDoc, -1) {
		docTokens[tok] = true
	}
	for _, name := range slices.Sorted(maps.Keys(documented)) {
		if docTokens[name] {
			continue
		}
		out = append(out, Violation{
			Policy: "acks-helper-lift",
			File:   "internal/check/acks.go",
			Detail: "M-0159/AC-3 class 4f: " + name + " is policed as an ackedSHAs consumer but WalkAcknowledgedSHAs' doc comment does not name it — that doc is where the consumer set is enumerated for readers, so add it there in the same commit",
		})
	}
	// The reverse: a name the doc still presents as a consumer but which no
	// longer appears in either policed list. Left unchecked, the doc keeps
	// advertising wiring that nothing verifies.
	for _, tok := range slices.Sorted(maps.Keys(docTokens)) {
		if documented[tok] || !ackedSHAsDocRosterPattern.MatchString(tok) {
			continue
		}
		out = append(out, Violation{
			Policy: "acks-helper-lift",
			File:   "internal/check/acks.go",
			Detail: "M-0159/AC-3 class 4f: WalkAcknowledgedSHAs' doc comment names " + tok + " as an ackedSHAs consumer, but it is in neither ackedSHAsConsumers nor ackedSHAsBodyConsumers — either add it to the list that applies or drop it from the doc, so the enumerated set and the policed set stay the same set",
		})
	}

	// Direction 2: every ackedSHAs[...] reader in internal/check/ is policed.
	// Methods are scanned too, not just plain functions, so a receiver-bearing
	// rule cannot slip past — class 4d scans receivers for the same reason.
	bodyPoliced := map[string]bool{}
	for _, name := range ackedSHAsBodyConsumers {
		bodyPoliced[name] = true
	}
	gatherPoliced := map[string]bool{}
	for _, name := range ackedSHAsConsumers {
		gatherPoliced[name] = true
	}
	for _, f := range checkInternalProd {
		ffset := token.NewFileSet()
		fileAST, err := parser.ParseFile(ffset, f.AbsPath, f.Contents, parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", f.Path, err)
		}
		for _, decl := range fileAST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			name := fn.Name.Name
			if fn.Body != nil && !bodyPoliced[name] && bodyIndexesAckedSHAs(fn.Body) {
				out = append(out, Violation{
					Policy: "acks-helper-lift",
					File:   f.Path,
					Line:   ffset.Position(fn.Pos()).Line,
					Detail: "M-0159/AC-3 class 4f: " + name + " reads the ackedSHAs map but is not in ackedSHAsBodyConsumers, so class 4d never checks that its body keeps reading it — a refactor that drops the read would silently stop silencing acknowledged commits; add it to ackedSHAsBodyConsumers in internal/policies/acks_helper_lift.go and to WalkAcknowledgedSHAs' doc comment",
				})
			}
			// Direction 3: exported + takes the map as a parameter =>
			// the gather layer calls it, so classes 4a-4c must cover it.
			if fn.Recv == nil && ast.IsExported(name) && !gatherPoliced[name] && declaresAckedSHAsParam(fn) {
				out = append(out, Violation{
					Policy: "acks-helper-lift",
					File:   f.Path,
					Line:   ffset.Position(fn.Pos()).Line,
					Detail: "M-0159/AC-3 class 4f: " + name + " is exported and takes an `ackedSHAs map[string]bool` parameter, so the CLI gather layer feeds it, but it is not in ackedSHAsConsumers — classes 4a-4c therefore never check that the gather layer passes the single computed map to it, and a call site that drops the argument would silently stop silencing acknowledged commits; add it to ackedSHAsConsumers in internal/policies/acks_helper_lift.go and to WalkAcknowledgedSHAs' doc comment",
				})
			}
		}
	}
	return out, nil
}

// identifierTokenPattern matches whole Go identifiers in doc-comment prose,
// so direction 1 compares rosters token-by-token rather than by substring.
var identifierTokenPattern = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// ackedSHAsDocRosterPattern recognizes the identifier shapes that name a
// consumer in WalkAcknowledgedSHAs' doc — the exported `Run*` rules and the
// two unexported leaf predicates. It keeps direction 1's reverse check from
// treating ordinary prose words as claimed consumers, which is why the check
// is a roster comparison and not a scan for arbitrary capitalized words.
var ackedSHAsDocRosterPattern = regexp.MustCompile(`^(Run[A-Z]|FSMHistoryConsistent$|illegalTransitionFindings$|forcedUntraileredFindings$)`)

// bodyIndexesAckedSHAs reports whether body contains an `ackedSHAs[...]`
// index expression — the per-SHA lookup shape the consuming rules use.
func bodyIndexesAckedSHAs(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		idx, ok := n.(*ast.IndexExpr)
		if !ok {
			return true
		}
		if id, ok := idx.X.(*ast.Ident); ok && id.Name == "ackedSHAs" {
			found = true
			return false
		}
		return true
	})
	return found
}

// declaresAckedSHAsParam reports whether fn takes a parameter named
// ackedSHAs whose type is map[string]bool — the gather layer's seam
// contract, matched on the type as well as the name so an unrelated
// parameter that happens to share the name does not count.
func declaresAckedSHAsParam(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Params == nil {
		return false
	}
	for _, field := range fn.Type.Params.List {
		mt, ok := field.Type.(*ast.MapType)
		if !ok {
			continue
		}
		kt, kok := mt.Key.(*ast.Ident)
		vt, vok := mt.Value.(*ast.Ident)
		if !kok || !vok || kt.Name != "string" || vt.Name != "bool" {
			continue
		}
		for _, n := range field.Names {
			if n.Name == "ackedSHAs" {
				return true
			}
		}
	}
	return false
}

// policeEntitiesWalkerSingleCompute extends the acks-helper-lift
// chokepoint to WalkAcknowledgedSHAEntities, the per-(SHA, entity) ack
// walker added by G-0231 item 3 and consumed by the provenance-
// untrailered-entity-commit rule. It pins the SAME single-compute /
// one-consumer / no-recompute contract WalkAcknowledgedSHAs carries,
// scoped to the three structural classes that apply to a single-
// consumer walker (G-0239):
//
//	E1. internal/check/acks.go must declare WalkAcknowledgedSHAEntities
//	    as a top-level exported FuncDecl — the CLI gather layer consumes
//	    it across the package boundary exactly as WalkAcknowledgedSHAs.
//
//	E2. The CLI gather layer (internal/cli/check/) must call
//	    check.WalkAcknowledgedSHAEntities exactly once. Zero means the
//	    per-(SHA, entity) ack map is never computed and the
//	    provenance-untrailered rule degrades to "nil map → no
//	    suppression", re-firing historical findings as errors; more
//	    than one is a redundant recompute.
//
//	E3. No internal/check/ non-test file except acks.go may call
//	    WalkAcknowledgedSHAEntities (bare identifier, same package) — a
//	    rule recomputing the map internally defeats the single-compute
//	    claim.
//
// Unlike WalkAcknowledgedSHAs, whose consumers are enumerated in
// ackedSHAsConsumers, the entities walker has a SINGLE consumer inside
// the gather flow (RunProvenanceCheck), so the provenance-wiring
// classes (4a-4f) do not apply here; the
// parameter flow into RunProvenanceCheck is policed by that rule's
// behavioral ack tests.
//
// "Inside the gather flow" is the operative scope: internal/verb also
// calls this walker directly, to answer whether an acknowledgment it is
// about to write already exists (M-0281/AC-4). That call takes no map
// from the gather layer and feeds no rule, so it neither participates in
// nor threatens the single-compute invariant — but it does mean the
// walker has a caller this policy deliberately does not police.
//
// A future second consumer or a relocation of the walker
// requires updating this helper in the same commit — that visibility is
// the chokepoint.
func policeEntitiesWalkerSingleCompute(acksFile *FileEntry, cliCheckProdFiles, checkInternalProd []*FileEntry) ([]Violation, error) {
	const walker = "WalkAcknowledgedSHAEntities"
	var out []Violation

	// E1: acks.go declares the walker as a top-level FuncDecl.
	// (acksFile == nil is already flagged by the SHA-walker class 1;
	// don't duplicate the missing-file violation here.)
	if acksFile != nil {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, acksFile.AbsPath, acksFile.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
		}
		found := false
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if fn.Name.Name == walker {
				found = true
				break
			}
		}
		if !found {
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   "internal/check/acks.go",
				Detail: "G-0239 requires acks.go to declare " + walker + " as a top-level exported function (the per-(SHA, entity) ack walker added by G-0231 item 3; the CLI gather layer consumes it across the package boundary, exactly as WalkAcknowledgedSHAs)",
			})
		}
	}

	// E3: no rule-internal recompute. Scan internal/check/ non-test
	// files (acks.go is already excluded from checkInternalProd) for a
	// bare-identifier call to the walker (same-package call shape).
	for _, f := range checkInternalProd {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || id.Name != walker {
				return true
			}
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "G-0239 forbids rule-internal recompute of " + walker + " (the call must come from the CLI gather layer ONCE so the per-(SHA, entity) map flows in through the ackedSHAEntities parameter); this call recomputes the map and defeats the single-compute claim",
			})
			return true
		})
	}

	// E2: exactly one gather-layer call. Scan internal/cli/check/ non-
	// test files for the cross-package selector call
	// check.WalkAcknowledgedSHAEntities.
	type gatherCall struct {
		file string
		line int
	}
	var gatherCalls []gatherCall
	for _, f := range cliCheckProdFiles {
		fset := token.NewFileSet()
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			return nil, perr
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
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if pkg.Name == "check" && sel.Sel.Name == walker {
				gatherCalls = append(gatherCalls, gatherCall{f.Path, fset.Position(call.Pos()).Line})
			}
			return true
		})
	}
	switch len(gatherCalls) {
	case 0:
		out = append(out, Violation{
			Policy: "acks-helper-lift",
			File:   "internal/cli/check/",
			Detail: "G-0239 requires the CLI gather layer to call check." + walker + " exactly once; found zero call sites — the per-(SHA, entity) ack map is never computed and provenance-untrailered-entity-commit degrades to 'nil map → no suppression', re-firing acknowledged historical findings as errors",
		})
	case 1:
		// happy path
	default:
		for _, cs := range gatherCalls {
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   cs.file,
				Line:   cs.line,
				Detail: "G-0239 requires the CLI gather layer to call check." + walker + " exactly once (single-compute claim); this is one of multiple call sites — consolidate",
			})
		}
	}

	return out, nil
}

// passesAckedAtHit reports whether the recorded consumer call site
// actually had an ackedSHAs identifier among its arguments.
//
// The hit's hasProvenance field encodes (passes-arg AND
// provenance-resolved), so hasProvenance=false alone cannot say which
// half failed. This recovers the distinction from the recorded call
// expression, which is why consumerHit stores the call rather than its
// enclosing function: one function can call several consumers, and a
// sibling call's ackedSHAs argument would otherwise answer for this one,
// reporting a dropped argument (class 4b) as a fabricated identifier
// (class 4c).
func passesAckedAtHit(h *consumerHit) bool {
	if h == nil || h.call == nil {
		return false
	}
	for _, arg := range h.call.Args {
		if id, ok := arg.(*ast.Ident); ok && id.Name == "ackedSHAs" {
			return true
		}
	}
	return false
}

// consumerHit captures one consumer call site, holding the call
// expression itself so the provenance check can distinguish
// passes-arg from fabricated-identifier without re-walking the
// enclosing function. Recording the enclosing FuncDecl instead
// would force a re-walk that cannot tell this call from a sibling
// call to another consumer in the same function.
type consumerHit struct {
	file          string
	line          int
	call          *ast.CallExpr
	hasProvenance bool
}
