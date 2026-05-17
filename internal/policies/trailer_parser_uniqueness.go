package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// trailerParserFuncNames is the closed set of function names that
// would constitute a re-duplication of the canonical trailer parser
// (gitops.ParseTrailers). Includes the exported and unexported
// variants in both prior shapes — pre-M-0113 the canonical name was
// unexported parseTrailers in gitops, and an exported ParseTrailerLines
// lived in cliutil. Any of these declared outside internal/gitops/
// is a regression of M-0113/AC-2.
var trailerParserFuncNames = map[string]bool{
	"ParseTrailers":     true,
	"parseTrailers":     true,
	"ParseTrailerLines": true,
	"parseTrailerLines": true,
}

// PolicyTrailerParserUniqueness asserts the trailer-line parser is
// declared only in internal/gitops/ (M-0113/AC-2). Any other package
// declaring a top-level function whose name matches one of the
// trailer-parser-shape names is a re-duplication of the canonical
// implementation and is flagged.
//
// Test files are out of scope: an external _test.go file may want
// to wrap or shadow the parser for fixture purposes without that
// being a re-duplication of the production symbol. WalkGoFiles is
// invoked with excludeTests=true.
//
// Background: pre-M-0113 the parser lived twice — gitops.parseTrailers
// (unexported) and cliutil.ParseTrailerLines (exported, byte-identical
// body). M-0113 consolidated the two into the single exported
// gitops.ParseTrailers and this policy is the mechanical guard
// against the duplication reappearing.
func PolicyTrailerParserUniqueness(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasPrefix(f.Path, "internal/gitops/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if !trailerParserFuncNames[fn.Name.Name] {
				continue
			}
			out = append(out, Violation{
				Policy: "trailer-parser-uniqueness",
				File:   f.Path,
				Line:   fset.Position(fn.Pos()).Line,
				Detail: "function " + fn.Name.Name + " duplicates the canonical gitops.ParseTrailers; route callers through gitops.ParseTrailers instead",
			})
		}
	}
	return out, nil
}
