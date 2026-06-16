package policies

import (
	"os"
	"path/filepath"
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

	t.Run("readSourceLines error when working-tree file removed", func(t *testing.T) {
		t.Parallel()
		const headSrc = "package foo\n\nfunc Add(a, b int) int {\n\tif a < 0 {\n\t\treturn 0\n\t}\n\treturn a + b\n}\n"
		profile := "mode: atomic\n" + fixtureModule + "/internal/foo/bar.go:4.12,6.3 1 0\n"
		root, baseSHA, profilePath := covFixture(t,
			"package foo\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n",
			headSrc, profile)
		// The file is committed at HEAD but absent on disk → readSourceLines fails.
		if rmErr := os.Remove(filepath.Join(root, "internal", "foo", "bar.go")); rmErr != nil {
			t.Fatalf("remove: %v", rmErr)
		}
		_, err := branchCoverageViolations(root, profilePath, baseSHA)
		if err == nil {
			t.Fatal("want error for removed source file, got nil")
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
	src := []string{"line1", "line2 //coverage:ignore reason", "line3"}

	if !blockHasCoverageIgnore(coverBlock{StartLine: 1, EndLine: 3}, src) {
		t.Error("expected ignore directive within span to be found")
	}
	if blockHasCoverageIgnore(coverBlock{StartLine: 1, EndLine: 1}, src) {
		t.Error("unexpected match outside the annotated line")
	}
	// Out-of-range span must not panic and must report no match.
	if blockHasCoverageIgnore(coverBlock{StartLine: 1, EndLine: 99}, []string{"only"}) {
		t.Error("out-of-range span should not match")
	}
}
