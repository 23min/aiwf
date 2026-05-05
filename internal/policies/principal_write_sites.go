package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicyPrincipalWriteSitesGuardHuman asserts that every function
// that writes the TrailerPrincipal or TrailerOnBehalfOf trailer
// references "human/" somewhere in its body. The kernel rule:
// principal and on-behalf-of are human-only by design.
//
// The verb-side coherence check (CheckTrailerCoherence) and the
// standing rules already enforce this at runtime. The policy's
// job is to catch a regression at the source: a new site that
// writes one of these trailers without the matching guard.
func PolicyPrincipalWriteSitesGuardHuman(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		// gitops.go owns the trailer constants themselves; not a
		// write site for the trailers.
		if strings.HasPrefix(f.Path, "internal/gitops/") {
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
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				continue
			}
			body := string(f.Contents[start:end])
			writesPrincipal := strings.Contains(body, "TrailerPrincipal") ||
				strings.Contains(body, "TrailerOnBehalfOf")
			if !writesPrincipal {
				continue
			}
			// Skip read sites: a function that mentions the constant
			// in a SWITCH or comparison (e.g. inside indexing helpers)
			// is reading, not writing. Heuristic: writing happens
			// inside a Trailer{Key: ..., Value: ...} composite or as
			// the LHS of an assignment to plan.Trailers. We approximate
			// by requiring `Key: gitops.TrailerPrincipal` or
			// `Key: gitops.TrailerOnBehalfOf` in the body.
			isWrite := strings.Contains(body, "Key: gitops.TrailerPrincipal") ||
				strings.Contains(body, "Key: gitops.TrailerOnBehalfOf")
			if !isWrite {
				continue
			}
			hasGuard := strings.Contains(body, "human/") ||
				strings.Contains(body, "actorIsHuman") ||
				strings.Contains(body, "actorIsNonHuman")
			if !hasGuard {
				out = append(out, Violation{
					Policy: "principal-write-sites-guard-human",
					File:   f.Path,
					Line:   fset.Position(fn.Pos()).Line,
					Detail: fn.Name.Name +
						" writes TrailerPrincipal or TrailerOnBehalfOf but does not reference \"human/\"; the principal slot is human-only and the call must be guarded",
				})
			}
		}
	}
	return out, nil
}
