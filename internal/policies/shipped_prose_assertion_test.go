package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// parseSyntheticPackage parses named sources as one package for the pure core.
func parseSyntheticPackage(t *testing.T, srcs map[string]string) (*token.FileSet, []*ast.File, map[*ast.File]string) {
	t.Helper()
	fset := token.NewFileSet()
	var files []*ast.File
	paths := map[*ast.File]string{}
	names := make([]string, 0, len(srcs))
	for n := range srcs {
		names = append(names, n)
	}
	// Deterministic order: the fixpoint over readers must not depend on it,
	// and a test that silently relied on map order would hide it if it did.
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, n := range names {
		f, err := parser.ParseFile(fset, n, srcs[n], parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", n, err)
		}
		files = append(files, f)
		paths[f] = "pkg/" + n
	}
	return fset, files, paths
}

// fixtureHeader is the shared preamble: a shipped fixture path and the helper
// that reads it, mirroring how the real packages reach these bytes.
const fixtureHeader = `package pkg

const ritualPath = "internal/skills/embedded-rituals/plugins/p/skills/s/SKILL.md"

func readSkill(t *testing.T, rel string) string {
	data, _ := os.ReadFile(rel)
	return string(data)
}

func loadSkill(t *testing.T, rel string) skillRec {
	data, _ := os.ReadFile(rel)
	return skillRec{Content: data}
}

func frontmatterField(body, field string) string { return body }
func extractMarkdownSection(body string, level int, heading string) string { return body }
`

// TestDetectProseAssertions drives the pure core. Each case names one way the
// rule must discriminate, because the value of a ban is entirely in what it
// declines to fire on.
func TestDetectProseAssertions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		body  string
		wantN int
	}{
		{
			name: "a literal phrase over a shipped read fires",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	if !strings.Contains(body, "decision is decision") {
		t.Error("missing")
	}
}`,
			wantN: 1,
		},
		{
			name: "a phrase list ranged over fires once per assertion",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	for _, want := range []string{"alpha phrase", "beta phrase"} {
		if !strings.Contains(body, want) {
			t.Error("missing")
		}
	}
}`,
			wantN: 1,
		},
		{
			// A heading extraction scopes a claim rather than making one. On
			// its own it is scoping a structural check — here, a count — which
			// is the shape D-0050 asks for.
			name: "a heading extraction scoping a structural claim does not fire",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if strings.Count(workflow, "\n### ") != 9 {
		t.Error("wrong step count")
	}
}`,
			wantN: 0,
		},
		{
			// Alongside a prose assertion the same extraction goes too: once
			// the body assertion is deleted it degrades to asserting that a
			// heading exists.
			name: "a heading extraction scoping a prose assertion fires with it",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if !strings.Contains(workflow, "run the gate first") {
		t.Error("missing")
	}
}`,
			wantN: 2,
		},
		{
			name: "a case-folded needle is still test-authored",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	for _, marker := range []string{"approves the gate"} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(marker)) {
			t.Error("missing")
		}
	}
}`,
			wantN: 1,
		},
		{
			name: "taint carries through a narrowed section",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	section := extractMarkdownSection(body, 2, "Workflow")
	lower := strings.ToLower(section)
	if !strings.Contains(lower, "run the gate") {
		t.Error("missing")
	}
}`,
			wantN: 2, // the heading scope, and the phrase inside it
		},
		{
			name: "a needle drawn from a second document does not fire",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	other := readSkill(t, "internal/skills/embedded-rituals/plugins/p/skills/other/SKILL.md")
	heading := firstHeading(other)
	if !strings.Contains(body, heading) {
		t.Error("dangling citation")
	}
}`,
			wantN: 0,
		},
		{
			name: "a shipped path used as data, with no read, does not fire",
			body: `func TestX(t *testing.T) {
	edits := []edit{{Path: ritualPath}}
	got := analyze(edits)
	if !strings.Contains(got[0].Detail, "no trailer here") {
		t.Error("wrong detail")
	}
}`,
			wantN: 0,
		},
		{
			name: "a non-content field of a record derived from a read does not fire",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	found := analyze(body)
	if !strings.Contains(found.Hint, "revert the edit") {
		t.Error("wrong hint")
	}
}`,
			wantN: 0,
		},
		{
			name: "a Content field of a record derived from a read does fire",
			body: `func TestX(t *testing.T) {
	skill := loadSkill(t, ritualPath)
	if !strings.Contains(string(skill.Content), "advisory by design") {
		t.Error("missing")
	}
}`,
			wantN: 1,
		},
		{
			name: "a markdown delimiter is structure, not prose",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") {
			continue
		}
	}
}`,
			wantN: 0,
		},
		{
			name: "content reached through a map still fires",
			body: `func TestX(t *testing.T) {
	byName := map[string]string{}
	byName["s"] = readSkill(t, ritualPath)
	if !strings.Contains(byName["s"], "some shipped phrase") {
		t.Error("missing")
	}
}`,
			wantN: 1,
		},
		{
			// The table-of-cases spelling, where the needle arrives as a field
			// of the ranged struct rather than the range variable itself.
			name: "a needle reached as a struct field is still test-authored",
			body: `func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	for _, m := range []struct{ name, needle string }{
		{"the self-contained framing", "self-contained"},
	} {
		if !strings.Contains(body, m.needle) {
			t.Error("missing")
		}
	}
}`,
			wantN: 1,
		},
		{
			name: "a non-test function is out of scope",
			body: `func helperX(t *testing.T) {
	body := readSkill(t, ritualPath)
	if !strings.Contains(body, "decision is decision") {
		t.Error("missing")
	}
}`,
			wantN: 0,
		},
		{
			name: "an allowlisted test is exempt",
			body: `func TestDeployerCard_FrontmatterDescriptionNamesReleaseTriggers(t *testing.T) {
	body := readSkill(t, ritualPath)
	desc := frontmatterField(body, "description")
	for _, p := range []string{"cut a release", "let's ship"} {
		if !strings.Contains(desc, p) {
			t.Error("missing")
		}
	}
}`,
			wantN: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fset, files, paths := parseSyntheticPackage(t, map[string]string{
				"a_test.go": fixtureHeader + "\n" + tt.body + "\n",
			})
			got := detectProseAssertions(fset, files, paths)
			if len(got) != tt.wantN {
				t.Fatalf("got %d violations, want %d: %+v", len(got), tt.wantN, got)
			}
			for _, v := range got {
				if v.Policy != "shipped-prose-assertion" {
					t.Errorf("Policy = %q, want shipped-prose-assertion", v.Policy)
				}
				if v.Line == 0 || v.File == "" {
					t.Errorf("a violation must name file and line; got %+v", v)
				}
			}
		})
	}
}

// TestDetectProseAssertions_EmbeddedTreeNeedsNoRead pins the skills package's
// access pattern: shipped bytes arrive through a `//go:embed` directive, so a
// rule keyed on file reads alone would see that whole package as touching
// nothing.
func TestDetectProseAssertions_EmbeddedTreeNeedsNoRead(t *testing.T) {
	t.Parallel()

	src := `package pkg

//go:embed embedded-guidance/aiwf-guidance.md
var guidanceEmbed []byte

func GuidanceBytes() []byte { return guidanceEmbed }

func TestGuidance(t *testing.T) {
	got := string(GuidanceBytes())
	if !strings.Contains(got, "single source of truth") {
		t.Error("missing")
	}
}
`
	fset, files, paths := parseSyntheticPackage(t, map[string]string{"g_test.go": src})
	got := detectProseAssertions(fset, files, paths)
	if len(got) != 1 {
		t.Fatalf("want 1 violation for the embedded-tree read, got %d: %+v", len(got), got)
	}

	// Proof the directive is what carries it: an embed of something else is
	// not a shipped surface, and the same assertion must go quiet.
	fset2, files2, paths2 := parseSyntheticPackage(t, map[string]string{
		"g_test.go": strings.Replace(src, "embedded-guidance/aiwf-guidance.md", "testdata/sample.md", 1),
	})
	if got2 := detectProseAssertions(fset2, files2, paths2); len(got2) != 0 {
		t.Errorf("an embed of a non-shipped path must not fire; got %+v", got2)
	}
}

// TestDetectProseAssertions_DetailNamesTheDisposition pins that the message
// tells the reader what to do, including which of the two survivals might
// apply — a bare "not allowed" would send them to the allowlist by default,
// which is the wrong repair for a cross-document check.
func TestDetectProseAssertions_DetailNamesTheDisposition(t *testing.T) {
	t.Parallel()

	fset, files, paths := parseSyntheticPackage(t, map[string]string{
		"a_test.go": fixtureHeader + `
func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	if !strings.Contains(body, "decision is decision") {
		t.Error("missing")
	}
}
`,
	})
	got := detectProseAssertions(fset, files, paths)
	if len(got) != 1 {
		t.Fatalf("want 1 violation, got %d", len(got))
	}
	d := got[0].Detail
	for _, want := range []string{"TestX", "delete the assertion", "derivedExpectationExemptions", "triggerPhraseExemptions"} {
		if !strings.Contains(d, want) {
			t.Errorf("Detail must mention %q so the repair is unambiguous; got %q", want, d)
		}
	}
	if !strings.Contains(d, `"decision is decision"`) {
		t.Errorf("Detail must quote the offending phrase; got %q", d)
	}
}

// TestPolicyShippedProseAssertion_Seam is AC-1's firing fixture: a planted
// assertion in a test that reads an embedded-skill fixture fires and names the
// file and line; removing the plant returns the scan to silence.
func TestPolicyShippedProseAssertion_Seam(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pkg := filepath.Join(root, "internal", "demo")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	planted := fixtureHeader + `
func TestPlanted(t *testing.T) {
	body := readSkill(t, ritualPath)
	if !strings.Contains(body, "a phrase the skill must carry") {
		t.Error("missing")
	}
}
`
	write := func(src string) {
		if err := os.WriteFile(filepath.Join(pkg, "demo_test.go"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write(planted)
	got, err := PolicyShippedProseAssertion(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 violation for the planted assertion, got %d: %+v", len(got), got)
	}
	if got[0].File != "internal/demo/demo_test.go" {
		t.Errorf("File = %q, want the repo-relative test path", got[0].File)
	}
	before, _, found := strings.Cut(planted, "a phrase the skill must carry")
	if !found {
		t.Fatal("fixture no longer carries the planted phrase")
	}
	wantLine := 1 + strings.Count(before, "\n")
	if got[0].Line != wantLine {
		t.Errorf("Line = %d, want %d (the line carrying the phrase)", got[0].Line, wantLine)
	}

	write(strings.Replace(planted, `	if !strings.Contains(body, "a phrase the skill must carry") {
		t.Error("missing")
	}
`, "	_ = body\n", 1))
	got, err = PolicyShippedProseAssertion(root)
	if err != nil {
		t.Fatalf("policy error after removing the plant: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("removing the plant must return the scan to silence; got %+v", got)
	}
}

// TestPolicyShippedProseAssertion_SkipsUnparseableTrees confirms the walk
// surfaces a broken directory rather than reporting a silent pass.
func TestPolicyShippedProseAssertion_Errors(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pkg := filepath.Join(root, "internal", "broken")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "x_test.go"), []byte("package !!!"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := PolicyShippedProseAssertion(root); err == nil {
		t.Error("want an error for a source that does not parse, got nil")
	}

	if _, err := PolicyShippedProseAssertion(filepath.Join(root, "does-not-exist")); err == nil {
		t.Error("want an error for a missing root, got nil")
	}
}

// TestPolicyShippedProseAssertion_SortsByFileThenLine pins the reporting order.
// A scan walking packages in directory order would otherwise report a file's
// findings wherever its package happened to land, which makes a diff of two
// runs unreadable.
func TestPolicyShippedProseAssertion_SortsByFileThenLine(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	two := fixtureHeader + `
func TestFirst(t *testing.T) {
	body := readSkill(t, ritualPath)
	if !strings.Contains(body, "the earlier phrase") {
		t.Error("missing")
	}
	if !strings.Contains(body, "the later phrase") {
		t.Error("missing")
	}
}
`
	for _, pkg := range []string{"zeta", "alpha"} {
		dir := filepath.Join(root, "internal", pkg)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "x_test.go"), []byte(two), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := PolicyShippedProseAssertion(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("want 4 violations across the two packages, got %d: %+v", len(got), got)
	}
	for i := 1; i < len(got); i++ {
		prev, cur := got[i-1], got[i]
		if cur.File < prev.File || (cur.File == prev.File && cur.Line < prev.Line) {
			t.Errorf("violations out of order at %d: %s:%d after %s:%d",
				i, cur.File, cur.Line, prev.File, prev.Line)
		}
	}
	if got[0].File != "internal/alpha/x_test.go" {
		t.Errorf("first violation is from %q; alpha sorts before zeta", got[0].File)
	}
}

// TestScanPackageForProseAssertions_UnreadableDir confirms a directory the scan
// cannot read is an error rather than a silent empty pass.
func TestScanPackageForProseAssertions_UnreadableDir(t *testing.T) {
	t.Parallel()

	if _, err := scanPackageForProseAssertions(t.TempDir(), "no/such/package"); err == nil {
		t.Error("want an error for an unreadable package directory, got nil")
	}
}

// TestDescribeNeedle_TruncatesLongPhrase keeps a violation's Detail readable
// when the asserted phrase is a whole sentence.
func TestDescribeNeedle_TruncatesLongPhrase(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("phrase ", 12)
	fset, files, paths := parseSyntheticPackage(t, map[string]string{
		"a_test.go": fixtureHeader + `
func TestX(t *testing.T) {
	body := readSkill(t, ritualPath)
	if !strings.Contains(body, "` + long + `") {
		t.Error("missing")
	}
}
`,
	})
	got := detectProseAssertions(fset, files, paths)
	if len(got) != 1 {
		t.Fatalf("want 1 violation, got %d", len(got))
	}
	if !strings.Contains(got[0].Detail, "…") {
		t.Errorf("a long phrase must be elided in Detail; got %q", got[0].Detail)
	}
	if strings.Contains(got[0].Detail, long) {
		t.Error("Detail repeats the whole phrase instead of eliding it")
	}
}

// TestPolicyShippedProseAssertion_TestdataIsNotScanned pins that fixture trees
// under testdata are excluded: they hold deliberately malformed sources, and a
// scan that walked into them would report a parse error for every one.
func TestPolicyShippedProseAssertion_TestdataIsNotScanned(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	td := filepath.Join(root, "internal", "demo", "testdata")
	if err := os.MkdirAll(td, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "golden_test.go"), []byte("package !!!"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := PolicyShippedProseAssertion(root)
	if err != nil {
		t.Fatalf("testdata must be skipped, not parsed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want no violations, got %+v", got)
	}
}

// TestShippedProseAssertionAllowlist_IsClosedAndLive keeps the exemption set
// honest from both directions. Every entry must name a test that still exists,
// so the list cannot accumulate dead exemptions that quietly widen it; and
// every rationale must state the dispatch claim, since that is the only ground
// D-0070 admits.
func TestShippedProseAssertionAllowlist_IsClosedAndLive(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	defined := map[string]string{}
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
		data, rerr := os.ReadFile(path) //nolint:gosec // walking a repo-relative tree under test
		if rerr != nil {
			return rerr
		}
		rel, _ := filepath.Rel(root, path)
		for _, m := range regexp.MustCompile(`(?m)^func (Test\w+)\(`).FindAllStringSubmatch(string(data), -1) {
			defined[m[1]] = filepath.ToSlash(rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking for test functions: %v", err)
	}
	if len(defined) == 0 {
		t.Fatal("found no test functions; the scan is vacuous")
	}

	classes := []struct {
		name    string
		entries map[string]string
		grounds []string
	}{
		{"triggerPhraseExemptions", triggerPhraseExemptions, []string{"trigger", "dispatch"}},
		{"derivedExpectationExemptions", derivedExpectationExemptions, []string{"derive"}},
	}
	for _, c := range classes {
		if len(c.entries) == 0 {
			t.Errorf("%s is empty; if the class has no members, delete the class rather than leaving an open door", c.name)
		}
		for name, why := range c.entries {
			if _, ok := defined[name]; !ok {
				t.Errorf("%s names %s, which no longer exists — drop the entry rather than leaving the exemption standing", c.name, name)
			}
			low := strings.ToLower(why)
			stated := false
			for _, g := range c.grounds {
				if strings.Contains(low, g) {
					stated = true
				}
			}
			if !stated {
				t.Errorf("%s entry for %s must state the %v ground that earns the exemption; got %q", c.name, name, c.grounds, why)
			}
			if _, dup := triggerPhraseExemptions[name]; dup && c.name != "triggerPhraseExemptions" {
				t.Errorf("%s is exempt under two classes; one test has one ground", name)
			}
		}
	}
}

// TestPolicy_ShippedProseAssertion is the CI gate: no test in the tree asserts
// that shipped prose carries a phrase, outside the closed allowlist.
func TestPolicy_ShippedProseAssertion(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyShippedProseAssertion)
}
