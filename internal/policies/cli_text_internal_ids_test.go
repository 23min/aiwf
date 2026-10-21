package policies

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPolicy_CLITextCarriesNoInternalIDs(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyCLITextCarriesNoInternalIDs)
}

// TestPolicyCLITextCarriesNoInternalIDs_Rules pins one row per rule the
// policy decides: which shapes fire, which a consumer may legitimately read,
// which literals are data rather than text, and where a finding is reported.
func TestPolicyCLITextCarriesNoInternalIDs_Rules(t *testing.T) {
	t.Parallel()
	const bt = "`"
	cases := []struct {
		name      string
		files     map[string]string
		wantLines []int
	}{
		{
			name:      "a digit-bearing id in text fires",
			files:     map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"priority level (G-0078)\"\n"},
			wantLines: []int{3},
		},
		{
			name:      "a placeholder below canonical width fires",
			files:     map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"composite ids (M-NNN/AC-N) accepted\"\n"},
			wantLines: []int{3},
		},
		{
			name:  "a canonical placeholder passes",
			files: map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"aiwf promote M-NNNN/AC-N met\"\n"},
		},
		{
			name:      "every unhyphenated gap label in a literal fires, not only the first",
			files:     map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"the G24 and G42 recovery paths\"\n"},
			wantLines: []int{3, 3},
		},
		{
			name: "every source path in a literal fires, not only the first",
			files: map[string]string{
				"internal/cli/x/x.go": "package x\n\nvar s = \"see docs/design/a.md and cmd/aiwf/main.go\"\n",
				"docs/design/a.md":    "fixture\n",
				"cmd/aiwf/main.go":    "package main\n",
			},
			wantLines: []int{3, 3},
		},
		{
			name:      "text is plain, not Markdown: an id in a link destination still fires",
			files:     map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"see [the notes](G-0078) here\"\n"},
			wantLines: []int{3},
		},
		{
			name:      "an interpreted literal reports its own line, whatever newlines it escapes",
			files:     map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"first\\nsee G24\"\n"},
			wantLines: []int{3},
		},
		{
			name: "a relative path into aiwf's source tree fires",
			files: map[string]string{
				"internal/cli/x/x.go": "package x\n\nvar s = \"install via go install ./cmd/aiwf\"\n" +
					"var u = \"see ../internal/cli/check.go.\"\n",
				"cmd/aiwf/main.go": "package main\n",
			},
			wantLines: []int{3, 4},
		},
		{
			name:      "a literal that opens with a path is judged from its first byte",
			files:     map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"internal/cli is where the verbs live\"\n"},
			wantLines: []int{3},
		},
		{
			name:  "a path naming nothing in aiwf's tree passes, as a consumer's own does",
			files: map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"paths: [internal/billing/**]\"\n"},
		},
		{
			name:  "a bare top-level directory passes, even where aiwf has one",
			files: map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"Go code under internal/ in your repo\"\n"},
		},
		{
			name: "a path that ends a sentence still fires",
			files: map[string]string{
				"internal/cli/x/x.go": "package x\n\nvar s = \"the entry point is cmd/aiwf.\"\n",
				"cmd/aiwf/main.go":    "package main\n",
			},
			wantLines: []int{3},
		},
		{
			name: "the consumer's own ADR directory passes, even where aiwf has one",
			files: map[string]string{
				"internal/cli/x/x.go": "package x\n\nvar s = \"moves docs/adr/ADR-NNNN-<slug>.md\"\n",
				"docs/adr/x.md":       "fixture\n",
			},
		},
		{
			name: "an installable module path passes, even where aiwf has the directory",
			files: map[string]string{
				"internal/cli/x/x.go": "package x\n\nvar s = \"go install github.com/23min/aiwf/cmd/aiwf@latest\"\n",
				"cmd/aiwf/main.go":    "package main\n",
			},
		},
		{
			name:  "a literal with no whitespace is data, not text",
			files: map[string]string{"internal/cli/x/x.go": "package x\n\nvar args = []string{\"move\", \"E-01\", \"internal/skills/\"}\n"},
		},
		{
			name:  "comments are not judged",
			files: map[string]string{"internal/cli/x/x.go": "package x\n\n// see G-0078 and docs/design/x.md\nvar s = \"plain text\"\n"},
		},
		{
			name: "every package whose text an operator reads is judged as the CLI's is",
			files: map[string]string{
				"internal/verb/v.go":     "package verb\n\nvar s = \"refusing (ADR-0010)\"\n",
				"internal/config/c.go":   "package config\n\nvar s = \"the sentinel (ADR-0021)\"\n",
				"internal/aiwfyaml/a.go": "package aiwfyaml\n\nvar s = \"does not match C-NNN format\"\n",
				"internal/gitops/g.go":   "package gitops\n\nvar s = \"must be hex (per M-0161/AC-6)\"\n",
			},
			wantLines: []int{3, 3, 3, 3},
		},
		{
			name: "tests, test support and packages no operator reads are not judged",
			files: map[string]string{
				"internal/cli/x/x_test.go":           "package x\n\nvar s = \"see G-0078\"\n",
				"internal/cli/cliutil/testutil/t.go": "package testutil\n\nvar s = \"see G-0078\"\n",
				"internal/entity/e.go":               "package entity\n\nvar s = \"see G-0078\"\n",
			},
		},
		{
			name:  "a file that does not parse is left to the compiler",
			files: map[string]string{"internal/cli/x/x.go": "package x\n\nvar s = \"see G-0078\n"},
		},
		{
			name: "a raw literal reports the source line the token sits on",
			files: map[string]string{
				"internal/cli/x/x.go": "package x\n\nvar s = " + bt + "first line\n" +
					"second cites E-0043\nthird names G24\ndocs/design/x.md opens the fourth" + bt + "\n",
				"docs/design/x.md": "fixture\n",
			},
			wantLines: []int{4, 5, 6},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := PolicyCLITextCarriesNoInternalIDs(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			var got []int
			for _, v := range vs {
				if v.Policy == "cli-text-internal-id" {
					got = append(got, v.Line)
				}
			}
			if diff := cmp.Diff(tc.wantLines, got); diff != "" {
				t.Errorf("violation lines mismatch (-want +got):\n%s\nviolations: %+v", diff, vs)
			}
		})
	}
}

// TestPolicyCLITextCarriesNoInternalIDs_UnreadableRootIsAnError: a root the
// walk cannot read is reported as an error, never as a clean pass.
func TestPolicyCLITextCarriesNoInternalIDs_UnreadableRootIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := PolicyCLITextCarriesNoInternalIDs(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("policy returned no error for a root that does not exist")
	}
}
