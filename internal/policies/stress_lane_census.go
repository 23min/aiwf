package policies

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PolicyStressLaneCensus asserts that every test carrying the `stress`
// build tag earns its place there: it starts a process, waits on a
// clock, or runs a goroutine.
//
// Why: the tag moves a test off the every-push path onto the on-demand
// `make stress-tests` path, and it applies per *file*, not per test. A
// file that holds one real-subprocess scenario driver alongside a
// small fabricated-input decision test takes both when it is tagged.
// The decision test then stops running on every push while still
// passing whenever anyone runs it, so nothing reports the loss — no
// failure, no warning, just less coverage than the suite appears to
// have (G-0468).
//
// Detection signal: a tagged test is legitimate if its body reaches a
// process, a clock, or a goroutine. Rather than enumerate every helper
// that shells out, the list keys on the seams a test cannot avoid.
// Most are impure whatever they are handed — obtaining a binary, or
// touching a real git repo — and any of them excuses a test on its
// own.
//
// Three are not: Setup, RunScenario and RunRepeated take a Scenario,
// so their impurity lives in the argument rather than in the callee,
// and this package's own hermetic tests drive all three with fake
// scenarios. Those count only alongside a real scenario constructor in
// the same body, which is what separates driving the harness from
// driving the thing the harness runs.
//
// Accepted detection limits. The scan reads each test's own body, so a
// test whose only impurity hides one call deep inside a package-local
// helper reads as hermetic and is flagged — that has happened once
// here already, for a test polling through waitForTempFile, and the
// fix is to name the helper below rather than to tag around it. The
// scan keys on unaliased `exec` and `time` imports. Both limits fail
// toward a red gate naming a specific test, which is the direction
// that gets noticed; the silent direction is what this policy exists
// to close.
//
// Pins G-0468.
func PolicyStressLaneCensus(root string) ([]Violation, error) {
	var vs []Violation
	for _, pkg := range []string{
		filepath.Join("internal", "stresstest"),
		filepath.Join("cmd", "stresstest"),
	} {
		pkgVs, err := stressLanePackageViolations(root, pkg)
		if err != nil {
			return nil, err
		}
		vs = append(vs, pkgVs...)
	}
	return vs, nil
}

// stressLanePackageViolations scans one package directory, named
// relative to root. A directory that is absent yields nothing, so the
// policy is inert in a tree that does not carry the harness.
func stressLanePackageViolations(root, pkgRel string) ([]Violation, error) {
	dir := filepath.Join(root, pkgRel)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var vs []Violation
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(pkgRel, e.Name()))
		fileVs, scanErr := stressLaneViolations(filepath.Join(dir, e.Name()), rel)
		if scanErr != nil {
			return nil, scanErr
		}
		vs = append(vs, fileVs...)
	}
	return vs, nil
}

// stressLaneViolations reports the tests in one `stress`-tagged file
// whose bodies reach nothing slow. An untagged file yields none. rel
// is path's repo-relative name, carried in rather than recomputed.
func stressLaneViolations(path, rel string) ([]Violation, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if !hasStressBuildTag(file) {
		return nil, nil
	}

	var vs []Violation
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || !isTestFunc(fn) {
			continue
		}
		if bodyReachesSlowWork(fn.Body) {
			continue
		}
		vs = append(vs, Violation{
			Policy: "stress-lane-census",
			File:   rel,
			Line:   fset.Position(fn.Pos()).Line,
			Detail: fn.Name.Name + " carries the stress build tag but starts no process, waits on no clock, and runs no goroutine — a decision test in the on-demand lane stops running on every push without anything reporting it (G-0468). Move it to an untagged sibling file, or, if it drives real work through a helper, name that helper in slowWorkIdents (internal/policies/stress_lane_census.go).",
		})
	}
	return vs, nil
}

// isTestFunc reports whether fn is a test the `go test` runner would
// call: a top-level func named Test... taking exactly one *testing.T.
// The signature check is what excludes TestMain, which takes a
// *testing.M and drives the binary rather than asserting anything.
func isTestFunc(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
		return false
	}
	params := fn.Type.Params.List
	if len(params) != 1 {
		return false
	}
	star, ok := params[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "T"
}

// hasStressBuildTag reports whether file's build constraint selects
// the `stress` tag. Evaluating the constraint rather than matching its
// text is what keeps `//go:build !stress` — a file pinned to the
// default lane, the opposite of this policy's subject — from reading
// as tagged. Only the comments ahead of the package clause are
// considered, since that is where a build constraint can live.
func hasStressBuildTag(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() > file.Package {
			return false
		}
		for _, c := range group.List {
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				continue
			}
			if expr.Eval(func(tag string) bool { return tag == "stress" }) {
				return true
			}
		}
	}
	return false
}

// slowWorkIdents are the package-local seams that are impure whatever
// they are handed: acquiring or building a binary, or touching a real
// git repo. Each excuses a tagged test on its own. The list names the
// package's process-touching seams rather than only those a tagged
// test happens to use today, so adding a driver that reaches for one
// does not have to touch this policy.
var slowWorkIdents = map[string]bool{
	"sharedTestBinary":               true,
	"sharedLockHolderBinary":         true,
	"BuildBinary":                    true,
	"BuildLockHolder":                true,
	"runRun":                         true,
	"runAiwfJSON":                    true,
	"runGit":                         true,
	"gitInitAndConfig":               true,
	"gitCaptureOutput":               true,
	"newVerbSequenceTestRepo":        true,
	"newSiblingWorktreesFixture":     true,
	"newBareOriginWithClonesFixture": true,
	"seedActivationEpic":             true,
}

// scenarioDrivenIdents take a Scenario, so what they do depends on
// what they are given: this package's own hermetic tests drive all
// three with fake scenarios and no subprocess at all. They excuse a
// tagged test only alongside a real scenario constructor.
var scenarioDrivenIdents = map[string]bool{
	"Setup":       true,
	"RunScenario": true,
	"RunRepeated": true,
}

// scenarioConstructor matches this package's scenario constructors,
// each of which returns a driver that runs real subprocesses.
var scenarioConstructor = regexp.MustCompile(`^New\w*Scenario$`)

// slowWorkSelectors are the stdlib calls that make a test slow on
// their own, keyed by package and function name.
var slowWorkSelectors = map[string]map[string]bool{
	"exec": {"Command": true, "CommandContext": true},
	"time": {"Sleep": true, "After": true, "NewTimer": true, "NewTicker": true, "Tick": true},
}

// bodyReachesSlowWork reports whether body starts a process, waits on
// a clock, or runs a goroutine.
func bodyReachesSlowWork(body *ast.BlockStmt) bool {
	var unconditional, scenarioDriven, constructsScenario bool
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GoStmt:
			unconditional = true
		case *ast.Ident:
			switch {
			case slowWorkIdents[node.Name]:
				unconditional = true
			case scenarioDrivenIdents[node.Name]:
				scenarioDriven = true
			case scenarioConstructor.MatchString(node.Name):
				constructsScenario = true
			}
		case *ast.SelectorExpr:
			pkg, ok := node.X.(*ast.Ident)
			if !ok {
				return true
			}
			if slowWorkSelectors[pkg.Name][node.Sel.Name] {
				unconditional = true
			}
		}
		return true
	})
	return unconditional || (scenarioDriven && constructsScenario)
}
