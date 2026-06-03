package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// aiwfx_wrap_milestone_test.go — M-0160 wrap cycle, G-0219 + G-0220
// follow-up.
//
// Structural drift-check pins for the aiwfx-wrap-milestone SKILL.md
// step 11 ("After merge") trailer prescription. Mirrors
// internal/policies/aiwfx_wrap_epic_test.go's
// TestAiwfxWrapEpic_AC6_StructuralMergeStepDriftCheck pattern: walks
// the markdown heading hierarchy to locate the merge-step section,
// then asserts the trailer prescription appears INSIDE that section
// (not just somewhere in the file). Per CLAUDE.md §"Substring
// assertions are not structural assertions".
//
// The prescription itself landed at commit 5cf007f5 during M-0160
// wrap; this test is the mechanical backstop that prevents silent
// regression of the new content. Without this test, a future SKILL.md
// edit could drop the wrap-milestone trailer prescription and CI
// wouldn't notice — exactly the discipline gap G-0220 names.
//
// Discovered via G-0219 (the prescription asymmetry between
// aiwfx-wrap-milestone and aiwfx-wrap-epic) + G-0220 (ritual SKILL.md
// edits without structural AC pins have no mechanical backstop).

// aiwfxWrapMilestoneFixturePath is the canonical authoring location
// for the `aiwfx-wrap-milestone` skill body — the embedded ritual
// snapshot the aiwf binary ships. Per G-0182, AC content assertions
// read the embedded bytes directly rather than a duplicated fixture
// under internal/policies/testdata/.
const aiwfxWrapMilestoneFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md"

// loadAiwfxWrapMilestoneFixture reads the fixture relative to repo
// root. The tests under this file are seam-tests against the
// authored skill body — they assert the doctrinal content G-0219's
// fix established, scoped to the relevant markdown section per
// CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions".
func loadAiwfxWrapMilestoneFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxWrapMilestoneFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxWrapMilestoneFixturePath, err)
	}
	return string(data)
}

// TestAiwfxWrapMilestone_StructuralMergeStepDriftCheck pins the
// G-0219 fix structurally: the trailered-merge instructions live
// inside the merge-step section ("After merge"), not floating
// elsewhere. A reshuffle of the SKILL.md that moved the trailered
// commit into an unrelated section would fail this test even if a
// flat grep over the file still passed.
//
// Concretely: locate the merge-step subsection inside `## Workflow`
// (heading text references the merge action), then assert each
// required trailer flag appears *inside that subsection*. The
// trailer keys are quoted from CLAUDE.md §"Commit conventions"
// verbatim — variant casings (e.g., `Aiwf-Verb`) would fail the
// kernel's trailer-keys policy and must fail this test too.
//
// Mirrors TestAiwfxWrapEpic_AC6_StructuralMergeStepDriftCheck for
// the wrap-milestone surface; the two together pin the
// wrap-{epic,milestone} ritual symmetry G-0219 established.
func TestAiwfxWrapMilestone_StructuralMergeStepDriftCheck(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapMilestoneFixture(t)

	// Step 1: the parent section exists.
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("SKILL.md must have a `## Workflow` section that contains the merge step as a subsection")
	}

	// Step 2: the merge-step subsection is reachable via a `### `
	// heading whose text references the merge action.
	merge := findWrapMilestoneMergeStepSection(body)
	if merge == "" {
		t.Fatal("`## Workflow` must contain a `### …merge…` subsection that documents the epic-integration merge")
	}

	// Step 3: each required trailer flag is named *inside* the
	// merge-step subsection. The keys are quoted from CLAUDE.md
	// §"Commit conventions" verbatim — variant casings would fail
	// the kernel's trailer-keys policy and must fail this test too.
	requiredTrailerFlags := []string{
		`--trailer "aiwf-verb: wrap-milestone"`,
		`--trailer "aiwf-entity: M-NNNN"`,
		`--trailer "aiwf-actor: human/`,
	}
	for _, flag := range requiredTrailerFlags {
		if !strings.Contains(merge, flag) {
			t.Errorf("merge-step subsection must name the trailer flag %q (in the right section, not just somewhere in the file)", flag)
		}
	}

	// Step 4: the trailered commit *follows* the staged merge.
	// Ordering matters — a fixture that documented the trailer
	// flags first and then a plain `git merge --no-ff` would
	// produce an untrailered commit at run time. Assert by index.
	stageIdx := strings.Index(merge, "git merge --no-ff --no-commit")
	commitIdx := strings.Index(merge, `--trailer "aiwf-verb: wrap-milestone"`)
	if stageIdx < 0 || commitIdx < 0 {
		t.Fatal("merge-step subsection must contain both the staged-merge command and the trailer-emitting commit")
	}
	if stageIdx > commitIdx {
		t.Error("the staged-merge (`--no-commit`) must appear *before* the trailered `git commit` so the commit-emitting step is the one carrying trailers")
	}

	// Step 5: Conventional Commits-shaped subject for the wrap
	// commit. The skill instruction's subject template
	// (chore(milestone): wrap M-NNNN — <title>) is what CLAUDE.md
	// §"Commit conventions" requires; assert by the leading shape.
	if !regexp.MustCompile(`chore\(milestone\):\s+wrap\s+M-NNNN`).MatchString(merge) {
		t.Error("merge-step subsection must use a Conventional Commits subject template `chore(milestone): wrap M-NNNN — <title>`")
	}
}

// findWrapMilestoneMergeStepSection returns the body of the
// merge-step subsection inside `## Workflow`. The subsection's
// `### ` heading is identified by a case-insensitive substring match
// on "merge". Returns "" if no matching subsection is found.
//
// Distinct from aiwfx_wrap_epic_test.go's findMergeStepSection
// (which matches on "merge" AND "epic branch", scoped to the
// epic-to-trunk merge step) because the wrap-milestone skill's
// merge step is the milestone-to-epic merge — its heading text is
// "After merge" rather than mentioning "epic branch" explicitly.
// A single-substring "merge" match is sufficient because the
// wrap-milestone skill's only `###` heading containing "merge" IS
// the relevant one.
func findWrapMilestoneMergeStepSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	lines := strings.Split(workflow, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "merge") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}
