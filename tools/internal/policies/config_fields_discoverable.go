package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

// PolicyConfigFieldsAreDiscoverable asserts that every yaml-tagged
// field on a struct in tools/internal/config/config.go appears in at
// least one channel an AI assistant routinely consults: an embedded
// skill, the binary's printHelp output (cmd/aiwf/main.go), CLAUDE.md
// / tools/CLAUDE.md, or any markdown under docs/pocv3/.
//
// Mirrors PolicyFindingCodesAreDiscoverable for a parallel kernel
// surface: aiwf.yaml is the consumer-facing knob set, and a knob the
// docs don't mention is, by definition, undocumented. The motivation
// is recent: G37 layer (a) added `allocate.trunk` and required
// manual updates to the aiwf-add SKILL.md to keep it discoverable;
// nothing structurally guarded that. This policy does.
//
// Heuristic: AST-walk config.go, collect every struct-field yaml tag
// (the name portion before any `,omitempty` or other modifier), then
// require the bare tag value (e.g. `allocate`, `trunk`,
// `aiwf_version`) appears in the discoverability haystack. Embedded
// fields that share a tag name with their parent block (e.g. the
// `trunk:` field under the `allocate:` block) are documented if the
// parent block's name appears alongside the field name anywhere in
// the haystack — a check page that says "allocate.trunk" satisfies
// both the `allocate` and the `trunk` requirements.
//
// Tag values explicitly excluded: `actor` (legacy field, kept on
// Config purely so `aiwf doctor` can render its deprecation note;
// per design-decisions.md it is no longer documented as a live
// surface).
func PolicyConfigFieldsAreDiscoverable(root string) ([]Violation, error) {
	prodFiles, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}

	tags := collectConfigYAMLTags(prodFiles)

	haystack, err := readDiscoverabilityChannels(root)
	if err != nil {
		return nil, err
	}
	hay := string(haystack)

	excluded := map[string]bool{
		"actor": true, // legacy / deprecation-note-only
	}

	var out []Violation
	for tag := range tags {
		if excluded[tag] {
			continue
		}
		if strings.Contains(hay, tag) {
			continue
		}
		out = append(out, Violation{
			Policy: "config-fields-discoverable",
			File:   "tools/internal/config/config.go",
			Detail: "yaml field " + tag + " is declared on a Config struct but not mentioned in any AI-discoverable channel (embedded skills, aiwf <verb> --help, CLAUDE.md, or docs/pocv3/**/*.md)",
		})
	}
	return out, nil
}

// collectConfigYAMLTags AST-walks tools/internal/config/config.go
// (and any peer .go file in the same package, for forward
// compatibility) and returns the set of yaml tag names declared on
// struct fields. The tag name is everything before the first comma:
// `aiwf_version` from `yaml:"aiwf_version"`,
// `actor` from `yaml:"actor,omitempty"`.
func collectConfigYAMLTags(files []FileEntry) map[string]struct{} {
	out := map[string]struct{}{}
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "tools/internal/config/") {
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
				if field.Tag == nil {
					continue
				}
				raw := strings.Trim(field.Tag.Value, "`")
				yamlTag := reflect.StructTag(raw).Get("yaml")
				name := strings.SplitN(yamlTag, ",", 2)[0]
				if name == "" || name == "-" {
					continue
				}
				out[name] = struct{}{}
			}
			return true
		})
	}
	return out
}
