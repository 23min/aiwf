package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// gitopsImportPath is the package whose commit-construction primitives
// this policy tracks.
const gitopsImportPath = "github.com/23min/aiwf/internal/gitops"

// commitPrimitives are every exported gitops function that builds a
// commit. Naming only CommitVerbChange would hold the seam against the
// call a verb is supposed to make while ignoring the three that would
// bypass it — a caller reaching for a lower-level primitive is exactly
// the shape this policy exists to catch.
var commitPrimitives = map[string]bool{
	"CommitVerbChange": true,
	"CommitTree":       true,
	"Commit":           true,
	"CommitAllowEmpty": true,
}

// verbCommitCall names the primitive a verb commit is supposed to route
// through, for the violation messages.
const verbCommitCall = "gitops.CommitVerbChange"

// coherenceGuardCall is the guard that must run ahead of it.
const coherenceGuardCall = "CheckForceTrailerCoherence"

// commitSite is one function that builds a verb commit.
type commitSite struct {
	File      string
	Line      int
	Func      string
	HasGuard  bool
	IsTheSeam bool
}

// verbCommitSites returns every production function whose body builds a
// verb commit, with whether that body also runs the coherence guard.
// The second result names files that contain the commit call but do not
// parse — skipping those would hide the very violation being scanned
// for, so the policy reports them rather than passing over them.
//
// Returned separately from the policy so a test can assert the scan
// still finds a site: a scan over an empty population reports no
// violations, which is indistinguishable from a tree that passed.
func verbCommitSites(root string) (sites []commitSite, unparseable []string, err error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, nil, err
	}
	fset := token.NewFileSet()
	for _, f := range files {
		// gitops composes its own primitives; the seam is about who
		// calls into the package from outside it.
		if strings.HasPrefix(f.Path, "internal/gitops/") {
			continue
		}
		if !mentionsCommitPrimitive(f.Contents) {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			unparseable = append(unparseable, f.Path)
			continue
		}
		// The local name gitops is bound to in this file. An aliased
		// import renames the selector, so a textual scan for
		// "gitops.Commit…" would miss the call entirely.
		local := gitopsLocalName(astFile)
		if local == "" {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			commits, guarded := scanFuncBody(fn.Body, local)
			if !commits {
				continue
			}
			sites = append(sites, commitSite{
				File:     f.Path,
				Line:     fset.Position(fn.Pos()).Line,
				Func:     fn.Name.Name,
				HasGuard: guarded,
				// fn.Recv == nil because the seam is the package-level
				// verb.Apply. A method that happens to be named Apply is
				// a second commit site wearing the seam's name, which is
				// the thing this policy is looking for rather than an
				// exception to it.
				IsTheSeam: fn.Recv == nil && fn.Name.Name == "Apply" && strings.HasPrefix(f.Path, "internal/verb/"),
			})
		}
	}
	return sites, unparseable, nil
}

// mentionsCommitPrimitive is the cheap pre-filter deciding which files
// are worth parsing — and, for a file that fails to parse, whether its
// failure is worth reporting.
func mentionsCommitPrimitive(contents []byte) bool {
	s := string(contents)
	for name := range commitPrimitives {
		if strings.Contains(s, name) {
			return true
		}
	}
	return false
}

// gitopsLocalName returns the identifier gitops is bound to in this
// file — its alias where one is given, else the package name. Empty
// when the file does not import it at all.
func gitopsLocalName(astFile *ast.File) string {
	for _, imp := range astFile.Imports {
		if strings.Trim(imp.Path.Value, `"`) != gitopsImportPath {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "gitops"
	}
	return ""
}

// scanFuncBody reports whether this function builds a commit through a
// gitops primitive, and whether it also calls the coherence guard.
//
// Both are answered from call expressions rather than from the body's
// text, so a primitive named in a comment or in a string literal is not
// mistaken for a call — internal/verb/apply.go names CommitTree in
// comments inside function bodies, which a textual scan cannot tell
// apart from the real thing.
func scanFuncBody(body *ast.BlockStmt, gitopsLocal string) (commits, guarded bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			pkg, isIdent := fn.X.(*ast.Ident)
			if isIdent && pkg.Name == gitopsLocal && commitPrimitives[fn.Sel.Name] {
				commits = true
			}
		case *ast.Ident:
			if fn.Name == coherenceGuardCall {
				guarded = true
			}
		}
		return true
	})
	return commits, guarded
}

// PolicyCoherenceGuardChokepoint holds the sovereign-force guard at the
// verb-commit seam (M-0291/AC-3).
//
// The no-bypass property is structural rather than policed once the
// guard sits inside verb.Apply: any caller reaching Apply is covered
// without being enumerated, including callers outside the CLI
// dispatcher layer. What remains checkable, and what this policy holds,
// is that the seam stays singular and that the guard's name appears in
// it.
//
// Two clauses, because either alone is satisfiable while the property
// is false. A single commit site proves nothing if the guard has been
// removed from it; a present guard proves nothing if a second function
// builds its own commit alongside.
//
// Scope worth stating, since the doc above would otherwise read as more
// than it is. The scan resolves calls, so an aliased import is caught
// and a primitive named in a comment or a string literal is not
// mistaken for one. It still holds presence rather than ordering: a
// guard placed after the commit call, or one whose error was discarded,
// would satisfy it. Ordering is pinned behaviorally instead, by the
// unmoved-HEAD and no-write-landed assertions in
// internal/verb/apply_coherence_test.go.
//
// It resolves direct calls only. A primitive taken as a function value,
// held in a variable, passed as an argument, or reached through a
// dot-import is not seen. None of those forms appears in the tree, and
// each would be a strange way to build a commit — but a caller
// determined to bypass the seam has them.
func PolicyCoherenceGuardChokepoint(root string) ([]Violation, error) {
	sites, unparseable, err := verbCommitSites(root)
	if err != nil {
		return nil, err
	}

	var out []Violation
	for _, path := range unparseable {
		out = append(out, Violation{
			Policy: "coherence-guard-chokepoint",
			File:   path,
			Line:   0,
			Detail: "file contains " + verbCommitCall + " but does not parse, so it cannot be scanned; " +
				"an unparseable commit site is reported rather than skipped, because skipping it would " +
				"hide exactly the violation this policy looks for",
		})
	}

	if len(sites) == 0 {
		return append(out, Violation{
			Policy: "coherence-guard-chokepoint",
			File:   "internal/verb/apply.go",
			Line:   0,
			Detail: "no verb-commit site found in the tree: the scan is orphaned and cannot fire. " +
				"If the commit seam moved or was renamed, repoint verbCommitCall at the call that now " +
				"builds a commit carrying a verb's provenance trailers",
		}), nil
	}

	for _, s := range sites {
		switch {
		case !s.IsTheSeam:
			out = append(out, Violation{
				Policy: "coherence-guard-chokepoint",
				File:   s.File,
				Line:   s.Line,
				Detail: s.Func + " builds a verb commit outside verb.Apply, so its trailer set never reaches " +
					"the coherence guard. Return a *Plan and let Apply commit it, rather than committing here",
			})
		case !s.HasGuard:
			out = append(out, Violation{
				Policy: "coherence-guard-chokepoint",
				File:   s.File,
				Line:   s.Line,
				Detail: s.Func + " is the commit seam but does not call " + coherenceGuardCall +
					": every verb commit routes through a chokepoint that checks nothing. " +
					"Restore the guard ahead of the commit",
			})
		}
	}
	return out, nil
}
