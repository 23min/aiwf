package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicyBranchCoverageAudit_Env drives the env-fed entry point.
// Serial (t.Setenv panics under t.Parallel) and documented in
// setup_test.go's skip-list.
func TestPolicyBranchCoverageAudit_Env(t *testing.T) {
	// Unset profile → no-op.
	t.Setenv("AIWF_COVERAGE_PROFILE", "")
	vs, err := PolicyBranchCoverageAudit(t.TempDir())
	if err != nil {
		t.Fatalf("unset profile: unexpected error: %v", err)
	}
	if vs != nil {
		t.Fatalf("unset profile: want nil violations, got %+v", vs)
	}

	// Set profile + base → delegates to branchCoverageViolations and
	// surfaces the uncovered changed branch.
	const baseSrc = "package foo\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"
	const headSrc = "package foo\n\nfunc Add(a, b int) int {\n\tif a < 0 {\n\t\treturn 0\n\t}\n\treturn a + b\n}\n"
	profile := "mode: atomic\n" + fixtureModule + "/internal/foo/bar.go:4.12,6.3 1 0\n"
	root, baseSHA, profilePath := covFixture(t, baseSrc, headSrc, profile)

	t.Setenv("AIWF_COVERAGE_PROFILE", profilePath)
	t.Setenv("AIWF_COVERAGE_BASE", baseSHA)
	vs, err = PolicyBranchCoverageAudit(root)
	if err != nil {
		t.Fatalf("set profile: unexpected error: %v", err)
	}
	if len(vs) != 1 || vs[0].Line != 4 {
		t.Fatalf("set profile: want one violation at line 4, got %+v", vs)
	}
}

func TestBranchCoverageViolations_Errors(t *testing.T) {
	t.Parallel()

	t.Run("modulePath error when go.mod absent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		runGit := repoGitRunner(t, root)
		writeFile := repoFileWriter(t, root)
		runGit("init")
		runGit("config", "user.email", "test@example.com")
		runGit("config", "user.name", "aiwf-test")
		writeFile("x.go", "package x\n")
		runGit("add", "-A")
		runGit("commit", "-m", "base")
		base := trimLine(runGit("rev-parse", "HEAD"))
		writeFile("x.go", "package x\n\n// changed\n")
		runGit("add", "-A")
		runGit("commit", "-m", "head")

		_, err := branchCoverageViolations(root, filepath.Join(root, "coverage.out"), base)
		if err == nil {
			t.Fatal("want error for missing go.mod, got nil")
		}
	})

	t.Run("changedLines error on bad base ref", func(t *testing.T) {
		t.Parallel()
		root, _, profilePath := covFixture(t,
			"package foo\n\nfunc Add() int { return 1 }\n",
			"package foo\n\nfunc Add() int { return 2 }\n",
			"mode: atomic\n")
		_, err := branchCoverageViolations(root, profilePath, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
		if err == nil {
			t.Fatal("want error for nonexistent base ref, got nil")
		}
	})

	t.Run("removed working-tree file is out of scope", func(t *testing.T) {
		t.Parallel()
		const headSrc = "package foo\n\nfunc Add(a, b int) int {\n\tif a < 0 {\n\t\treturn 0\n\t}\n\treturn a + b\n}\n"
		profile := "mode: atomic\n" + fixtureModule + "/internal/foo/bar.go:4.12,6.3 1 0\n"
		root, baseSHA, profilePath := covFixture(t,
			"package foo\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n",
			headSrc, profile)
		// Deleting the file makes it a deletion against the working tree,
		// and a file that is gone has no statements left to audit. The
		// profile still names it, so this pins that scope — not the
		// stale profile — decides.
		if rmErr := os.Remove(filepath.Join(root, "internal", "foo", "bar.go")); rmErr != nil {
			t.Fatalf("remove: %v", rmErr)
		}
		vs, err := branchCoverageViolations(root, profilePath, baseSHA)
		if err != nil {
			t.Fatalf("a deleted file must drop out of scope, not error: %v", err)
		}
		if len(vs) != 0 {
			t.Errorf("want no violations for a deleted file, got %+v", vs)
		}
	})

	t.Run("coverageIgnoreLines error when a changed file is unreadable", func(t *testing.T) {
		t.Parallel()
		const headSrc = "package foo\n\nfunc Add(a, b int) int {\n\tif a < 0 {\n\t\treturn 0\n\t}\n\treturn a + b\n}\n"
		profile := "mode: atomic\n" + fixtureModule + "/internal/foo/bar.go:4.12,6.3 1 0\n"
		root, baseSHA, profilePath := covFixture(t,
			"package foo\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n",
			headSrc, profile)
		// Replace the file with a dangling symlink: git still reports it
		// changed, and the read fails for every uid, so the branch is
		// reachable without depending on file modes.
		bar := filepath.Join(root, "internal", "foo", "bar.go")
		if rmErr := os.Remove(bar); rmErr != nil {
			t.Fatalf("remove: %v", rmErr)
		}
		if lnErr := os.Symlink(filepath.Join(root, "no-such-target"), bar); lnErr != nil {
			t.Fatalf("symlink: %v", lnErr)
		}
		_, err := branchCoverageViolations(root, profilePath, baseSHA)
		if err == nil {
			t.Fatal("want error for an unreadable source file, got nil")
		}
	})
}

func TestModulePath(t *testing.T) {
	t.Parallel()

	t.Run("happy", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/m\n\ngo 1.24\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := modulePath(root)
		if err != nil {
			t.Fatal(err)
		}
		if got != "example.com/m" {
			t.Errorf("module = %q, want example.com/m", got)
		}
	})

	t.Run("missing go.mod", func(t *testing.T) {
		t.Parallel()
		if _, err := modulePath(t.TempDir()); err == nil {
			t.Fatal("want error for missing go.mod, got nil")
		}
	})

	t.Run("no module directive", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("go 1.24\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := modulePath(root); err == nil {
			t.Fatal("want error for go.mod without module directive, got nil")
		}
	})
}

func TestParseCoverProfile(t *testing.T) {
	t.Parallel()

	t.Run("groups by relpath, skips noise", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		profile := "mode: atomic\n" +
			"\n" + // blank line skipped
			"this is not a coverage line\n" + // malformed → skipped
			"other.com/zzz/a.go:1.1,2.2 1 1\n" + // wrong module prefix → skipped
			"example.com/m/internal/foo/bar.go:3.2,5.3 2 0\n" +
			"example.com/m/internal/foo/bar.go:7.2,7.10 1 4\n"
		p := filepath.Join(root, "cov.out")
		if err := os.WriteFile(p, []byte(profile), 0o644); err != nil {
			t.Fatal(err)
		}
		blocks, err := parseCoverProfile(p, "example.com/m")
		if err != nil {
			t.Fatal(err)
		}
		got := blocks["internal/foo/bar.go"]
		if len(got) != 2 {
			t.Fatalf("want 2 blocks, got %d (%+v)", len(got), blocks)
		}
		if got[0] != (coverBlock{StartLine: 3, EndLine: 5, Count: 0}) {
			t.Errorf("block[0] = %+v", got[0])
		}
		if _, ok := blocks["a.go"]; ok {
			t.Error("wrong-prefix path leaked into results")
		}
	})

	t.Run("open error", func(t *testing.T) {
		t.Parallel()
		if _, err := parseCoverProfile(filepath.Join(t.TempDir(), "nope.out"), "x"); err == nil {
			t.Fatal("want error for missing profile, got nil")
		}
	})

	// A multi-binary run (`go test -coverpkg=./pkgs ./multi/...`)
	// concatenates one profile per binary, so the same block appears
	// many times — count 0 from binaries that never ran it, count >0
	// from the one that did. The merge must sum these so the block reads
	// as covered, not as a count-0 occurrence.
	t.Run("merges duplicate blocks by span (summing counts)", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		profile := "mode: atomic\n" +
			"example.com/m/internal/foo/bar.go:3.2,5.3 2 0\n" + // binary A: not run
			"example.com/m/internal/foo/bar.go:3.2,5.3 2 4\n" + // binary B: covered
			"example.com/m/internal/foo/bar.go:3.2,5.3 2 0\n" + // binary C: not run
			"example.com/m/internal/foo/bar.go:7.2,7.9 1 0\n" // genuinely uncovered
		p := filepath.Join(root, "cov.out")
		if err := os.WriteFile(p, []byte(profile), 0o644); err != nil {
			t.Fatal(err)
		}
		blocks, err := parseCoverProfile(p, "example.com/m")
		if err != nil {
			t.Fatal(err)
		}
		got := blocks["internal/foo/bar.go"]
		if len(got) != 2 {
			t.Fatalf("want 2 merged blocks, got %d (%+v)", len(got), got)
		}
		// First span merged to a positive count; second stays 0.
		if got[0].StartLine != 3 || got[0].Count == 0 {
			t.Errorf("block[0] = %+v, want StartLine 3 with positive merged count", got[0])
		}
		if got[1].StartLine != 7 || got[1].Count != 0 {
			t.Errorf("block[1] = %+v, want StartLine 7 with count 0", got[1])
		}
	})
}

func TestChangedLines(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runGit := repoGitRunner(t, root)
	writeFile := repoFileWriter(t, root)
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "aiwf-test")
	// keep.go will be modified with a pure-removal hunk; gone.go will be
	// deleted (exercises the /dev/null + empty-curFile path).
	writeFile("keep.go", "package k\n\nfunc A() {}\nfunc B() {}\nfunc C() {}\n")
	writeFile("gone.go", "package k\n\nfunc Z() {}\n")
	runGit("add", "-A")
	runGit("commit", "-m", "base")
	base := trimLine(runGit("rev-parse", "HEAD"))

	// Remove B() (a pure deletion within keep.go → +N,0 hunk) and add a
	// new line at the end (a real addition).
	writeFile("keep.go", "package k\n\nfunc A() {}\nfunc C() {}\nfunc D() {}\n")
	if err := os.Remove(filepath.Join(root, "gone.go")); err != nil {
		t.Fatal(err)
	}
	runGit("add", "-A")
	runGit("commit", "-m", "head")

	changed, err := changedLines(root, base)
	if err != nil {
		t.Fatalf("changedLines: %v", err)
	}
	if _, ok := changed["gone.go"]; ok {
		t.Error("deleted file should contribute no added lines")
	}
	// keep.go must have at least one added/modified line recorded.
	if len(changed["keep.go"]) == 0 {
		t.Errorf("keep.go: expected changed lines, got none (%+v)", changed)
	}
}

// newGoFixtureRepo returns an initialized repo with one committed Go
// file, plus the committed SHA and the closures the caller needs to keep
// mutating it.
func newGoFixtureRepo(t *testing.T, seed map[string]string) (root, base string, runGit func(...string) string, writeFile func(string, string)) {
	t.Helper()
	root = t.TempDir()
	runGit = repoGitRunner(t, root)
	writeFile = repoFileWriter(t, root)
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "aiwf-test")
	for rel, content := range seed {
		writeFile(rel, content)
	}
	runGit("add", "-A")
	runGit("commit", "-m", "base")
	return root, trimLine(runGit("rev-parse", "HEAD")), runGit, writeFile
}

// TestChangedLines_ScopeIsTheWorkingTree pins that the diff target is the
// working tree rather than HEAD, across all three states a change can be
// in relative to HEAD.
//
// Every caller classifies content it reads off disk — the coverage audit
// against a profile produced by running the suite over the working tree,
// the comment scan by parsing the same files. Diffing HEAD lines that
// numbering up against content nobody measured, and reports nothing at
// all for a change that is not committed yet, which is the state the tree
// is in when `make ci` runs.
func TestChangedLines_ScopeIsTheWorkingTree(t *testing.T) {
	t.Parallel()
	root, base, runGit, writeFile := newGoFixtureRepo(t, map[string]string{
		"committed.go": "package k\n\nfunc A() {}\n",
		"staged.go":    "package k\n\nfunc B() {}\n",
		"dirty.go":     "package k\n\nfunc C() {}\n",
	})

	writeFile("committed.go", "package k\n\nfunc A() { _ = 1 }\n")
	runGit("commit", "-am", "committed change")
	writeFile("staged.go", "package k\n\nfunc B() { _ = 2 }\n")
	runGit("add", "staged.go")
	writeFile("dirty.go", "package k\n\nfunc C() { _ = 3 }\n")

	changed, err := changedLines(root, base)
	if err != nil {
		t.Fatalf("changedLines: %v", err)
	}
	for _, rel := range []string{"committed.go", "staged.go", "dirty.go"} {
		if !changed[rel][3] {
			t.Errorf("%s: the edited line 3 must be in scope, got %+v", rel, changed[rel])
		}
	}
}

// TestChangedLines_ExcludesUntrackedFiles pins the scope boundary of the
// shared helper. The coverage audit adds untracked files itself; the two
// comment history-attrition scans that also call this must not get them,
// or an untracked scratch file would fail `make check-fast` — and the
// whole-tree scan documents its subject as every *tracked* Go file.
func TestChangedLines_ExcludesUntrackedFiles(t *testing.T) {
	t.Parallel()
	root, base, _, writeFile := newGoFixtureRepo(t, map[string]string{"keep.go": "package k\n"})
	writeFile("scratch.go", "package k\n\nfunc Scratch() {}\n")

	changed, err := changedLines(root, base)
	if err != nil {
		t.Fatalf("changedLines: %v", err)
	}
	if _, ok := changed["scratch.go"]; ok {
		t.Error("changedLines must stay tracked-only; untracked files belong to the coverage audit alone")
	}
}

// TestAddUntrackedGoLines pins the audit-only widening: a file git diff
// cannot see at any revision still enters the scope, because a file just
// written is where an untested statement is most likely to live. Ignored
// files stay out — they are not part of the build.
func TestAddUntrackedGoLines(t *testing.T) {
	t.Parallel()

	t.Run("untracked files are wholly changed, ignored ones are not", func(t *testing.T) {
		t.Parallel()
		root, _, _, writeFile := newGoFixtureRepo(t, map[string]string{
			"keep.go":    "package k\n",
			".gitignore": "ignored.go\n",
		})
		const fresh = "package k\n\nfunc New() int {\n\treturn 7\n}\n"
		writeFile("fresh.go", fresh)
		writeFile("ignored.go", "package k\n\nfunc Ignored() int {\n\treturn 8\n}\n")

		changed := map[string]map[int]bool{}
		if err := addUntrackedGoLines(root, changed); err != nil {
			t.Fatalf("addUntrackedGoLines: %v", err)
		}
		// A newline-terminated file has exactly Count("\n") lines; one
		// past that is the trailing empty split element, not a line.
		lines := strings.Count(fresh, "\n")
		for ln := 1; ln <= lines; ln++ {
			if !changed["fresh.go"][ln] {
				t.Errorf("fresh.go: line %d of an untracked file must be in scope, got %+v", ln, changed["fresh.go"])
			}
		}
		if changed["fresh.go"][lines+1] {
			t.Errorf("fresh.go: line %d is past EOF and must not be recorded", lines+1)
		}
		if _, ok := changed["ignored.go"]; ok {
			t.Error("a gitignored file must stay out of scope (git ls-files --exclude-standard)")
		}
	})

	t.Run("an unreadable listed file is skipped", func(t *testing.T) {
		t.Parallel()
		root, _, _, writeFile := newGoFixtureRepo(t, map[string]string{"keep.go": "package k\n"})
		writeFile("readable.go", "package k\n\nfunc R() {}\n")
		// A dangling symlink is listed by ls-files and fails every read,
		// regardless of uid.
		if err := os.Symlink(filepath.Join(root, "no-such-target"), filepath.Join(root, "broken.go")); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		changed := map[string]map[int]bool{}
		if err := addUntrackedGoLines(root, changed); err != nil {
			t.Fatalf("an unreadable untracked file must not fail the scan: %v", err)
		}
		if _, ok := changed["broken.go"]; ok {
			t.Error("an unreadable untracked file must be skipped, not recorded")
		}
		if len(changed["readable.go"]) == 0 {
			t.Error("the readable untracked file beside it must still be in scope")
		}
	})

	t.Run("a root git cannot resolve is an error", func(t *testing.T) {
		t.Parallel()
		changed := map[string]map[int]bool{}
		if err := addUntrackedGoLines(t.TempDir(), changed); err == nil {
			t.Error("listing untracked files outside a repository must error, not report none")
		}
	})
}

// TestBranchCoverageViolations_FiresOnAnUntrackedFile pins that the audit
// applies the widening its shared helper deliberately omits.
func TestBranchCoverageViolations_FiresOnAnUntrackedFile(t *testing.T) {
	t.Parallel()
	root, base, _, writeFile := newGoFixtureRepo(t, map[string]string{
		"go.mod":              "module " + fixtureModule + "\n\ngo 1.24\n",
		"internal/foo/bar.go": "package foo\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n",
	})

	// A brand-new, never-staged file carrying an untested guard.
	writeFile("internal/foo/baz.go", "package foo\n\nfunc Sub(a, b int) int {\n\tif a < 0 {\n\t\treturn 0\n\t}\n\treturn a - b\n}\n")

	profilePath := filepath.Join(root, "coverage.out")
	profile := "mode: atomic\n" + fixtureModule + "/internal/foo/baz.go:4.12,6.3 1 0\n"
	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	vs, err := branchCoverageViolations(root, profilePath, base)
	if err != nil {
		t.Fatalf("branchCoverageViolations: %v", err)
	}
	if len(vs) != 1 || vs[0].File != "internal/foo/baz.go" {
		t.Fatalf("want one violation in the untracked file, got %+v", vs)
	}
}

// TestBranchCoverageViolations_FiresOnAnUncommittedChange is the vacuity
// pin for running the gate inside `make ci`: that target runs before the
// ritual's commit gate, so a HEAD-scoped audit would report green on
// precisely the change it was asked to judge.
func TestBranchCoverageViolations_FiresOnAnUncommittedChange(t *testing.T) {
	t.Parallel()
	root, base, _, writeFile := newGoFixtureRepo(t, map[string]string{
		"go.mod":              "module " + fixtureModule + "\n\ngo 1.24\n",
		"internal/foo/bar.go": "package foo\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n",
	})

	// The untested guard lands in the working tree only — never committed.
	writeFile("internal/foo/bar.go", "package foo\n\nfunc Add(a, b int) int {\n\tif a < 0 {\n\t\treturn 0\n\t}\n\treturn a + b\n}\n")

	profilePath := filepath.Join(root, "coverage.out")
	profile := "mode: atomic\n" + fixtureModule + "/internal/foo/bar.go:4.12,6.3 1 0\n"
	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	vs, err := branchCoverageViolations(root, profilePath, base)
	if err != nil {
		t.Fatalf("branchCoverageViolations: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("want exactly one violation for the uncommitted guard, got %d: %+v", len(vs), vs)
	}
	if vs[0].File != "internal/foo/bar.go" || vs[0].Line != 4 {
		t.Errorf("violation = %s:%d, want internal/foo/bar.go:4", vs[0].File, vs[0].Line)
	}
}

// TestTrailingTrimmedLen pins the line count the untracked-file scan
// marks as changed. readSourceLines splits on "\n", so whether the file
// ends in a newline decides if the last element is a line or an artifact.
func TestTrailingTrimmedLen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  []string
		want int
	}{
		{name: "newline-terminated file", src: []string{"package k", "", "func A() {}", ""}, want: 3},
		{name: "file with no trailing newline", src: []string{"package k", "", "func A() {}"}, want: 3},
		{name: "empty file", src: []string{""}, want: 0},
		{name: "no content at all", src: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := trailingTrimmedLen(tt.src); got != tt.want {
				t.Errorf("trailingTrimmedLen(%q) = %d, want %d", tt.src, got, tt.want)
			}
		})
	}
}

func TestNewFilePath(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"+++ b/internal/foo/bar.go": "internal/foo/bar.go",
		"+++ /dev/null":             "",
	}
	for in, want := range cases {
		if got := newFilePath(in); got != want {
			t.Errorf("newFilePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseHunkRange(t *testing.T) {
	t.Parallel()
	cases := []struct {
		header           string
		wantStart, wantN int
	}{
		{"@@ -1,3 +4,5 @@", 4, 5},
		{"@@ -1 +4 @@ func foo()", 4, 1}, // no explicit new length → defaults to 1
		{"@@ -1,2 +3,0 @@", 3, 0},        // pure deletion
		{"not a hunk header", 0, 0},
	}
	for _, c := range cases {
		gotStart, gotN := parseHunkRange(c.header)
		if gotStart != c.wantStart || gotN != c.wantN {
			t.Errorf("parseHunkRange(%q) = (%d,%d), want (%d,%d)", c.header, gotStart, gotN, c.wantStart, c.wantN)
		}
	}
}

func TestReadSourceLines(t *testing.T) {
	t.Parallel()

	t.Run("happy", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		p := filepath.Join(root, "f.txt")
		if err := os.WriteFile(p, []byte("a\nb\nc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		lines, err := readSourceLines(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) < 3 || lines[0] != "a" || lines[2] != "c" {
			t.Errorf("lines = %q", lines)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		if _, err := readSourceLines(filepath.Join(t.TempDir(), "nope")); err == nil {
			t.Fatal("want error for missing file, got nil")
		}
	})
}

func TestSortedKeys(t *testing.T) {
	t.Parallel()
	in := map[string][]coverBlock{"c": nil, "a": nil, "b": nil}
	got := sortedKeys(in)
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedKeys = %v, want %v", got, want)
		}
	}
}

func TestBlockHasCoverageIgnore(t *testing.T) {
	t.Parallel()
	ignored := map[int]bool{2: true}

	if !blockHasCoverageIgnore(coverBlock{StartLine: 1, EndLine: 3}, ignored) {
		t.Error("expected ignore directive within span to be found")
	}
	if blockHasCoverageIgnore(coverBlock{StartLine: 1, EndLine: 1}, ignored) {
		t.Error("unexpected match outside the annotated line")
	}
	// A span reaching past every annotated line must report no match.
	if blockHasCoverageIgnore(coverBlock{StartLine: 3, EndLine: 99}, ignored) {
		t.Error("span holding no annotated line should not match")
	}
	// Both ends of the span are inclusive. The profile emits single-line
	// blocks (StartLine == EndLine), so an exclusive end would drop the
	// annotation on exactly the statement it was written for.
	if !blockHasCoverageIgnore(coverBlock{StartLine: 1, EndLine: 2}, ignored) {
		t.Error("a directive on the span's last line must exempt the block")
	}
	if !blockHasCoverageIgnore(coverBlock{StartLine: 2, EndLine: 2}, ignored) {
		t.Error("a directive on a single-line block must exempt it")
	}
}
