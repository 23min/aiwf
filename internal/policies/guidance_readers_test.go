package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readerFixtureHeader is a synthetic package reaching guidance the ways a
// test can: a helper that reads it from the root, a helper that reads a
// fixture repository's copy, a policy passed by value, and a package
// constant naming it. A blank package value names it too, and lends it to
// no one.
const readerFixtureHeader = `package pkg

const claudePath = "CLAUDE.md"

var _ = "CLAUDE.md"

func readClaude(t *testing.T) string {
	data, _ := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	return string(data)
}

func readFixture(t *testing.T) string {
	data, _ := os.ReadFile(filepath.Join(t.TempDir(), "CLAUDE.md"))
	return string(data)
}

func runPolicy(t *testing.T, fn func(string) error) { _ = fn(repoRoot(t)) }

func PolicyReadsClaude(root string) error {
	_, err := os.ReadFile(filepath.Join(root, claudePath))
	return err
}

func pingPong(n int) int { return pongPing(n) }

func pongPing(n int) int { return pingPong(n) }

type doc struct{}

func (doc) load(t *testing.T) string { return readClaude(t) }

func audit(root string, paths []string) {}

type quiet struct{}

func (quiet) fetch(t *testing.T) string { return "" }

type other struct{}

func (other) fetch(t *testing.T) string { return readClaude(t) }
`

// TestReadersInPackage is M-0333 AC-6's rule for what reads development
// guidance: a test that, itself or through any function or method of its
// package it calls or passes along, names a guidance document and resolves
// the repository root.
func TestReadersInPackage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, body string
		reader     bool
	}{
		{"a direct read from the root", `data, _ := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md")); _ = data`, true},
		{"a read through a helper", `_ = readClaude(t)`, true},
		{"a read through a method", `_ = doc{}.load(t)`, true},
		{"a policy passed by value", `runPolicy(t, PolicyReadsClaude)`, true},
		{"a nested CLAUDE.md from the root", `data, _ := os.ReadFile(filepath.Join(repoRoot(t), "internal/pkg/CLAUDE.md")); _ = data`, true},
		{"AGENTS.md from the root", `data, _ := os.ReadFile(filepath.Join(repoRoot(t), "AGENTS.md")); _ = data`, true},
		{"a routed document from the root", `data, _ := os.ReadFile(filepath.Join(repoRoot(t), "docs/dev/testing.md")); _ = data`, true},
		{"a relative path climbing to the root", `data, _ := os.ReadFile("../../CLAUDE.md"); _ = data`, true},
		{"a fixture repository's copy", `_ = readFixture(t)`, false},
		{"a literal naming the file anywhere counts, failing closed", `for _, p := range []string{"CLAUDE.md"} { _ = p }; _ = repoRoot(t)`, true},
		{"a path list passed along after resolving the root", `root := repoRoot(t); audit(root, []string{"CLAUDE.md", "ROADMAP.md"})`, true},
		{"the second of two same-named methods reads", `_ = other{}.fetch(t)`, true},
		{"a cycle of helpers settles", `_ = pingPong(1)`, false},
		{"a document outside the guidance set", `data, _ := os.ReadFile(filepath.Join(repoRoot(t), "docs/other.md")); _ = data`, false},
		{"a blank package value lends its path to no test", `_ = repoRoot(t)`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, files, paths := parseSyntheticPackage(t, map[string]string{
				"header.go": readerFixtureHeader,
				"a_test.go": "package pkg\n\nfunc TestSubject(t *testing.T) {\n\t" + tt.body + "\n}\n",
			})
			got := readersInPackage(files, paths, fixtureNamesGuidance)
			if _, isReader := got["TestSubject"]; isReader != tt.reader {
				t.Errorf("reader = %v, want %v", isReader, tt.reader)
			}
		})
	}
}

// TestReadersInPackage_OnlyTestsAreReported pins what counts as a test: a
// top-level Test function in a _test.go file other than TestMain. A method
// named like a test, TestMain, and a Test function outside a test file read
// the guidance here and are not reported.
func TestReadersInPackage_OnlyTestsAreReported(t *testing.T) {
	t.Parallel()
	const reads = "{ _ = readClaude(t) }\n"
	_, files, paths := parseSyntheticPackage(t, map[string]string{
		"header.go": readerFixtureHeader + "\nfunc TestInSource(t *testing.T) " + reads,
		"a_test.go": "package pkg\n\nfunc TestSubject(t *testing.T) " + reads +
			"\nfunc (doc) TestMethod(t *testing.T) " + reads +
			"\nfunc TestMain(m *testing.M) { var t *testing.T; _ = readClaude(t) }\n",
	})
	got := readersInPackage(files, paths, fixtureNamesGuidance)
	if len(got) != 1 || got["TestSubject"] == "" {
		t.Errorf("readers = %v, want only TestSubject", got)
	}
}

// fixtureNamesGuidance is the guidance-path predicate the fixtures use: the
// entry points at any depth, the router, and one routed document.
func fixtureNamesGuidance(s string) bool {
	return namesGuidancePath(s, map[string]bool{"docs/dev/testing.md": true})
}

// TestGuidanceReaderViolations pins the list's two directions: a reader
// the list does not name is reported on its file, and an entry naming no
// reader is reported on the list.
func TestGuidanceReaderViolations(t *testing.T) {
	t.Parallel()
	readers := map[string]string{"pkg.TestListed": "pkg/a_test.go", "pkg.TestNew": "pkg/b_test.go"}
	list := map[string]guidanceReaderEntry{
		"pkg.TestListed":  {readerRelationship, "fixture"},
		"pkg.TestRetired": {readerPin, "fixture"},
	}
	vs := guidanceReaderViolations(readers, list)
	if got, want := violationFiles(vs), []string{"pkg/b_test.go", guidanceReaderListFile}; !equalStrings(got, want) {
		t.Errorf("violation files = %v, want %v", got, want)
	}
	// The unlisted-reader finding cites the decision in force.
	if len(vs) > 0 && !strings.Contains(vs[0].Detail, "D-0102") {
		t.Errorf("the finding must cite D-0102; got %q", vs[0].Detail)
	}
	for _, v := range vs {
		if v.Policy != "guidance-readers" {
			t.Errorf("Policy = %q, want guidance-readers", v.Policy)
		}
	}
}

// TestGuidanceReaderTests_Errors pins that an unreadable or unparseable
// package is an error, not a silent pass.
func TestGuidanceReaderTests_Errors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile := repoFileWriter(t, root)
	writeFile("pkg/a_test.go", "package pkg\n\nfunc TestX(t *testing.T) {\n")
	if _, err := PolicyGuidanceReaders(root); err == nil {
		t.Error("an unparseable test file: want an error")
	}
	if os.Geteuid() != 0 {
		denied := filepath.Join(root, "pkg")
		if err := os.Chmod(denied, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })
		if _, err := guidanceReaderTests(root); err == nil {
			t.Error("an unreadable package: want an error")
		}
	}
}

// TestPolicy_GuidanceReaders holds the list equal to the tree's readers.
func TestPolicy_GuidanceReaders(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyGuidanceReaders)
}

// TestRepoNamesGuidance pins that a document the router links to is
// guidance, alongside the entry points at any depth and the router itself,
// however the literal spells its path; a router below the root is not the
// router.
func TestRepoNamesGuidance(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoFileWriter(t, root)(".guidance/project.md", "Read [t](../docs/dev/testing.md).\n")
	names := repoNamesGuidance(root)
	for p, want := range map[string]bool{
		"docs/dev/testing.md": true, "CLAUDE.md": true, "sub/AGENTS.md": true,
		".guidance/project.md": true, "docs/other.md": false,
		"../../docs/dev/testing.md": true, "./docs/dev/testing.md": true,
		"sub/.guidance/project.md": false,
	} {
		if got := names(p); got != want {
			t.Errorf("names(%q) = %v, want %v", p, got, want)
		}
	}
}
