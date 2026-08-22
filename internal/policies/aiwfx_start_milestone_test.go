package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// aiwfxStartMilestoneFixturePath is the canonical authoring location
// for the `aiwfx-start-milestone` skill body — the embedded ritual
// snapshot the aiwf binary ships. Per G-0182 (same pattern as
// aiwfx-start-epic), AC content assertions read the embedded bytes
// directly rather than a duplicated fixture under
// internal/policies/testdata/. ADR-0014 retired the marketplace
// channel; ADR-0016 retired the upstream authoring channel — the
// embedded snapshot is the source of truth.
const aiwfxStartMilestoneFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md"

// loadAiwfxStartMilestoneFixture reads the fixture relative to repo
// root. Tests under this file assert M-0105's AC content claims,
// scoped to the relevant markdown section per CLAUDE.md
// §"Substring assertions are not structural assertions".
func loadAiwfxStartMilestoneFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxStartMilestoneFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxStartMilestoneFixturePath, err)
	}
	return string(data)
}

// findStartMilestonePreflightSection locates the `### 1. Preflight`
// subsection inside `## Workflow`. Heading-content driven (case-
// insensitive match on "preflight") so a future reshuffle that
// moves the step to a different number does not silently break the
// structural check.
func findStartMilestonePreflightSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "preflight") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindStartMilestonePreflightSection_BranchCoverage covers the
// defensive return arms of findStartMilestonePreflightSection that
// the happy-path fixture test does not reach.
func TestFindStartMilestonePreflightSection_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want string
	}{
		{"missing-workflow", "prose only", ""},
		{"workflow-without-preflight-heading", "## Workflow\n\n### 1. Some other step\n\nbody\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := findStartMilestonePreflightSection(tc.body); got != tc.want {
				t.Errorf("findStartMilestonePreflightSection(%q) = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}

// findStartMilestoneAuthorizeSection locates the sovereign-authorize
// subsection inside `## Workflow` (the new step 4 added by M-0105).
// Heading-content driven on "sovereign" + "authoriz".
func findStartMilestoneAuthorizeSection(body string) string {
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

// TestFindStartMilestoneAuthorizeSection_BranchCoverage covers the
// defensive return arms.
func TestFindStartMilestoneAuthorizeSection_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want string
	}{
		{"missing-workflow", "prose only", ""},
		{"workflow-without-authorize-heading", "## Workflow\n\n### 1. Other\n\nbody\n", ""},
		{
			// Heading mentions "sovereign" but not "authoriz" — the
			// promote step.
			name: "only-sovereign-promote-heading",
			body: "## Workflow\n\n### 3. Sovereign promote on parent epic branch\n\nbody\n",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := findStartMilestoneAuthorizeSection(tc.body); got != tc.want {
				t.Errorf("findStartMilestoneAuthorizeSection(%q) = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestAiwfxStartMilestone_M0105_AC1_FixtureAndWorkflow pins
// M-0105/AC-1: the fixture SKILL.md exists at the canonical
// authoring location with frontmatter declaring
// `name: aiwfx-start-milestone` plus a non-empty `description:`,
// and the body contains a `## Workflow` section holding exactly 8
// named orchestration steps.
//
// M-0105 reshaped the workflow from 6 steps to 8: the old steps 1
// (preflight) + 2 (promote) + 3 (branch setup) + 4 (implementation)
// + 5 (self-review) + 6 (hand off) become 1 (preflight, tightened)
// + 2 (delegation prompt, new) + 3 (sovereign promote on parent) +
// 4 (sovereign authorize on parent, new, only if delegating) + 5
// (cut milestone branch) + 6 (implementation) + 7 (readiness check,
// reframed from "self-review" per G-0271) + 8 (hand off). The
// sequencing implements ADR-0010.
//
// The 8-step count is asserted structurally — exactly the integers
// 1..8 appear as `### N.` subheadings under `## Workflow`, with no
// gaps and no extras.
func TestAiwfxStartMilestone_M0105_AC1_FixtureAndWorkflow(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	if name := frontmatterField(body, "name"); name != "aiwfx-start-milestone" {
		t.Errorf("AC-1: frontmatter `name:` must be `aiwfx-start-milestone` (got %q)", name)
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
	want := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	for _, n := range want {
		if !seen[n] {
			t.Errorf("AC-1: `## Workflow` must contain a `### %s.` step heading", n)
		}
	}
	if len(matches) != len(want) {
		t.Errorf("AC-1: `## Workflow` must contain exactly %d numbered step headings; got %d", len(want), len(matches))
	}

	if strings.TrimSpace(workflow) == "" {
		t.Error("AC-1: `## Workflow` section must have content beyond headings")
	}
}

// TestStartSkills_G0224_LiveRefusalCodesNamed pins G-0224: both start
// rituals name the refusal code the kernel actually emits
// (`rung-pair-illegal`, from PreflightRungPairError) rather than the
// dead `branch-not-found` code — PreflightBranchNotFoundError is
// defined but never constructed, subsumed by the rung-pair check in
// internal/verb/authorize.go. Scoped to each skill's sovereign-
// authorize subsection (where the refusal sentence lives), plus a
// whole-body guard against the dead code reappearing anywhere.
func TestStartSkills_G0224_LiveRefusalCodesNamed(t *testing.T) {
	t.Parallel()

	epicBody := loadAiwfxStartEpicFixture(t)
	msBody := loadAiwfxStartMilestoneFixture(t)
	cases := []struct {
		skill   string
		body    string
		section string
	}{
		{"aiwfx-start-epic", epicBody, findSovereignAuthorizeSection(epicBody)},
		{"aiwfx-start-milestone", msBody, findStartMilestoneAuthorizeSection(msBody)},
	}
	for _, c := range cases {
		t.Run(c.skill, func(t *testing.T) {
			t.Parallel()
			if c.section == "" {
				t.Fatalf("G-0224: %s must contain a sovereign-authorize subsection", c.skill)
			}
			if strings.Contains(c.body, "branch-not-found") {
				t.Errorf("G-0224: %s must not name the dead `branch-not-found` code (the authorize preflight never constructs it)", c.skill)
			}
			if !strings.Contains(c.section, "rung-pair-illegal") {
				t.Errorf("G-0224: %s sovereign-authorize subsection must name the live `rung-pair-illegal` refusal code", c.skill)
			}
		})
	}
}
