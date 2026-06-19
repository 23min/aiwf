package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicyValidateCheckIsNeverWrites asserts that functions whose names
// belong to the read-only "query" naming families — Validate*, Is*,
// Check*, Has* — never call a filesystem- or git-mutating primitive,
// anywhere under internal/.
//
// This widens the naming contract that verbs_validate_then_write.go
// pins for the verb layer (G-0235). A function named IsValid, HasRef,
// CheckTrailerCoherence, or ValidateID promises its caller it is a
// pure query: it reads state and answers a question. If such a
// function quietly writes a cache file, stages a path, or removes a
// directory, the name lies and the lie ships silently — the exact
// "IsValid quietly writes a cache file" regression this policy exists
// to catch.
//
// Forbidden primitives (matched by AST selector, not substring, so a
// read-only sibling that merely shares a prefix — gitops.AddCommitSHA,
// which returns a path's birth commit — is never mistaken for the
// gitops.Add writer):
//
//   - os.WriteFile, os.Create, os.Remove, os.RemoveAll, os.Mkdir,
//     os.MkdirAll, os.Rename, and a write-mode os.OpenFile
//     (O_WRONLY/O_RDWR/O_APPEND/O_CREATE/O_TRUNC).
//   - pathutil.AtomicWriteFile — the sanctioned write path is still a
//     write; a query must not reach even the atomic writer.
//   - gitops.Mv, gitops.Add, gitops.Restore, gitops.Commit,
//     gitops.CommitAllowEmpty, gitops.Init, gitops.StashStaged,
//     gitops.StashPop.
//
// Family membership is word-boundary aware: a name matches a family
// only when the character after the family prefix is uppercase (or the
// name equals the prefix exactly), so Issue (not Is*), Hash (not Has*),
// and similar nouns that continue lowercase are not swept in.
//
// There is deliberately no allowlist: the naming contract admits no
// exception. A function that genuinely must write should not be named
// for a query — rename it (or split the write out) rather than exempt
// it here.
//
// Scope is every non-test production file under internal/ (WalkGoFiles
// already excludes _test.go and the policies package itself). Known
// blind spots match the sibling AST policies: an aliased os import, a
// write primitive reached through a method value, and an os.OpenFile
// flag passed as an opaque variable rather than named os.O_* constants.
func PolicyValidateCheckIsNeverWrites(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err //coverage:ignore WalkGoFiles errors only on a filesystem walk failure; not reachable with a valid tree root.
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/") {
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
			if !inQueryFamily(fn.Name.Name) {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				prim, ok := writePrimitive(call)
				if !ok {
					return true
				}
				out = append(out, Violation{
					Policy: "validate-check-is-never-writes",
					File:   f.Path,
					Line:   fset.Position(call.Pos()).Line,
					Detail: fn.Name.Name + " is a query-family function but calls " + prim +
						"; Validate*/Is*/Check*/Has* names must not mutate filesystem or git state",
				})
				return true
			})
		}
	}
	return out, nil
}

// inQueryFamily reports whether name belongs to a read-only query
// naming family: Validate*, Is*, Check*, Has*. Membership is
// word-boundary aware — the family prefix must be followed by an
// uppercase letter or end the name — so Issue, Hash, and similar nouns
// that merely begin with the letters are excluded.
func inQueryFamily(name string) bool {
	for _, fam := range []string{"Validate", "Is", "Check", "Has"} {
		if name == fam {
			return true
		}
		if rest, ok := strings.CutPrefix(name, fam); ok && rest != "" && isCapitalized(rest) {
			return true
		}
	}
	return false
}

// writePrimitive reports the qualified name of the filesystem- or
// git-mutating primitive a call invokes, if any. The match is on the
// selector's package identifier and method name (e.g. os.WriteFile),
// not a substring, so a read-only sibling that shares a prefix
// (gitops.AddCommitSHA vs gitops.Add) is not misclassified.
func writePrimitive(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	name := sel.Sel.Name
	switch pkg.Name {
	case "os":
		switch name {
		case "WriteFile", "Create", "Remove", "RemoveAll", "Mkdir", "MkdirAll", "Rename":
			return "os." + name, true
		case "OpenFile":
			if len(call.Args) >= 2 && hasOSWriteFlag(call.Args[1]) {
				return "os.OpenFile (write mode)", true
			}
		}
	case "pathutil":
		if name == "AtomicWriteFile" {
			return "pathutil.AtomicWriteFile", true
		}
	case "gitops":
		switch name {
		case "Mv", "Add", "Restore", "Commit", "CommitAllowEmpty", "Init", "StashStaged", "StashPop":
			return "gitops." + name, true
		}
	}
	return "", false
}
