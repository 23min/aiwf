package policies

import (
	"os"
	"os/exec"
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
				// Naming the marker is not enough: the placement rule is
				// what an operator gets wrong, and a message that leaves
				// them blocked after following it invites --no-verify.
				for _, want := range []string{historyOKMarker, "must open"} {
					if !strings.Contains(v.Detail, want) {
						t.Errorf("detail must name %q so the fix is self-explaining; got %q", want, v.Detail)
					}
				}
			}
			if diff := cmp.Diff(tt.wantLines, got); diff != "" {
				t.Errorf("reported lines mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAddedCommentLines_EscapeScope pins which comments the escape opens, at
// the layer that applies it. The matcher's own contract is TestHasDirectiveComment;
// what is pinned here is the group scope it feeds — one directive line covers
// the paragraph it sits in, and nothing else does.
func TestAddedCommentLines_EscapeScope(t *testing.T) {
	t.Parallel()

	// offender is appended below every group and is the line whose exemption
	// each case turns on. It carries a phrase from the trigger set, which the
	// control below re-establishes per case rather than assuming.
	const offender = "// used to be a third arm"

	tests := []struct {
		name string
		// group is a comment block placed above a func; offender follows it,
		// joining the same comment group unless the group ends in a blank line.
		group string
		// changed is the diff's changed-line set, nil meaning every line. A
		// case naming specific lines is testing what the escape covers when
		// the diff did not touch the directive itself.
		changed    []int
		wantExempt bool
	}{
		{
			name:       "a directive line exempts the whole group",
			group:      "//history:ok legacy on-disk format, still parsed on read\n// a second line of the same note",
			wantExempt: true,
		},
		{
			name:       "a directive line below the offender still exempts the group",
			group:      "// a note that opens the group\n//history:ok legacy on-disk format, still parsed on read",
			wantExempt: true,
		},
		{
			// gofmt reflows a doc comment carrying a directive into this
			// shape, moving the directive below a bare `//` separator. Since
			// no directive exists in the tree today, every one that lands
			// will be in it.
			name:       "a bare // separator does not split the group",
			group:      "// a note that opens the group\n//\n//history:ok legacy on-disk format, still parsed on read",
			wantExempt: true,
		},
		{
			// The everyday case: editing the prose of an already-escaped note.
			// Scoping the exemption scan to changed lines would block the push
			// unless the operator retyped the directive line.
			name:       "an untouched directive still exempts a line the diff touched",
			group:      "//history:ok legacy on-disk format, still parsed on read",
			changed:    []int{4},
			wantExempt: true,
		},
		{
			name:       "prose naming the escape does not exempt",
			group:      "// F is documented; see the history:ok escape in CLAUDE.md.",
			wantExempt: false,
		},
		{
			name:       "a longer word opening with the marker does not exempt",
			group:      "//history:okay",
			wantExempt: false,
		},
		{
			// A marker on an interior line of a block comment is text, not a
			// directive — the same reading Go gives //go:build. A block
			// comment demonstrating the escape must not silence the gate.
			name:       "a marker inside a block comment does not exempt",
			group:      "/*\n//history:ok legacy on-disk format\n*/",
			wantExempt: false,
		},
		{
			name:       "a neighbouring group's directive does not reach across",
			group:      "//history:ok legacy on-disk format, still parsed on read\n\n// an unrelated note",
			wantExempt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "seam.go")
			src := "package p\n\n" + tt.group + "\n" + offender + "\nfunc F() {}\n"
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatalf("writing fixture: %v", err)
			}

			changed := map[int]bool{}
			if tt.changed == nil {
				for i := 1; i <= strings.Count(src, "\n")+1; i++ {
					changed[i] = true
				}
			} else {
				for _, ln := range tt.changed {
					changed[ln] = true
				}
			}

			lines, err := addedCommentLines(path, changed)
			if err != nil {
				t.Fatalf("addedCommentLines: %v", err)
			}

			// Locate the offending line rather than inferring from silence.
			// Absent, the assertions below would be satisfied by a fixture the
			// scan never reached — which is what "no findings" alone means.
			var off *commentLine
			for i := range lines {
				if lines[i].text == offender {
					off = &lines[i]
					break
				}
			}
			if off == nil {
				t.Fatalf("the offending line never reached the scan; scanned %+v\nsource:\n%s", lines, src)
			}

			// Control: the offending line must be one the detector fires on
			// when unexempted. Without this, an offender that stopped
			// triggering would make every wantExempt case pass vacuously.
			if got := detectHistoryAttrition("seam.go", []commentLine{{line: off.line, text: off.text}}); len(got) != 1 {
				t.Fatalf("fixture control: %q produced %d findings unexempted, want 1", off.text, len(got))
			}

			if off.exempt != tt.wantExempt {
				t.Errorf("offending line exempt = %v, want %v; source:\n%s", off.exempt, tt.wantExempt, src)
			}

			wantFindings := 1
			if tt.wantExempt {
				wantFindings = 0
			}
			if got := detectHistoryAttrition("seam.go", lines); len(got) != wantFindings {
				t.Errorf("end to end: %d findings, want %d; got %+v\nsource:\n%s", len(got), wantFindings, got, src)
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

// TestPolicy_CommentHistoryAttritionTree is the whole-tree gate. Unlike its
// diff-scoped sibling it takes no environment, so it runs in the ordinary
// policy suite and fails the build on an offending comment anywhere in the
// tree — not only on lines the current change touched.
func TestPolicy_CommentHistoryAttritionTree(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyCommentHistoryAttritionTree)
}

// TestEmptyTreeOID_MatchesGit pins that the base the whole-tree scan diffs
// against really is the empty tree: every tracked Go file must appear in that
// diff. A wrong base would silently narrow the gate to nothing while still
// reporting green.
func TestEmptyTreeOID_MatchesGit(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	oid, err := emptyTreeOID(root)
	if err != nil {
		t.Fatalf("emptyTreeOID: %v", err)
	}
	changed, err := changedLines(root, oid)
	if err != nil {
		t.Fatalf("changedLines against the empty tree: %v", err)
	}

	// The file list comes from HEAD rather than the index, because that is
	// what changedLines diffs the empty tree against. `git ls-files` reads the
	// index, so with a merge staged but not yet committed — exactly the state
	// the pre-commit hook runs in — it reports files HEAD does not carry, and
	// this assertion would fail on a tree that is perfectly fine.
	out, err := exec.Command("git", "-C", root, "ls-tree", "-r", "--name-only", "HEAD").Output()
	if err != nil {
		t.Fatalf("git ls-tree: %v", err)
	}
	var tracked int
	for _, ln := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if !strings.HasSuffix(ln, ".go") {
			continue
		}
		// A file HEAD carries but the working tree no longer has is a pending
		// deletion. changedLines diffs the empty tree against the working
		// tree, so a deleted file yields no hunk and is legitimately
		// unscannable — the commit that removes it is exactly when HEAD and
		// the working tree disagree, and flagging it here blocks that commit.
		if _, statErr := os.Stat(filepath.Join(root, ln)); os.IsNotExist(statErr) {
			continue
		}
		tracked++
		if changed[ln] == nil {
			t.Errorf("%s is in HEAD but absent from the empty-tree diff — the whole-tree scan would skip it", ln)
		}
	}
	if tracked == 0 {
		t.Fatal("no tracked Go files found; the assertion would be vacuous")
	}
}

// TestPolicyCommentHistoryAttritionTree_NonRepoErrors pins that the gate
// fails loudly where it cannot do its job. Resolving the base succeeds
// anywhere (hashing a fixed input needs no repository), so the failure
// surfaces at the diff — and it must surface, because a gate that reported a
// clean tree when it had scanned nothing is worse than no gate.
func TestPolicyCommentHistoryAttritionTree_NonRepoErrors(t *testing.T) {
	t.Parallel()

	if _, err := PolicyCommentHistoryAttritionTree(t.TempDir()); err == nil {
		t.Fatal("want an error outside a git repository, got a clean report")
	}
}

// TestCommentHistoryAudit_TargetIsWired pins the focused whole-tree target:
// it must exist, drive the whole-tree policy (one matcher, not a second
// scanner that can drift), and not suppress its exit code — the same finding
// already fails the ordinary suite, so a target reporting green while the
// build goes red would be lying.
func TestCommentHistoryAudit_TargetIsWired(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}

	target := extractMakeTarget(string(data), "comment-history-audit")
	if target == "" {
		t.Fatal("Makefile must define a comment-history-audit target for the rubric's mechanical companion")
	}
	if !strings.Contains(target, "TestPolicy_CommentHistoryAttritionTree") {
		t.Error("the audit target must drive the whole-tree policy, not a separate scanner that can drift from it")
	}
	if strings.Contains(target, "|| true") {
		t.Error("the audit target must not suppress its exit code — the same finding fails the ordinary policy suite")
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
