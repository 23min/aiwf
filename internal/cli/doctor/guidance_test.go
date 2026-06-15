package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guidanceFixture builds a temp repo: optionally with the materialized
// guidance file, and a CLAUDE.md that optionally imports it (M-0165).
func guidanceFixture(t *testing.T, withGuidanceFile, withImport bool) string {
	t.Helper()
	root := t.TempDir()
	if withGuidanceFile {
		if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".claude", "aiwf-guidance.md"), []byte("guidance"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	claude := "# project\n"
	if withImport {
		claude += "@.claude/aiwf-guidance.md\n"
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(claude), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// AC-1: an unwired tree (guidance file present, CLAUDE.md does not import
// it) yields an advisory naming the exact fix command.
func TestGuidanceImportReport_UnwiredEmitsAdvisory(t *testing.T) {
	t.Parallel()
	root := guidanceFixture(t, true, false)
	out := strings.Join(appendGuidanceImportReport(nil, root), "\n")
	if !strings.Contains(out, "claudemd-guidance-unwired") {
		t.Errorf("AC-1: expected the unwired advisory; got:\n%s", out)
	}
	if !strings.Contains(out, "aiwf init") {
		t.Errorf("AC-1: advisory must name the exact fix command `aiwf init`; got:\n%s", out)
	}
}

// AC-2 + AC-3: the wired and absent states yield no unwired advisory, and
// all three states (unwired / wired / absent) are exercised.
func TestGuidanceImportReport_States(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		withGuidanceFile bool
		withImport       bool
		wantUnwired      bool
	}{
		{"unwired", true, false, true},
		{"wired", true, true, false},
		{"absent", false, false, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := guidanceFixture(t, tc.withGuidanceFile, tc.withImport)
			out := strings.Join(appendGuidanceImportReport(nil, root), "\n")
			if got := strings.Contains(out, "claudemd-guidance-unwired"); got != tc.wantUnwired {
				t.Errorf("AC-2/AC-3 [%s]: unwired advisory present=%v, want %v; out:\n%s", tc.name, got, tc.wantUnwired, out)
			}
		})
	}
}

// Branch coverage: guidance fragment present but CLAUDE.md absent (read
// error) is treated as unwired.
func TestGuidanceImportReport_GuidancePresentNoClaudeMd(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "aiwf-guidance.md"), []byte("g"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(appendGuidanceImportReport(nil, root), "\n")
	if !strings.Contains(out, "claudemd-guidance-unwired") {
		t.Errorf("guidance present but no CLAUDE.md should be unwired; got:\n%s", out)
	}
}
