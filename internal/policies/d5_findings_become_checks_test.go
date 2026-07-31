package policies

import (
	"strings"
	"testing"
)

// Structural tests for the D5 "findings become checks" force (G-0489) and the
// two surfaces that carry it: the wf-codebase-health rubric where the force is
// stated canonically, and the wf-review-code verdict block that applies it at
// the disposition point. The always-on guidance priming subset is pinned too,
// mirroring how the H1 reuse force is pinned by its sibling economy test.
//
// Path literals are the shared consts declared by sibling policy tests
// (wfCodebaseHealthFixturePath, wfReviewCodeFixturePath, g0343GuidanceFixturePath)
// rather than redeclared here; those declarations are also what discharge the
// skill-edit-structural-test-backstop for these two SKILL.md edits.
//
// Two matching disciplines run here, and mixing them up produces a test that
// looks strict and pins nothing:
//
//   - Prose assertions run against whitespace-flattened, emphasis-stripped text,
//     so they pin the prescription rather than the author's line wrapping or
//     bold placement.
//   - Assertions about a paragraph's *label* run against the raw section and
//     require the label to start a line. Flattening erases the paragraph
//     boundary such a claim rests on, so a flattened label check passes even
//     when the label has been demoted into a mid-sentence aside.
//
// Every assertion is scoped to a named markdown section, never a flat body
// grep, per CLAUDE.md §"Substring assertions are not structural assertions".

// TestWfCodebaseHealth_SectionDHasFiveForces pins that section D carries the
// D5 force alongside the original four. A D5 dropped or renumbered out of the
// section fails here rather than silently leaving the rubric's other surfaces
// citing a force that no longer exists.
func TestWfCodebaseHealth_SectionDHasFiveForces(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfCodebaseHealthFixturePath)

	sec := extractMarkdownSection(body, 2, "D. Tests that pin behavior")
	if sec == "" {
		t.Fatal("wf-codebase-health must have a `## D. Tests that pin behavior…` section")
	}
	if got := countSubHeadings(sec, 3); got != 5 {
		t.Errorf("section D has %d `###` forces; want exactly 5 (D1-D4 plus D5 findings-become-checks)", got)
	}
	if !strings.Contains(sec, "D5. Findings become checks") {
		t.Error("section D must carry a `### D5. Findings become checks` force")
	}
}

// TestWfCodebaseHealth_D5RequiresAPinningCheck pins the ratchet itself: a
// confirmed defect leaves a check that fails without the fix. Without this
// clause the force degrades into "fix it", which is the pre-D5 state the whole
// rule exists to replace.
func TestWfCodebaseHealth_D5RequiresAPinningCheck(t *testing.T) {
	t.Parallel()
	d5 := readD5Force(t)

	if !strings.Contains(d5, "fails without the fix") {
		t.Error("D5 must require a check that fails without the fix — the standard that makes a fix a ratchet")
	}
	if !strings.Contains(d5, "silent correction") {
		t.Error("D5 must name the silent correction as what a disposed finding must never be")
	}
}

// TestWfCodebaseHealth_D5NamesTheUnpinnableEscape pins the escape hatch. A
// ratchet with no stated escape is unworkable for defects that genuinely
// cannot be pinned, and an unworkable rule is ignored wholesale rather than
// selectively. Asserted as the normative phrasing rather than the two nouns
// alone, which the Moves bullet would also satisfy.
func TestWfCodebaseHealth_D5NamesTheUnpinnableEscape(t *testing.T) {
	t.Parallel()
	d5 := readD5Force(t)

	if !strings.Contains(strings.ToLower(d5), "becomes a recorded decision or a tracked issue") {
		t.Error("D5 must state that an unpinnable defect becomes a recorded decision or a tracked issue")
	}
}

// TestWfCodebaseHealth_D5SeparatesDefectFromJudgment pins the classification
// that makes the other two clauses operable: only an objective defect can be
// encoded, and a judgment disagreement must not recur as free-floating opinion.
func TestWfCodebaseHealth_D5SeparatesDefectFromJudgment(t *testing.T) {
	t.Parallel()
	d5 := strings.ToLower(readD5Force(t))

	for phrase, why := range map[string]string{
		"objective defect":      "D5 must name the objective-defect class that goes to the oracle",
		"judgment disagreement": "D5 must name the judgment-disagreement class that cannot be encoded as it stands",
		"fresh opinion":         "D5 must forbid a disposed judgment finding returning as a fresh opinion",
	} {
		if !strings.Contains(d5, phrase) {
			t.Error(why)
		}
	}
}

// TestWfCodebaseHealth_D5CarriesLabelledStopRule pins the stop rule as its own
// labelled paragraph inside D5 — the shape chosen so calling surfaces can cite
// the terminator precisely without it becoming separately adoptable. The label
// is checked against the raw section at line start: a flattened check would
// still pass with the label demoted into a mid-sentence aside, which is exactly
// the shape the "labelled paragraph" claim rules out.
func TestWfCodebaseHealth_D5CarriesLabelledStopRule(t *testing.T) {
	t.Parallel()

	if !hasLineStartingWith(readD5ForceRaw(t), "**Stop rule.**") {
		t.Fatal("D5 must open a paragraph with the `**Stop rule.**` label, so callers can cite it precisely")
	}

	d5 := strings.ToLower(readD5Force(t))
	if !strings.Contains(d5, "whole surface") {
		t.Error("the stop rule must require a whole-surface pass, not a re-scan narrowed to what changed")
	}
	if !strings.Contains(d5, `"no findings ever"`) {
		t.Error("the stop rule must reject the zero-findings reading — judgment findings are unbounded")
	}
	if !strings.Contains(d5, "already fixed, pinned, or tracked") {
		t.Error("the stop rule must bound its defect set by disposition, or a deliberately deferred defect blocks convergence forever")
	}
}

// TestWfReviewCode_VerdictClassifiesOnTwoAxes pins the verdict block's kind
// axis alongside the urgency axis it already had. Urgency alone cannot route a
// finding: it says when to act, never whether the claim can be encoded at all.
func TestWfReviewCode_VerdictClassifiesOnTwoAxes(t *testing.T) {
	t.Parallel()
	verdict := readReviewVerdict(t)

	for _, want := range []string{"Defect", "Judgment", "Blocking", "Track for later"} {
		if !strings.Contains(verdict, want) {
			t.Errorf("the verdict block must classify findings with a %q category", want)
		}
	}
	if !strings.Contains(strings.ToLower(verdict), "two axes") {
		t.Error("the verdict block must state that findings carry two classification axes")
	}
}

// TestWfReviewCode_VerdictRequiresPinningAndCitesD5 pins the disposition rule
// at the point findings are actually disposed of, and pins the citation so the
// review skill defers to the canonical statement instead of restating a
// wording that can drift away from it. The citation is asserted as the full
// skill-plus-force composite: a bare "D5" is a two-character grep an unrelated
// token would satisfy.
func TestWfReviewCode_VerdictRequiresPinningAndCitesD5(t *testing.T) {
	t.Parallel()
	verdict := readReviewVerdict(t)

	if !strings.Contains(verdict, "fails without it") {
		t.Error("the verdict block must require a blocking defect's fix to carry a check that fails without it")
	}
	if !strings.Contains(verdict, "`wf-codebase-health` D5") {
		t.Error("the verdict block must cite wf-codebase-health D5 by name rather than restate the rule independently")
	}
}

// TestWfReviewCode_VerdictCarriesFullSurfaceLoopTerminator pins that a verdict
// closes one pass rather than the review, and that the deciding pass is
// full-surface. A terminator evaluated only against a narrowing re-scan
// certifies a slice, which reads as convergence while the rest goes unsampled.
// Label checked raw, for the reason given on the stop-rule test.
func TestWfReviewCode_VerdictCarriesFullSurfaceLoopTerminator(t *testing.T) {
	t.Parallel()

	if !hasLineStartingWith(readReviewVerdictRaw(t), "**When the loop ends.**") {
		t.Fatal("the verdict block must open a paragraph stating when the review loop ends, not only how one pass is scored")
	}

	verdict := strings.ToLower(readReviewVerdict(t))
	if !strings.Contains(verdict, "whole surface") {
		t.Error("the loop terminator must require a whole-surface deciding pass")
	}
	if !strings.Contains(verdict, "stop rule") {
		t.Error("the loop terminator must route the stop/continue decision through D5's stop rule")
	}
}

// TestWfReviewCode_OutputFormatCarriesKindAndDisposition pins the template a
// reviewer actually emits. The verdict prose can explain the two axes perfectly
// while the template collects neither — and the template is the mechanism by
// which the classification binds to a real report, so it needs its own check.
func TestWfReviewCode_OutputFormatCarriesKindAndDisposition(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfReviewCodeFixturePath)

	format := extractMarkdownSection(body, 2, "Output format")
	if format == "" {
		t.Fatal("wf-review-code must have an `## Output format` section")
	}
	for _, want := range []string{"[defect]", "[judgment]", "· pin:", "· dispose:"} {
		if !strings.Contains(format, want) {
			t.Errorf("the output template must carry %q, or the two-axis classification never reaches the emitted report", want)
		}
	}
}

// TestWfReviewCode_ConstraintBindsKindToDisposition pins the 🛑 constraint that
// makes the classification load-bearing rather than descriptive, including its
// escape hatch. A constraint stating the pin absolutely would contradict the
// body's own carve-out for a defect that cannot be pinned, and the 🛑 block is
// the part a reader weights hardest.
func TestWfReviewCode_ConstraintBindsKindToDisposition(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfReviewCodeFixturePath)

	section := extractMarkdownSection(body, 2, "Constraints")
	if section == "" {
		t.Fatal("wf-review-code must have a `## Constraints` section")
	}
	flat := strings.ToLower(flattenMarkdownProse(section))
	if !strings.Contains(flat, "every finding carries its kind") {
		t.Error("the constraints must require every finding to carry its kind — defect or judgment")
	}
	if !strings.Contains(flat, "the decision not to pin it is recorded") {
		t.Error("the kind constraint must preserve the unpinnable-defect escape the body grants, or it contradicts its own file")
	}
}

// TestEmbeddedGuidance_PrimingCarriesFindingsBecomeChecks pins that the
// always-on priming subset carries D5, so the ratchet is primed every turn
// rather than only reachable by opening the full rubric. Mirrors the sibling
// assertion that pins the H1 reuse force into the same subset.
func TestEmbeddedGuidance_PrimingCarriesFindingsBecomeChecks(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, g0343GuidanceFixturePath)

	priming := extractMarkdownSection(body, 2, "Code-health priming")
	if priming == "" {
		t.Fatal("aiwf-guidance.md must have a `## Code-health priming` section")
	}
	low := strings.ToLower(flattenMarkdownProse(priming))
	if !strings.Contains(low, "findings become checks") {
		t.Error("the always-on priming subset must carry the findings-become-checks force")
	}
	if !strings.Contains(low, "fails without the fix") {
		t.Error("the primed D5 line must carry the pinning standard, not only the force's name")
	}
}

// readD5ForceRaw returns the `### D5.` force with its line structure intact,
// for assertions about paragraph labels.
func readD5ForceRaw(t *testing.T) string {
	t.Helper()
	body := readVerbSkill(t, wfCodebaseHealthFixturePath)
	d5 := extractMarkdownSection(body, 3, "D5.")
	if d5 == "" {
		t.Fatal("wf-codebase-health must have a `### D5. …` findings-become-checks force")
	}
	return d5
}

// readD5Force returns the D5 force normalized for prose matching.
func readD5Force(t *testing.T) string {
	t.Helper()
	return normalizeProse(readD5ForceRaw(t))
}

// readReviewVerdictRaw returns wf-review-code's verdict step with its line
// structure intact, for assertions about paragraph labels.
func readReviewVerdictRaw(t *testing.T) string {
	t.Helper()
	body := readVerbSkill(t, wfReviewCodeFixturePath)
	verdict := extractMarkdownSection(body, 3, "8. Verdict")
	if verdict == "" {
		t.Fatal("wf-review-code must have a `### 8. Verdict` step")
	}
	return verdict
}

// readReviewVerdict returns the verdict step normalized for prose matching.
func readReviewVerdict(t *testing.T) string {
	t.Helper()
	return normalizeProse(readReviewVerdictRaw(t))
}

// normalizeProse prepares markdown for a phrase assertion that should pin a
// prescription rather than its typography: whitespace collapses so a phrase
// matches across a hard-wrapped line, and emphasis markers drop so bold
// placement inside a phrase is not load-bearing. Reflowing a paragraph or
// moving a `**` should never red a test that has nothing to say about either.
func normalizeProse(s string) string {
	return flattenMarkdownProse(strings.ReplaceAll(s, "*", ""))
}

// flattenMarkdownProse collapses every run of whitespace to a single space, so
// a multi-word phrase assertion matches across a hard-wrapped source line.
func flattenMarkdownProse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// hasLineStartingWith reports whether any line of raw markdown begins with
// prefix, ignoring leading indentation. Used for paragraph-label assertions,
// where "the label opens a line" is the property under test.
func hasLineStartingWith(raw, prefix string) bool {
	for line := range strings.SplitSeq(raw, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}
