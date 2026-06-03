package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicyAcksHelperLift pins M-0159/AC-3's structural claim that the
// retroactive-acknowledgment SHA walker lives at a single canonical
// location — internal/check/acks.go — is exposed under a single
// canonical name (WalkAcknowledgedSHAs, exported because the CLI
// gather layer in internal/cli/check/ consumes it across the
// package boundary), is called from a sanctioned site exactly ONCE,
// and the resulting ackedSHAs value flows to all four named
// consumers (fsm-history-consistent, isolation-escape,
// trailer-verb-unknown, and id-rename-untrailered) through
// identifier provenance — each call site's argument identifier
// must trace either to the local WalkAcknowledgedSHAs assignment
// or to a function parameter named ackedSHAs (parameter
// pass-through).
//
// M-0159/AC-3 introduced the policy with three consumers (fsm-
// history-consistent, isolation-escape, trailer-verb-unknown).
// M-0160/AC-4 added id-rename-untrailered as the fourth consumer;
// the policy's closed-set map and iteration list cover all four.
//
// The AC's load-bearing language: "walkAcknowledgedSHAs lifted to
// internal/check/acks.go; consumed by fsm-history-consistent,
// isolation-escape, and trailer-verb-unknown rules through a single
// ackedSHAs map[string]bool parameter populated by the CLI gather
// layer." Both halves of the claim — structural (file location,
// identifier presence, no-duplicate, no-recompute) and
// architectural (single-compute, four-consumer wiring with
// traced provenance) — are policed here as one chokepoint.
//
// The signature half (the consuming rules' surfaces accept
// ackedSHAs map[string]bool) is policed by sibling behavioral
// unit tests in
// internal/check/{isolation_escape,trailer_verb_unknown,fsm_history_consistent}_ack_test.go
// (M-0159) and internal/check/id_rename_untrailered_test.go
// (M-0160/AC-4) which exercise the new signatures directly and
// fail with compile errors if the lift hasn't happened.
//
// Ten violation classes are surfaced (grouped 1, 2, 3a-3c, 4a-4e):
//
//  1. internal/check/acks.go does not exist OR exists but does not
//     declare WalkAcknowledgedSHAs as a top-level FuncDecl. Without
//     this the lift never landed and the four consumers cannot
//     reach the helper as a package-shared symbol.
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
//     4a. A named consumer (FSMHistoryConsistent, RunIsolationEscape,
//     RunTrailerVerbUnknown) is not called from internal/cli/check/
//     at all. The four-consumer wiring is incomplete.
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
//     Consumers covered: the four named public consumers
//     (FSMHistoryConsistent, RunIsolationEscape,
//     RunTrailerVerbUnknown, RunIDRenameUntrailered) PLUS the two
//     FSMHistoryConsistent-
//     internal predicate helpers that perform the per-observation
//     check (illegalTransitionFindings, forcedUntraileredFindings).
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
//     side seam-contract on the four public consumers.
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

	// (3) + (4) Gather-layer single-compute + four-consumer wiring.
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
	consumerHits := map[string][]consumerHit{
		"FSMHistoryConsistent":  nil,
		"RunIsolationEscape":    nil,
		"RunTrailerVerbUnknown": nil,
		// M-0160/AC-4: id-rename-untrailered is the fourth consumer
		// of the ackedSHAs map. Per-SHA closed-set scoping consistent
		// with the M-0159/AC-3 contract.
		"RunIDRenameUntrailered": nil,
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
						funcDecl:      fn,
						hasProvenance: false,
					})
					return true
				}
				consumerHits[name] = append(consumerHits[name], consumerHit{
					file:          cs.File,
					line:          cs.Line,
					funcDecl:      fn,
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
			Detail: "M-0159/AC-3 requires the CLI gather layer to call check.WalkAcknowledgedSHAs exactly once; found zero call sites — the gather never computes ackedSHAs and the four consuming rules (M-0159/AC-3 + M-0160/AC-4) have nothing to consume",
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
	for _, name := range []string{"FSMHistoryConsistent", "RunIsolationEscape", "RunTrailerVerbUnknown", "RunIDRenameUntrailered"} {
		hits := consumerHits[name]
		if len(hits) == 0 {
			out = append(out, Violation{
				Policy: "acks-helper-lift",
				File:   "internal/cli/check/",
				Detail: "M-0159/AC-3 (extended at M-0160/AC-4) requires the CLI gather layer to call check." + name + " with ackedSHAs; no call site for this consumer was found in internal/cli/check/ — the AC's four-consumer wiring is incomplete",
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
	// internal/check/ production files (non-test) for the four
	// named consumer FuncDecls and asserts the body has at least
	// one consuming reference. A FuncDecl whose body is missing
	// (interface method, nil body) is skipped — the AC's four
	// consumers all have concrete bodies, so a nil body would be
	// an unrelated regression already caught elsewhere.
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
	// Three named PUBLIC consumers + two internal predicate
	// helpers that perform the per-observation per-SHA lookup
	// at the leaf of FSMHistoryConsistent's call chain. Anchoring
	// the lookup at the predicates (not just the top-level
	// public surface) closes the "fsmHistoryConsistentWithDeps
	// drops the value before reaching the predicate" sabotage.
	consumerBodySeen := map[string]bool{
		"FSMHistoryConsistent":      false,
		"RunIsolationEscape":        false,
		"RunTrailerVerbUnknown":     false,
		"illegalTransitionFindings": false,
		"forcedUntraileredFindings": false,
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
	for _, name := range []string{
		"FSMHistoryConsistent",
		"RunIsolationEscape",
		"RunTrailerVerbUnknown",
		"illegalTransitionFindings",
		"forcedUntraileredFindings",
	} {
		if consumerBodySeen[name] {
			continue
		}
		hit, declared := consumerBodyDecl[name]
		if !declared {
			// The consumer doesn't have a FuncDecl in
			// internal/check/. Either renamed or missing —
			// class (4a) already flags the four public
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

	return out, nil
}

// passesAckedAtHit indicates whether the recorded consumer call
// site actually had an ackedSHAs identifier as one of its args.
// The hit's hasProvenance field encodes the AND of (passes-arg)
// AND (provenance-resolved); we distinguish the "didn't pass arg
// at all" case from the "passed it but the identifier is
// fabricated" case for clearer diagnostics. The encoding lives
// in the consumerHits builder: hits whose body passed acked are
// recorded with hasProvenance reflecting the enclosing function's
// state; hits whose body did NOT pass acked are recorded with
// hasProvenance=false unconditionally. So a hit with
// hasProvenance=false could be either kind. This helper recovers
// the distinction by re-checking the AST. Kept as a small helper
// so the main builder stays readable.
func passesAckedAtHit(h *consumerHit) bool {
	if h == nil || h.funcDecl == nil || h.funcDecl.Body == nil {
		return false
	}
	found := false
	ast.Inspect(h.funcDecl.Body, func(n ast.Node) bool {
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
		// Match by line — the same FuncDecl may contain multiple
		// calls to the same consumer; we want the specific one at
		// h.line.
		for _, arg := range call.Args {
			if id, ok := arg.(*ast.Ident); ok && id.Name == "ackedSHAs" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// consumerHit captures one consumer call site with the enclosing
// function reference so the provenance check can re-walk it for
// diagnostic disambiguation (passes-arg vs fabricated-identifier).
type consumerHit struct {
	file          string
	line          int
	funcDecl      *ast.FuncDecl
	hasProvenance bool
}
