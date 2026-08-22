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
