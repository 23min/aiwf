package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// zeroPadDigitVerb matches a zero-pad numeric format verb inside a
// Printf-style format string — %0*d (width supplied as an arg) or
// %0Nd (a literal width, e.g. %04d). This is the shape
// entity.AllocateID's own formatting uses (`"%s%0*d"`,
// internal/entity/allocate.go), and the shape the deleted
// internal/verb/import.go helpers duplicated before G-0426's fix
// (M-0269/AC-1). An AC sub-id (`"AC-%d"`) is not zero-padded and so
// does not match — AC ids live in a different, positionally-scoped id
// space with no entity.Kind entry, genuinely outside
// entity.AllocateID's domain.
var zeroPadDigitVerb = regexp.MustCompile(`%0[0-9*]*d`)

// PolicyMintIDsViaAllocate asserts that internal/verb/ — the layer
// responsible for entity id allocation — never independently
// reconstructs the "scan existing ids, +1, zero-pad format" shape
// entity.AllocateID (internal/entity/allocate.go) already owns.
// G-0426 (M-0269/AC-1) was exactly this: import.go hand-rolled its
// own id-numbering helpers instead of calling entity.AllocateID, so
// its auto-id path never saw the cross-branch id view. This policy is
// the mechanical backstop against that bug class recurring.
//
// Detection is a zero-pad-digit-verb Sprintf call — the fmt.Sprintf
// shape both entity.AllocateID and the deleted helpers used. A file
// whose only zero-pad Sprintf calls are legitimate re-display of an
// already-existing id (parsed from on-disk text, not a highest+1
// scan) is allowlisted by path with a one-line rationale, matching
// the repo's other AST policies (e.g. atomic_write_chokepoint.go).
//
// Scope is internal/verb/*.go only — the verb layer's own
// package, where id allocation logic belongs; internal/entity/ (the
// allocator's own definition) and internal/cli/** (the dispatcher
// layer, which never touches id numbering directly) are out of
// scope by construction, not by exemption.
func PolicyMintIDsViaAllocate(root string) ([]Violation, error) {
	// File-path allowlist. Key is the repo-relative forward-slash
	// path; value is the rationale.
	allow := map[string]string{
		// padToCanonical zero-pads the digit tail of an id already
		// present in on-disk text (regex-captured from an existing
		// reference), not a highest+1 scan over the tree — re-display
		// of an existing id, not minting a new one. It exists precisely
		// because entity.Canonicalize refuses narrow legacy ids below
		// the per-kind grammar minimum that rewidth is migrating (see
		// padToCanonical's own doc comment, internal/verb/rewidth.go).
		"internal/verb/rewidth.go": "padToCanonical re-pads an id already present in on-disk text; not a highest+1 mint",
	}
	files, err := WalkGoFiles(root, true)
	if err != nil { //coverage:ignore WalkGoFiles only errors on a filesystem-level fault (unreadable dir, mid-walk file removal) — not portably triggerable in a unit test; mirrors the identical unexercised guard on every sibling AST policy (e.g. atomic_write_chokepoint.go)
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") {
			continue
		}
		if _, ok := allow[f.Path]; ok {
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
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "fmt" || sel.Sel.Name != "Sprintf" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			format, uerr := strconv.Unquote(lit.Value)
			if uerr != nil || !zeroPadDigitVerb.MatchString(format) {
				return true
			}
			out = append(out, Violation{
				Policy: "mint-ids-via-allocate",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "fmt.Sprintf with a zero-pad id-format verb outside entity.AllocateID; " +
					"route id minting through entity.AllocateID (G-0426) or allowlist the file with a rationale",
			})
			return true
		})
	}
	return out, nil
}
