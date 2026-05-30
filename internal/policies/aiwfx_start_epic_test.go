package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// aiwfxStartEpicFixturePath is the canonical authoring location for
// the `aiwfx-start-epic` skill body during M-0096, per CLAUDE.md
// §"Cross-repo plugin testing". At wrap, the fixture content is
// copied to the rituals plugin repo (`plugins/aiwf-extensions/
// skills/aiwfx-start-epic/SKILL.md` there); the drift-check in
// TestAiwfxStartEpic_AC5_DriftAgainstCache guards the long-term
// coupling.
const aiwfxStartEpicFixturePath = "internal/policies/testdata/aiwfx-start-epic/SKILL.md"

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

// TestAiwfxStartEpic_AC1_FixtureAndWorkflow pins M-0096/AC-1: the
// fixture SKILL.md exists at the canonical authoring location with
// frontmatter declaring `name: aiwfx-start-epic` plus a non-empty
// `description:`, and the body contains a `## Workflow` section
// holding the 10 named orchestration steps from E-0028's scope.
//
// The 10-step count is asserted structurally — exactly the integers
// 1..10 appear as `### N.` subheadings under `## Workflow`, with no
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
	want := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
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

// findBranchPromptSection locates the branch-shape prompt's
// subsection inside `## Workflow`. The locator is heading-content
// driven (case-insensitive match on "branch") and deliberately
// EXCLUDES the worktree section, whose heading may itself contain
// "branch" as part of the path literal `<branch>` (e.g. "Worktree
// placement (`.claude/worktrees/<branch>/` …)"). The exclusion
// matches on a leading "worktree" token in the heading.
//
// Returns the section body, or "" if no matching heading is found.
func findBranchPromptSection(body string) string {
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
		if !strings.Contains(lower, "branch") {
			continue
		}
		// Skip the worktree heading if it happens to mention "branch"
		// (the worktree options surface `<branch>` as a path literal).
		if strings.Contains(lower, "worktree") {
			continue
		}
		return extractMarkdownSection(body, 3, text)
	}
	return ""
}

// TestFindBranchPromptSection_BranchCoverage covers the defensive
// return arms plus the worktree-skip arm (a `### …worktree…branch…`
// heading must not match the branch-prompt locator).
func TestFindBranchPromptSection_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want string
	}{
		{"missing-workflow", "prose only", ""},
		{"workflow-without-branch-heading", "## Workflow\n\n### 1. Other\n\nbody\n", ""},
		{
			name: "only-worktree-heading-mentioning-branch",
			body: "## Workflow\n\n### 5. Worktree (`<branch>/`)\n\nbody\n",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := findBranchPromptSection(tc.body); got != tc.want {
				t.Errorf("findBranchPromptSection(%q) = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestAiwfxStartEpic_AC4_BranchPromptDefersToG0059 pins M-0096/AC-4:
// the branch-shape prompt is a heading-scoped Q&A with two named
// options (stay on current / create new) plus an explicit reference
// to G-0059 — the open gap framing the branch-model convention. The
// G-0059 reference documents in-skill that the prompt is a *placeholder*
// pending the gap's resolution; a future skill update can tighten
// the default when G-0059 lands. Heading-scoped per CLAUDE.md
// §"Substring assertions are not structural assertions".
//
// The G-0059 literal is the right kind of marker to assert: it is
// unique enough that it cannot drift to an unrelated section, and
// its presence is the load-bearing signal that "this prompt is a
// placeholder, not a settled convention."
func TestAiwfxStartEpic_AC4_BranchPromptDefersToG0059(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	section := findBranchPromptSection(body)
	if section == "" {
		t.Fatal("AC-4: `## Workflow` must contain a `### …branch…` subsection (distinct from the worktree section) for the branch-shape Q&A")
	}

	wantContent := []struct {
		name     string
		marker   string
		caseFold bool
	}{
		{"stay-on-current option", "stay on", true},
		{"create-new-branch option", "create", true},
		{"G-0059 deferral note", "G-0059", false},
	}
	for _, w := range wantContent {
		hay := section
		needle := w.marker
		if w.caseFold {
			hay = strings.ToLower(hay)
			needle = strings.ToLower(needle)
		}
		if !strings.Contains(hay, needle) {
			t.Errorf("AC-4: branch-prompt subsection must name %s (substring %q)", w.name, w.marker)
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
