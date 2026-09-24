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

// verdictReturners are the strings functions that answer a question about a
// document — bool or index — rather than handing back part of it. Only these,
// and local helpers that are not document-text helpers, can be assertions; a
// call yielding text is narrowing the document for a later claim.
var verdictReturners = map[string]bool{
	"strings.Contains":    true,
	"strings.ContainsAny": true,
	"strings.HasPrefix":   true,
	"strings.HasSuffix":   true,
	"strings.EqualFold":   true,
	"strings.Index":       true,
	"strings.LastIndex":   true,
	"strings.Count":       true,
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

// proseSurface is one set of documents the prose-assertion engine guards,
// and what it refuses over them.
type proseSurface struct {
	// policy is the Violation.Policy id the surface reports under.
	policy string
	// namesPath reports whether a string literal names one of the
	// surface's documents.
	namesPath func(string) bool
	// embedRoots counts a `//go:embed` of an embedded tree as naming the
	// surface, which is how the skills package reaches its own bytes.
	embedRoots bool
	// rooted counts a path only when it is built from this repository's
	// root, for a surface of the repository's own files.
	rooted bool
	// presenceOnly refuses only an assertion that a phrase is present; an
	// assertion that one is absent is a ban, which pins no reading.
	presenceOnly bool
	// exempt names the test functions the surface does not judge.
	exempt func(string) bool
	// noun names the documents in a Detail.
	noun string
	// remedy is the Detail text after the offending assertion is named.
	remedy string
}

// shippedSurface is D-0070's surface: the embedded trees that materialize
// into consumer repositories.
var shippedSurface = proseSurface{
	policy:     "shipped-prose-assertion",
	namesPath:  namesShippedPath,
	embedRoots: true,
	exempt:     shippedProseAssertionExempt,
	noun:       "shipped-surface prose",
	remedy:     "D-0070 retires this class: delete the assertion. Two things are not this: a check that draws its needle from a second document is already out of scope and needs no change, and one deriving its expectation by running the code belongs on derivedExpectationExemptions. A trigger phrase deciding whether an assistant reaches for the skill belongs on triggerPhraseExemptions.",
}

// proseFinding is one refused assertion with the test function holding it.
type proseFinding struct {
	fn string
	v  Violation
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
// runs the shipped-surface analysis across them.
func scanPackageForProseAssertions(root, relDir string) ([]Violation, error) {
	found, err := scanPackageFor(root, relDir, shippedSurface)
	return proseViolations(found), err
}

// proseViolations drops the function names a caller does not need.
func proseViolations(found []proseFinding) []Violation {
	var out []Violation
	for _, f := range found {
		out = append(out, f.v)
	}
	return out
}

// scanPackageFor parses one directory's Go sources together and runs the
// analysis for surface across them.
func scanPackageFor(root, relDir string, surface proseSurface) ([]proseFinding, error) {
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
	return detectProseFindings(surface, fset, files, paths), nil
}

// detectProseAssertions is the pure core: given one package's parsed files, it
// reports every containment whose haystack carries shipped-surface content and
// whose needle was written into the test.
func detectProseAssertions(fset *token.FileSet, files []*ast.File, paths map[*ast.File]string) []Violation {
	return proseViolations(detectProseFindings(shippedSurface, fset, files, paths))
}

// detectProseFindings is the pure core for any surface.
func detectProseFindings(surface proseSurface, fset *token.FileSet, files []*ast.File, paths map[*ast.File]string) []proseFinding {
	pathConsts := shippedPathConsts(files, surface)
	readers, selfShipped := contentReaders(files, pathConsts, surface)
	textHelpers := documentTextHelpers(files)

	var out []proseFinding
	for _, f := range files {
		rel := paths[f]
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || !strings.HasPrefix(fd.Name.Name, "Test") {
				continue
			}
			if surface.exempt(fd.Name.Name) {
				continue
			}
			sc := &scopeTaint{
				surface:     surface,
				shipped:     map[string]bool{},
				pathIdents:  map[string]bool{},
				rootIdents:  map[string]bool{},
				localConsts: map[string]bool{},
				lits:        literalNeedles(files, fd.Body),
				readers:     readers, selfShipped: selfShipped, pathConsts: pathConsts,
				textHelpers: textHelpers,
			}
			sc.propagate(fd.Body)
			for _, v := range sc.assertions(fset, fd, rel) {
				out = append(out, proseFinding{fn: fd.Name.Name, v: v})
			}
		}
	}
	return out
}

// scopeTaint tracks, within one test function, which identifiers carry
// shipped-surface content and which carry a phrase the test itself wrote.
type scopeTaint struct {
	surface     proseSurface
	shipped     map[string]bool
	lits        map[string]bool
	readers     map[string]bool
	selfShipped map[string]bool
	pathConsts  map[string]bool
	textHelpers map[string]bool
	// pathIdents are locals that hold a shipped-surface path rather than its
	// content: `p := ritualPath`, or a table row whose fields include one.
	// Without them a path reaching the reader through anything but its own
	// constant name is invisible, which excludes the table-driven form this
	// repo mandates for two or more cases.
	pathIdents map[string]bool
	// rootIdents are locals bound to the repository root: `root :=
	// repoRoot(t)`.
	rootIdents map[string]bool
	// localConsts are function-local constants and variables declared
	// with a value naming a surface path.
	localConsts map[string]bool
}

// namesShippedPath reports whether e names a surface path, whether as a
// literal, a package constant, or a local carrying one.
func (sc *scopeTaint) namesShippedPath(e ast.Expr) bool {
	return sc.anyArgNamesShippedPath([]ast.Expr{e})
}

// anyArgNamesShippedPath reports whether the arguments together name a
// surface path. A local already carrying a path names one on its own. A
// literal or constant does too, except on a surface whose documents are
// this repository's own files, where the path must also be built from the
// repository root: a file of the same name in a test's fixture repository
// is a test of code, not of the guidance.
func (sc *scopeTaint) anyArgNamesShippedPath(args []ast.Expr) bool {
	named, rooted := false, !sc.surface.rooted
	for _, a := range args {
		carried := false
		ast.Inspect(a, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				if sc.pathIdents[x.Name] {
					carried = true
				}
				if sc.pathConsts[x.Name] || sc.localConsts[x.Name] {
					named = true
				}
				if sc.rootIdents[x.Name] {
					rooted = true
				}
			case *ast.SelectorExpr:
				// A field of a table row that carries a path — `tc.path`.
				if id, ok := x.X.(*ast.Ident); ok && sc.pathIdents[id.Name] {
					carried = true
				}
			case *ast.CallExpr:
				if isRepoRootCall(x) {
					rooted = true
				}
			case *ast.BasicLit:
				if x.Kind == token.STRING && sc.surface.namesPath(litValue(x)) {
					named = true
				}
			}
			return true
		})
		if carried {
			return true
		}
	}
	return named && rooted
}

// isRepoRootCall reports whether a call resolves this repository's root:
// repoRoot and its per-package siblings.
func isRepoRootCall(ce *ast.CallExpr) bool {
	return strings.HasPrefix(calleeFuncName(ce.Fun), "repoRoot")
}

// mentionsRepoRoot reports whether e resolves the repository root or uses a
// local bound to it.
func (sc *scopeTaint) mentionsRepoRoot(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if isRepoRootCall(x) {
				found = true
			}
		case *ast.Ident:
			if sc.rootIdents[x.Name] {
				found = true
			}
		}
		return true
	})
	return found
}

// anyArgCarriesShipped reports whether some argument carries shipped content.
func (sc *scopeTaint) anyArgCarriesShipped(args []ast.Expr) bool {
	for _, a := range args {
		if sc.carriesShipped(a) {
			return true
		}
	}
	return false
}

// propagate seeds taint from reads and carries it across assignments. Several
// passes settle chains such as body → section → lowercased section, which the
// AST walk can visit out of dependency order.
func (sc *scopeTaint) propagate(body *ast.BlockStmt) {
	for pass := 0; pass < 4; pass++ {
		ast.Inspect(body, func(n ast.Node) bool {
			switch st := n.(type) {
			case *ast.ValueSpec:
				// A function-local `const` or `var` naming a path stands for
				// it within this function, as a package constant does across
				// the package.
				for i, nm := range st.Names {
					if i < len(st.Values) && nm.Name != "_" && exprNamesShippedPath(st.Values[i], sc.pathConsts, sc.surface.namesPath) {
						sc.localConsts[nm.Name] = true
					}
				}
			case *ast.AssignStmt:
				for i, rhs := range st.Rhs {
					if i < len(st.Lhs) && sc.mentionsRepoRoot(rhs) {
						if id, ok := st.Lhs[i].(*ast.Ident); ok && id.Name != "_" {
							sc.rootIdents[id.Name] = true
						}
					}
				}
				// A local rebound to a shipped path carries it onward.
				for i, rhs := range st.Rhs {
					if i < len(st.Lhs) && sc.namesShippedPath(rhs) {
						if id, ok := st.Lhs[i].(*ast.Ident); ok && id.Name != "_" {
							sc.pathIdents[id.Name] = true
						}
					}
				}
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
				// A table whose rows carry a shipped path makes the row
				// variable a path carrier, so `tc.path` resolves.
				if sc.namesShippedPath(st.X) {
					if id, ok := st.Value.(*ast.Ident); ok && id.Name != "_" {
						sc.pathIdents[id.Name] = true
					}
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
			return sc.anyArgNamesShippedPath(x.Args)
		}
		if sc.readers[name] && (sc.selfShipped[name] || sc.anyArgNamesShippedPath(x.Args)) {
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
		line := fset.Position(needle.Pos()).Line
		detail := fmt.Sprintf("%s asserts via %s that %s contains %s. %s",
			fd.Name.Name, call, sc.surface.noun, describeNeedle(needle), sc.surface.remedy)
		// Each id is spelled inline so the firing-fixture inventory sees
		// both policies.
		if sc.surface.policy == "guidance-prose-assertion" {
			out = append(out, Violation{Policy: "guidance-prose-assertion", File: rel, Line: line, Detail: detail})
			return
		}
		out = append(out, Violation{Policy: "shipped-prose-assertion", File: rel, Line: line, Detail: detail})
	}
	// A call standing in a condition decides whether the test fails, so it is
	// making a claim. A call whose result is bound to a name is narrowing the
	// document for a later claim to be made about. That distinction is what
	// separates asserting prose from scoping to a section, and it holds for
	// this repo's own document helpers as readily as for the strings package —
	// which matters, because enumerating helper names by hand is exactly how a
	// rule like this ends up with a hole in it.
	deciding := decidingCalls(fd.Body)
	presence := presenceCalls(fd.Body)

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := calleeFuncName(ce.Fun)
		// Reporting and formatting calls take a document alongside a format
		// string; the literal there is a message, not a claim about the
		// document. Skipping them is what lets the haystack be sought in any
		// argument position below.
		if isReportingCall(name) {
			return true
		}
		// The haystack need not be the first argument: `mustContain(t, body,
		// "…")` is the dominant assertion-helper signature, and keying on
		// argument zero exits the rule the moment a repeated assertion is
		// extracted into a helper — the very move H1 asks for.
		if len(ce.Args) < 2 || !sc.anyArgCarriesShipped(ce.Args) {
			return true
		}
		for i, arg := range ce.Args {
			if sc.carriesShipped(arg) || !sc.isTestAuthoredNeedle(arg) {
				continue
			}
			_ = i
			// A ban on a phrase pins no reading, so a surface that refuses
			// only presence lets an absence assertion stand — and it is not
			// scoping either, so nothing is held back for it.
			if sc.surface.presenceOnly && deciding[ce] && !presence[ce] {
				break
			}
			if deciding[ce] && !reshapers[name] && sc.isAsserting(name) {
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
		ifs, ok := n.(*ast.IfStmt)
		// Position alone does not make a call an assertion — what matters is
		// whether failing the condition fails the test. A locator reaches a
		// `break` or a `continue`; an assertion reaches t.Error or t.Fatal.
		// Keying on the branch's consequence is what tells a search for the
		// row to inspect apart from a claim about what the row says.
		if !ok || !branchFailsTheTest(ifs) {
			return true
		}
		mark(ifs.Cond)
		return true
	})
	return out
}

// presenceCalls returns the deciding calls whose failing arm runs when the
// phrase is absent — an assertion that it is present. A call negated in the
// condition, compared against false, or an index compared as not found
// flips which way the condition reads.
func presenceCalls(body *ast.BlockStmt) map[*ast.CallExpr]bool {
	out := map[*ast.CallExpr]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		bodyFails, elseFails := armFailsTheTest(ifs.Body), armFailsTheTest(ifs.Else)
		// trueWhenAbsent maps each call in the condition to whether the
		// condition is true exactly when that call's phrase is absent.
		trueWhenAbsent := map[*ast.CallExpr]bool{}
		conditionPolarity(ifs.Cond, false, trueWhenAbsent)
		for ce, absent := range trueWhenAbsent {
			if (bodyFails && absent) || (elseFails && !absent) {
				out[ce] = true
			}
		}
		return true
	})
	return out
}

// conditionPolarity records, for each call in e, whether e reads true when
// the call's phrase is absent, given that neg says whether e itself sits
// under an odd number of negations.
func conditionPolarity(e ast.Expr, neg bool, out map[*ast.CallExpr]bool) {
	switch x := e.(type) {
	case *ast.ParenExpr:
		conditionPolarity(x.X, neg, out)
	case *ast.UnaryExpr:
		if x.Op == token.NOT {
			conditionPolarity(x.X, !neg, out)
			return
		}
		conditionPolarity(x.X, neg, out)
	case *ast.BinaryExpr:
		if ce, absentWhenTrue, ok := comparisonPolarity(x); ok {
			out[ce] = neg != absentWhenTrue
			return
		}
		conditionPolarity(x.X, neg, out)
		conditionPolarity(x.Y, neg, out)
	case *ast.CallExpr:
		out[x] = neg
		for _, a := range x.Args {
			conditionPolarity(a, neg, out)
		}
	}
}

// comparisonPolarity reads a comparison between a call and a constant: a
// verdict against true or false, or an index or count against the value
// meaning "not found". It reports the call and whether the comparison is
// true when the phrase is absent.
func comparisonPolarity(b *ast.BinaryExpr) (*ast.CallExpr, bool, bool) {
	ce, ok := b.X.(*ast.CallExpr)
	other := b.Y
	if !ok {
		return nil, false, false
	}
	switch v := other.(type) {
	case *ast.Ident:
		// A verdict compared against a boolean constant.
		switch {
		case (v.Name == "false" && b.Op == token.EQL) || (v.Name == "true" && b.Op == token.NEQ):
			return ce, true, true
		case (v.Name == "true" && b.Op == token.EQL) || (v.Name == "false" && b.Op == token.NEQ):
			return ce, false, true
		}
	case *ast.BasicLit, *ast.UnaryExpr:
		// An index, or a count, compared against its not-found value.
		lit := constIntValue(other)
		switch {
		case lit == -1 && (b.Op == token.EQL || b.Op == token.LEQ):
			return ce, true, true
		case lit == -1 && (b.Op == token.NEQ || b.Op == token.GTR):
			return ce, false, true
		case lit == 0 && b.Op == token.LSS:
			return ce, true, true
		case lit == 0 && b.Op == token.GEQ:
			return ce, false, true
		case lit == 0 && b.Op == token.EQL:
			return ce, true, true
		case lit == 0 && (b.Op == token.NEQ || b.Op == token.GTR):
			return ce, false, true
		}
	}
	return nil, false, false
}

// constIntValue returns the value of an integer literal, allowing a leading
// minus; anything else reads as a value no comparison above names.
func constIntValue(e ast.Expr) int {
	neg := false
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.SUB {
		neg, e = true, u.X
	}
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 99
	}
	n, err := strconv.Atoi(lit.Value)
	if err != nil { //coverage:ignore an INT token the parser accepted always converts
		return 99
	}
	if neg {
		return -n
	}
	return n
}

// branchFailsTheTest reports whether either arm of an if reports a failure.
func branchFailsTheTest(ifs *ast.IfStmt) bool {
	return armFailsTheTest(ifs.Body) || armFailsTheTest(ifs.Else)
}

// armFailsTheTest reports whether one arm of an if reports a failure.
func armFailsTheTest(n ast.Node) bool {
	found := false
	check := func(n ast.Node) {
		if n == nil {
			return
		}
		ast.Inspect(n, func(m ast.Node) bool {
			ce, ok := m.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch name := sel.Sel.Name; {
			case strings.HasPrefix(name, "Error"), strings.HasPrefix(name, "Fatal"):
				found = true
			}
			return true
		})
	}
	check(n)
	return found
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
func shippedPathConsts(files []*ast.File, surface proseSurface) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Doc == nil || !surface.embedRoots {
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
		// Package-level declarations only: the map is keyed by name across
		// the whole package, so a function-local constant would lend its
		// path to every same-named identifier elsewhere. A local one is
		// followed inside its own function by propagate.
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, sp := range gd.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, nm := range vs.Names {
					if i < len(vs.Values) && exprNamesShippedPath(vs.Values[i], out, surface.namesPath) {
						out[nm.Name] = true
					}
				}
			}
		}
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
func contentReaders(files []*ast.File, pathConsts map[string]bool, surface proseSurface) (readers, selfShipped map[string]bool) {
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
			reads, shipped, rooted := false, false, !surface.rooted
			ast.Inspect(f.decl.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					if isReadFileCall(x.Fun) {
						reads = true
					}
					if isRepoRootCall(x) {
						rooted = true
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
					if x.Kind == token.STRING && surface.namesPath(litValue(x)) {
						shipped = true
					}
				}
				return true
			})
			if reads && !readers[f.name] {
				readers[f.name], changed = true, true
			}
			if reads && shipped && rooted && !selfShipped[f.name] {
				selfShipped[f.name], changed = true, true
			}
		}
	}
	return readers, selfShipped
}

// literalNeedles maps each identifier in a test bound only to string literals
// the test wrote — directly, or as elements of a composite literal ranged over,
// which is how a phrase list reaches a containment call.
func literalNeedles(files []*ast.File, body *ast.BlockStmt) map[string]bool {
	lits := map[string]bool{}
	// Package-level declarations count: a phrase list hoisted to a package var
	// is the same needle, and scoping the scan to the function body is how a
	// corpus member hides one `var` away from the test that uses it.
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || (gd.Tok != token.CONST && gd.Tok != token.VAR) {
				continue
			}
			for _, sp := range gd.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue //coverage:ignore a const or var declaration's specs are ValueSpecs by construction; the guard is here so a hand-built AST cannot panic the walk.
				}
				for i, nm := range vs.Names {
					if i < len(vs.Values) && isLiteralSource(vs.Values[i], lits) {
						lits[nm.Name] = true
					}
				}
			}
		}
	}
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

func exprNamesShippedPath(e ast.Expr, pathConsts map[string]bool, namesPath func(string) bool) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if pathConsts[x.Name] {
				found = true
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING && namesPath(litValue(x)) {
				found = true
			}
		}
		return true
	})
	return found
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

// isReportingCall reports whether a call emits a message rather than testing
// one. A `t.Fatalf("read %s: %v", path, err)` carries a document and a literal
// side by side without asserting anything about the document.
func isReportingCall(name string) bool {
	pkg, fn, ok := strings.Cut(name, ".")
	if !ok {
		return false
	}
	switch pkg {
	case "t", "tb", "b", "fmt", "log", "slog":
		return true
	}
	// A method on the test handle reached through another receiver name.
	return strings.HasPrefix(fn, "Error") || strings.HasPrefix(fn, "Fatal") ||
		strings.HasPrefix(fn, "Log") || strings.HasPrefix(fn, "Skip")
}

// isAsserting reports whether a call can be the assertion itself rather than
// the narrowing that precedes one. A call handing back document text is always
// narrowing, wherever it sits — inlining `extractMarkdownSection(...)` into the
// condition does not turn a section lookup into a claim about prose.
func (sc *scopeTaint) isAsserting(name string) bool {
	if verdictReturners[name] {
		return true
	}
	if strings.HasPrefix(name, "strings.") {
		return false
	}
	return !sc.textHelpers[name]
}
