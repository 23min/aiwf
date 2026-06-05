package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestM0162_AC4_AllowlistClaimsResolve pins the cross-AC-4
// reviewer R1-T4 finding: the allowlist at bijectionAllowlist()
// has prose claims of the form "primary test TestX in
// internal/cli/<dir>/" — without mechanical verification, those
// claims rot silently when the named test is renamed or moved.
//
// This test parses each allowlist entry's prose, extracts the
// expected test function name (the first identifier matching
// `Test\w+` after "primary test "), and the expected package
// directory (the path between "in " and a trailing slash). It
// then walks that directory's *_test.go files and asserts at
// least one function with that name exists.
//
// Sabotage-verifiable: rename `TestIsolationEscape_AC1_AICommitOnMainFires`
// or move `internal/check/isolation_escape_test.go` and this test
// fires naming the missing test.
//
// The allowlist entry for branch-cell-override-f-nnnn-waiver is
// exempted (the F-NNNN milestone family is outside E-0030 scope,
// per the inherited M-0158/AC-5 exception). The named-rule cells
// (branch-cell-isolation-escape-*) have AC-3-era ordinal
// counterparts (m0161-ac3-c1..14, etc.) carrying the actual Pin
// calls; their allowlist entries describe a cell-family
// pairing not a specific test function — for those, the test
// extracts the directory only and asserts at least one matching
// test in that directory.
//
// Patterns handled:
//   - "primary test TestX in internal/cli/Y/" → assert TestX
//     exists in internal/cli/Y/
//   - "primary verb test in internal/verb/" → assert at least
//     one test exists in internal/verb/ (cell-family claim)
//   - "primary unit test in internal/<dir>/" → assert at least
//     one test exists in that dir (cell-family claim)
//
// This test enforces the discoverability claim, not behavioral
// correctness — the named test could be wholly unrelated to the
// cell. Verifying the test↔cell semantic link would require
// reading the test's source for cell-ID literals (also covered
// by AC-4's bijection invariant 2, indirectly).
func TestM0162_AC4_AllowlistClaimsResolve(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	allow := bijectionAllowlist()

	// Regex for "TestX" — captures Go test function names including
	// the truncated "..." placeholder form (e.g.,
	// "TestAuthorize_..._BranchMissing_Refuses"). The character
	// class `[\w.]` admits letters/digits/underscore/dot so the
	// match extends through the placeholder.
	testNameRe := regexp.MustCompile(`primary test (Test[\w.]+)`)
	// Regex for "in internal/<path>/" — capture the path.
	dirRe := regexp.MustCompile(`in (internal/[a-z][a-z0-9_/-]*[a-z0-9])/?`)

	for cellID, prose := range allow {
		// Documented exception inherited from M-0158/AC-5 — outside
		// E-0030 scope, no verifiable claim.
		if cellID == "branch-cell-override-f-nnnn-waiver" {
			continue
		}

		dirMatch := dirRe.FindStringSubmatch(prose)
		if dirMatch == nil {
			t.Errorf("M-0162/AC-4 allowlist[%q]: prose %q does not name a directory (`in internal/...`); the bijection scope-narrowing claim is unverifiable", cellID, prose)
			continue
		}
		dir := dirMatch[1]
		absDir := filepath.Join(root, filepath.FromSlash(dir))

		tnMatch := testNameRe.FindStringSubmatch(prose)
		if tnMatch != nil {
			// Specific test name claim — verify the named test exists.
			testName := tnMatch[1]
			// Strip a trailing "..._<something>" placeholder if the
			// claim used the truncated form (e.g.,
			// "TestAuthorize_..._BranchMissing_Refuses"). The "..."
			// segment can match any test function whose name contains
			// the prefix AND the suffix.
			if strings.Contains(testName, "...") {
				parts := strings.Split(testName, "...")
				if len(parts) != 2 {
					t.Errorf("M-0162/AC-4 allowlist[%q]: malformed truncated test name %q", cellID, testName)
					continue
				}
				if !anyTestMatches(t, absDir, parts[0], parts[1]) {
					t.Errorf("M-0162/AC-4 allowlist[%q]: no test in %s matches pattern %q*%q\n  prose: %q", cellID, dir, parts[0], parts[1], prose)
				}
				continue
			}
			if !testExistsInDir(t, absDir, testName) {
				t.Errorf("M-0162/AC-4 allowlist[%q]: test %s not found in %s\n  prose: %q", cellID, testName, dir, prose)
			}
			continue
		}

		// Cell-family claim (no specific TestX named) — assert at
		// least one test exists in the directory.
		if !atLeastOneTestInDir(t, absDir) {
			t.Errorf("M-0162/AC-4 allowlist[%q]: no test functions found in %s\n  prose: %q", cellID, dir, prose)
		}
	}
}

// testExistsInDir reports whether dir contains a *_test.go file
// with a top-level test function named exactly testName.
func testExistsInDir(t *testing.T, dir, testName string) bool {
	t.Helper()
	found := false
	if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || found || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return nil
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fd.Name.Name == testName {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return found
}

// anyTestMatches reports whether any test in dir has a name
// matching prefix*suffix.
func anyTestMatches(t *testing.T, dir, prefix, suffix string) bool {
	t.Helper()
	found := false
	if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || found || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return nil
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			name := fd.Name.Name
			if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return found
}

// atLeastOneTestInDir reports whether dir contains at least one
// test function (Test* in any *_test.go).
func atLeastOneTestInDir(t *testing.T, dir string) bool {
	t.Helper()
	found := false
	if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || found || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return nil
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if strings.HasPrefix(fd.Name.Name, "Test") {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return found
}
