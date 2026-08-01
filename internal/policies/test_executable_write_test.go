package policies

import (
	"go/ast"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_TestExecutableWrite is the CI gate entry point. It runs the
// diff-scoped audit against the live tree using the base ref supplied
// via AIWF_COVERAGE_BASE. Without a base (the default in the broad `go
// test ./...` job) it skips — the authoritative invocation is the
// dedicated CI coverage-gate step and `make coverage-gate`.
func TestPolicy_TestExecutableWrite(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		t.Skip("AIWF_COVERAGE_BASE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicyTestExecutableWrite)
}

// TestPolicyTestExecutableWrite_Env drives the env-fed entry point so the
// wrapper body is exercised during profile generation. Serial (t.Setenv
// panics under t.Parallel) and documented in setup_test.go's skip-list.
func TestPolicyTestExecutableWrite_Env(t *testing.T) {
	// Unset base → no-op.
	t.Setenv("AIWF_COVERAGE_BASE", "")
	vs, err := PolicyTestExecutableWrite(t.TempDir())
	if err != nil {
		t.Fatalf("unset base: unexpected error: %v", err)
	}
	if vs != nil {
		t.Fatalf("unset base: want nil violations, got %+v", vs)
	}

	// Set base → delegates and surfaces the offending write.
	root, runGit, writeFile, baseSHA := skillFixtureBase(t)
	writeFile("internal/seam/seam_test.go", "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755)\n}\n")
	runGit("add", "-A")
	runGit("commit", "-m", "head")

	t.Setenv("AIWF_COVERAGE_BASE", baseSHA)
	vs, err = PolicyTestExecutableWrite(root)
	if err != nil {
		t.Fatalf("set base: unexpected error: %v", err)
	}
	if len(vs) != 1 || vs[0].Line != 4 {
		t.Fatalf("set base: want one violation on line 4, got %+v", vs)
	}
	if vs[0].Policy != "test-executable-write" {
		t.Errorf("policy id = %q, want test-executable-write", vs[0].Policy)
	}
}

// TestTestExecutableWriteViolations_Seam drives the full IO shell — a
// synthetic git repo, `git diff <base>`, changedLines, the AST scan —
// across the discriminations the policy exists to make.
func TestTestExecutableWriteViolations_Seam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      string
		body      string
		wantLines []int
	}{
		{
			name:      "executable write in a test fires",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755)\n}\n",
			wantLines: []int{4},
		},
		{
			name: "the 0o700 spelling fires too",
			path: "internal/seam/seam_test.go",
			body: "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o700)\n}\n",
			// Keying on the executable mask, not the 0o755 literal, is
			// what catches this — it was already in the tree.
			wantLines: []int{4},
		},
		{
			name:      "a non-executable mode is silent",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o644)\n}\n",
			wantLines: nil,
		},
		{
			name:      "production source is out of scope",
			path:      "internal/seam/seam.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755)\n}\n",
			wantLines: nil,
		},
		{
			name:      "the helper itself is not os.WriteFile",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = testsupport.WriteExecutable(p, b)\n}\n",
			wantLines: nil,
		},
		{
			name:      "a same-named method on another package is out of scope",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = fs.WriteFile(p, b, 0o755)\n}\n",
			wantLines: nil,
		},
		{
			name:      "a non-literal mode is out of scope",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, mode)\n}\n",
			wantLines: nil,
		},
		{
			name:      "exec:ok with a reason exempts",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755) //exec:ok the mode is the subject\n}\n",
			wantLines: nil,
		},
		{
			name:      "a bare exec:ok is not an escape",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755) //exec:ok\n}\n",
			wantLines: []int{4},
		},
		{
			// A longer word opening with the marker's letters is not the
			// directive with "ay" for a reason.
			name:      "a word merely starting with the marker is not an escape",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755) //exec:okay\n}\n",
			wantLines: []int{4},
		},
		{
			name:      "exec:ok on the line above exempts",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t//exec:ok the mode is the subject\n\t_ = os.WriteFile(p, b, 0o755)\n}\n",
			wantLines: nil,
		},
		{
			// A trailing marker annotates its own line only; carrying it
			// downward would exempt the next call, which nobody annotated.
			name:      "a trailing exec:ok does not exempt the next call",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755) //exec:ok the mode is the subject\n\t_ = os.WriteFile(q, b, 0o755)\n}\n",
			wantLines: []int{5},
		},
		{
			// Prose that mentions the escape is not the escape.
			name:      "prose mentioning exec:ok does not exempt",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t// see the exec:ok escape in CLAUDE.md\n\t_ = os.WriteFile(p, b, 0o755)\n}\n",
			wantLines: []int{5},
		},
		{
			// Including a doc comment, which would otherwise exempt the
			// function's first body line.
			name:      "a doc comment mentioning exec:ok does not exempt",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\n// F writes a stand-in; use exec:ok when the mode is the subject.\nfunc F() { _ = os.WriteFile(p, b, 0o755) }\n",
			wantLines: []int{4},
		},
		{
			name:      "a file that does not parse is the compiler's finding",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F( {\n\t_ = os.WriteFile(p, b, 0o755)\n",
			wantLines: nil,
		},
		{
			name:      "two writes both fire",
			path:      "internal/seam/seam_test.go",
			body:      "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755)\n\t_ = os.WriteFile(q, b, 0o755)\n}\n",
			wantLines: []int{4, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, baseSHA := skillFixtureBase(t)
			writeFile(tt.path, tt.body)
			runGit("add", "-A")
			runGit("commit", "-m", "head")

			vs, err := testExecutableWriteViolations(root, baseSHA)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertViolationLines(t, vs, tt.wantLines)
		})
	}
}

// TestTestExecutableWriteViolations_OnlyFlagsChangedLines pins the
// diff-scoping itself: an offending write already present in the base is
// untouched by a later edit elsewhere in the same file. This is what
// makes the rule adoptable on a tree that carries the pattern in roughly
// a hundred places.
func TestTestExecutableWriteViolations_OnlyFlagsChangedLines(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, baseSHA := skillFixtureBase(t)

	const rel = "internal/seam/seam_test.go"
	writeFile(rel, "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755)\n}\n")
	runGit("add", "-A")
	runGit("commit", "-m", "pre-existing offender")
	preexisting := trimLine(runGit("rev-parse", "HEAD"))

	// Append an unrelated function; the offending line is untouched.
	writeFile(rel, "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755)\n}\n\nfunc G() {}\n")
	runGit("add", "-A")
	runGit("commit", "-m", "unrelated edit")

	vs, err := testExecutableWriteViolations(root, preexisting)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertViolationLines(t, vs, nil)

	// Against the original base the same line is inside the diff, so the
	// silence above is scoping rather than a detector that never fires.
	vs, err = testExecutableWriteViolations(root, baseSHA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertViolationLines(t, vs, []int{4})
}

// TestTestExecutableWriteViolations_UntrackedTestFile pins that a fixture
// written but not yet committed is in scope — the gate runs before the
// commit, so catching it only afterwards would be too late to help.
func TestTestExecutableWriteViolations_UntrackedTestFile(t *testing.T) {
	t.Parallel()
	root, _, writeFile, baseSHA := skillFixtureBase(t)
	writeFile("internal/seam/seam_test.go", "package p\n\nfunc F() {\n\t_ = os.WriteFile(p, b, 0o755)\n}\n")

	vs, err := testExecutableWriteViolations(root, baseSHA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertViolationLines(t, vs, []int{4})
}

// TestTestExecutableWriteViolations_NoBaseIsANoOp pins the two
// "no comparison point" spellings the CI jobs actually pass.
func TestTestExecutableWriteViolations_NoBaseIsANoOp(t *testing.T) {
	t.Parallel()
	for _, base := range []string{"", "  ", zeroSHA} {
		vs, err := testExecutableWriteViolations(t.TempDir(), base)
		if err != nil {
			t.Fatalf("base %q: unexpected error: %v", base, err)
		}
		if vs != nil {
			t.Errorf("base %q: want nil violations, got %+v", base, vs)
		}
	}
}

// TestTestExecutableWriteViolations_SurfacesADiffFailure pins that a
// base ref git cannot resolve is reported rather than silently read as
// "nothing changed" — a scan that no-ops on a broken base would report
// green while auditing nothing.
func TestTestExecutableWriteViolations_SurfacesADiffFailure(t *testing.T) {
	t.Parallel()
	// Not a git repo, so `git diff` in it fails.
	if _, err := testExecutableWriteViolations(t.TempDir(), "HEAD"); err == nil {
		t.Fatal("expected an error diffing against a base git cannot resolve")
	}
}

// TestDetectBareExecutableWrites_SkipsAFileItCannotRead pins the
// disposition of a path that is in the changed set but not on disk —
// a file removed between the diff and the read, or a dangling symlink.
// There is nothing to audit either way, and the sibling scan in
// comment_history_attrition.go resolves it the same way.
func TestDetectBareExecutableWrites_SkipsAFileItCannotRead(t *testing.T) {
	t.Parallel()
	got := detectBareExecutableWrites(t.TempDir(), "internal/seam/gone_test.go", map[int]bool{1: true})
	if got != nil {
		t.Fatalf("want no violations for a file that is not there, got %+v", got)
	}
}

// TestOwnLine covers the own-line/trailing discrimination directly,
// including the bounds guards. Those are reachable: a //line directive
// moves reported positions off the file they were read from, and
// without the guards the slice panics.
func TestOwnLine(t *testing.T) {
	t.Parallel()
	lines := []string{"\t//exec:ok why", "\t_ = os.WriteFile(p, b, 0o755) //exec:ok why"}

	tests := []struct {
		name   string
		line   int
		column int
		want   bool
	}{
		{"a comment alone on its line", 1, 2, true},
		{"a comment trailing code", 2, 32, false},
		{"a line number below the source", 0, 1, false},
		{"a line number past the source", 3, 1, false},
		{"a column past the end of its line", 1, 500, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ownLine(lines, tt.line, tt.column); got != tt.want {
				t.Errorf("ownLine(%d, %d) = %v, want %v", tt.line, tt.column, got, tt.want)
			}
		})
	}
}

// TestIsExecutableMode_RejectsAnUnparseableLiteral pins the guard on
// ParseInt: a valid Go integer literal can still overflow the mode's
// width, and a policy that panicked or mis-read it would misjudge the
// call.
func TestIsExecutableMode_RejectsAnUnparseableLiteral(t *testing.T) {
	t.Parallel()
	// A syntactically valid Go int literal far wider than a file mode.
	lit := &ast.BasicLit{Kind: token.INT, Value: "0o77777777777777777777"}
	if isExecutableMode(lit) {
		t.Error("a literal too wide to parse as a mode must not be read as executable")
	}
}

// TestTestExecutableWrite_WiredIntoCoverageGate pins that the gate runs
// at the integration boundary: the policy test is named in the
// coverage-gate run-pattern of both the CI workflow and the Makefile
// target. Without this a future edit could drop it from the run set and
// it would silently never fire.
//
// Scoped to the run-pattern line via coverageGateRunLine rather than a
// file-wide substring, which would also be satisfied by an incidental
// mention of the name elsewhere in the file.
func TestTestExecutableWrite_WiredIntoCoverageGate(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	const testName = "TestExecutableWrite"

	for _, f := range []string{".github/workflows/go.yml", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		line := coverageGateRunLine(t, f, string(data))
		if !strings.Contains(line, testName) {
			t.Errorf("%s: coverage-gate run-pattern does not include %s:\n  %s", f, testName, line)
		}
	}
}

// TestAdoptedPackagesRouteThroughWriteExecutable pins the adoption this
// policy was landed alongside: no bare os.WriteFile with an executable
// mode may return to a package where the ETXTBSY flake actually bit. The
// diff-scoped policy cannot express this — it forgives any line a change
// does not touch — which is why the packages that have to stay clean
// need their own whole-file assertion.
//
// It inherits the detector's scope, so the write-then-chmod and
// os.OpenFile spellings of the same hazard pass it. Neither appears in
// these packages today; both would need the detector widened first.
func TestAdoptedPackagesRouteThroughWriteExecutable(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	for _, pkg := range adoptedPackages {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, filepath.FromSlash(pkg))
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("reading %s: %v", pkg, err)
			}
			scanned := 0
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
					continue
				}
				scanned++
				rel := path.Join(pkg, e.Name())
				all := everyLine(t, filepath.Join(root, filepath.FromSlash(rel)))
				for _, v := range detectBareExecutableWrites(root, rel, all) {
					t.Errorf("%s:%d: %s", v.File, v.Line, v.Detail)
				}
			}
			// Without this the subtest passes just as happily if the
			// package is renamed out from under it and nothing is
			// scanned at all.
			if scanned == 0 {
				t.Fatalf("scanned no test files under %s; the adoption assertion is vacuous", pkg)
			}
		})
	}
}

// adoptedPackages are the packages held clean whole-file rather than
// only diff-scoped: internal/stresstest is where the ETXTBSY flake was
// measured, and internal/contractverify is where it recurred.
var adoptedPackages = []string{
	"internal/stresstest",
	"internal/contractverify",
}

// everyLine returns a changed-line set covering the whole file, turning
// the diff-scoped detector into a whole-file scan.
func everyLine(t *testing.T, file string) map[int]bool {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}
	set := map[int]bool{}
	for i := range strings.Count(string(data), "\n") + 1 {
		set[i+1] = true
	}
	return set
}

// assertViolationLines compares the reported lines against want,
// reporting the whole violation on a mismatch so a failure names the
// text rather than only a number.
func assertViolationLines(t *testing.T, vs []Violation, want []int) {
	t.Helper()
	if len(vs) != len(want) {
		t.Fatalf("got %d violations, want %d: %+v", len(vs), len(want), vs)
	}
	for i, v := range vs {
		if v.Line != want[i] {
			t.Errorf("violation %d on line %d, want line %d: %s", i, v.Line, want[i], v.Detail)
		}
	}
}
