package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PolicyTestSetupPresence asserts that every test-bearing package
// under internal/ carries a `setup_test.go` file containing a
// `func TestMain(m *testing.M)` declaration. The policy is the
// chokepoint for M-0091's parallelism convention: a new test file
// in any internal/* package must inherit the parallel-by-default
// shape, and TestMain is the seam where the GIT identity env is
// seeded once via os.Setenv (t.Setenv panics under t.Parallel).
//
// What "test-bearing" means here: a directory directly containing
// at least one *_test.go file. The check skips fixture/data dirs
// (testdata/) and any dir whose go files are all non-test. Nested
// test-bearing subpackages are checked independently.
//
// Why AST-level instead of substring: per CLAUDE.md's "Substring
// assertions are not structural assertions", a flat grep for
// `func TestMain(` would match a function hidden under a build tag,
// a misplaced helper inside another file, or even a comment. The
// go/parser walk anchors the assertion to the actual package-level
// declaration shape `func TestMain(m *testing.M)`.
//
// Scope is internal/* only. cmd/aiwf/'s test discipline is
// captured by its setup_test.go skip-list (M-0092), which is more
// nuanced per-test than a presence check would capture; widening
// the scope would require extending this policy with the cmd-side
// audit logic, deferred until real friction surfaces.
//
// Pins M-0093/AC-2.
func PolicyTestSetupPresence(root string) ([]Violation, error) {
	internalRoot := filepath.Join(root, "internal")

	// Discover test-bearing directories: walk internal/ and collect
	// any directory that holds at least one *_test.go file.
	testDirs := map[string]bool{}
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			// Skip testdata/ subtrees — they hold fixture files, not
			// test code, and shouldn't be expected to host a TestMain.
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		testDirs[filepath.Dir(path)] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	var vs []Violation
	for dir := range testDirs {
		rel, relErr := filepath.Rel(root, dir)
		if relErr != nil {
			rel = dir
		}
		setupPath := filepath.Join(dir, "setup_test.go")
		setupRel := filepath.Join(rel, "setup_test.go")

		if _, statErr := os.Stat(setupPath); os.IsNotExist(statErr) {
			vs = append(vs, Violation{
				Policy: "test-setup-presence",
				File:   rel,
				Detail: "test-bearing package missing setup_test.go (see CLAUDE.md *Test discipline*; landed M-0091/M-0092, locked M-0093/AC-2)",
			})
			continue
		} else if statErr != nil {
			return nil, statErr
		}

		// Parse the file with PackageClauseOnly | ParseComments would
		// drop function declarations; we need full parsing to find
		// TestMain. ParseFile with mode 0 reads the full body which
		// is fine — setup_test.go files are short by convention.
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, setupPath, nil, parser.ParseComments)
		if parseErr != nil {
			vs = append(vs, Violation{
				Policy: "test-setup-presence",
				File:   setupRel,
				Detail: "setup_test.go failed to parse: " + parseErr.Error(),
			})
			continue
		}

		if !hasTestMainDecl(file) {
			vs = append(vs, Violation{
				Policy: "test-setup-presence",
				File:   setupRel,
				Detail: "setup_test.go missing top-level `func TestMain(m *testing.M)` declaration (see CLAUDE.md *Test discipline*)",
			})
		}
	}
	return vs, nil
}

// hasTestMainDecl reports whether file declares a top-level
// `func TestMain(m *testing.M)`. The check requires:
//   - name == "TestMain"
//   - receiver == nil (free function, not a method)
//   - exactly one parameter
//   - parameter type is *testing.M (selector expr "testing.M"
//     wrapped in a star expr)
//
// Other shapes (different name, wrong arity, method) fail the check
// because they don't satisfy Go's `go test` TestMain contract — the
// runtime would silently fall through to the default m.Run() and
// the env-setup wouldn't fire.
func hasTestMainDecl(file *ast.File) bool {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name == nil || fn.Name.Name != "TestMain" {
			continue
		}
		if fn.Recv != nil {
			continue
		}
		if fn.Type == nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
			continue
		}
		param := fn.Type.Params.List[0]
		star, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok {
			continue
		}
		if pkg.Name == "testing" && sel.Sel != nil && sel.Sel.Name == "M" {
			return true
		}
	}
	return false
}
