package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// aiwfxStartEpicFixturePath is the canonical authoring location for
// the `aiwfx-start-epic` skill body — the embedded ritual snapshot
// the aiwf binary ships. Per G-0182, AC content assertions read the
// embedded bytes directly rather than a duplicated fixture under
// internal/policies/testdata/. ADR-0014 retired the marketplace
// channel; the pending ADR-0016 follow-up retires the upstream
// authoring channel — in both states, the embedded snapshot is the
// source of truth.
const aiwfxStartEpicFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md"

// loadAiwfxStartEpicFixture reads the fixture relative to repo root.
// Tests under this file assert the doctrinal content M-0096's ACs
// require, scoped to the relevant markdown section per CLAUDE.md
// §"Substring assertions are not structural assertions".
func loadAiwfxStartEpicFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxStartEpicFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxStartEpicFixturePath, err)
	}
	return string(data)
}

// TestAiwfxStartEpic_AC1_FixtureAndWorkflow pins M-0096/AC-1
// (updated by M-0104/AC-1): the fixture SKILL.md exists at the
// canonical authoring location with frontmatter declaring
// `name: aiwfx-start-epic` plus a non-empty `description:`, and the
// body contains a `## Workflow` section holding the named orchestration
// steps.
//
// M-0104 reduced the step count from 10 to 9 by merging the old
// worktree-placement (step 5) and branch-shape (step 6) Q&A steps
// into a single worktree-placement-and-branch-creation step at the
// new step 8 — the branch shape is now settled by ADR-0010 and no
// longer surfaced as a separate prompt.
//
// The 9-step count is asserted structurally — exactly the integers
// 1..9 appear as `### N.` subheadings under `## Workflow`, with no
// gaps and no extras. A flat substring search for the word "Workflow"
// would pass even if the steps were renumbered or missing; the
// numbered-heading enumeration ensures the structural promise holds.
func TestAiwfxStartEpic_AC1_FixtureAndWorkflow(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	if name := frontmatterField(body, "name"); name != "aiwfx-start-epic" {
		t.Errorf("AC-1: frontmatter `name:` must be `aiwfx-start-epic` (got %q)", name)
	}
	if desc := frontmatterField(body, "description"); desc == "" {
		t.Error("AC-1: frontmatter `description:` must be non-empty")
	}

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("AC-1: body must contain a `## Workflow` section")
	}

	stepHeading := regexp.MustCompile(`(?m)^### (\d+)\.\s`)
	matches := stepHeading.FindAllStringSubmatch(workflow, -1)
	seen := map[string]bool{}
	for _, m := range matches {
		seen[m[1]] = true
	}
	want := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	for _, n := range want {
		if !seen[n] {
			t.Errorf("AC-1: `## Workflow` must contain a `### %s.` step heading", n)
		}
	}
	if len(matches) != len(want) {
		t.Errorf("AC-1: `## Workflow` must contain exactly %d numbered step headings; got %d", len(want), len(matches))
	}

	// Belt-and-braces: assert the workflow body is non-trivial so a
	// future "shrink the fixture to just headings" regression doesn't
	// pass the structural check vacuously.
	if strings.TrimSpace(workflow) == "" {
		t.Error("AC-1: `## Workflow` section must have content beyond headings")
	}
}

// findWorktreePromptSection locates the worktree-placement prompt's
// subsection inside `## Workflow`. The locator is heading-content
// driven (not step-number driven) so a future reshuffle that moves
// the prompt to a different step number does not silently break the
// structural drift check — what matters is that the prompt exists
// under a heading naming "worktree", not which step number carries
// it.
//
// Returns the section body, or "" if no `### …worktree…` heading
// is found under `## Workflow`.
func findWorktreePromptSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "worktree") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// findSovereignPromotionSection locates the sovereign-promotion
// subsection inside `## Workflow`. The locator is heading-content
// driven (case-insensitive match on both "sovereign" and "promot")
// so a future reshuffle that moves the step to a different number
// does not silently break the structural check.
//
// Returns the section body, or "" if no matching heading is found.
func findSovereignPromotionSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		lower := strings.ToLower(text)
		if strings.Contains(lower, "sovereign") && strings.Contains(lower, "promot") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindSovereignPromotionSection_BranchCoverage covers the
// defensive return arms of findSovereignPromotionSection that the
// happy-path fixture test does not reach.
func TestFindSovereignPromotionSection_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want string
	}{
		{"missing-workflow", "prose only", ""},
		{"workflow-without-promote-heading", "## Workflow\n\n### 1. Some other step\n\nbody\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := findSovereignPromotionSection(tc.body); got != tc.want {
				t.Errorf("findSovereignPromotionSection(%q) = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestAiwfxStartEpic_AC3_SovereignPromotionStep pins M-0096/AC-3: the
// sovereign-promotion step (step 8 in E-0028's scope) names the
// `aiwf promote E-NN active` verb, references the M-0095 rule's
// substance (the `human/` actor requirement), and points at the
// `--force --reason "..."` override path. Heading-scoped per CLAUDE.md
// §"Substring assertions are not structural assertions"; the rule
// substance and override hint must live inside the promotion step,
// not float in an unrelated section.
//
// The test asserts substance, not id — "M-0095" as a literal string
// can move (an ADR might supersede the milestone's mechanical
// chokepoint, the milestone id could be reallocated); the rule's
// *content* (human-only + --force --reason override) is what readers
// land on the section to learn.
func TestAiwfxStartEpic_AC3_SovereignPromotionStep(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	section := findSovereignPromotionSection(body)
	if section == "" {
		t.Fatal("AC-3: `## Workflow` must contain a `### …sovereign…promot…` subsection that holds the activation verb")
	}

	wantContent := []struct {
		name   string
		marker string
	}{
		{"the activation verb", "aiwf promote E-NN active"},
		{"the human/ actor requirement", "human/"},
		{"the --force --reason override path", "--force --reason"},
	}
	for _, w := range wantContent {
		if !strings.Contains(section, w.marker) {
			t.Errorf("AC-3: sovereign-promotion subsection must name %s (substring %q)", w.name, w.marker)
		}
	}
}

// findSovereignAuthorizeSection locates the sovereign-authorize
// subsection inside `## Workflow`. Mirrors findSovereignPromotionSection's
// shape — heading-content driven (case-insensitive match on both
// "sovereign" and "authoriz") so a future reshuffle that moves the
// step to a different number does not silently break the structural
// check.
//
// Distinct from the sovereign-promotion locator because the heading
// for the authorize step is a peer, not a sub-step.
//
// Returns the section body, or "" if no matching heading is found.
func findSovereignAuthorizeSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		lower := strings.ToLower(text)
		if strings.Contains(lower, "sovereign") && strings.Contains(lower, "authoriz") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindSovereignAuthorizeSection_BranchCoverage covers the
// defensive return arms of findSovereignAuthorizeSection that the
// happy-path fixture test does not reach.
func TestFindSovereignAuthorizeSection_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want string
	}{
		{"missing-workflow", "prose only", ""},
		{"workflow-without-authorize-heading", "## Workflow\n\n### 1. Some other step\n\nbody\n", ""},
		{
			// Heading mentions "sovereign" but not "authoriz" — the
			// promotion step's heading; locator must not match.
			name: "only-sovereign-promotion-heading",
			body: "## Workflow\n\n### 6. Sovereign promotion\n\nbody\n",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := findSovereignAuthorizeSection(tc.body); got != tc.want {
				t.Errorf("findSovereignAuthorizeSection(%q) = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestAiwfxStartEpic_M0104_AC2_G0059Removed_ADR0010Referenced pins
// M-0104/AC-2: the stale "G-0059 frames the open question of which
// branch-model convention aiwf should bless" paragraph at the
// original step 6 is removed; the replacement names ADR-0010
// explicitly.
//
// Two-sided assertion. G-0059 absence is checked over the WHOLE
// fixture body (substring is unambiguous and the only legitimate
// reason it would re-appear is precisely the regression this test
// catches). ADR-0010 presence is asserted under `## Workflow` to
// scope it to the orchestration prose — the marker most worth pinning
// is the workflow-side commitment, not a stray frontmatter or
// constraints-section mention.
func TestAiwfxStartEpic_M0104_AC2_G0059Removed_ADR0010Referenced(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	if strings.Contains(body, "G-0059") {
		t.Error("M-0104/AC-2: fixture body must not contain `G-0059` — the deferral paragraph was retired per ADR-0010")
	}

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("M-0104/AC-2: body must contain a `## Workflow` section")
	}
	if !strings.Contains(workflow, "ADR-0010") {
		t.Error("M-0104/AC-2: `## Workflow` must reference `ADR-0010` (the branch-model decision that replaced the G-0059 deferral)")
	}
}

// TestAiwfxStartEpic_M0104_AC3_WorkflowHeadingsInNewOrder pins
// M-0104/AC-3: the workflow headings, parsed structurally, appear
// in the new order — preflight → delegation prompt → sovereign
// promote → sovereign authorize → worktree placement → hand-off
// (with the 4 preflight items unfolded as steps 1..4 each).
//
// Heading-content driven per CLAUDE.md §"Substring assertions are
// not structural assertions": each expected step asserts that the
// i-th `### N.` heading under `## Workflow` contains a distinctive
// lowercase token. The order is what's pinned; the exact wording
// is allowed to evolve so long as the conceptual sequence holds.
//
// A regression that reorders steps (e.g., moves "worktree" before
// "sovereign promote", regressing the M-0103-driven sequencing
// invariant) fires this test on the misplaced step's token mismatch.
func TestAiwfxStartEpic_M0104_AC3_WorkflowHeadingsInNewOrder(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("M-0104/AC-3: body must contain a `## Workflow` section")
	}

	// Extract the ordered list of `### N. <heading text>` headings.
	stepHeading := regexp.MustCompile(`(?m)^### \d+\.\s+(.+)$`)
	matches := stepHeading.FindAllStringSubmatch(workflow, -1)
	gotHeadings := make([]string, 0, len(matches))
	for _, m := range matches {
		gotHeadings = append(gotHeadings, strings.ToLower(strings.TrimSpace(m[1])))
	}

	// The expected ordering — distinctive tokens per step. Pinning
	// the token rather than the full heading text leaves room for
	// small wording polish without test churn while still catching
	// any reorder.
	wantOrderTokens := []string{
		"preflight",        // step 1
		"drafted-milestone", // step 2
		"aiwf check",       // step 3
		"tests/build",      // step 4
		"delegation",       // step 5 — new (was step 7)
		"sovereign promot", // step 6 — was step 8
		"sovereign authoriz", // step 7 — was step 9
		"worktree",         // step 8 — was step 5 (now merged with branch)
		"hand-off",         // step 9 — was step 10
	}
	if len(gotHeadings) != len(wantOrderTokens) {
		t.Fatalf("M-0104/AC-3: expected %d workflow steps in the new ordering; got %d (headings: %q)",
			len(wantOrderTokens), len(gotHeadings), gotHeadings)
	}
	for i, tok := range wantOrderTokens {
		if !strings.Contains(gotHeadings[i], tok) {
			t.Errorf("M-0104/AC-3: step %d heading %q does not contain expected token %q (full ordering: %q)",
				i+1, gotHeadings[i], tok, gotHeadings)
		}
	}
}

// TestAiwfxStartEpic_M0104_AC5_SovereignAuthorizeStepNamesOverride
// pins M-0104/AC-5: the new sovereign-authorize step (the one
// introduced by ADR-0010's sequencing) names `--force --reason` as
// the override path. The promotion step (step 6) already pins this
// via TestAiwfxStartEpic_AC3_SovereignPromotionStep; this test pins
// the same discipline on the authorize step (step 7) — both are
// sovereign acts on `main`, both must surface the override.
//
// Heading-scoped per CLAUDE.md §"Substring assertions are not
// structural assertions": the override hint must live INSIDE the
// authorize step, not float somewhere else in the body where a
// reader looking at step 7 in isolation would miss it.
//
// The section must also name the M-0104/AC-4 carve-out's two
// preconditions — main + ritual `--branch` — so a reader who hits
// step 7 understands why the verb does not refuse despite the
// future-branch shape.
func TestAiwfxStartEpic_M0104_AC5_SovereignAuthorizeStepNamesOverride(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	section := findSovereignAuthorizeSection(body)
	if section == "" {
		t.Fatal("M-0104/AC-5: `## Workflow` must contain a `### …sovereign…authoriz…` subsection that holds the delegation verb (step 7)")
	}

	wantContent := []struct {
		name   string
		marker string
	}{
		{"the delegation verb", "aiwf authorize"},
		{"--force --reason override path", "--force --reason"},
		{"--branch flag (the future-binding the carve-out permits)", "--branch"},
		{"main checkout precondition (operator on main)", "main"},
	}
	for _, w := range wantContent {
		if !strings.Contains(section, w.marker) {
			t.Errorf("M-0104/AC-5: sovereign-authorize subsection must name %s (substring %q)", w.name, w.marker)
		}
	}
}

// TestFindWorktreePromptSection_BranchCoverage exercises the
// defensive return arms of findWorktreePromptSection that the
// happy-path fixture test does not reach (missing `## Workflow`,
// `## Workflow` present but no `### …worktree…` heading). Cheap
// insurance per CLAUDE.md §"Test untested code paths before
// declaring code paths done" — every reachable branch has a test.
func TestFindWorktreePromptSection_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing-workflow",
			body: "no headings here, just prose",
			want: "",
		},
		{
			name: "workflow-without-worktree-heading",
			body: "## Workflow\n\n### 1. Some other step\n\nbody\n",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := findWorktreePromptSection(tc.body); got != tc.want {
				t.Errorf("findWorktreePromptSection(%q) = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestAiwfxStartEpic_AC2_WorktreePromptOptions pins M-0096/AC-2: the
// worktree-placement prompt is a heading-scoped Q&A with three named
// options — *no worktree (work on main)*, `.claude/worktrees/<branch>/`,
// and `../aiwf-<branch>/`. The assertion is heading-scoped (not flat
// substring) per CLAUDE.md §"Substring assertions are not structural
// assertions"; the literal path strings could plausibly appear in
// unrelated sections (e.g. an anti-pattern example) so the locator
// scopes the claim to the prompt's own subsection.
//
// The three option markers are chosen so they:
//   - cannot all appear unintentionally in a non-prompt section
//     (the three together carry the prompt's signature);
//   - tolerate small wording variations in the surrounding prose
//     (each marker is a path literal, not a sentence fragment).
func TestAiwfxStartEpic_AC2_WorktreePromptOptions(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	section := findWorktreePromptSection(body)
	if section == "" {
		t.Fatal("AC-2: `## Workflow` must contain a `### …worktree…` subsection that holds the placement Q&A")
	}

	// The three named placements per E-0028's scope. Each marker is
	// a path-shaped or doctrinal literal that disambiguates the option
	// from prose elsewhere in the skill. Prose markers ("no worktree")
	// match case-insensitively so a Title-Case bullet still hits;
	// path markers (`.claude/worktrees/`, `../aiwf-`) match
	// case-sensitively because the path strings are not free prose.
	wantOptions := []struct {
		name     string
		marker   string
		caseFold bool
	}{
		{"no worktree (work on main)", "no worktree", true},
		{".claude/worktrees/<branch>/", ".claude/worktrees/", false},
		{"../aiwf-<branch>/", "../aiwf-", false},
	}
	for _, opt := range wantOptions {
		hay := section
		needle := opt.marker
		if opt.caseFold {
			hay = strings.ToLower(hay)
			needle = strings.ToLower(needle)
		}
		if !strings.Contains(hay, needle) {
			t.Errorf("AC-2: worktree-prompt subsection must name the %s option (marker substring %q)", opt.name, opt.marker)
		}
	}
}
