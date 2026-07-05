package policies

import (
	"strings"
	"testing"
)

// wf_patch_changelog_test.go pins G-0365: wf-patch's wrap sequence must
// add a CHANGELOG.md entry for every patch, mandatory even for
// internal-only changes, since a patch (unlike a milestone) has no
// parent epic to roll its change into later — its own wrap is the
// only chance the change is ever recorded. Modeled on
// aiwfx-wrap-epic's own CHANGELOG step (its own `### ` heading; see
// aiwfx_wrap_epic_test.go).

// TestWfPatch_ChangelogEntryStep asserts the `### 4. Add a CHANGELOG
// entry` step exists inside `## Workflow`, positioned between
// "Implement the change" (step 3) and "Verify locally" (step 5), and
// documents: the `[Unreleased]` target section, the three
// Keep-a-Changelog category headings, and the always-required (no
// skip) mandate with its internal-only minimal-form allowance. Per
// CLAUDE.md *Substring assertions are not structural assertions*,
// content checks are scoped to the step's own section.
func TestWfPatch_ChangelogEntryStep(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("wf-patch must have a `## Workflow` section")
	}

	implementIdx, changelogIdx, verifyIdx := -1, -1, -1
	for i, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.ToLower(strings.TrimPrefix(line, "### "))
		switch {
		case implementIdx < 0 && strings.Contains(text, "implement the change"):
			implementIdx = i
		case changelogIdx < 0 && strings.Contains(text, "changelog"):
			changelogIdx = i
		case verifyIdx < 0 && strings.Contains(text, "verify locally"):
			verifyIdx = i
		}
	}
	if implementIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Implement the change…` step")
	}
	if changelogIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …CHANGELOG…` step")
	}
	if verifyIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Verify locally…` step")
	}
	if implementIdx >= changelogIdx || changelogIdx >= verifyIdx {
		t.Errorf("the CHANGELOG step must sit between implement and verify-locally (implement=%d, changelog=%d, verify=%d)", implementIdx, changelogIdx, verifyIdx)
	}

	changelog := extractMarkdownSection(body, 3, "4. Add a CHANGELOG entry")
	if changelog == "" {
		t.Fatal("could not extract the `### 4. Add a CHANGELOG entry` section")
	}

	wantPresent := []string{
		"[Unreleased]",
		"### Added — G-NNNN",
		"### Changed — G-NNNN",
		"### Fixed — G-NNNN",
		"user-visible delta",
	}
	for _, w := range wantPresent {
		if !strings.Contains(changelog, w) {
			t.Errorf("CHANGELOG-entry step must mention %q", w)
		}
	}

	if !strings.Contains(changelog, "no skip") {
		t.Error("CHANGELOG-entry step must state the step always runs — no skip")
	}
	if !strings.Contains(strings.ToLower(changelog), "internal-only") {
		t.Error("CHANGELOG-entry step must document the internal-only minimal one-line-form allowance")
	}
	if !strings.Contains(changelog, "has no parent to roll up into") {
		t.Error("CHANGELOG-entry step must explain why the step is mandatory: a patch has no parent epic to roll its change into")
	}
}

// TestWfPatch_ChangelogEntryAntiPatternAndConstraint asserts the
// mandate is reinforced outside the step itself: an anti-pattern
// bullet names the "it's internal, skip the entry" temptation, and
// the Constraints section states the rule as a standing constraint,
// not just a one-time procedural step — matching how the skill
// already reinforces its other gates (commit/wrap/push) in both
// places.
func TestWfPatch_ChangelogEntryAntiPatternAndConstraint(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	antiPatterns := extractMarkdownSection(body, 2, "Anti-patterns")
	if antiPatterns == "" {
		t.Fatal("wf-patch must have an `## Anti-patterns` section")
	}
	if !strings.Contains(strings.ToLower(antiPatterns), "changelog") {
		t.Error("`## Anti-patterns` must name the temptation to skip the CHANGELOG entry")
	}

	constraints := extractMarkdownSection(body, 2, "Constraints")
	if constraints == "" {
		t.Fatal("wf-patch must have a `## Constraints` section")
	}
	if !strings.Contains(constraints, "CHANGELOG.md") {
		t.Error("`## Constraints` must state the mandatory CHANGELOG.md entry rule")
	}
	if !strings.Contains(constraints, "[Unreleased]") {
		t.Error("`## Constraints` must name `[Unreleased]` as the entry's target section")
	}
}
