package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestMakefileVersionStamp pins the version the Makefile stamps into a
// binary: `<branch>@<git describe>` inside a git checkout, and nothing
// at all outside one, so that version.Current falls back to the
// version Go records in the binary's build info.
func TestMakefileVersionStamp(t *testing.T) {
	t.Parallel()

	noGit := t.TempDir()
	makefile, err := os.ReadFile(filepath.Join(repoRoot(t), "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(noGit, "Makefile"), makefile, 0o644); err != nil {
		t.Fatalf("copy Makefile: %v", err)
	}

	cases := []struct {
		name string
		dir  string
		want *regexp.Regexp
	}{
		{"inside a git checkout", repoRoot(t), regexp.MustCompile(`version\.Stamp=[^"@\s]+@[^"\s]+"`)},
		{"outside any git checkout", noGit, regexp.MustCompile(`version\.Stamp="`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// The ceiling stops git from finding a repository above the
			// temporary directory, whatever the test host's layout.
			recipe := makeDryRun(t, tc.dir, "diag-aiwf", "GIT_CEILING_DIRECTORIES="+filepath.Dir(noGit))
			if !tc.want.MatchString(recipe) {
				t.Errorf("diag-aiwf recipe does not match %s:\n%s", tc.want, recipe)
			}
		})
	}
}
