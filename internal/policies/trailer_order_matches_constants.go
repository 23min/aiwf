package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// PolicyTrailerOrderMatchesConstants asserts set-equality between
// the Trailer* constants declared in internal/gitops/trailers.go and
// the identifiers referenced inside the trailerOrder slice literal
// in the same file.
//
// Why this matters (G-0195 reviewer pass, follow-up direction):
// G-0195 closed the test-side drift — internal/cli/integration's
// canonicalTrailerKeys map went from hand-maintained to derived
// via gitops.CanonicalTrailerKeys(), which reads from trailerOrder.
// But trailerOrder itself is hand-maintained against the const
// block. A new TrailerXyz constant landing in the const block
// without an append to trailerOrder is silently invisible to every
// downstream consumer (CanonicalTrailerKeys returns the wrong set;
// SortedTrailers sends the new key to the unknown-keys tail;
// TestTrailerShapePerMutatingVerb's membership check silently
// drops the new trailer). Same drift shape, one layer up.
//
// The policy AST-parses internal/gitops/trailers.go and:
//
//   - collects every const identifier matching `Trailer*` whose
//     value is a string literal — the const-block set;
//   - locates the `var trailerOrder = []string{…}` declaration and
//     collects the identifiers inside the slice literal — the
//     order-slice set;
//   - reports a violation per set-difference (constants missing
//     from trailerOrder, identifiers in trailerOrder that don't
//     resolve to a Trailer* constant).
//
// The policy stays narrow: scope is internal/gitops/trailers.go
// specifically; if a future refactor splits the constants across
// files, the policy fails closed with a discoverability violation
// rather than silently widening.
func PolicyTrailerOrderMatchesConstants(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, false)
	if err != nil {
		return nil, err
	}
	var target *FileEntry
	for i := range files {
		if files[i].Path == "internal/gitops/trailers.go" {
			target = &files[i]
			break
		}
	}
	if target == nil {
		return []Violation{{
			Policy: "trailer-order-matches-constants",
			File:   "internal/gitops/trailers.go",
			Detail: "file not found; policy cannot scan for drift",
		}}, nil
	}

	fset := token.NewFileSet()
	astFile, perr := parser.ParseFile(fset, target.Path, target.Contents, parser.SkipObjectResolution)
	if perr != nil {
		return []Violation{{
			Policy: "trailer-order-matches-constants",
			File:   target.Path,
			Detail: "parse error: " + perr.Error(),
		}}, nil
	}

	consts := map[string]int{}   // identifier → line
	orderIDs := map[string]int{} // identifier → line
	var orderLine int

	for _, decl := range astFile.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		switch gen.Tok {
		case token.CONST:
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				// Assumes single-name single-value specs (the shape every
				// Trailer* constant in the current trailers.go uses). A
				// parallel-assignment form (`const TrailerFoo, TrailerBar
				// = "a", "b"`) would be silently skipped. If a future
				// author adopts that form, this branch must grow paired
				// iteration over Names/Values to keep the drift caught.
				if len(vs.Names) != 1 || len(vs.Values) != 1 {
					continue
				}
				name := vs.Names[0].Name
				if !strings.HasPrefix(name, "Trailer") {
					continue
				}
				lit, ok := vs.Values[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				consts[name] = fset.Position(vs.Pos()).Line
			}
		case token.VAR:
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if len(vs.Names) != 1 || vs.Names[0].Name != "trailerOrder" {
					continue
				}
				orderLine = fset.Position(vs.Pos()).Line
				if len(vs.Values) != 1 {
					continue
				}
				comp, ok := vs.Values[0].(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, elt := range comp.Elts {
					id, ok := elt.(*ast.Ident)
					if !ok {
						continue
					}
					orderIDs[id.Name] = fset.Position(id.Pos()).Line
				}
			}
		}
	}

	if len(consts) == 0 {
		return []Violation{{
			Policy: "trailer-order-matches-constants",
			File:   target.Path,
			Detail: "no Trailer* string constants found; policy cannot scan for drift",
		}}, nil
	}
	if len(orderIDs) == 0 {
		return []Violation{{
			Policy: "trailer-order-matches-constants",
			File:   target.Path,
			Line:   orderLine,
			Detail: "trailerOrder slice literal not found or empty; policy cannot scan for drift",
		}}, nil
	}

	var out []Violation

	// Constants missing from trailerOrder.
	var missingFromOrder []string
	for name := range consts {
		if _, ok := orderIDs[name]; !ok {
			missingFromOrder = append(missingFromOrder, name)
		}
	}
	sort.Strings(missingFromOrder)
	for _, name := range missingFromOrder {
		out = append(out, Violation{
			Policy: "trailer-order-matches-constants",
			File:   target.Path,
			Line:   consts[name],
			Detail: "constant " + name + " is not listed in trailerOrder; SortedTrailers will send it to the unknown-keys tail and CanonicalTrailerKeys will omit it",
		})
	}

	// Identifiers in trailerOrder that don't resolve to a Trailer* constant.
	var phantomInOrder []string
	for name := range orderIDs {
		if _, ok := consts[name]; !ok {
			phantomInOrder = append(phantomInOrder, name)
		}
	}
	sort.Strings(phantomInOrder)
	for _, name := range phantomInOrder {
		out = append(out, Violation{
			Policy: "trailer-order-matches-constants",
			File:   target.Path,
			Line:   orderIDs[name],
			Detail: "trailerOrder references " + name + " which is not a Trailer* string constant in this file",
		})
	}

	return out, nil
}
