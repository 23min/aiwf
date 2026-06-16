package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PolicyGitTestEnvHardened asserts that every test-bearing package
// under internal/ whose tests shell out to a subprocess seeds its
// TestMain with a call to testsupport.HardenGitTestEnv().
//
// Why: test fixtures that build a repo in a t.TempDir() and run git
// against it invoke exec.Command with cmd.Dir set but no cmd.Env, so
// they inherit the ambient process environment, and they let git run
// its default background auto-gc. Two flake classes follow, both with
// the same symptom ("invalid object / Error building trees", "directory
// not empty"):
//
//   - G-0250: when the suite runs inside a git hook (the pre-commit
//     policy hook, or `make ci` launched mid-commit), git has exported
//     the locator vars (GIT_DIR/GIT_INDEX_FILE/...). Those override
//     cwd-based discovery, so fixture git commands operate against the
//     parent repo's shared git state and parallel tests race on one
//     index / object DB / lockfile.
//   - G-0251: git spawns a detached `git gc` after commits; under load
//     that background gc races the fixture's later git commands and
//     t.TempDir's RemoveAll.
//
// testsupport.HardenGitTestEnv scrubs the locator vars and disables
// auto-gc for the test binary's lifetime; this policy is the chokepoint
// that keeps a future exec-bearing test package from silently dropping
// the guard.
//
// Detection signal: a package is "exec-bearing" if any of its
// *_test.go files contains an exec.Command / exec.CommandContext call.
// We deliberately key on exec usage rather than a literal "git" first
// argument — internal/check's test fixtures shell git via
// exec.Command(args[0], ...) where args[0] is "git" passed by callers,
// which a literal-arg scan would miss. Hardening is harmless for the
// rare package that execs a non-git binary, so the broad signal costs
// nothing and closes the detection gap.
//
// Accepted detection limits (no occurrence today): the scan keys on
// the unaliased import name `exec`, so an aliased `os/exec` import
// would evade it; and a package that shells git only through a helper
// in another package (no exec.Command in its own *_test.go) would not
// be flagged — add the HardenGitTestEnv() call by hand in that case.
//
// Scope is internal/* only — the same scope as PolicyTestSetupPresence,
// whose TestMain-presence guarantee this policy layers on. There is no
// allowlist: every exec-bearing package under internal/ hardens today,
// and the call is a no-op where unneeded, so an exemption has no
// current motivating case (YAGNI). Add one — with a recorded rationale
// and its own coverage — if a real exec-bearing package ever needs to
// opt out.
//
// Pins G-0250 and G-0251.
func PolicyGitTestEnvHardened(root string) ([]Violation, error) {
	internalRoot := filepath.Join(root, "internal")

	testDirs := map[string]bool{}
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			testDirs[filepath.Dir(path)] = true
		}
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

		execs, scanErr := dirExecsSubprocess(dir)
		if scanErr != nil {
			return nil, scanErr
		}
		if !execs {
			continue
		}

		setupRel := filepath.ToSlash(filepath.Join(rel, "setup_test.go"))
		setupPath := filepath.Join(dir, "setup_test.go")
		if _, statErr := os.Stat(setupPath); statErr != nil {
			if os.IsNotExist(statErr) {
				vs = append(vs, Violation{
					Policy: "git-test-env-harden",
					File:   setupRel,
					Detail: "exec-bearing test package has no setup_test.go to host the testsupport.HardenGitTestEnv() call (G-0250/G-0251)",
				})
				continue
			}
			return nil, statErr
		}

		ok, scanErr := testMainCallsHarden(setupPath)
		if scanErr != nil {
			return nil, scanErr
		}
		if !ok {
			vs = append(vs, Violation{
				Policy: "git-test-env-harden",
				File:   setupRel,
				Detail: "test package shells out to a subprocess but its TestMain does not call testsupport.HardenGitTestEnv(); ambient git-locator env vars leak into fixture git commands under a git-hook run (G-0250) and background auto-gc races fixtures under load (G-0251)",
			})
		}
	}
	return vs, nil
}

// dirExecsSubprocess reports whether any *_test.go file in dir contains
// an exec.Command / exec.CommandContext call.
func dirExecsSubprocess(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, perr := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if perr != nil {
			return false, perr
		}
		found := false
		ast.Inspect(file, func(n ast.Node) bool {
			if found {
				return false
			}
			if call, ok := n.(*ast.CallExpr); ok && isExecCommandCall(call) {
				found = true
				return false
			}
			return true
		})
		if found {
			return true, nil
		}
	}
	return false, nil
}

// isExecCommandCall reports whether call is `exec.Command(...)` or
// `exec.CommandContext(...)` — the os/exec subprocess constructors as
// invoked under the package's conventional import name.
func isExecCommandCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "exec" {
		return false
	}
	return sel.Sel != nil && (sel.Sel.Name == "Command" || sel.Sel.Name == "CommandContext")
}

// testMainCallsHarden reports whether the setup_test.go at path declares
// a TestMain whose body calls testsupport.HardenGitTestEnv().
func testMainCallsHarden(path string) (bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return false, err
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name == nil || fn.Name.Name != "TestMain" || fn.Body == nil {
			continue
		}
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if found {
				return false
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "testsupport" {
				return true
			}
			if sel.Sel != nil && sel.Sel.Name == "HardenGitTestEnv" {
				found = true
				return false
			}
			return true
		})
		return found, nil
	}
	return false, nil
}
