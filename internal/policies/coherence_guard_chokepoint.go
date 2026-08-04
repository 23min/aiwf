package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// verbCommitCall is the one call that builds a commit carrying a verb's
// provenance trailers. Every other commit-construction primitive in
// gitops is reachable only from inside that package.
const verbCommitCall = "gitops.CommitVerbChange("

// coherenceGuardCall is the guard that must run ahead of it.
const coherenceGuardCall = "CheckSovereignForceCoherence("

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
		if !strings.Contains(string(f.Contents), verbCommitCall) {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			unparseable = append(unparseable, f.Path)
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) { //coverage:ignore defensive: positions come from the same fset and contents the file was parsed from, so they are in range for any file that parsed
				continue
			}
			body := string(f.Contents[start:end])
			if !strings.Contains(body, verbCommitCall) {
				continue
			}
			sites = append(sites, commitSite{
				File:      f.Path,
				Line:      fset.Position(fn.Pos()).Line,
				Func:      fn.Name.Name,
				HasGuard:  strings.Contains(body, coherenceGuardCall),
				IsTheSeam: fn.Name.Name == "Apply" && strings.HasPrefix(f.Path, "internal/verb/"),
			})
		}
	}
	return sites, unparseable, nil
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
// than it is: this is a textual scan, so it holds presence, not
// ordering — a guard placed after the commit call, or one whose error
// was discarded, would satisfy it. Ordering is pinned behaviorally
// instead, by the unmoved-HEAD and no-write-landed assertions in
// internal/verb/apply_coherence_test.go.
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
