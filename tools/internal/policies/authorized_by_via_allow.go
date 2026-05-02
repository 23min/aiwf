package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicyAuthorizedByWriteSitesUseAllow asserts that every function
// that writes the TrailerAuthorizedBy trailer also references
// `Allow(` or `gateAndDecorate` — the kernel's authorization-rule
// entrypoint and its cmd-side wrapper. A site that hand-stamps an
// authorize SHA without running the allow-rule check is the same
// regression class the standing rule (provenance-authorization-
// missing / -out-of-scope) catches at read time, but at write
// time.
func PolicyAuthorizedByWriteSitesUseAllow(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasPrefix(f.Path, "tools/internal/gitops/") {
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
			writesAuthorizedBy := strings.Contains(body, "Key: gitops.TrailerAuthorizedBy")
			if !writesAuthorizedBy {
				continue
			}
			referencesAllow := strings.Contains(body, "Allow(") ||
				strings.Contains(body, "gateAndDecorate") ||
				strings.Contains(body, "decorateAndFinish") ||
				strings.Contains(body, "AllowResult")
			// Also accept when the file contains `gateAndDecorate` —
			// the function may delegate to a helper in the same file.
			fileBody := string(f.Contents)
			if !referencesAllow && (strings.Contains(fileBody, "gateAndDecorate") ||
				strings.Contains(fileBody, "AllowResult")) {
				referencesAllow = true
			}
			if !referencesAllow {
				out = append(out, Violation{
					Policy: "authorized-by-via-allow",
					File:   f.Path,
					Line:   fset.Position(fn.Pos()).Line,
					Detail: fn.Name.Name +
						" writes TrailerAuthorizedBy without going through Allow / gateAndDecorate; the allow-rule check must run before stamping a scope SHA",
				})
			}
		}
	}
	return out, nil
}
