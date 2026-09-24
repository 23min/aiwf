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

// TestGuidanceProseLedger_NoStaleEntries holds the ledger to shrinking:
// every entry must name a test the scan still flags over the whole tree,
// so retiring a pin forces its entry's deletion. New entries are held at
// review, as the firing-fixture ledger's are.
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

// TestConditionPolarity pins how a condition reads against a phrase: which
// spellings mean "the phrase is absent" when the condition is true.
func TestConditionPolarity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cond   string
		absent bool
	}{
		{`strings.Contains(d, "x")`, false},
		{`!strings.Contains(d, "x")`, true},
		{`strings.Contains(d, "x") == false`, true},
		{`strings.Contains(d, "x") != true`, true},
		{`strings.Contains(d, "x") == true`, false},
		{`strings.Contains(d, "x") != false`, false},
		{`strings.Index(d, "x") == -1`, true},
		{`strings.Index(d, "x") <= -1`, true},
		{`strings.Index(d, "x") != -1`, false},
		{`strings.Index(d, "x") > -1`, false},
		{`strings.Index(d, "x") < 0`, true},
		{`strings.Index(d, "x") >= 0`, false},
		{`strings.Count(d, "x") == 0`, true},
		{`strings.Count(d, "x") > 0`, false},
		{`!(strings.Contains(d, "x") && ok)`, true},
		{`strings.Index(d, "x") > limit`, false},
	}
	for _, tt := range tests {
		t.Run(tt.cond, func(t *testing.T) {
			t.Parallel()
			e, err := parser.ParseExpr(tt.cond)
			if err != nil {
				t.Fatal(err)
			}
			got := map[*ast.CallExpr]bool{}
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
			if got[verdict] != tt.absent {
				t.Errorf("true when absent = %v, want %v", got[verdict], tt.absent)
			}
		})
	}
}

// TestGuidanceProseAssertion_Errors covers the paths that cannot answer:
// a base naming no commit and a tree whose walk fails. The entry point no-ops without a base.
func TestGuidanceProseAssertion_Errors(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile("pkg/a_test.go", "package pkg\n")
	writeFile(".guidance/project.md", "Read [t](../docs/dev/testing.md).\n")
	runGit("add", "-A")
	runGit("commit", "-q", "-m", "seed")
	if _, err := guidanceProseViolations(root, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", nil); err == nil {
		t.Error("a base naming no commit: want an error")
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
