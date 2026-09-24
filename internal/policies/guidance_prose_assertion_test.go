package policies

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guidanceFixtureHeader is a synthetic package reaching development
// guidance the ways real tests do: a literal path, a helper that reads the
// file for the test, and a routed on-demand document.
const guidanceFixtureHeader = `package pkg

func readClaude(t *testing.T) string {
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	return string(data)
}

func derived() string { return "" }

func markdownSection(content, heading string) string { return content }
`

// fixtureNamesGuidance is the guidance-path predicate the pure-core
// fixtures use: the two root entry points, nested ones, the router, and
// one routed document.
func fixtureNamesGuidance(s string) bool {
	return namesGuidancePath(s, map[string]bool{"docs/dev/testing.md": true})
}

// TestDetectGuidanceProseAssertions is M-0333 AC-6's rule over test
// source: a test asserting a phrase it wrote is present in development
// guidance is refused; an absence assertion, and an expectation drawn
// from code or another artefact, are not; a test named in the ledger is
// carried until its pin is retired.
func TestDetectGuidanceProseAssertions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want int
	}{
		{
			name: "a presence assertion over CLAUDE.md is refused",
			body: `func TestPin(t *testing.T) {
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "a presence assertion over AGENTS.md is refused",
			body: `func TestPin(t *testing.T) {
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), "AGENTS.md"))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "a nested CLAUDE.md counts",
			body: `func TestPin(t *testing.T) {
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), "internal/pkg/CLAUDE.md"))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "a routed on-demand document counts",
			body: `func TestPin(t *testing.T) {
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), "docs/dev/testing.md"))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			// The section lookup scoping the assertion is reported with it,
			// as the shipped surface reports it: it goes when the pin goes.
			name: "a helper that reads the file is caught at the assertion",
			body: `func TestPin(t *testing.T) {
	section := markdownSection(readClaude(t), "## Rules")
	if !strings.Contains(section, "keep this sentence") {
		t.Error("missing")
	}
}`,
			want: 2,
		},
		{
			name: "an absence assertion is allowed",
			body: `func TestBan(t *testing.T) {
	if strings.Contains(readClaude(t), "a retired sentence") {
		t.Error("retired text is back")
	}
}`,
		},
		{
			name: "an absence assertion through an index comparison is allowed",
			body: `func TestBan(t *testing.T) {
	if strings.Index(readClaude(t), "a retired sentence") >= 0 {
		t.Error("retired text is back")
	}
}`,
		},
		{
			name: "a presence assertion through an index comparison is refused",
			body: `func TestPin(t *testing.T) {
	if strings.Index(readClaude(t), "keep this sentence") < 0 {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "an expectation derived from code is allowed",
			body: `func TestRelationship(t *testing.T) {
	want := derived()
	if !strings.Contains(readClaude(t), want) {
		t.Error("missing")
	}
}`,
		},
		{
			name: "a document outside the guidance set is not judged",
			body: `func TestOther(t *testing.T) {
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), "docs/other.md"))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
		},
		{
			name: "a fixture repository's CLAUDE.md is a test of code, not guidance",
			body: `func TestInitPreservesContent(t *testing.T) {
	dir := t.TempDir()
	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if !strings.Contains(string(data), "user content") {
		t.Error("lost")
	}
}`,
		},
		{
			name: "a root bound to a local still roots the path",
			body: `func TestPin(t *testing.T) {
	root := repoRoot(t)
	data, _ := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "a function-local path constant joined to the root counts",
			body: `func TestPin(t *testing.T) {
	const relPath = "CLAUDE.md"
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), relPath))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "a function-local constant names nothing outside its function",
			body: `func load(t *testing.T, relPath string) string {
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), relPath))
	return string(data)
}

func TestSkill(t *testing.T) {
	if !strings.Contains(load(t, "internal/skills/embedded/x/SKILL.md"), "a trigger phrase") {
		t.Error("missing")
	}
}

func TestOther(t *testing.T) {
	const relPath = "CLAUDE.md"
	_ = relPath
}`,
		},
		{
			name: "a reader that roots the path itself counts",
			body: `func readRepoFile(t *testing.T, rel string) string {
	raw, _ := os.ReadFile(filepath.Join(repoRoot(t), rel))
	return string(raw)
}

func TestPin(t *testing.T) {
	if !strings.Contains(readRepoFile(t, "CLAUDE.md"), "keep this sentence") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "a ledger entry is carried",
			body: `func TestGrandfatheredFixturePin(t *testing.T) {
	if !strings.Contains(readClaude(t), "keep this sentence") {
		t.Error("missing")
	}
}`,
		},
	}
	ledger := map[string]string{"TestGrandfatheredFixturePin": "fixture"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fset, files, paths := parseSyntheticPackage(t, map[string]string{
				"header.go": guidanceFixtureHeader,
				"a_test.go": "package pkg\n\n" + tt.body + "\n",
			})
			got := detectGuidanceProseAssertions(fset, files, paths, fixtureNamesGuidance, ledger)
			if len(got) != tt.want {
				t.Fatalf("got %d violations %+v, want %d", len(got), got, tt.want)
			}
			for _, v := range got {
				if v.Policy != "guidance-prose-assertion" {
					t.Errorf("violation Policy = %q, want guidance-prose-assertion", v.Policy)
				}
			}
		})
	}
}

// TestGuidanceProseAssertion_DiffScoped drives M-0333 AC-6 through git: a
// pin already in the tree at the base is not judged, a test file the range
// adds with a pin is, and one the working tree modifies is too.
func TestGuidanceProseAssertion_DiffScoped(t *testing.T) {
	t.Parallel()
	pin := func(name string) string {
		return "package pkg\n\nimport (\n\t\"os\"\n\t\"path/filepath\"\n\t\"strings\"\n\t\"testing\"\n)\n\nfunc " + name +
			"(t *testing.T) {\n\tdata, _ := os.ReadFile(filepath.Join(repoRoot(t), \"CLAUDE.md\"))\n\tif !strings.Contains(string(data), \"keep this sentence\") {\n\t\tt.Error(\"missing\")\n\t}\n}\n"
	}
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile("CLAUDE.md", "keep this sentence\n")
	writeFile("pkg/old_test.go", pin("TestOldPin"))
	runGit("add", "-A")
	runGit("commit", "-q", "-m", "seed")
	base := trimLine(runGit("rev-parse", "HEAD"))

	if got := guidanceProseFiles(t, root, base); len(got) != 0 {
		t.Fatalf("a pin already at the base must not be judged; got %v", got)
	}
	writeFile("pkg/new_test.go", pin("TestNewPin"))
	runGit("add", "-A")
	runGit("commit", "-q", "-m", "add a pin")
	if got, want := guidanceProseFiles(t, root, base), []string{"pkg/new_test.go"}; !equalStrings(got, want) {
		t.Errorf("a committed new pin: files = %v, want %v", got, want)
	}
	writeFile("pkg/old_test.go", pin("TestOldPin")+"\n// touched\n")
	if got, want := guidanceProseFiles(t, root, base), []string{"pkg/new_test.go", "pkg/old_test.go"}; !equalStrings(got, want) {
		t.Errorf("an uncommitted modification: files = %v, want %v", got, want)
	}
	if got := guidanceProseFiles(t, root, ""); len(got) != 0 {
		t.Errorf("no base no-ops; got %v", got)
	}
}

func guidanceProseFiles(t *testing.T, root, base string) []string {
	t.Helper()
	vs, err := guidanceProseViolations(root, base, map[string]string{})
	if err != nil {
		t.Fatalf("guidanceProseViolations: %v", err)
	}
	return violationFiles(vs)
}

// TestGuidanceProseLedger_NoStaleEntries holds the ledger to the tree in
// both directions: every entry must name a test the scan still flags over
// the whole tree, so retiring a pin forces its entry's deletion, and every
// flagged test must be an entry. New entries are held at review, as the
// firing-fixture ledger's are.
func TestGuidanceProseLedger_NoStaleEntries(t *testing.T) {
	t.Parallel()
	flagged, err := guidanceProseFlaggedTests(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	for name := range guidanceProseLedger {
		if !flagged[name] {
			t.Errorf("ledger entry %s names no prose-presence pin over development guidance; delete the entry", name)
		}
	}
	// The other direction: a pin created by a change outside its own test
	// file — a constant moved to name CLAUDE.md — escapes the diff-scoped
	// gate, and is caught here instead.
	for name := range flagged {
		if _, ok := guidanceProseLedger[name]; !ok {
			t.Errorf("%s pins development-guidance prose and is not in the ledger; state its claim as a relationship check or an observation", name)
		}
	}
}

// TestPolicy_GuidanceProseAssertion is the gate entry point, run with the
// base in AIWF_COVERAGE_BASE by the coverage-gate step and
// `make coverage-gate`; without one it skips.
func TestPolicy_GuidanceProseAssertion(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		t.Skip("AIWF_COVERAGE_BASE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicyGuidanceProseAssertion)
}

// TestGuidanceProseAssertion_WiredIntoCoverageGate pins that the ban runs
// at the integration boundary: it is named in the coverage-gate
// run-pattern of both the CI workflow and the Makefile target.
func TestGuidanceProseAssertion_WiredIntoCoverageGate(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, f := range []string{".github/workflows/go.yml", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if line := coverageGateRunLine(t, f, string(data)); !strings.Contains(line, "|GuidanceProseAssertion)") && !strings.Contains(line, "|GuidanceProseAssertion|") {
			t.Errorf("%s: coverage-gate run-pattern does not include GuidanceProseAssertion:\n  %s", f, line)
		}
	}
}

// TestConditionPolarity pins how a condition reads against a phrase: it
// is evaluated with the call at its not-found value, on either side of a
// comparison, and anything else is unknown.
func TestConditionPolarity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cond string
		want polarity
	}{
		{`strings.Contains(d, "x")`, polPresent},
		{`!strings.Contains(d, "x")`, polAbsent},
		{`strings.Contains(d, "x") == false`, polAbsent},
		{`false == strings.Contains(d, "x")`, polAbsent},
		{`strings.Contains(d, "x") != true`, polAbsent},
		{`strings.Contains(d, "x") == true`, polPresent},
		{`strings.Index(d, "x") == -1`, polAbsent},
		{`-1 == strings.Index(d, "x")`, polAbsent},
		{`strings.Index(d, "x") != -1`, polPresent},
		{`strings.Index(d, "x") < 0`, polAbsent},
		{`strings.Index(d, "x") >= 0`, polPresent},
		{`strings.Index(d, "x") != 0`, polAbsent},
		{`strings.Index(d, "x") == 0`, polPresent},
		{`strings.Count(d, "x") == 0`, polAbsent},
		{`0 == strings.Count(d, "x")`, polAbsent},
		{`strings.Count(d, "x") < 1`, polAbsent},
		{`strings.Count(d, "x") != 1`, polAbsent},
		{`strings.Count(d, "x") >= 1`, polPresent},
		{`0 < strings.Count(d, "x")`, polPresent},
		{`strings.Count(d, "x") == 0x0`, polAbsent},
		{`!(strings.Contains(d, "x") && ok)`, polAbsent},
		{`strings.Index(d, "x") > limit`, polUnknown},
		{`strings.Contains(d, "x") != tc.want`, polUnknown},
		{`strings.Count(d, "x") > 99999999999999999999`, polUnknown},
		{`strings.Contains(d, "x") + 1`, polPresent},
	}
	for _, tt := range tests {
		t.Run(tt.cond, func(t *testing.T) {
			t.Parallel()
			e, err := parser.ParseExpr(tt.cond)
			if err != nil {
				t.Fatal(err)
			}
			got := map[*ast.CallExpr]polarity{}
			conditionPolarity(e, false, got)
			var verdict *ast.CallExpr
			for ce := range got {
				if name := calleeFuncName(ce.Fun); strings.HasPrefix(name, "strings.") {
					verdict = ce
				}
			}
			if verdict == nil {
				t.Fatal("no strings call recorded")
			}
			if got[verdict] != tt.want {
				t.Errorf("polarity = %v, want %v", got[verdict], tt.want)
			}
		})
	}
}

// TestGuidanceProseAssertion_PathShapes pins the path forms the table
// discipline produces: a path named before it is rooted, a table row, and
// a root handed back by a helper that does not read the file itself.
func TestGuidanceProseAssertion_PathShapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, body string
	}{
		{
			name: "a local path rooted at the read",
			body: `func TestPin(t *testing.T) {
	p := "CLAUDE.md"
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), p))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
		},
		{
			name: "a table row rooted at the read",
			body: `func TestPin(t *testing.T) {
	tests := []struct{ path, phrase string }{{"CLAUDE.md", "keep this sentence"}}
	root := repoRoot(t)
	for _, tc := range tests {
		data, _ := os.ReadFile(filepath.Join(root, tc.path))
		if !strings.Contains(string(data), "keep this sentence") {
			t.Error("missing")
		}
	}
}`,
		},
		{
			name: "a table row handed to a reader that roots it",
			body: `func readRepoFile(t *testing.T, rel string) string {
	raw, _ := os.ReadFile(filepath.Join(repoRoot(t), rel))
	return string(raw)
}

func TestPin(t *testing.T) {
	for _, tc := range []struct{ path string }{{"CLAUDE.md"}} {
		if !strings.Contains(readRepoFile(t, tc.path), "keep this sentence") {
			t.Error("missing")
		}
	}
}`,
		},
		{
			name: "a root from a helper that does not read",
			body: `func sharedRoot(t *testing.T) (string, int) {
	return repoRoot(t), 0
}

func TestPin(t *testing.T) {
	root, _ := sharedRoot(t)
	data, _ := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if !strings.Contains(string(data), "keep this sentence") {
		t.Error("missing")
	}
}`,
		},
		{
			name: "a presence assertion written with the call on the right",
			body: `func TestPin(t *testing.T) {
	if false == strings.Contains(readClaude(t), "keep this sentence") {
		t.Error("missing")
	}
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fset, files, paths := parseSyntheticPackage(t, map[string]string{
				"header.go": guidanceFixtureHeader,
				"a_test.go": "package pkg\n\n" + tt.body + "\n",
			})
			if got := detectGuidanceProseAssertions(fset, files, paths, fixtureNamesGuidance, nil); len(got) == 0 {
				t.Error("want the pin reported, got none")
			}
		})
	}
}

// TestDetectProseAssertions_LocalVarHoldsTheDocument pins, on the shipped
// surface, that a local var initialized from a read carries the document
// to the assertion.
func TestDetectProseAssertions_LocalVarHoldsTheDocument(t *testing.T) {
	t.Parallel()
	fset, files, paths := parseSyntheticPackage(t, map[string]string{
		"header.go": fixtureHeader,
		"a_test.go": `package pkg

func TestPin(t *testing.T) {
	var body = readSkill(t, ritualPath)
	section := extractMarkdownSection(body, 2, "Steps")
	if !strings.Contains(section, "keep this sentence") {
		t.Error("missing")
	}
}
`,
	})
	if got := detectProseAssertions(fset, files, paths); len(got) == 0 {
		t.Error("want the pin reported, got none")
	}
}

// TestGuidanceProseAssertion_Errors covers the paths that cannot answer: a
// base naming no commit, an unparseable test file, and a tree whose walk
// fails. A file under testdata is a fixture and is not read. The entry point no-ops without a base.
func TestGuidanceProseAssertion_Errors(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile("pkg/a_test.go", "package pkg\n")
	writeFile(".guidance/project.md", "Read [t](../docs/dev/testing.md).\n")
	runGit("add", "-A")
	runGit("commit", "-q", "-m", "seed")
	base := trimLine(runGit("rev-parse", "HEAD"))
	if _, err := guidanceProseViolations(root, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", nil); err == nil {
		t.Error("a base naming no commit: want an error")
	}
	writeFile("pkg/testdata/broken_test.go", "package pkg\n\nfunc TestX(t *testing.T) {\n")
	if _, err := guidanceProseViolations(root, base, nil); err != nil {
		t.Errorf("a broken file under testdata is a fixture, not a test: %v", err)
	}
	writeFile("pkg/wip_test.go", "package pkg\n\nfunc TestWIP(t *testing.T) {\n")
	if _, err := guidanceProseViolations(root, base, nil); err == nil {
		t.Error("an unparseable changed test file: want an error")
	}
	if _, err := guidanceProseFlaggedTests(root); err == nil {
		t.Error("an unparseable test file in the tree: want an error")
	}
	if err := os.Remove(filepath.Join(root, "pkg", "wip_test.go")); err != nil {
		t.Fatal(err)
	}
	if !repoNamesGuidance(root)("docs/dev/testing.md") {
		t.Error("a routed document must name guidance")
	}
	if os.Geteuid() != 0 {
		denied := filepath.Join(root, "pkg")
		if err := os.Chmod(denied, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })
		if _, err := guidanceProseFlaggedTests(root); err == nil {
			t.Error("an unwalkable tree: want an error")
		}
	}
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		if vs, err := PolicyGuidanceProseAssertion(repoRoot(t)); err != nil || len(vs) != 0 {
			t.Errorf("without a base: %v, %v; want none", vs, err)
		}
	}
}
