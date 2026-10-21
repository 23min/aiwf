package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// versionGlobalNames is the closed set of package-level identifier
// names that, when declared as a string var, act as a binary-version
// source — an ldflags `-X` target or an alternate version reader. The
// set covers the conventional names a future contributor reaches for
// when (re)introducing a version global. Matching is case-insensitive.
var versionGlobalNames = map[string]bool{
	"version":      true,
	"stamp":        true,
	"buildversion": true,
	"buildstamp":   true,
	"gitversion":   true,
	"appversion":   true,
}

// PolicyVersionSingleSource keeps internal/version.Stamp the only
// version global: it forbids any production file outside
// internal/version (and internal/policies, which WalkGoFiles skips)
// from declaring a package-level var named like a version/stamp
// global and declared `string`, or declared without a type and
// initialized with a string literal.
//
// This is the chokepoint behind G-0235 candidate B (the C1
// version-source split). A `make install` binary carries two
// versions: the ldflags stamp and the one Go records in its build
// info. A second version global lets one surface print the first
// while another prints the second, so the same binary reports two
// strings. Every surface resolves through version.Current(), and this
// policy fails CI if a second such global appears.
//
// The companion PolicyEnvelopeVersionSource pins the *read* side (the
// JSON envelope's Version field must come from a function call's
// `.Version`, in practice version.Current()); this policy pins the
// *declaration* side (no parallel string var to read in the first
// place). Neither catches a non-envelope surface that calls
// runtime/debug.ReadBuildInfo itself or reads a const or a var
// outside the shape above.
//
// Scope: package-level (file-scope) var declarations only. A local
// variable named version inside a function body is not a version
// source and is not flagged. internal/version is the one legitimate
// home for the Stamp global and is allowlisted by path.
func PolicyVersionSingleSource(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err //coverage:ignore WalkGoFiles errors only on a filesystem walk failure; not reachable with a valid tree root.
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		// internal/version is the single legitimate home for the Stamp
		// global; everything else is a parallel source.
		if strings.HasPrefix(filepath.ToSlash(f.Path), "internal/version/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		// Only file-scope declarations — astFile.Decls holds top-level
		// decls; function-body vars never appear here.
		for _, decl := range astFile.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue //coverage:ignore a VAR GenDecl's specs are always *ast.ValueSpec in well-formed Go; defensive guard.
				}
				for i, name := range vs.Names {
					if !versionGlobalNames[strings.ToLower(name.Name)] {
						continue
					}
					if !valueSpecIsString(vs, i) {
						continue
					}
					out = append(out, Violation{
						Policy: "version-single-source",
						File:   f.Path,
						Line:   fset.Position(name.Pos()).Line,
						Detail: "package-level string var " + name.Name +
							" is a parallel binary-version source; the version stamp lives only in internal/version.Stamp and every surface resolves it through version.Current() (G-0235 candidate B)",
					})
				}
			}
		}
	}
	return out, nil
}

// valueSpecIsString reports whether the i-th name in a ValueSpec is
// string-typed: either an explicit `string` type on the spec, or a
// string-literal initializer in the matching position when no type is
// written. This narrows the match to version *string* globals and
// skips, e.g., a `var version int` counter that happens to share the
// name.
func valueSpecIsString(vs *ast.ValueSpec, i int) bool {
	if ident, ok := vs.Type.(*ast.Ident); ok {
		return ident.Name == "string"
	}
	// No explicit type: infer from the initializer in the same slot.
	if vs.Type == nil && i < len(vs.Values) {
		if lit, ok := vs.Values[i].(*ast.BasicLit); ok {
			return lit.Kind == token.STRING
		}
	}
	return false
}
