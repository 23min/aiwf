package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// verbScaffoldCLIPrefix scopes the guard to the verb layer: every aiwf
// verb lives under internal/cli/, including both documented
// non-members. A non-verb package that legitimately calls a wrapped
// primitive (a stress scenario, a diagnostic tool) is not a verb and
// is deliberately out of this guard's scope — the AC is "no verb
// hand-rolls the scaffold", not "the primitive is unreachable".
const verbScaffoldCLIPrefix = "internal/cli/"

// verbScaffoldCliutilPrefix is the package the two scaffolds' shared
// helpers and their wrapped primitives live in today. Detection keys
// on `cliutil.`-qualified selectors, so cliutil's own unqualified
// internal calls are excluded for free; the relocation anchor asserts
// the primitives are still declared here (see below).
const verbScaffoldCliutilPrefix = "internal/cli/cliutil/"

// verbScaffold names one of the two per-verb scaffolds E-0072
// single-sourced. A verb reconstructs the scaffold inline iff it calls
// one of its primitives directly instead of routing through the shared
// helper; the primitive set is the detection key.
type verbScaffold struct {
	// name is the human label used in the finding Detail.
	name string
	// helper is the shared entry point a verb should call instead.
	helper string
	// primitives are the cliutil func names that constitute the
	// scaffold. The detection is an OR over the set — a verb that calls
	// any one of them directly has hand-rolled (part of) the scaffold;
	// requiring all of them would miss a partial re-inline.
	primitives []string
	// allow maps a repo-relative forward-slash path to the rationale
	// for a documented non-member. The rationale travels beside the
	// exemption so the two never drift apart.
	allow map[string]string
}

// verbScaffolds is the closed set of scaffolds the guard pins. Adding a
// third scaffold is a deliberate edit here, not an open extension point.
func verbScaffolds() []verbScaffold {
	return []verbScaffold{
		{
			name:       "diagnostic block",
			helper:     "cliutil.BeginVerbDiag",
			primitives: []string{"ResolveLogger", "EmitVerbOutcome"},
			allow: map[string]string{
				// upgrade emits two decoupled diagnostic outcomes — an
				// install.completed the moment the binary lands, then a
				// deferred, installSucceeded-guarded install.failed — both
				// under a custom "install" prefix and decoupled from Run's
				// own final exit code. BeginVerbDiag's single deferred
				// finish(&code, &sha) captures one outcome at return and
				// can't express either the split emit or the custom prefix.
				"internal/cli/upgrade/upgrade.go": "emits two decoupled diagnostic outcomes under a custom \"install\" prefix (install.completed before the update re-exec, a guarded install.failed) — BeginVerbDiag's single deferred finish can't express it; documented M-0278 non-member",
			},
		},
	}
}

// verbScaffoldViolation is the single Violation construction site for
// this policy: every firing path (re-inline detection and relocation
// anchor) funnels through it, so one firing fixture covers the policy
// id for the firing-fixture-presence meta-gate.
func verbScaffoldViolation(file string, line int, detail string) Violation {
	return Violation{
		Policy: "verb-scaffold-single-seam",
		File:   file,
		Line:   line,
		Detail: detail,
	}
}

// PolicyVerbScaffoldSingleSeam asserts M-0280: no verb under
// internal/cli/ hand-rolls either of the two scaffolds E-0072
// single-sourced (the diagnostic block behind cliutil.BeginVerbDiag,
// the root/actor prelude behind cliutil.ResolvePrelude). Detection is
// per-scaffold: a `cliutil.`-qualified call to any of the scaffold's
// wrapped primitives, from a non-allowlisted file, is a re-inline.
//
// Two checks:
//
//  1. Re-inline detection — every file under internal/cli/ (except the
//     cliutil package itself, whose unqualified internal calls the
//     qualified key never matches) is scanned for direct primitive
//     calls; a documented non-member is exempted by an allowlist entry
//     carrying its rationale.
//  2. Relocation anchor — every keyed primitive must still be declared
//     in package cliutil. Detection keys on the `cliutil.`-qualified
//     selector, so a future cliutil split that relocates a primitive
//     would leave the key matching nothing and the guard green
//     vacuously; the anchor turns that silent rot into a loud failure
//     that forces the guard's package key to be updated with the move.
//
// Known blind spots, consistent with the repo's other AST policies
// (e.g. atomic_write_chokepoint.go, commit_construction_seam.go): an
// aliased cliutil import (`cu "…/cliutil"`) and a from-scratch
// reimplementation of a primitive's internals (rather than a call to
// it) are not matched. The realistic regression — a copy-paste of the
// existing block, which does call the primitives — is caught.
func PolicyVerbScaffoldSingleSeam(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err //coverage:ignore WalkGoFiles errors only on a filesystem walk failure; not reachable with a valid tree root.
	}
	scaffolds := verbScaffolds()
	var out []Violation
	fset := token.NewFileSet()

	// declaredInCliutil collects package cliutil's top-level func names
	// for the relocation anchor.
	declaredInCliutil := map[string]bool{}

	for _, f := range files {
		if !strings.HasPrefix(f.Path, verbScaffoldCLIPrefix) {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		if strings.HasPrefix(f.Path, verbScaffoldCliutilPrefix) {
			for _, decl := range astFile.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
					declaredInCliutil[fn.Name.Name] = true
				}
			}
			// cliutil's own files call the primitives unqualified, which
			// the cliutil-qualified detection below never matches — nothing
			// to scan here for re-inlines.
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
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "cliutil" {
				return true
			}
			for _, s := range scaffolds {
				if !containsString(s.primitives, sel.Sel.Name) {
					continue
				}
				if _, exempt := s.allow[f.Path]; exempt {
					continue
				}
				out = append(out, verbScaffoldViolation(f.Path, fset.Position(call.Pos()).Line,
					"calls cliutil."+sel.Sel.Name+" directly, hand-rolling the "+s.name+
						" — route through "+s.helper+" (or allowlist this file with a rationale)"))
			}
			return true
		})
	}

	for _, s := range scaffolds {
		for _, prim := range s.primitives {
			if !declaredInCliutil[prim] {
				out = append(out, verbScaffoldViolation(verbScaffoldCliutilPrefix, 0,
					"primitive cliutil."+prim+" is no longer declared in package cliutil — the "+s.name+
						" guard keys on the cliutil-qualified selector; relocating the primitive requires "+
						"updating this guard's package key so it keeps firing"))
			}
		}
	}

	return out, nil
}

// containsString reports whether s contains v.
func containsString(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
