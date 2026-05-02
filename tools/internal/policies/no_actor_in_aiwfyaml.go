package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// forbiddenAiwfYAMLFieldNames are field names whose presence on
// any aiwfyaml struct would imply storing identity in the project
// config — the I2.5 design explicitly migrated identity to runtime
// derivation (git config user.email). A new field with one of
// these names is a regression.
//
// The list is exact-match against AST field names. Substring
// matching would catch unrelated fields ("ContractActor", "Actor"
// inside a doc string).
var forbiddenAiwfYAMLFieldNames = map[string]bool{
	"Actor":     true,
	"Identity":  true,
	"Principal": true,
	"Agent":     true,
}

// PolicyNoActorFieldsInAiwfYAML asserts that no struct in
// tools/internal/aiwfyaml/ declares a field named after an
// identity-shape concept. Identity is runtime-derived per
// docs/pocv3/design/provenance-model.md; storing it in the YAML
// would re-introduce the bug step 1 of I2.5 fixed.
//
// The package's existing LegacyActor field (a deprecation hatch
// that reads but never writes the old `actor:` key) is whitelisted
// by name.
func PolicyNoActorFieldsInAiwfYAML(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "tools/internal/aiwfyaml/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					if !forbiddenAiwfYAMLFieldNames[name.Name] {
						continue
					}
					out = append(out, Violation{
						Policy: "no-actor-fields-in-aiwfyaml",
						File:   f.Path,
						Line:   fset.Position(name.Pos()).Line,
						Detail: "struct field " + name.Name +
							" — identity must stay runtime-derived from git config user.email; do not store it in aiwf.yaml",
					})
				}
			}
			return true
		})
	}
	return out, nil
}
