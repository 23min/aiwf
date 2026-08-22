package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// shippedSurfaceMarkers name the embedded trees that materialize into a
// consumer's `.claude/`. They are the surface set the skill-body-id check
// scans: skill and ritual bodies with their frontmatter, entity templates,
// role-agent cards, the always-on guidance fragment, and the statusline.
//
// Both spellings appear because the two packages reach the same bytes
// differently: a policy test names a repo-relative path, while the skills
// package reads its own go:embed roots.
var shippedSurfaceMarkers = []string{
	"internal/skills/embedded",
	"embedded-rituals",
	"embedded-guidance",
	"embedded-statusline",
}

// documentTextFields name the struct fields that hold a document's own bytes,
// as opposed to something a reader derived from them. The skills package hands
// back a record whose Content is the shipped file; a check hands back findings
// whose Path and Hint it composed itself.
var documentTextFields = map[string]bool{
	"Content":  true,
	"Contents": true,
	"Body":     true,
}

// reshapers take a document and hand back a reshaped copy. They carry a
// phrase without testing for one, so they never report on their own.
var reshapers = map[string]bool{
	"strings.ReplaceAll": true,
	"strings.Replace":    true,
	"strings.Trim":       true,
	"strings.TrimLeft":   true,
	"strings.TrimRight":  true,
	"strings.TrimPrefix": true,
	"strings.TrimSuffix": true,
	"strings.Join":       true,
	"strings.Repeat":     true,
}

// PolicyShippedProseAssertion reports a test assertion that reads a shipped
// surface and compares its content against a phrase written into the test.
//
// D-0070 retires that class. The measurement behind it: across roughly
// fourteen months the corpus of such assertions recorded no catch, while the
// drift it failed to prevent was filed as gaps. The assertions pin a reading
// rather than a rule, and a reading drifts in more ways than an assertion can
// enumerate — the phrase can pre-exist elsewhere in scope, a later edit can
// give it a second occurrence, the negator that makes it binding can sit
// outside the asserted span.
//
// The rule is checked over test source rather than over prose, so no rewording
// satisfies it, and it is a ban rather than a mandate: it costs once, when
// someone reaches for the shape, instead of charging every shipped surface for
// a proof that it still reads a particular way.
//
// What survives is the relationship check — an assertion comparing two
// artefacts, which fails when either moves and which no rewording satisfies
// falsely. It reaches the second artefact one of two ways:
//
//   - Through a second document, in which case nothing exempts it: the needle
//     comes from that document rather than the test, so the rule never fires.
//   - Through code, by computing the expectation from the behaviour under test.
//     The needle is then a literal — a stable identifier, not a phrase — so the
//     rule does fire and the check carries an entry in
//     derivedExpectationExemptions.
//
// Trigger phrases are the one exemption that is not a relationship check. The
// phrasings in a skill's `## When to use` section and its `description:`
// frontmatter decide whether an assistant reaches for the skill at all, and
// each carries an entry in triggerPhraseExemptions. Exempting those two
// locations wholesale was measured and rejected: it would have covered nine
// further assertions that merely sit there without bearing on dispatch.
func PolicyShippedProseAssertion(root string) ([]Violation, error) {
	dirs, err := testPackageDirs(root)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, dir := range dirs {
		vs, perr := scanPackageForProseAssertions(root, dir)
		if perr != nil {
			return nil, perr
		}
		out = append(out, vs...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

// testPackageDirs returns every repo-relative directory holding a Go test
// file. Grouping by directory matters: a fixture path constant and the helper
// that reads it routinely live in different files of the same package.
func testPackageDirs(root string) ([]string, error) {
	seen := map[string]bool{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "vendor", "node_modules", ".git", ".claude", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, rerr := filepath.Rel(root, filepath.Dir(path))
		if rerr != nil {
			return rerr //coverage:ignore filepath.Rel fails only on a path outside root, which Walk cannot produce.
		}
		seen[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s for test packages: %w", root, err)
	}
	dirs := make([]string, 0, len(seen))
	for d := range seen {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs, nil
}

// scanPackageForProseAssertions parses one directory's Go sources together and
// runs the analysis across them.
func scanPackageForProseAssertions(root, relDir string) ([]Violation, error) {
	dir := filepath.Join(root, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", relDir, err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	paths := map[*ast.File]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		// ParseComments, because a `//go:embed` directive is a comment and is
		// the only thing tying an embedded tree to the variable holding it.
		f, perr := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if perr != nil {
			return nil, fmt.Errorf("parsing %s/%s: %w", relDir, name, perr) //coverage:ignore a source that does not parse fails the build long before any policy runs.
		}
		files = append(files, f)
		paths[f] = relDir + "/" + name
	}
	return detectProseAssertions(fset, files, paths), nil
}

// detectProseAssertions is the pure core: given one package's parsed files, it
// reports every containment whose haystack carries shipped-surface content and
// whose needle was written into the test.
func detectProseAssertions(fset *token.FileSet, files []*ast.File, paths map[*ast.File]string) []Violation {
	pathConsts := shippedPathConsts(files)
	readers, selfShipped := contentReaders(files, pathConsts)
	textHelpers := documentTextHelpers(files)

	var out []Violation
	for _, f := range files {
		rel := paths[f]
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || !strings.HasPrefix(fd.Name.Name, "Test") {
				continue
			}
			if shippedProseAssertionExempt(fd.Name.Name) {
				continue
			}
			sc := &scopeTaint{
				shipped: map[string]bool{},
				lits:    literalNeedles(fd.Body),
				readers: readers, selfShipped: selfShipped, pathConsts: pathConsts,
				textHelpers: textHelpers,
			}
			sc.propagate(fd.Body)
			out = append(out, sc.assertions(fset, fd, rel)...)
		}
	}
	return out
}

// scopeTaint tracks, within one test function, which identifiers carry
// shipped-surface content and which carry a phrase the test itself wrote.
type scopeTaint struct {
	shipped     map[string]bool
	lits        map[string]bool
	readers     map[string]bool
	selfShipped map[string]bool
	pathConsts  map[string]bool
	textHelpers map[string]bool
}

// propagate seeds taint from reads and carries it across assignments. Several
// passes settle chains such as body → section → lowercased section, which the
// AST walk can visit out of dependency order.
func (sc *scopeTaint) propagate(body *ast.BlockStmt) {
	for pass := 0; pass < 4; pass++ {
		ast.Inspect(body, func(n ast.Node) bool {
			switch st := n.(type) {
			case *ast.AssignStmt:
				if len(st.Rhs) == 1 && len(st.Lhs) > 1 {
					if sc.carriesShipped(st.Rhs[0]) {
						for _, l := range st.Lhs {
							sc.mark(l)
						}
					}
					return true
				}
				for i, rhs := range st.Rhs {
					if i < len(st.Lhs) && sc.carriesShipped(rhs) {
						sc.mark(st.Lhs[i])
					}
				}
			case *ast.RangeStmt:
				if sc.carriesShipped(st.X) {
					sc.mark(st.Value)
				}
			}
			return true
		})
	}
}

// mark taints an assignment target. Writing shipped content into a container
// taints the container, which is how `byName[s.Name] = string(s.Content)`
// keeps its bytes in scope when a later assertion indexes back into it.
func (sc *scopeTaint) mark(l ast.Expr) {
	switch x := l.(type) {
	case *ast.Ident:
		if x.Name != "_" {
			sc.shipped[x.Name] = true
		}
	case *ast.IndexExpr:
		sc.mark(x.X)
	}
}

// carriesShipped reports whether e evaluates to content read from a shipped
// surface. A read is the only seed — naming a shipped path is not enough, or
// every test constructing a fixture record would qualify.
func (sc *scopeTaint) carriesShipped(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return sc.shipped[x.Name]
	case *ast.ParenExpr:
		return sc.carriesShipped(x.X)
	case *ast.StarExpr:
		return sc.carriesShipped(x.X)
	case *ast.UnaryExpr:
		return sc.carriesShipped(x.X)
	case *ast.IndexExpr:
		return sc.carriesShipped(x.X)
	case *ast.SliceExpr:
		return sc.carriesShipped(x.X)
	case *ast.BinaryExpr:
		return sc.carriesShipped(x.X) || sc.carriesShipped(x.Y)
	case *ast.SelectorExpr:
		// A field carries the document only when it holds document text. A
		// reader handing back records it derived taints its own result, but
		// `finding.Hint` is a message the code composed, not shipped prose.
		return documentTextFields[x.Sel.Name] && sc.carriesShipped(x.X)
	case *ast.CallExpr:
		name := calleeFuncName(x.Fun)
		if isReadFileCall(x.Fun) {
			return anyExprNamesShippedPath(x.Args, sc.pathConsts)
		}
		if sc.readers[name] &&
			(sc.selfShipped[name] || anyExprNamesShippedPath(x.Args, sc.pathConsts)) {
			return true
		}
		// Only a call handing document text back passes the taint on. One that
		// projects some other property of a record — a name, a path — yields a
		// fact about the document rather than the document, and an assertion on
		// that is not a reading of prose.
		if !sc.carriesText(name) {
			return false
		}
		for _, a := range x.Args {
			if sc.carriesShipped(a) {
				return true
			}
		}
	}
	return false
}

// carriesText reports whether a call hands its argument's document text back.
// The strings package does by construction; a local helper does when it
// returns a lone string or byte slice, which is the shape of "give me this
// section" rather than "give me this property".
func (sc *scopeTaint) carriesText(name string) bool {
	return strings.HasPrefix(name, "strings.") || name == "string" || sc.textHelpers[name]
}

// assertions reports each containment in fd comparing shipped content against
// a test-authored phrase.
func (sc *scopeTaint) assertions(fset *token.FileSet, fd *ast.FuncDecl, rel string) []Violation {
	var out, scopes []Violation
	report := func(needle ast.Expr, call string) {
		out = append(out, Violation{
			Policy: "shipped-prose-assertion",
			File:   rel,
			Line:   fset.Position(needle.Pos()).Line,
			Detail: fmt.Sprintf(
				"%s asserts via %s that shipped-surface prose contains %s. D-0070 retires this class: delete the assertion. Two things are not this: a check that draws its needle from a second document is already out of scope and needs no change, and one deriving its expectation by running the code belongs on derivedExpectationExemptions. A trigger phrase deciding whether an assistant reaches for the skill belongs on triggerPhraseExemptions.",
				fd.Name.Name, call, describeNeedle(needle)),
		})
	}
	// A call standing in a condition decides whether the test fails, so it is
	// making a claim. A call whose result is bound to a name is narrowing the
	// document for a later claim to be made about. That distinction is what
	// separates asserting prose from scoping to a section, and it holds for
	// this repo's own document helpers as readily as for the strings package —
	// which matters, because enumerating helper names by hand is exactly how a
	// rule like this ends up with a hole in it.
	deciding := decidingCalls(fd.Body)

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := calleeFuncName(ce.Fun)
		if len(ce.Args) < 2 || !sc.carriesShipped(ce.Args[0]) {
			return true
		}
		for _, arg := range ce.Args[1:] {
			if !sc.isTestAuthoredNeedle(arg) {
				continue
			}
			if deciding[ce] && !reshapers[name] {
				report(arg, name)
				break
			}
			// Scoping. D-0070 deletes it alongside the assertion it scoped —
			// "once the body assertion is gone it degrades to asserting the
			// heading exists" — so it is held back and emitted only if this
			// function turns out to assert prose too. Scoping a *structural*
			// claim, such as a count of numbered step headings, is the shape
			// D-0050 asks for and stays.
			before := len(out)
			report(arg, name)
			scopes = append(scopes, out[before:]...)
			out = out[:before]
			break
		}
		return true
	})
	if len(out) > 0 {
		out = append(out, scopes...)
	}
	return out
}

// decidingCalls returns the calls sitting in a position that decides whether
// the test fails: an if-condition, or either side of a comparison.
func decidingCalls(body *ast.BlockStmt) map[*ast.CallExpr]bool {
	out := map[*ast.CallExpr]bool{}
	mark := func(e ast.Expr) {
		ast.Inspect(e, func(n ast.Node) bool {
			if ce, ok := n.(*ast.CallExpr); ok {
				out[ce] = true
			}
			return true
		})
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.IfStmt:
			mark(x.Cond)
		case *ast.BinaryExpr:
			switch x.Op {
			case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
				mark(x.X)
				mark(x.Y)
			}
		}
		return true
	})
	return out
}

// isTestAuthoredNeedle reports whether the needle traces back only to string
// literals in the test source, and carries a word rather than a markdown
// delimiter. A needle derived from another document makes the call a
// relationship check, which stays.
func (sc *scopeTaint) isTestAuthoredNeedle(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.BasicLit:
		return x.Kind == token.STRING && carriesWord(litValue(x))
	case *ast.Ident:
		return sc.lits[x.Name]
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			return sc.lits[id.Name]
		}
	case *ast.CallExpr:
		// A case-folded or trimmed needle is the same needle. Without this
		// the commonest spelling of the pattern — comparing both sides
		// lowercased — walks straight past the rule.
		if needleTransforms[calleeFuncName(x.Fun)] && len(x.Args) == 1 {
			return sc.isTestAuthoredNeedle(x.Args[0])
		}
	}
	return false
}

// needleTransforms are the string functions that reshape a needle without
// changing where it came from.
var needleTransforms = map[string]bool{
	"strings.ToLower":   true,
	"strings.ToUpper":   true,
	"strings.TrimSpace": true,
}

// shippedPathConsts collects the names that stand for a shipped surface: those
// bound to a literal naming one of the shipped trees, and those a `//go:embed`
// directive binds to one.
//
// The embed case is not an optimization. The skills package reaches its own
// shipped bytes through `//go:embed` rather than a file read, so a rule looking
// only for reads sees that whole package as touching nothing.
func shippedPathConsts(files []*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Doc == nil {
				continue
			}
			if !embedsShippedTree(gd.Doc) {
				continue
			}
			for _, sp := range gd.Specs {
				if vs, ok := sp.(*ast.ValueSpec); ok {
					for _, nm := range vs.Names {
						out[nm.Name] = true
					}
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			vs, ok := n.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for i, nm := range vs.Names {
				if i < len(vs.Values) && exprNamesShippedPath(vs.Values[i], out) {
					out[nm.Name] = true
				}
			}
			return true
		})
	}
	return out
}

// embedsShippedTree reports whether a declaration's doc comment carries a
// `//go:embed` directive naming a shipped tree. Bare `embedded` counts here,
// where the directive's argument is unambiguously an embed root, though it is
// too broad to match as a path substring anywhere else.
func embedsShippedTree(doc *ast.CommentGroup) bool {
	for _, c := range doc.List {
		text, ok := strings.CutPrefix(c.Text, "//go:embed ")
		if !ok {
			continue
		}
		for _, arg := range strings.Fields(text) {
			if arg == "embedded" || strings.HasPrefix(arg, "embedded/") || namesShippedPath(arg) {
				return true
			}
		}
	}
	return false
}

// contentReaders finds the functions that read a file and hand back its bytes.
// selfShipped names those that read a shipped surface without being told which
// one, so a bare call to them already yields shipped content.
//
// A reader that hands back derived records rather than bytes still counts —
// documentTextFields is what keeps its findings out of scope.
func contentReaders(files []*ast.File, pathConsts map[string]bool) (readers, selfShipped map[string]bool) {
	readers, selfShipped = map[string]bool{}, map[string]bool{}
	type fn struct {
		name string
		decl *ast.FuncDecl
	}
	var fns []fn
	for _, f := range files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || fd.Type.Results == nil || len(fd.Type.Results.List) == 0 {
				continue
			}
			if resultsCarryDiagnostics(fd.Type.Results) {
				continue
			}
			fns = append(fns, fn{fd.Name.Name, fd})
		}
	}
	for changed := true; changed; {
		changed = false
		for _, f := range fns {
			reads, shipped := false, false
			ast.Inspect(f.decl.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					if isReadFileCall(x.Fun) {
						reads = true
					}
					if name := calleeFuncName(x.Fun); readers[name] {
						reads = true
						if selfShipped[name] {
							shipped = true
						}
					}
				case *ast.Ident:
					if pathConsts[x.Name] {
						shipped = true
						// An embedded tree needs no read to be reached: naming
						// the variable already yields the shipped bytes.
						reads = true
					}
				case *ast.BasicLit:
					if x.Kind == token.STRING && namesShippedPath(litValue(x)) {
						shipped = true
					}
				}
				return true
			})
			if reads && !readers[f.name] {
				readers[f.name], changed = true, true
			}
			if reads && shipped && !selfShipped[f.name] {
				selfShipped[f.name], changed = true, true
			}
		}
	}
	return readers, selfShipped
}

// literalNeedles maps each identifier in a test bound only to string literals
// the test wrote — directly, or as elements of a composite literal ranged over,
// which is how a phrase list reaches a containment call.
func literalNeedles(body *ast.BlockStmt) map[string]bool {
	lits := map[string]bool{}
	for pass := 0; pass < 3; pass++ {
		ast.Inspect(body, func(n ast.Node) bool {
			switch st := n.(type) {
			case *ast.ValueSpec:
				for i, nm := range st.Names {
					if i < len(st.Values) && isLiteralSource(st.Values[i], lits) {
						lits[nm.Name] = true
					}
				}
			case *ast.AssignStmt:
				for i, rhs := range st.Rhs {
					if i >= len(st.Lhs) {
						break //coverage:ignore Go admits either one right-hand value or one per target, so a parsed assignment never has more values than targets; the bound is here so a hand-built AST cannot panic the walk.
					}
					if id, ok := st.Lhs[i].(*ast.Ident); ok && isLiteralSource(rhs, lits) {
						lits[id.Name] = true
					}
				}
			case *ast.RangeStmt:
				if id, ok := st.Value.(*ast.Ident); ok && isLiteralSource(st.X, lits) {
					lits[id.Name] = true
				}
			}
			return true
		})
	}
	return lits
}

// isLiteralSource reports whether e is a string literal, an identifier already
// known to hold one, or a composite literal whose every element is a string
// literal (including one field of a struct element).
func isLiteralSource(e ast.Expr, lits map[string]bool) bool {
	switch x := e.(type) {
	case *ast.BasicLit:
		return x.Kind == token.STRING
	case *ast.Ident:
		return lits[x.Name]
	case *ast.CompositeLit:
		sawLit := false
		for _, el := range x.Elts {
			switch v := el.(type) {
			case *ast.BasicLit:
				if v.Kind != token.STRING {
					return false
				}
				sawLit = true
			case *ast.CompositeLit:
				for _, inner := range v.Elts {
					val := inner
					if kv, ok := inner.(*ast.KeyValueExpr); ok {
						val = kv.Value
					}
					bl, ok := val.(*ast.BasicLit)
					if !ok || bl.Kind != token.STRING {
						return false
					}
					sawLit = true
				}
			default:
				return false
			}
		}
		return sawLit
	}
	return false
}

// isReadFileCall reports whether the callee is any ReadFile — os.ReadFile,
// fs.ReadFile, or a method on an embed.FS.
func isReadFileCall(fun ast.Expr) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "ReadFile"
}

func namesShippedPath(s string) bool {
	for _, m := range shippedSurfaceMarkers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

func exprNamesShippedPath(e ast.Expr, pathConsts map[string]bool) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if pathConsts[x.Name] {
				found = true
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING && namesShippedPath(litValue(x)) {
				found = true
			}
		}
		return true
	})
	return found
}

func anyExprNamesShippedPath(args []ast.Expr, pathConsts map[string]bool) bool {
	for _, a := range args {
		if exprNamesShippedPath(a, pathConsts) {
			return true
		}
	}
	return false
}

// carriesWord reports whether s holds at least three letters. Parsing code
// matches on markdown delimiters — "#", "|", "\n\n" — and those are structure,
// not prose.
func carriesWord(s string) bool {
	n := 0
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			n++
		}
	}
	return n >= 3
}

func litValue(b *ast.BasicLit) string {
	if v, err := strconv.Unquote(b.Value); err == nil {
		return v
	}
	return b.Value //coverage:ignore the parser only produces quotable string literals.
}

func describeNeedle(e ast.Expr) string {
	if bl, ok := e.(*ast.BasicLit); ok {
		v := litValue(bl)
		if len(v) > 48 {
			v = v[:48] + "…"
		}
		return strconv.Quote(v)
	}
	return "a phrase list written in the test"
}

// calleeFuncName renders a call's callee as `pkg.Func`, `Func`, or `recv.Func`.
func calleeFuncName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			return id.Name + "." + x.Sel.Name
		}
		return x.Sel.Name
	}
	return ""
}

// documentTextHelpers names the package's own functions that hand back a
// document's text: those returning a lone string or byte slice. A helper
// returning anything else — a list of names, an index, a record — yields a
// property of the document rather than the document itself.
func documentTextHelpers(files []*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Type.Results == nil || len(fd.Type.Results.List) == 0 {
				continue
			}
			switch t := fd.Type.Results.List[0].Type.(type) {
			case *ast.Ident:
				if t.Name == "string" {
					out[fd.Name.Name] = true
				}
			case *ast.ArrayType:
				if id, ok := t.Elt.(*ast.Ident); ok && id.Name == "byte" {
					out[fd.Name.Name] = true
				}
			}
		}
	}
	return out
}

// resultsCarryDiagnostics reports whether a result list yields records a check
// or policy derived — findings and violations — rather than document text.
func resultsCarryDiagnostics(res *ast.FieldList) bool {
	for _, f := range res.List {
		t := f.Type
		if arr, ok := t.(*ast.ArrayType); ok {
			t = arr.Elt
		}
		if star, ok := t.(*ast.StarExpr); ok {
			t = star.X
		}
		name := ""
		switch v := t.(type) {
		case *ast.Ident:
			name = v.Name
		case *ast.SelectorExpr:
			name = v.Sel.Name
		}
		if name == "Violation" || name == "Finding" {
			return true
		}
	}
	return false
}
