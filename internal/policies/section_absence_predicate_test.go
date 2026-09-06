package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicySectionAbsenceSinglePredicate_ThisRepoIsClean asserts the
// live tree carries one heading scan, in the file that owns it.
func TestPolicySectionAbsenceSinglePredicate_ThisRepoIsClean(t *testing.T) {
	t.Parallel()
	vs, err := PolicySectionAbsenceSinglePredicate(repoRoot(t))
	if err != nil {
		t.Fatalf("PolicySectionAbsenceSinglePredicate: %v", err)
	}
	for _, v := range vs {
		t.Errorf("%s:%d %s", v.File, v.Line, v.Detail)
	}
}

// TestPolicySectionAbsenceSinglePredicate_FiresOnASecondScan drives the
// policy over a tree carrying the shape it bans, so its silence on the
// live tree means the scan is absent rather than the policy inert.
//
// Each row is one classification rule, not one spelling. Six ways a scan
// gets written and five things that are not one: a pattern asking which
// acceptance criteria a body carries, a heading rendered into a message,
// a classifier testing a heading of any level, and the one function
// exempt by name for using the heading as a terminator.
//
// The tolerant rows matter most. A scan accepting `##` without a space,
// or a regexp carrying `(?m)` for a whole-body match, is looser than the
// parser — which is the drift direction this exists to catch, and the
// spelling a new author reaches for first.
func TestPolicySectionAbsenceSinglePredicate_FiresOnASecondScan(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  string
		want int
	}{
		{
			name: "a prefix test on a body's lines",
			src:  "package check\n\nimport \"strings\"\n\nfunc hasGoal(line string) bool {\n\treturn strings.HasPrefix(line, \"## \")\n}\n",
			want: 1,
		},
		{
			name: "the same test on bytes",
			src:  "package check\n\nimport \"bytes\"\n\nfunc hasGoal(line []byte) bool {\n\treturn bytes.HasPrefix(line, []byte(\"## \"))\n}\n",
			want: 1,
		},
		{
			name: "a prefix test more tolerant than the parser",
			src:  "package check\n\nimport \"strings\"\n\nfunc hasGoal(line string) bool {\n\treturn strings.HasPrefix(line, \"##\")\n}\n",
			want: 1,
		},
		{
			name: "a search through the whole body",
			src:  "package check\n\nimport \"strings\"\n\nfunc sections(b string) []string {\n\treturn strings.Split(b, \"\\n## \")\n}\n",
			want: 1,
		},
		{
			name: "a literal extracted to a constant",
			src:  "package check\n\nimport \"strings\"\n\nconst h2 = \"## \"\n\nfunc hasGoal(line string) bool {\n\treturn strings.HasPrefix(line, h2)\n}\n",
			want: 1,
		},
		{
			name: "a regexp anchored on the heading",
			src:  "package check\n\nimport \"regexp\"\n\nvar h2 = regexp.MustCompile(`^##\\s+(.+)$`)\n",
			want: 1,
		},
		{
			name: "the same regexp written for a whole body",
			src:  "package check\n\nimport \"regexp\"\n\nvar h2 = regexp.MustCompile(`(?m)^## (.+)$`)\n",
			want: 1,
		},
		{
			name: "the AC-heading pattern, which asks a different question",
			src:  "package check\n\nimport \"regexp\"\n\nvar h3 = regexp.MustCompile(`(?m)^###\\s+AC-(\\d+)`)\n",
			want: 0,
		},
		{
			name: "a heading rendered into a message",
			src:  "package check\n\nfunc quoted(name string) string {\n\treturn \"## \" + name\n}\n",
			want: 0,
		},
		{
			name: "a classifier testing a heading of any level",
			src:  "package check\n\nimport \"strings\"\n\nfunc isHeading(line string) bool {\n\treturn strings.HasPrefix(line, \"#\")\n}\n",
			want: 0,
		},
		{
			name: "the exempt AC-body terminator",
			src:  "package check\n\nimport \"strings\"\n\nfunc scanACBodies(line string) bool {\n\treturn strings.HasPrefix(line, \"## \")\n}\n",
			want: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for _, pkg := range sectionScanPackages {
				if err := os.MkdirAll(filepath.Join(root, pkg), 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", pkg, err)
				}
			}
			if err := os.WriteFile(filepath.Join(root, "internal", "check", "rule.go"), []byte(tc.src), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			vs, err := PolicySectionAbsenceSinglePredicate(root)
			if err != nil {
				t.Fatalf("PolicySectionAbsenceSinglePredicate: %v", err)
			}
			if len(vs) != tc.want {
				t.Errorf("violations = %d, want %d: %+v", len(vs), tc.want, vs)
			}
		})
	}
}

// TestPolicySectionAbsenceSinglePredicate_RefusesRatherThanReportsClean
// pins that the ban fails loudly when it cannot read what it judges.
//
// A silent skip is the shape that makes a ban vacuous without anything
// going red: rename or move one of the packages it scans and it would
// report a clean tree forever, which is indistinguishable from the
// tree being clean.
func TestPolicySectionAbsenceSinglePredicate_RefusesRatherThanReportsClean(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{
			name:  "a scanned package is not there",
			setup: func(t *testing.T, root string) { t.Helper() },
		},
		{
			name: "a scanned file does not parse",
			setup: func(t *testing.T, root string) {
				t.Helper()
				for _, pkg := range sectionScanPackages {
					if err := os.MkdirAll(filepath.Join(root, pkg), 0o755); err != nil {
						t.Fatalf("mkdir %s: %v", pkg, err)
					}
				}
				if err := os.WriteFile(filepath.Join(root, "internal", "check", "rule.go"), []byte("package check\n\nfunc ("), 0o600); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			tc.setup(t, root)
			if _, err := PolicySectionAbsenceSinglePredicate(root); err == nil {
				t.Error("expected an error, got nil — the policy reported a clean tree it could not read")
			}
		})
	}
}
