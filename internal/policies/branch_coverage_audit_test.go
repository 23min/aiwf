package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

// fixtureModule is the synthetic module path used by the
// branchCoverageViolations fixtures. The coverage-profile lines below
// are prefixed with it; the parser strips it to recover repo-relative
// paths.
const fixtureModule = "example.com/cov"

// covFixture builds a throwaway git repo at a t.TempDir with a base
// commit (baseSrc) and a HEAD commit (headSrc) for internal/foo/bar.go,
// writes profileContent to coverage.out, and returns the root and the
// base commit SHA. The returned baseSHA is what a caller passes as the
// baseRef to branchCoverageViolations.
func covFixture(t *testing.T, baseSrc, headSrc, profileContent string) (root, baseSHA, profilePath string) {
	t.Helper()
	root = t.TempDir()
	runGit := repoGitRunner(t, root)
	writeFile := repoFileWriter(t, root)

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "aiwf-test")
	writeFile("go.mod", "module "+fixtureModule+"\n\ngo 1.24\n")
	writeFile("internal/foo/bar.go", baseSrc)
	runGit("add", "-A")
	runGit("commit", "-m", "base")
	baseSHA = trimLine(runGit("rev-parse", "HEAD"))

	writeFile("internal/foo/bar.go", headSrc)
	runGit("add", "-A")
	runGit("commit", "-m", "head")

	profilePath = filepath.Join(root, "coverage.out")
	if wErr := os.WriteFile(profilePath, []byte(profileContent), 0o644); wErr != nil {
		t.Fatalf("write profile: %v", wErr)
	}
	return root, baseSHA, profilePath
}

// repoGitRunner returns a closure that runs git in root and fails the
// test on any non-zero exit.
func repoGitRunner(t *testing.T, root string) func(args ...string) string {
	t.Helper()
	return func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
}

// repoFileWriter returns a closure that writes a repo-relative file,
// creating parent directories as needed.
func repoFileWriter(t *testing.T, root string) func(rel, content string) {
	t.Helper()
	return func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if mkErr := os.MkdirAll(filepath.Dir(p), 0o755); mkErr != nil {
			t.Fatalf("mkdir %s: %v", p, mkErr)
		}
		if wErr := os.WriteFile(p, []byte(content), 0o644); wErr != nil {
			t.Fatalf("write %s: %v", p, wErr)
		}
	}
}

func trimLine(s string) string {
	for s != "" && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

func violationLines(vs []Violation) []int {
	out := make([]int, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Line)
	}
	sort.Ints(out)
	return out
}

func TestBranchCoverageViolations(t *testing.T) {
	t.Parallel()

	const baseSrc = "package foo\n\n" +
		"func Add(a, b int) int {\n" +
		"\treturn a + b\n" +
		"}\n"

	// headSrc adds a guard branch: lines 4-6 are new/changed.
	//  1 package foo
	//  2
	//  3 func Add(a, b int) int {
	//  4 \tif a < 0 {
	//  5 \t\treturn 0
	//  6 \t}
	//  7 \treturn a + b
	//  8 }
	const headSrc = "package foo\n\n" +
		"func Add(a, b int) int {\n" +
		"\tif a < 0 {\n" +
		"\t\treturn 0\n" +
		"\t}\n" +
		"\treturn a + b\n" +
		"}\n"

	// Same as headSrc but the guard body carries a //coverage:ignore.
	const headSrcIgnored = "package foo\n\n" +
		"func Add(a, b int) int {\n" +
		"\tif a < 0 { //coverage:ignore defensive guard, not reachable in fixtures\n" +
		"\t\treturn 0\n" +
		"\t}\n" +
		"\treturn a + b\n" +
		"}\n"

	const prefix = fixtureModule + "/internal/foo/bar.go"

	tests := []struct {
		name      string
		headSrc   string
		profile   string
		wantLines []int
	}{
		{
			name:    "uncovered changed branch fires",
			headSrc: headSrc,
			// Block covering the new guard (lines 4-6), count 0.
			profile: "mode: atomic\n" +
				prefix + ":4.12,6.3 1 0\n" +
				prefix + ":7.2,7.13 1 1\n",
			wantLines: []int{4},
		},
		{
			name:    "covered changed branch clean",
			headSrc: headSrc,
			profile: "mode: atomic\n" +
				prefix + ":4.12,6.3 1 1\n" +
				prefix + ":7.2,7.13 1 1\n",
			wantLines: nil,
		},
		{
			name:    "uncovered changed branch annotated is clean",
			headSrc: headSrcIgnored,
			profile: "mode: atomic\n" +
				prefix + ":4.12,6.3 1 0\n" +
				prefix + ":7.2,7.13 1 1\n",
			wantLines: nil,
		},
		{
			name:    "uncovered block on unchanged line is clean",
			headSrc: headSrc,
			// Mark only line 7 (return a+b) — which existed in base and
			// was merely shifted, not added — as uncovered. The diff's
			// new-file added lines are 4-6, so line 7 is out of scope.
			profile: "mode: atomic\n" +
				prefix + ":4.12,6.3 1 1\n" +
				prefix + ":7.2,7.13 1 0\n",
			wantLines: nil,
		},
		{
			name:    "file not in profile is clean",
			headSrc: headSrc,
			profile: "mode: atomic\n" +
				fixtureModule + "/internal/other/x.go:1.1,2.2 1 0\n",
			wantLines: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, baseSHA, profilePath := covFixture(t, baseSrc, tt.headSrc, tt.profile)
			vs, err := branchCoverageViolations(root, profilePath, baseSHA)
			if err != nil {
				t.Fatalf("branchCoverageViolations: %v", err)
			}
			got := violationLines(vs)
			if !equalInts(got, tt.wantLines) {
				t.Errorf("violation lines = %v, want %v (violations: %+v)", got, tt.wantLines, vs)
			}
		})
	}
}

func TestBranchCoverageViolations_BaseUnresolvable(t *testing.T) {
	t.Parallel()
	root, _, profilePath := covFixture(t,
		"package foo\n\nfunc Add(a, b int) int { return a + b }\n",
		"package foo\n\n// changed\nfunc Add(a, b int) int { return a + b }\n",
		"mode: atomic\n")

	// Empty and all-zero base refs mean "no comparison point" — the
	// audit must no-op rather than error (a fresh-branch push passes
	// the all-zero github.event.before).
	for _, base := range []string{"", "0000000000000000000000000000000000000000"} {
		vs, err := branchCoverageViolations(root, profilePath, base)
		if err != nil {
			t.Fatalf("base %q: unexpected error: %v", base, err)
		}
		if len(vs) != 0 {
			t.Errorf("base %q: got %d violations, want 0", base, len(vs))
		}
	}
}

func TestBranchCoverageViolations_MissingProfile(t *testing.T) {
	t.Parallel()
	root, baseSHA, _ := covFixture(t,
		"package foo\n\nfunc Add(a, b int) int { return a + b }\n",
		"package foo\n\n// changed\nfunc Add(a, b int) int { return a + b }\n",
		"mode: atomic\n")

	_, err := branchCoverageViolations(root, filepath.Join(root, "does-not-exist.out"), baseSHA)
	if err == nil {
		t.Fatal("expected an error for a missing profile, got nil")
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestPolicy_BranchCoverageAudit is the CI gate entry point. It runs
// the diff-scoped coverage audit against the live tree using the
// coverage profile and base ref supplied via environment. Without a
// profile (the default in the broad `go test ./...` job) it skips —
// the authoritative invocation is the dedicated CI coverage-gate step
// and the `make coverage-gate` target, both of which set
// AIWF_COVERAGE_PROFILE.
func TestPolicy_BranchCoverageAudit(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_PROFILE") == "" {
		t.Skip("AIWF_COVERAGE_PROFILE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicyBranchCoverageAudit)
}
