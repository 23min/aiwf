package policies

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// wf-measure-spec structural tests (M-0308). The skill lives under
// internal/skills/embedded-rituals/**, so referencing its path here also
// discharges the skill-edit-structural-test-backstop: the ritual's
// SKILL.md is referenced by the tests below.
//
// The expected side is derived, not typed. Verb names come from
// findAllVerbs, which AST-walks cli.NewRootCmd's AddCommand calls, so a
// rename in the command tree turns these red rather than leaving the
// ritual pointing at a command that no longer exists. Section shape is
// asserted by heading count and by each heading carrying content —
// positional properties a reader cannot satisfy by typing a word.
const wfMeasureSpecFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-measure-spec/SKILL.md"

// recordSectionHeading is the `## …` section that defines what a completed
// pass leaves behind. D-0066 makes the record a body section written into
// the measured entity, so this section is where the ritual says which verb
// lands it and what the two possible outcomes are.
const recordSectionHeading = "The record"

// TestWfMeasureSpec_RecordSectionNamesTheVerbThatLandsIt pins the first
// half of AC-1: a completed pass leaves a record against the entity whose
// claims it measured, so the ritual must name the verb that writes it.
//
// Both sides are derived. Every `aiwf <verb>` the record section names is
// resolved against the live command tree, and the body-writing verb is
// looked up there before being required in the section — so renaming the
// command fails this test instead of silently orphaning the prose.
func TestWfMeasureSpec_RecordSectionNamesTheVerbThatLandsIt(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfMeasureSpecFixturePath)

	record := extractMarkdownSection(body, 2, recordSectionHeading)
	if record == "" {
		t.Fatalf("wf-measure-spec must have a `## %s` section defining what a completed pass leaves behind", recordSectionHeading)
	}

	verbs, err := findAllVerbs(repoRoot(t))
	if err != nil {
		t.Fatalf("walk the cobra command tree: %v", err)
	}

	// The record is a body write, so the section must name the body-writing
	// command. Resolving it through the tree first is what makes the name
	// below a derivation rather than a literal: rename the command and this
	// fails here, naming the ritual that needs updating with it.
	const bodyWriteVerb = "edit-body"
	if _, ok := verbs[bodyWriteVerb]; !ok {
		t.Fatalf("no %q command in the cobra tree — the verb that lands the record was renamed; update wf-measure-spec's `## %s` section and this constant together", bodyWriteVerb, recordSectionHeading)
	}

	mentions := backtickedAiwfMentions(record, verbs)
	if len(mentions) == 0 {
		t.Fatalf("`## %s` names no aiwf command; a record nobody is told how to write is not a record", recordSectionHeading)
	}

	named := false
	for _, m := range mentions {
		if !m.resolved {
			t.Errorf("`## %s` names `aiwf %s`, which resolves to no command in the cobra tree", recordSectionHeading, m.path)
		}
		if m.path == bodyWriteVerb {
			named = true
		}
	}
	if !named {
		t.Errorf("`## %s` must name `aiwf %s` as the verb that lands the record", recordSectionHeading, bodyWriteVerb)
	}
}

// TestWfMeasureSpec_RecordSectionNamesBothOutcomes pins the second half of
// AC-1: a pass that measured nothing and a pass that was never run have to
// be distinguishable afterwards, which takes two named outcomes rather than
// one. Exactly two `###` sub-sections carry them, and each carries content —
// so gutting one to a bare heading fails here rather than passing because
// the heading survived.
func TestWfMeasureSpec_RecordSectionNamesBothOutcomes(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfMeasureSpecFixturePath)

	record := extractMarkdownSection(body, 2, recordSectionHeading)
	if record == "" {
		t.Fatalf("wf-measure-spec must have a `## %s` section defining what a completed pass leaves behind", recordSectionHeading)
	}

	if got := countSubHeadings(record, 3); got != 2 {
		t.Errorf("`## %s` has %d `###` outcomes; want exactly 2 — the pass that changed the entity and the pass that changed nothing", recordSectionHeading, got)
	}

	for _, sub := range emptySubSections(record, 3) {
		t.Errorf("outcome %q carries no content; a heading alone does not tell a reader what to write", sub)
	}
}

// TestEmptySubSections covers each arm of the scanner the outcome assertion
// relies on: content, no content, a deeper heading standing in for content,
// and a shallower heading closing the run.
func TestEmptySubSections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		section string
		want    []string
	}{
		{
			name:    "sub-section with prose is not empty",
			section: "### One\n\nsome prose\n",
			want:    nil,
		},
		{
			name:    "sub-section with only blank lines is empty",
			section: "### One\n\n   \n\n### Two\n\nprose\n",
			want:    []string{"One"},
		},
		{
			name:    "a deeper heading counts as content",
			section: "### One\n\n#### Detail\n",
			want:    nil,
		},
		{
			name:    "a shallower heading closes the run",
			section: "### One\n\n## Elsewhere\n\nprose\n",
			want:    []string{"One"},
		},
		{
			name:    "trailing sub-section is flushed",
			section: "### One\n\nprose\n\n### Two\n",
			want:    []string{"Two"},
		},
		{
			name:    "no sub-sections at all",
			section: "just prose, no headings\n",
			want:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := emptySubSections(tc.section, 3)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("emptySubSections() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// emptySubSections returns the heading text of every level-`level` sub-section
// of section whose body holds no non-blank line. Used to reject a section
// gutted to bare headings, which a heading count alone still accepts.
func emptySubSections(section string, level int) []string {
	var empty []string
	current := ""
	hasContent := false
	flush := func() {
		if current != "" && !hasContent {
			empty = append(empty, current)
		}
	}
	for _, ln := range strings.Split(section, "\n") {
		lvl := headingLevel(ln)
		switch {
		case lvl > 0 && lvl < level:
			// A shallower heading closes the run of sub-sections.
			flush()
			current = ""
			hasContent = false
		case lvl == level:
			flush()
			current = strings.TrimSpace(strings.TrimLeft(ln, "# "))
			hasContent = false
		case lvl > level:
			// A deeper heading is content of the sub-section it sits under.
			hasContent = true
		default:
			if strings.TrimSpace(ln) != "" {
				hasContent = true
			}
		}
	}
	flush()
	return empty
}
