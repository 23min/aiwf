package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// design_doc_symbol_citations_test.go — M-0332/AC-2. A normative design doc
// that stops restating a fact routes the reader to whatever owns it. Where
// that owner is Go, the route is a link to the declaring file whose text is
// the symbol, and the symbol is what rots: a rename leaves the link
// resolving and the citation naming nothing.
//
// PolicyDesignDocAnchors already stats every link target under docs/design,
// so the path half is held there and is not re-asserted here.

// goSymbolCitation matches a markdown link whose text is a backticked Go
// identifier and whose target is a .go file — the shape
// [`entity.PathKind`](../../internal/entity/entity.go). A package qualifier
// is matched but not captured, since only the last segment is declared in
// the file the link points at.
//
// A link whose text is a path rather than an identifier names no symbol and
// is left to PolicyDesignDocAnchors. The backticks and the absence of a
// slash are what separate the two.
var goSymbolCitation = regexp.MustCompile(
	"\\[`(?:[A-Za-z_][A-Za-z0-9_]*\\.)?([A-Za-z_][A-Za-z0-9_]*)`\\]\\(([^)]+\\.go)\\)",
)

// nextMarkdownHeading matches the start of any ATX heading line, so a
// section can be cut from the text following its own heading.
var nextMarkdownHeading = regexp.MustCompile(`(?m)^#{1,6} `)

// declaredNames returns the package-level identifiers a Go file declares.
// Package-level is the claim: a citation resolves against the package
// surface, so a name declared inside a function body is not one.
func declaredNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	names := map[string]bool{}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			names[d.Name.Name] = true
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					names[s.Name.Name] = true
				case *ast.ValueSpec:
					for _, n := range s.Names {
						names[n.Name] = true
					}
				}
			}
		}
	}
	return names
}

// TestDesignDocGoSymbolCitationsResolve pins that a symbol a design doc
// cites beside a link to its file is really declared there. Both sides
// derive: the name is read from the prose and the declarations from the Go
// source, so a rename on either side turns it red.
func TestDesignDocGoSymbolCitationsResolve(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	docs, err := walkMarkdown(filepath.Join(root, "docs", "design"))
	if err != nil {
		t.Fatalf("walk docs/design: %v", err)
	}
	checked := 0
	for _, doc := range docs {
		rel := relTo(root, doc.AbsPath)
		for _, m := range goSymbolCitation.FindAllStringSubmatch(string(doc.Contents), -1) {
			cited, target := m[1], m[2]
			if !declaredNames(t, filepath.Join(filepath.Dir(doc.AbsPath), target))[cited] {
				t.Errorf("%s cites `%s` beside a link to %s, which declares no such name.\n"+
					"Cite what the file declares, or point the link at the file that declares it.",
					rel, cited, target)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Error("no Go-symbol citation found under docs/design; the pattern no longer matches the house shape")
	}
}

// TestDesignDecisionsRoutesBodySectionsToTheOwner pins that the section
// which described body templates routes the reader to the Go declaration
// owning the required section set, rather than restating that set.
//
// Scoped to the section by heading rather than matched anywhere in the file:
// a citation elsewhere in the document does not help a reader who arrived
// here looking for the sections.
func TestDesignDecisionsRoutesBodySectionsToTheOwner(t *testing.T) {
	t.Parallel()
	const heading = "### Frontmatter schema and body templates"
	doc := filepath.Join("docs", "design", "design-decisions.md")
	data, err := os.ReadFile(filepath.Join(repoRoot(t), doc))
	if err != nil {
		t.Fatalf("read %s: %v", doc, err)
	}
	_, after, found := strings.Cut(string(data), heading+"\n")
	if !found {
		t.Fatalf("%s has no %q heading; the section this criterion scopes to is gone", doc, heading)
	}
	if !goSymbolCitation.MatchString(nextMarkdownHeading.Split(after, 2)[0]) {
		t.Errorf("%s §%q names no Go symbol beside a link to its declaring file.\n"+
			"The section no longer states the required sections, so it must route to what owns them.",
			doc, heading)
	}
}

// TestDeclaredNames pins that every package-level declaration kind a doc
// might cite is found, and that a name declared inside a function is not.
func TestDeclaredNames(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "fixture.go")
	src := `package fixture

type Shape struct{}

const Limit = 3

var Registry = map[string]int{}

func Render() string {
	local := "not a declaration"
	return local
}
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	names := declaredNames(t, path)
	for _, want := range []string{"Shape", "Limit", "Registry", "Render"} {
		if !names[want] {
			t.Errorf("declaredNames did not find %q; a doc citing it would report as unresolved", want)
		}
	}
	if names["local"] {
		t.Error("declaredNames found a function-local name; only package-level declarations are citable")
	}
}
