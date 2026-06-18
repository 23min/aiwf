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

// PolicyVersionSingleSource asserts that the running binary's version
// has exactly one source of truth: the Stamp global in
// internal/version, resolved everywhere through version.Current(). It
// forbids any *other* production package from declaring a
// package-level string var named like a version/stamp global.
//
// This is the chokepoint behind G-0235 candidate B (the C1
// version-source split). Before the fix, internal/cli carried a
// parallel `var Version` ldflags target that the human-facing `aiwf
// version` print read directly, while the JSON envelope read
// version.Current() (buildinfo). A `make install` binary — stamped via
// ldflags but built from a working tree — therefore reported two
// different strings: the stamp on the human path, "(devel)" in JSON.
// Relocating the stamp into internal/version.Stamp routed every
// surface through version.Current(); this policy keeps it that way by
// failing CI if a second version global reappears anywhere outside
// internal/version.
//
// The companion PolicyEnvelopeVersionSource pins the *read* side (the
// JSON envelope's Version field must come from version.Current());
// this policy pins the *declaration* side (no parallel source to read
// in the first place). Together they make "one binary, one version
// string on every surface" a mechanical guarantee rather than a
// present-state fact.
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
