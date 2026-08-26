package policies_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

// go_workflow_embed_coverage_test.go — G-0637 chokepoint pin.
//
// Content embedded with `go:embed` is compiled into the binary, so editing
// one of those files changes what ships even though no Go file changed. The
// policy suite also parses some of those bytes, so a prose edit can turn
// tests red. `.github/workflows/go.yml` — build, vet, race, lint and the
// coverage gates — is path-filtered, and a filter that does not name an
// embedded tree lets a change to it reach `main` and a release tag with no
// Go workflow having judged it.
//
// This is a relationship check rather than a list to maintain: the expected
// set is derived by reading the `go:embed` directives, so a newly embedded
// tree fails here until the workflow filter names it too. Every commit that
// adds a directive edits a `.go` file, which `**/*.go` already selects, so
// the check is guaranteed to run on the commit that needs it.

// embedRoot is one embedded tree: the repo-relative path a `go:embed`
// directive draws from, whether that path is a directory, and the directive
// site that asked for it. The directory case needs a subtree pattern to be
// covered; a single embedded file is covered by naming it.
type embedRoot struct {
	Path  string // repo-relative, slash-separated
	IsDir bool
	Site  string // file:line of the directive
}

// embedScan is what one walk found: the roots drawn from, and any directive
// whose pattern resolved to nothing on disk. An unresolved directive is
// reported rather than skipped — a root dropped from the expected set is
// never checked for coverage, so silence there reads as "covered".
type embedScan struct {
	Roots      []embedRoot
	Unresolved []string
}

// globMeta are the pattern metacharacters `go:embed` honours. A first path
// segment containing one cannot be joined onto the package directory and
// stat'ed, because it names a set rather than a path.
const globMeta = "*?["

// scanEmbedRoots walks tops (relative to repoRoot) for non-test `go:embed`
// directives and returns the path each draws from.
//
// The first path segment decides the root, because that whole directory is
// build-relevant: `tpl/*.tmpl` can gain a file the next commit embeds. When
// the first segment is itself a glob — `*.tmpl` at a package root — the
// package directory is what is drawn from.
func scanEmbedRoots(repoRoot string, tops []string) (embedScan, error) {
	seen := map[string]embedRoot{}
	var unresolved []string
	for _, top := range tops {
		root := filepath.Join(repoRoot, top)
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if perr != nil {
				return fmt.Errorf("parsing %s: %w", path, perr)
			}
			pkgDir := filepath.Dir(path)
			for _, group := range f.Comments {
				for _, c := range group.List {
					patterns, ok := strings.CutPrefix(c.Text, "//go:embed ")
					if !ok {
						continue
					}
					pos := fset.Position(c.Pos())
					siteRel, _ := filepath.Rel(repoRoot, pos.Filename)
					site := filepath.ToSlash(siteRel) + ":" + strconv.Itoa(pos.Line)
					for _, pat := range strings.Fields(patterns) {
						// `all:` opts directories whose names begin with
						// `_` or `.` into the embed; it does not change
						// which directory is drawn from.
						pat = strings.TrimPrefix(pat, "all:")
						seg, _, _ := strings.Cut(pat, "/")
						if seg == "" {
							continue
						}
						abs := filepath.Join(pkgDir, seg)
						if strings.ContainsAny(seg, globMeta) {
							abs = pkgDir
						}
						info, serr := os.Stat(abs)
						if serr != nil {
							unresolved = append(unresolved, site+" ("+pat+")")
							continue
						}
						// A `.go` root is already selected by `**/*.go`,
						// so the filter needs no entry for it.
						if !info.IsDir() && strings.HasSuffix(abs, ".go") {
							continue
						}
						rel, rerr := filepath.Rel(repoRoot, abs)
						if rerr != nil {
							unresolved = append(unresolved, site+" ("+pat+")")
							continue
						}
						relPath := filepath.ToSlash(rel)
						if _, dup := seen[relPath]; !dup {
							seen[relPath] = embedRoot{Path: relPath, IsDir: info.IsDir(), Site: site}
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return embedScan{}, err
		}
	}
	out := make([]embedRoot, 0, len(seen))
	for _, r := range seen {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	sort.Strings(unresolved)
	return embedScan{Roots: out, Unresolved: unresolved}, nil
}

// mapValue returns the value node for key in a YAML mapping node. It compares
// the key's raw scalar text, so `on:` is found under YAML 1.1's boolean
// resolution of that word as well as under 1.2's string one.
func mapValue(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

// workflowPaths returns the `paths:` filter for one trigger of go.yml.
//
// Two entry shapes are refused rather than read, both because coveredBy would
// otherwise answer over a pattern set that is not what the workflow applies:
//
// A `!`-negated entry excludes paths a preceding positive selected. coveredBy
// reads patterns independently, so a broad positive plus a narrow negative
// leaves a tree unwatched while the positive still reports it covered.
//
// An entry that is not a plain string carries its text somewhere other than
// the node's value — an unquoted `!foo` is a YAML *tag*, leaving Value empty —
// so reading Value alone yields "" and silently admits an entry nobody meant
// as a pattern. Refusing keeps both states loud instead of silently green.
func workflowPaths(doc *yaml.Node, trigger string) ([]string, error) {
	on := mapValue(doc, "on")
	if on == nil {
		return nil, fmt.Errorf("no `on:` block")
	}
	trig := mapValue(on, trigger)
	if trig == nil {
		return nil, fmt.Errorf("no `on.%s` trigger", trigger)
	}
	paths := mapValue(trig, "paths")
	if paths == nil || paths.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("`on.%s` has no `paths:` sequence", trigger)
	}
	out := make([]string, 0, len(paths.Content))
	for i, p := range paths.Content {
		if p.Kind != yaml.ScalarNode || p.Tag != "!!str" {
			return nil, fmt.Errorf("`on.%s.paths` entry %d is not a plain string "+
				"(kind %v, tag %q, value %q); this check reads patterns as strings, so an "+
				"entry carrying its text elsewhere would be read as empty",
				trigger, i, p.Kind, p.Tag, p.Value)
		}
		if strings.HasPrefix(p.Value, "!") {
			return nil, fmt.Errorf("`on.%s.paths` carries the negated entry %q; this check "+
				"does not interpret negation and would report a tree covered that the "+
				"negation excludes — decide the coverage question deliberately", trigger, p.Value)
		}
		out = append(out, p.Value)
	}
	return out, nil
}

// coveredBy reports whether some filter pattern selects every file the embed
// draws from. A path filter matches file paths, so a directory is covered only
// by a `<prefix>**` subtree pattern — naming the bare directory selects a file
// literally at that path and nothing beneath it. A single embedded file is
// covered by either shape. Patterns outside those two forms are not
// interpreted: an unrecognised one leaves the tree uncovered and fails the
// test, which is the safe direction for a check whose subject is a gap in
// coverage. Negation is refused upstream, in workflowPaths, because it is the
// one shape that would invert this reading rather than merely narrow it.
func coveredBy(path string, isDir bool, patterns []string) bool {
	for _, p := range patterns {
		if prefix, ok := strings.CutSuffix(p, "**"); ok && strings.HasPrefix(path+"/", prefix) {
			return true
		}
		if !isDir && p == path {
			return true
		}
	}
	return false
}

// TestWorkflowPaths pins which filter entries are read and which are refused.
// Refusal is the load-bearing half: an entry this check misreads produces a
// pattern set the workflow does not apply, and coveredBy then answers a
// question nobody asked.
func TestWorkflowPaths(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		yaml    string
		want    []string
		wantErr string // substring; empty means the call must succeed
	}{
		{
			name: "plain string entries are read",
			yaml: "on:\n  push:\n    paths:\n      - \"**/*.go\"\n      - internal/skills/**\n",
			want: []string{"**/*.go", "internal/skills/**"},
		},
		{
			// The text of an unquoted `!foo` lives in the node's tag, not
			// its value, so a check reading the value alone sees "" and
			// admits an entry that changes what the workflow selects.
			name:    "an unquoted negation is a YAML tag and is refused",
			yaml:    "on:\n  push:\n    paths:\n      - internal/**\n      - !internal/skills/**\n",
			wantErr: "not a plain string",
		},
		{
			name:    "a quoted negation is refused",
			yaml:    "on:\n  push:\n    paths:\n      - \"internal/**\"\n      - \"!internal/skills/**\"\n",
			wantErr: "negated entry",
		},
		{
			name:    "a non-string entry is refused",
			yaml:    "on:\n  push:\n    paths:\n      - internal/**\n      - ~\n",
			wantErr: "not a plain string",
		},
		{
			name:    "a nested sequence entry is refused",
			yaml:    "on:\n  push:\n    paths:\n      - [internal/**]\n",
			wantErr: "not a plain string",
		},
		{
			name:    "a trigger with no paths filter is refused",
			yaml:    "on:\n  push:\n    branches: [main]\n",
			wantErr: "has no `paths:` sequence",
		},
		{
			name:    "a missing trigger is refused",
			yaml:    "on:\n  pull_request:\n    paths:\n      - \"**/*.go\"\n",
			wantErr: "no `on.push` trigger",
		},
		{
			name:    "a document with no on block is refused",
			yaml:    "name: go\njobs: {}\n",
			wantErr: "no `on:` block",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var root yaml.Node
			if err := yaml.Unmarshal([]byte(tc.yaml), &root); err != nil {
				t.Fatalf("fixture does not parse: %v", err)
			}
			got, err := workflowPaths(root.Content[0], "push")
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("workflowPaths = %v, want error containing %q", got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("patterns (-want +got):\n%s", diff)
			}
		})
	}
}

// TestCoveredBy pins the matcher's own decision, which the workflow-level test
// below cannot: that test reports what the current filter happens to satisfy,
// so a matcher too generous in either direction still passes it.
func TestCoveredBy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		path     string
		isDir    bool
		patterns []string
		want     bool
	}{
		{
			// A path filter selects files. Naming the bare directory
			// selects a file literally at that path, so the tree beneath
			// it stays unwatched — the shape the failure message must
			// never invite.
			name: "bare directory does not cover the tree beneath it",
			path: "internal/x", isDir: true,
			patterns: []string{"internal/x"},
			want:     false,
		},
		{
			name: "subtree pattern covers the directory",
			path: "internal/x", isDir: true,
			patterns: []string{"internal/x/**"},
			want:     true,
		},
		{
			name: "an ancestor subtree pattern covers it too",
			path: "internal/x", isDir: true,
			patterns: []string{"internal/**"},
			want:     true,
		},
		{
			name: "a sibling sharing a name prefix does not cover it",
			path: "internal/xy", isDir: true,
			patterns: []string{"internal/x/**"},
			want:     false,
		},
		{
			// `*` does not cross a separator, so it selects the directory's
			// own entries and not the tree under them.
			name: "a single-star pattern does not cover the subtree",
			path: "internal/x", isDir: true,
			patterns: []string{"internal/x/*"},
			want:     false,
		},
		{
			name: "a Go-source pattern covers no embedded tree",
			path: "internal/x", isDir: true,
			patterns: []string{"**/*.go", "go.mod", "Makefile"},
			want:     false,
		},
		{
			name: "a single embedded file is covered by naming it",
			path: "internal/x/version.txt", isDir: false,
			patterns: []string{"internal/x/version.txt"},
			want:     true,
		},
		{
			name: "a single embedded file is covered by an enclosing subtree",
			path: "internal/x/version.txt", isDir: false,
			patterns: []string{"internal/x/**"},
			want:     true,
		},
		{
			name: "naming a directory's path does not cover it even among other patterns",
			path: "internal/x", isDir: true,
			patterns: []string{"go.mod", "internal/x", "Makefile"},
			want:     false,
		},
		{
			name: "no patterns covers nothing",
			path: "internal/x", isDir: true,
			patterns: nil,
			want:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := coveredBy(tc.path, tc.isDir, tc.patterns); got != tc.want {
				t.Errorf("coveredBy(%q, isDir=%v, %v) = %v, want %v",
					tc.path, tc.isDir, tc.patterns, got, tc.want)
			}
		})
	}
}

// writeEmbedFixture materializes one synthetic package: a parseable Go file
// carrying the directive, plus whatever the directive should find.
func writeEmbedFixture(t *testing.T, root, pkg, goFile, directive string, files ...string) {
	t.Helper()
	dir := filepath.Join(root, "internal", pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	src := "package " + pkg + "\n\n" + directive + "\nvar embedded string\n"
	if err := os.WriteFile(filepath.Join(dir, goFile), []byte(src), 0o644); err != nil {
		t.Fatalf("writing %s: %v", goFile, err)
	}
	for _, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", f, err)
		}
		// A `.go` payload has to parse: the walk reads every Go file it
		// meets, so an unparseable one would fail the scan rather than
		// exercise the case the fixture is for.
		body := "x\n"
		if strings.HasSuffix(f, ".go") {
			body = "package " + pkg + "\n"
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", f, err)
		}
	}
}

// TestScanEmbedRoots pins which path each directive shape draws from. The
// package-root glob is the load-bearing case: its first segment names a set
// rather than a path, so a derivation that joins it onto the package
// directory resolves nothing — and a root missing from the scan is a root
// never checked for coverage, which reads as covered.
func TestScanEmbedRoots(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	writeEmbedFixture(t, root, "adir", "p.go", "//go:embed assets", "assets/a.txt")
	writeEmbedFixture(t, root, "bglob", "p.go", "//go:embed tpl/*.tmpl", "tpl/a.tmpl")
	writeEmbedFixture(t, root, "crootglob", "p.go", "//go:embed *.tmpl", "a.tmpl")
	writeEmbedFixture(t, root, "dall", "p.go", "//go:embed all:hidden", "hidden/.keep")
	writeEmbedFixture(t, root, "efile", "p.go", "//go:embed one.txt", "one.txt")
	writeEmbedFixture(t, root, "fmissing", "p.go", "//go:embed nothing-here")
	writeEmbedFixture(t, root, "gtest", "p_test.go", "//go:embed skipped", "skipped/a.txt")
	// A `.go` root needs no filter entry — `**/*.go` already selects it.
	writeEmbedFixture(t, root, "hgo", "p.go", "//go:embed helper.go", "helper.go")
	// The first segment is the root, not the pattern's whole directory:
	// `a/` is what can gain a file the next commit embeds.
	writeEmbedFixture(t, root, "ideep", "p.go", "//go:embed a/b/c.txt", "a/b/c.txt")

	got, err := scanEmbedRoots(root, []string{"internal", "cmd"})
	if err != nil {
		t.Fatalf("scanEmbedRoots: %v", err)
	}

	type rootShape struct {
		Path  string
		IsDir bool
	}
	want := []rootShape{
		{"internal/adir/assets", true},
		{"internal/bglob/tpl", true},
		{"internal/crootglob", true},
		{"internal/dall/hidden", true},
		{"internal/efile/one.txt", false},
		{"internal/ideep/a", true},
	}
	gotShapes := make([]rootShape, 0, len(got.Roots))
	for _, r := range got.Roots {
		gotShapes = append(gotShapes, rootShape{r.Path, r.IsDir})
		if !strings.Contains(r.Site, ".go:") {
			t.Errorf("root %q has no directive site, got %q", r.Path, r.Site)
		}
	}
	if diff := cmp.Diff(want, gotShapes); diff != "" {
		t.Errorf("derived roots (-want +got):\n%s", diff)
	}

	if len(got.Unresolved) != 1 || !strings.Contains(got.Unresolved[0], "fmissing") {
		t.Errorf("unresolved = %v, want exactly the fmissing directive", got.Unresolved)
	}
}

func TestGoWorkflowCoversEveryEmbeddedTree(t *testing.T) {
	t.Parallel()
	repoRoot := repoRootForHook(t)

	raw, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "go.yml"))
	if err != nil {
		t.Fatalf("reading go.yml: %v", err)
	}
	var root yaml.Node
	if uerr := yaml.Unmarshal(raw, &root); uerr != nil {
		t.Fatalf("parsing go.yml: %v", uerr)
	}
	if len(root.Content) == 0 {
		t.Fatal("go.yml is empty")
	}
	doc := root.Content[0]

	scan, err := scanEmbedRoots(repoRoot, []string{"internal", "cmd"})
	if err != nil {
		t.Fatalf("scanning for go:embed directives: %v", err)
	}
	if len(scan.Roots) == 0 {
		t.Fatal("no go:embed directives found — the derivation is broken, not the workflow")
	}
	for _, u := range scan.Unresolved {
		t.Errorf("go:embed directive at %s names nothing on disk; its tree cannot be "+
			"checked for workflow coverage", u)
	}

	// Both triggers matter: push is what gates main and therefore a release
	// tag; pull_request is what gates a contributor's branch.
	for _, trigger := range []string{"push", "pull_request"} {
		patterns, perr := workflowPaths(doc, trigger)
		if perr != nil {
			t.Fatalf("go.yml: %v", perr)
		}
		for _, r := range scan.Roots {
			if coveredBy(r.Path, r.IsDir, patterns) {
				continue
			}
			// Name the pattern to add, not the bare path: a path filter
			// selects files, so a bare directory entry would satisfy
			// neither the workflow nor this check.
			want := r.Path
			if r.IsDir {
				want += "/**"
			}
			t.Errorf("go.yml `on.%s.paths` does not cover embedded %q (embedded at %s); "+
				"add %q — a change to it ships in the binary with no Go workflow run",
				trigger, r.Path, r.Site, want)
		}
	}
}
