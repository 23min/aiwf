package policies

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// shapeSkill wraps table rows in a severity-declaring section, which is
// what puts them in the policy's scope.
func shapeSkill(rows string) string {
	return "# aiwf-check\n\n## Findings (errors)\n\n| Code | Meaning |\n|---|---|\n" + rows
}

// TestSkillFindingsTableShape_Rows is the firing/silent matrix over one
// row at a time: what the row looks like, and whether the two
// remediation shapes — a third column, an inline Fix: clause — are
// caught while legitimate prose is left alone.
func TestSkillFindingsTableShape_Rows(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		row   string
		fires bool
		// mentions is a fragment the violation must carry, so the
		// message names which of the two shapes was found.
		mentions string
	}{
		{
			name: "two-cell row is the shape",
			row:  "| `probe` | what the rule means. |\n",
		},
		{
			name:     "third column fires",
			row:      "| `probe` | what the rule means. | run `aiwf promote`. |\n",
			fires:    true,
			mentions: "has 3 cells",
		},
		{
			name:     "inline Fix: clause fires",
			row:      "| `probe` | what the rule means. Fix: run `aiwf promote`. |\n",
			fires:    true,
			mentions: `"Fix:" clause`,
		},
		{
			name:     "a one-cell row fires, and the message agrees with its count",
			row:      "| `probe` |\n",
			fires:    true,
			mentions: "has 1 cell;",
		},
		{
			name:     "a lone pipe is a truncated row, reported rather than read past",
			row:      "|\n",
			fires:    true,
			mentions: "has 1 cell;",
		},
		{
			name: "a cell may carry an escaped pipe without reading as a third column",
			row:  "| `probe` | takes `<required\\|advisory\\|none>`. |\n",
		},
		{
			name: "prose between rows is not a row",
			row:  "\nSome prose under the heading, not a table row.\n",
		},
		{
			name: "the word fix without the clause marker is prose",
			row:  "| `probe` | the verb refuses until you fix the frontmatter. |\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, skillCheckPath), shapeSkill(tc.row))

			vs, err := PolicySkillFindingsTableShape(root)
			if err != nil {
				t.Fatalf("policy error: %v", err)
			}
			got := hasPolicyViolation(vs, "skill-findings-table-shape")
			if got != tc.fires {
				t.Fatalf("fired=%v, want %v; violations: %+v", got, tc.fires, vs)
			}
			if tc.mentions != "" && !violationMentions(vs, tc.mentions) {
				t.Errorf("violation should mention %q; got %+v", tc.mentions, vs)
			}
			if tc.fires && !violationMentions(vs, "internal/check/hint.go") {
				t.Errorf("violation should point at the hint table; got %+v", vs)
			}
		})
	}
}

// TestSkillFindingsTableShape_ScopedToSeverityDeclaringSections proves
// the policy reads only findings tables. The skill's other tables have
// their own shapes on purpose — the hook table under "What to run" is
// three columns — and constraining them would be this policy
// overreaching into layout it has no claim on.
func TestSkillFindingsTableShape_ScopedToSeverityDeclaringSections(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, skillCheckPath),
		"# aiwf-check\n\n## What to run\n\n| Hook | What it runs | What it catches |\n|---|---|---|\n"+
			"| `pre-push` | full `aiwf check` | everything. Fix: nothing. |\n"+
			shapeSkill("| `probe` | what the rule means. |\n"))

	vs, err := PolicySkillFindingsTableShape(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("a table outside a severity-declaring section must not fire; got %+v", vs)
	}
}

// TestSkillFindingsTableShape_EveryFindingsSectionIsInScope pins that
// scope is the severity marker rather than one heading's wording, so
// the warnings and provenance-errors tables are held to the same shape
// as the errors table they sit beside.
func TestSkillFindingsTableShape_EveryFindingsSectionIsInScope(t *testing.T) {
	t.Parallel()
	for _, heading := range []string{
		"## Findings (errors)",
		"## Findings (warnings)",
		"## Findings (conditional severity)",
		"## Provenance findings (errors)",
	} {
		t.Run(heading, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, skillCheckPath),
				"# aiwf-check\n\n"+heading+"\n\n| Code | Meaning | Typical fix |\n|---|---|---|\n"+
					"| `probe` | what the rule means. | run `aiwf promote`. |\n")

			vs, err := PolicySkillFindingsTableShape(root)
			if err != nil {
				t.Fatalf("policy error: %v", err)
			}
			if !hasPolicyViolation(vs, "skill-findings-table-shape") {
				t.Fatalf("a three-column table under %q must fire; got %+v", heading, vs)
			}
		})
	}
}

// TestSkillFindingsTableShape_SubheadingDoesNotEndTheSection pins that
// section tracking follows `## ` only. A `###` subheading inside a
// findings section is part of that section, so it must not carry its
// rows out of scope — the shape a fail-open would take, since a row the
// policy never reads is a row it never judges.
func TestSkillFindingsTableShape_SubheadingDoesNotEndTheSection(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, skillCheckPath),
		"# aiwf-check\n\n## Findings (errors)\n\n### A subheading inside the section\n\n"+
			"| Code | Meaning | Typical fix |\n|---|---|---|\n| `probe` | means. | run it. |\n")

	vs, err := PolicySkillFindingsTableShape(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if !hasPolicyViolation(vs, "skill-findings-table-shape") {
		t.Fatalf("rows after a `###` subheading are still in the section; got %+v", vs)
	}
}

// TestSkillFindingsTableShape_HeaderAndSeparatorAreJudged covers the
// rows that carry no finding code: a reinstated column shows up in the
// header and separator before any row uses it, and catching it there
// names the line an author would actually edit.
func TestSkillFindingsTableShape_HeaderAndSeparatorAreJudged(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, skillCheckPath),
		"# aiwf-check\n\n## Findings (errors)\n\n| Code | Meaning | Typical fix |\n|---|---|---|\n")

	vs, err := PolicySkillFindingsTableShape(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if len(vs) != 2 {
		t.Fatalf("header and separator should each fire; got %+v", vs)
	}
	if diff := cmp.Diff([]int{5, 6}, []int{vs[0].Line, vs[1].Line}); diff != "" {
		t.Errorf("violation lines (-want +got):\n%s", diff)
	}
}

// TestSkillFindingsTableShape_MissingSkillErrors pins the failure exit:
// an unreadable skill is an error rather than a clean tree, so a
// renamed or deleted file cannot read as compliance.
func TestSkillFindingsTableShape_MissingSkillErrors(t *testing.T) {
	t.Parallel()
	if _, err := PolicySkillFindingsTableShape(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected an error when the skill cannot be read")
	}
}

// TestSkillFindingsTableShape_LiveTree runs the policy over the repo.
func TestSkillFindingsTableShape_LiveTree(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicySkillFindingsTableShape)
}
