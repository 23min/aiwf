package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestDetectHistoryAttrition drives the pure core: given the comment lines a
// diff touched, which ones are reported. This is the layer that constructs
// the Violation, so it is also what discharges the firing-fixture meta-gate.
func TestDetectHistoryAttrition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lines     []commentLine
		wantLines []int
	}{
		{
			name:      "clean comment is silent",
			lines:     []commentLine{{line: 3, text: "// holds the lock across the rename"}},
			wantLines: nil,
		},
		{
			name:      "historical phrasing fires",
			lines:     []commentLine{{line: 7, text: "// this used to be a third arm"}},
			wantLines: []int{7},
		},
		{
			name:      "case is ignored",
			lines:     []commentLine{{line: 2, text: "// Prior To This Change it was a nil deref"}},
			wantLines: []int{2},
		},
		{
			name:      "exempt group is silent",
			lines:     []commentLine{{line: 9, text: "// used to be two fields here", exempt: true}},
			wantLines: nil,
		},
		{
			name: "two phrases on one line report once",
			lines: []commentLine{
				{line: 4, text: "// used to be worse — at one point it forked"},
			},
			wantLines: []int{4},
		},
		{
			name:      "held-out drift phrasing does not fire",
			lines:     []commentLine{{line: 5, text: "// fails if the fixture has drifted"}},
			wantLines: nil,
		},
		{
			name:      "held-out guard-rationale phrasing does not fire",
			lines:     []commentLine{{line: 6, text: "// the bug this guards against would drop the row"}},
			wantLines: nil,
		},
		{
			name: "each offending line reports separately",
			lines: []commentLine{
				{line: 4, text: "// used to be one"},
				{line: 5, text: "// still true today"},
				{line: 6, text: "// prior to this change it forked"},
			},
			wantLines: []int{4, 6},
		},
		{
			name:      "excluded broad phrasing does not fire",
			lines:     []commentLine{{line: 8, text: "// a stub used to exercise the error path"}},
			wantLines: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vs := detectHistoryAttrition("internal/example/seam.go", tt.lines)

			var got []int
			for _, v := range vs {
				got = append(got, v.Line)
				if v.Policy != "comment-history-attrition" {
					t.Errorf("violation carries policy %q, want comment-history-attrition", v.Policy)
				}
				if v.File != "internal/example/seam.go" {
					t.Errorf("violation carries file %q, want the path passed in", v.File)
				}
				if !strings.Contains(v.Detail, historyOKMarker) {
					t.Errorf("detail must name the escape marker so the fix is self-explaining; got %q", v.Detail)
				}
			}
			if diff := cmp.Diff(tt.wantLines, got); diff != "" {
				t.Errorf("reported lines mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestHasHistoryOK pins the escape's contract: the reason is mandatory, so a
// bare marker cannot silence the gate.
func TestHasHistoryOK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"absent", "// a plain comment", false},
		{"marker with reason", "//history:ok legacy on-disk format still in the wild", true},
		{"bare marker is not an escape", "//history:ok", false},
		{"marker with only spaces after it", "//history:ok   ", false},
		{"marker mid-line with reason", "// see below //history:ok supported older release", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasHistoryOK(tt.raw); got != tt.want {
				t.Errorf("hasHistoryOK(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

// commentScanFixture is a Go source file exercising every shape the scanner
// must get right: a multi-line //-group, a /* */ span, a string literal
// wearing comment markers, and an escaped group. Line numbers are load-
// bearing — the assertions below index into them.
const commentScanFixture = `package p

// doc line one
// this used to be three arms
func F() {}

/* block start
   used to be different
*/

var s = "// used to be a literal"

//history:ok legacy on-disk format, still parsed on read
// used to be two fields here
var t = 1
`

// TestAddedCommentLines covers the parse layer: comment text is recovered
// with its line numbers, string literals wearing comment markers stay out,
// a block comment contributes every line it spans, and the escape marks its
// whole group.
func TestAddedCommentLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "seam.go")
	if err := os.WriteFile(path, []byte(commentScanFixture), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	all := map[int]bool{}
	for i := 1; i <= 16; i++ {
		all[i] = true
	}

	got, err := addedCommentLines(path, all)
	if err != nil {
		t.Fatalf("addedCommentLines: %v", err)
	}

	want := []commentLine{
		{line: 3, text: "// doc line one"},
		{line: 4, text: "// this used to be three arms"},
		{line: 7, text: "/* block start"},
		{line: 8, text: "   used to be different"},
		{line: 9, text: "*/"},
		{line: 13, text: "//history:ok legacy on-disk format, still parsed on read", exempt: true},
		{line: 14, text: "// used to be two fields here", exempt: true},
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(commentLine{})); diff != "" {
		t.Errorf("scanned comment lines mismatch (-want +got):\n%s", diff)
	}

	// The literal on line 11 must never reach the detector, or every table of
	// forbidden phrasings in this repo would flag itself.
	for _, cl := range got {
		if strings.Contains(cl.text, "a literal") {
			t.Errorf("string literal leaked into the comment scan: %+v", cl)
		}
	}

	// End to end over the fixture: only the two unescaped offenders report.
	var lines []int
	for _, v := range detectHistoryAttrition("seam.go", got) {
		lines = append(lines, v.Line)
	}
	if diff := cmp.Diff([]int{4, 8}, lines); diff != "" {
		t.Errorf("fixture violations mismatch (-want +got):\n%s", diff)
	}
}

// TestAddedCommentLines_RespectsChangedSet pins the diff scoping at the
// parse layer: an offending comment the diff did not touch is invisible.
func TestAddedCommentLines_RespectsChangedSet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "seam.go")
	if err := os.WriteFile(path, []byte(commentScanFixture), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	got, err := addedCommentLines(path, map[int]bool{4: true})
	if err != nil {
		t.Fatalf("addedCommentLines: %v", err)
	}
	want := []commentLine{{line: 4, text: "// this used to be three arms"}}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(commentLine{})); diff != "" {
		t.Errorf("scanned comment lines mismatch (-want +got):\n%s", diff)
	}
}

// TestAddedCommentLines_UnparseableFile pins that a file that does not parse
// surfaces as an error for the caller to skip, rather than a panic.
func TestAddedCommentLines_UnparseableFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "broken.go")
	if err := os.WriteFile(path, []byte("package p\n\nfunc F( {\n"), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if _, err := addedCommentLines(path, map[int]bool{1: true}); err == nil {
		t.Fatal("want a parse error for malformed source, got nil")
	}
}

// TestCommentHistoryViolations_Seam drives the full IO shell against a real
// git repo: `git diff <base>` → changedLines → parse → detector. It proves
// the resolver is wired, not just the pure layers.
//
// It reuses skillFixtureBase (the sibling diff-scoped policy's repo builder)
// rather than standing up a second near-identical one.
func TestCommentHistoryViolations_Seam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		head      string
		wantLines []int
	}{
		{
			name:      "added offending comment fires",
			head:      "package p\n\n// this used to be a third arm\nfunc F() {}\n",
			wantLines: []int{3},
		},
		{
			name:      "escaped comment is silent",
			head:      "package p\n\n//history:ok legacy format still on disk\n// used to be a third arm\nfunc F() {}\n",
			wantLines: nil,
		},
		{
			name:      "clean comment is silent",
			head:      "package p\n\n// guards against a nil resolver\nfunc F() {}\n",
			wantLines: nil,
		},
		{
			name:      "unparseable file is skipped, not fatal",
			head:      "package p\n\n// this used to be a third arm\nfunc F( {\n",
			wantLines: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, baseSHA := skillFixtureBase(t)
			writeFile("internal/seam/seam.go", tt.head)
			runGit("add", "-A")
			runGit("commit", "-m", "head")

			vs, err := commentHistoryViolations(root, baseSHA)
			if err != nil {
				t.Fatalf("commentHistoryViolations: %v", err)
			}
			var got []int
			for _, v := range vs {
				got = append(got, v.Line)
				if v.File != "internal/seam/seam.go" {
					t.Errorf("violation carries file %q, want the repo-relative path", v.File)
				}
			}
			if diff := cmp.Diff(tt.wantLines, got); diff != "" {
				t.Errorf("reported lines mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestCommentHistoryViolations_UntouchedFileIsInvisible pins the property
// that makes the gate adoptable on a tree already carrying the pattern: a
// file the diff does not touch is never scanned.
func TestCommentHistoryViolations_UntouchedFileIsInvisible(t *testing.T) {
	t.Parallel()

	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile("internal/seam/old.go", "package p\n\n// this used to be a third arm\nfunc F() {}\n")
	runGit("add", "-A")
	runGit("commit", "-m", "pre-existing offender")
	baseSHA := trimLine(runGit("rev-parse", "HEAD"))

	writeFile("internal/seam/new.go", "package p\n\n// a clean addition\nfunc G() {}\n")
	runGit("add", "-A")
	runGit("commit", "-m", "head")

	vs, err := commentHistoryViolations(root, baseSHA)
	if err != nil {
		t.Fatalf("commentHistoryViolations: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("want no violations for an untouched file, got %+v", vs)
	}
}

// TestCommentHistoryViolations_NoBase pins the no-comparison-point contract
// both env shapes rely on.
func TestCommentHistoryViolations_NoBase(t *testing.T) {
	t.Parallel()

	for _, base := range []string{"", "   ", zeroSHA} {
		vs, err := commentHistoryViolations(t.TempDir(), base)
		if err != nil {
			t.Fatalf("base %q: unexpected error: %v", base, err)
		}
		if vs != nil {
			t.Errorf("base %q: want nil violations, got %+v", base, vs)
		}
	}
}

// TestCommentHistoryViolations_BadBase pins that an unresolvable base ref
// surfaces as an error rather than silently auditing nothing — a gate that
// no-ops on a broken invocation is worse than one that fails loudly.
func TestCommentHistoryViolations_BadBase(t *testing.T) {
	t.Parallel()

	root, _, _, _ := skillFixtureBase(t)
	if _, err := commentHistoryViolations(root, "refs/heads/no-such-ref"); err == nil {
		t.Fatal("want an error for an unresolvable base ref, got nil")
	}
}

// TestPolicy_CommentHistoryAttrition is the CI gate entry point. It runs the
// diff-scoped audit against the live tree using the base ref supplied via
// AIWF_COVERAGE_BASE. Without a base (the default in the broad
// `go test ./...` job) it skips — the authoritative invocation is the
// dedicated CI coverage-gate step and `make coverage-gate`.
func TestPolicy_CommentHistoryAttrition(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		t.Skip("AIWF_COVERAGE_BASE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicyCommentHistoryAttrition)
}

// TestPolicyCommentHistoryAttrition_Env drives the env-fed entry point so the
// wrapper body is exercised during profile generation. Serial (t.Setenv
// panics under t.Parallel) and documented in setup_test.go's skip-list.
func TestPolicyCommentHistoryAttrition_Env(t *testing.T) {
	// Unset base → no-op.
	t.Setenv("AIWF_COVERAGE_BASE", "")
	vs, err := PolicyCommentHistoryAttrition(t.TempDir())
	if err != nil {
		t.Fatalf("unset base: unexpected error: %v", err)
	}
	if vs != nil {
		t.Fatalf("unset base: want nil violations, got %+v", vs)
	}

	// Set base → delegates and surfaces the offending comment.
	root, runGit, writeFile, baseSHA := skillFixtureBase(t)
	writeFile("internal/seam/seam.go", "package p\n\n// this used to be a third arm\nfunc F() {}\n")
	runGit("add", "-A")
	runGit("commit", "-m", "head")

	t.Setenv("AIWF_COVERAGE_BASE", baseSHA)
	vs, err = PolicyCommentHistoryAttrition(root)
	if err != nil {
		t.Fatalf("set base: unexpected error: %v", err)
	}
	if len(vs) != 1 || vs[0].Line != 3 {
		t.Fatalf("set base: want one violation on line 3, got %+v", vs)
	}
}

// TestCommentHistoryAttrition_WiredIntoCoverageGate pins that the gate runs
// at the integration boundary: the policy test is named in the coverage-gate
// run-pattern of both the CI workflow and the Makefile target. Without this
// a future edit could drop it from the run set and it would silently never
// fire.
func TestCommentHistoryAttrition_WiredIntoCoverageGate(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	const testName = "CommentHistoryAttrition"

	for _, f := range []string{".github/workflows/go.yml", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if !strings.Contains(string(data), testName) {
			t.Errorf("%s does not name %s in its coverage-gate run pattern; the gate would never run", f, testName)
		}
	}
}

// TestWfCodebaseHealth_F2NamesTheMechanicalCompanion pins that the rubric's
// comment force routes to the scanner before the read, and — the part that
// matters for a skill materialized into repos with no such tooling — that it
// stays conditional with the read as the fallback, mirroring how wf-vacuity
// references a mutation harness.
//
// Scoped to the F2 section rather than grepped over the body: a bare
// substring would pass on a stray mention anywhere in a 750-line rubric.
func TestWfCodebaseHealth_F2NamesTheMechanicalCompanion(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfCodebaseHealthFixturePath)

	f2 := extractMarkdownSection(body, 3, "F2.")
	if f2 == "" {
		t.Fatal("wf-codebase-health must have a `### F2. …` comments principle")
	}
	low := strings.ToLower(f2)

	if !strings.Contains(low, "comment-history-audit") {
		t.Error("F2 must name a whole-tree audit target, or the rubric's comment force has no mechanical route")
	}
	if !strings.Contains(low, "if the project") {
		t.Error("F2 must condition the scanner on the project having one wired up — the skill materializes into repos that do not")
	}
	if !strings.Contains(low, "where no such scanner") {
		t.Error("F2 must give the read as the fallback when no scanner exists, or the force is unusable in those repos")
	}
	if !strings.Contains(low, "cannot classify") {
		t.Error("F2 must say a clean report is the floor, not the finding — otherwise the scanner reads as a substitute for the review")
	}
}

// TestCommentHistoryAudit_TargetIsWired pins the advisory whole-tree target:
// it must exist, drive the same policy test the gate uses (one matcher, not a
// second implementation), and stay advisory. A target that started failing the
// build would turn pre-existing debt into a blocked change.
func TestCommentHistoryAudit_TargetIsWired(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	mk := string(data)

	target := extractMakeTarget(mk, "comment-history-audit")
	if target == "" {
		t.Fatal("Makefile must define a comment-history-audit target for the rubric's mechanical companion")
	}
	if !strings.Contains(target, "TestPolicy_CommentHistoryAttrition") {
		t.Error("the audit target must drive the same policy the gate runs, not a separate scanner that can drift")
	}
	if !strings.Contains(target, "hash-object -t tree /dev/null") {
		t.Error("the audit target must use the empty tree as its base — that is what makes the diff-scoped policy cover the whole tree")
	}
	if !strings.Contains(target, "|| true") {
		t.Error("the audit target must stay advisory (exit 0); whole-tree findings are pre-existing debt, not a gate")
	}
}

// extractMakeTarget returns the recipe lines of a make target: everything from
// the `<name>:` line up to the next line that is neither indented nor blank.
func extractMakeTarget(makefile, name string) string {
	lines := strings.Split(makefile, "\n")
	var out []string
	inTarget := false
	for _, ln := range lines {
		if strings.HasPrefix(ln, name+":") {
			inTarget = true
			continue
		}
		if !inTarget {
			continue
		}
		if ln == "" || strings.HasPrefix(ln, "\t") || strings.HasPrefix(ln, " ") {
			out = append(out, ln)
			continue
		}
		break
	}
	return strings.Join(out, "\n")
}
