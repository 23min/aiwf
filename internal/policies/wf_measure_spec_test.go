package policies

import (
	"strings"
	"testing"
)

// wf-measure-spec structural tests (M-0308). The skill lives under
// internal/skills/embedded-rituals/**, so referencing its path here also
// discharges the skill-edit-structural-test-backstop: the ritual's
// SKILL.md is referenced by the tests below.
const wfMeasureSpecFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-measure-spec/SKILL.md"

// recordSectionHeading is the `## …` section that defines what a completed
// pass leaves behind.
const recordSectionHeading = "The record"

// recordedHeading is the heading the record is written under. It is a decided
// constant rather than a literal read back out of the ritual's prose: a later
// rule locates a completed pass by this heading, so the ritual has to state
// exactly this text and drifting it silently orphans that rule.
const recordedHeading = "## Spec measurement"

// TestWfMeasureSpec_RecordSectionNamesTheVerbThatLandsIt pins that the record
// is landed by a command that exists. The verb names are read from the live
// Cobra tree, so renaming the command turns this red instead of leaving the
// ritual pointing at nothing.
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

	// Looking the verb up in the tree before requiring it in the section is
	// what makes the name below a derivation rather than a literal: rename
	// the command and this fails here, naming the ritual to update with it.
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

// TestWfMeasureSpec_RecordSectionStatesTheDecidedHeading pins the record's
// shape. Its value is that a later reader finds a completed pass by one known
// heading, so naming a different heading — or describing the record without
// naming one — breaks what the record is for while leaving the rest green.
func TestWfMeasureSpec_RecordSectionStatesTheDecidedHeading(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfMeasureSpecFixturePath)

	record := extractMarkdownSection(body, 2, recordSectionHeading)
	if record == "" {
		t.Fatalf("wf-measure-spec must have a `## %s` section defining what a completed pass leaves behind", recordSectionHeading)
	}
	if !strings.Contains(record, recordedHeading) {
		t.Errorf("`## %s` never names %q as the heading the record is written under; a later rule locates the pass by that heading and has nothing to find without it", recordSectionHeading, recordedHeading)
	}
}

// TestWfMeasureSpec_RecordSectionNamesBothOutcomes pins that a pass which
// measured nothing and a pass that never ran stay distinguishable, which takes
// two named outcomes rather than one.
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
}
