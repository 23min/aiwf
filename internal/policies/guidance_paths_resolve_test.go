package policies

import (
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/23min/aiwf/internal/projectguidance"
)

// TestPolicy_GuidancePathsResolve holds the live tree to M-0337 AC-4: every
// repository path the development guidance cites exists.
func TestPolicy_GuidancePathsResolve(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyGuidancePathsResolve)
}

// TestCitedPath pins which backticked spans are repository paths (M-0337
// AC-4): a span with a slash whose first segment names an entry at the
// repository root, with a leading `./` and a trailing `/...` package
// pattern read as the path they name; a span with whitespace, an
// angle-bracket placeholder, or a first segment the root does not hold
// is no path.
func TestCitedPath(t *testing.T) {
	t.Parallel()
	roots := map[string]bool{"internal": true, "cmd": true, ".github": true}
	cases := []struct {
		span, want string
		ok         bool
	}{
		{"internal/policies/x.go", "internal/policies/x.go", true},
		{"./cmd/aiwf", "cmd/aiwf", true},
		{"internal/...", "internal", true},
		{".github/workflows/*", ".github/workflows/*", true},
		{"internal", "", false},
		{"go test ./internal/...", "", false},
		{"internal/run.sh go test ./...", "", false},
		{"internal/<name>/x.go", "", false},
		{"github.com/spf13/cobra", "", false},
		{"origin/main", "", false},
	}
	for _, tc := range cases {
		got, ok := citedPath(tc.span, roots)
		if got != tc.want || ok != tc.ok {
			t.Errorf("citedPath(%q) = %q, %v; want %q, %v", tc.span, got, ok, tc.want, tc.ok)
		}
	}
}

// TestPathResolves pins how a cited path is matched against the tree
// (M-0337 AC-4): a plain path must exist, a glob must match a file, and a
// brace group is expanded with every alternative required to resolve.
func TestPathResolves(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, rel := range []string{"internal/a/x.go", "internal/b/y.go", "internal/skills/embedded/s.md"} {
		mustWrite(t, filepath.Join(root, rel), "x\n")
	}
	cases := []struct {
		path string
		want bool
	}{
		{"internal/a/x.go", true},
		{"internal/a/", true},
		{"internal/a/z.go", false},
		{"internal/*/y.go", true},
		{"internal/**/embedded*", true},
		{"internal/{a,b}/", true},
		{"internal/{a,c}/", false},
		{"internal/skills/embedded{,-rituals}/**", false},
	}
	for _, tc := range cases {
		if got := pathResolves(root, tc.path); got != tc.want {
			t.Errorf("pathResolves(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestPolicyGuidancePathsResolve_ReportsStalePaths pins the policy over a
// repository (M-0337 AC-4): G-0436's two stale citations are reported
// against the guidance that cites them, at the line a reader finds them on
// below a managed block, with the fix; a path git ignores, as a file or
// under a pattern naming only directories, is not judged, since only a
// command such as `aiwf update` or a build creates it; and a directory
// outside any repository, or one that cannot be read, is an error.
func TestPolicyGuidancePathsResolve_ReportsStalePaths(t *testing.T) {
	t.Parallel()
	start, end, _ := projectguidance.GuidanceMarkers()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".gitignore"), "/bin/\n.agents/skills/aiwf-*/\n")
	mustWrite(t, filepath.Join(root, "bin", "other"), "x\n")
	mustWrite(t, filepath.Join(root, ".agents", "skills", "README.md"), "x\n")
	mustWrite(t, filepath.Join(root, "cmd", "aiwf", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "# Repo\n\n"+start+"\n`cmd/aiwf/managed.go`\n"+end+"\n\nThe drift test is `cmd/aiwf/completion_drift_test.go`; history lives in `cmd/aiwf/admin_cmd.go`.\nBuild `./bin/aiwf-diag`; skills are `.agents/skills/aiwf-*`; the entry point is `cmd/aiwf/main.go`.\n")
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	vs, err := PolicyGuidancePathsResolve(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	var details []string
	for _, v := range vs {
		if v.Policy != "guidance-paths-resolve" || v.File != "CLAUDE.md" || v.Line != 7 {
			t.Errorf("violation %+v, want guidance-paths-resolve on CLAUDE.md:7", v)
		}
		details = append(details, v.Detail)
	}
	want := []string{
		"cites cmd/aiwf/completion_drift_test.go, which does not exist; correct the path or remove the citation",
		"cites cmd/aiwf/admin_cmd.go, which does not exist; correct the path or remove the citation",
	}
	if !slices.Equal(details, want) {
		t.Errorf("details = %q, want %q", details, want)
	}
	outside := t.TempDir()
	mustWrite(t, filepath.Join(outside, "CLAUDE.md"), "See `missing/x.go`.\n")
	mustWrite(t, filepath.Join(outside, "missing", ".keep"), "")
	mustWrite(t, filepath.Join(outside, ".git"), "gitdir: "+filepath.Join(outside, "no-such-gitdir")+"\n")
	if _, err := PolicyGuidancePathsResolve(outside); err == nil {
		t.Error("a directory outside any repository must be an error, not a pass")
	}
	if _, err := PolicyGuidancePathsResolve(filepath.Join(outside, "absent")); err == nil {
		t.Error("a root that cannot be read must be an error, not a pass")
	}
}
