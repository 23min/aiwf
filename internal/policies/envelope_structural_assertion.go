package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// envelopeStructuralPath is the one file this policy pins — the
// canonical JSON envelope type every aiwf verb writes.
const envelopeStructuralPath = "internal/render/render.go"

// envelopeRequiredJSONKeys is the checked-in contract: the exact set
// of json tag names render.Envelope's fields must carry (M-0239/AC-4).
// A future contributor renaming a field's json tag (or adding/
// removing one) without updating this set fails CI here, before any
// downstream JSON consumer silently breaks against the new shape.
// This is a structural check (parses the Go type declaration itself)
// distinct from internal/cli/integration/envelope_schema_test.go's
// runtime check (drives every --format=json verb and diffs the
// marshaled output) — the latter catches a shape drift a verb
// introduces at the call site; this one catches the type declaration
// drifting from its own documented contract even before any verb's
// output is inspected.
var envelopeRequiredJSONKeys = []string{
	"tool", "version", "status", "findings", "result", "error", "metadata",
}

// PolicyEnvelopeStructuralAssertion asserts that render.Envelope's
// field json tags are exactly envelopeRequiredJSONKeys — no more, no
// fewer, no renamed key. See envelopeRequiredJSONKeys's doc comment
// for the regression class this catches.
func PolicyEnvelopeStructuralAssertion(root string) ([]Violation, error) {
	path := filepath.Join(root, filepath.FromSlash(envelopeStructuralPath))
	contents, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			//coverage:ignore the live repo always carries this file; only a
			// synthetic fixture tree that deliberately omits it could hit this,
			// and no such fixture exists — every fixture in this policy's own
			// test file writes the file before calling the policy.
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, path, contents, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	found := collectEnvelopeJSONTags(astFile)
	if found == nil {
		return []Violation{{
			Policy: "envelope-structural-assertion",
			File:   envelopeStructuralPath,
			Detail: "no `type Envelope struct` found — the canonical JSON envelope type this policy pins is missing or renamed",
		}}, nil
	}

	want := make(map[string]bool, len(envelopeRequiredJSONKeys))
	for _, k := range envelopeRequiredJSONKeys {
		want[k] = true
	}

	var out []Violation
	for k := range found {
		if !want[k] {
			out = append(out, Violation{
				Policy: "envelope-structural-assertion",
				File:   envelopeStructuralPath,
				Detail: fmt.Sprintf("Envelope carries json key %q, which is not in the pinned required-key set (%s) — update envelopeRequiredJSONKeys in this policy alongside any deliberate Envelope field change", k, strings.Join(envelopeRequiredJSONKeys, ", ")),
			})
		}
	}
	for _, k := range envelopeRequiredJSONKeys {
		if !found[k] {
			out = append(out, Violation{
				Policy: "envelope-structural-assertion",
				File:   envelopeStructuralPath,
				Detail: fmt.Sprintf("Envelope is missing the pinned required json key %q — a field was renamed or removed without updating envelopeRequiredJSONKeys in this policy", k),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Detail < out[j].Detail })
	return out, nil
}

// collectEnvelopeJSONTags AST-walks astFile for `type Envelope struct
// {...}` and returns the set of json tag names its fields carry (the
// name portion before any comma, e.g. "metadata" from
// `json:"metadata,omitempty"`). Returns nil when no Envelope struct is
// found in the file at all — distinct from an empty-but-present
// struct (a non-nil empty map), which the caller reports as a
// required-key mismatch rather than a missing-type error.
func collectEnvelopeJSONTags(astFile *ast.File) map[string]bool {
	var found map[string]bool
	ast.Inspect(astFile, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name == nil || ts.Name.Name != "Envelope" {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		found = map[string]bool{}
		for _, field := range st.Fields.List {
			if field.Tag == nil {
				continue
			}
			raw := strings.Trim(field.Tag.Value, "`")
			jsonTag := reflect.StructTag(raw).Get("json")
			name := strings.SplitN(jsonTag, ",", 2)[0]
			if name == "" || name == "-" {
				continue
			}
			found[name] = true
		}
		return false
	})
	return found
}
