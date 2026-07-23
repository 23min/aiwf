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

// TestAiwfxStartMilestone_M0105_AC2_PreflightAssertsParentEpicBranchPrecondition
// pins M-0105/AC-2: the preflight (step 1) explicitly names the
// "parent epic branch must exist locally AND be the current
// checkout" precondition, and points operators at `aiwfx-start-epic`
// when the parent branch is missing.
//
// Heading-scoped — the precondition must live INSIDE the preflight
// section, not float somewhere else where a reader scanning the
// preflight would miss it.
func TestAiwfxStartMilestone_M0105_AC2_PreflightAssertsParentEpicBranchPrecondition(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	preflight := findStartMilestonePreflightSection(body)
	if preflight == "" {
		t.Fatal("AC-2: `## Workflow` must contain a `### …preflight…` subsection (step 1)")
	}

	wantContent := []struct {
		name   string
		marker string
	}{
		{"parent epic branch identifier", "epic/E-NNNN"},
		{"existence requirement", "must exist"},
		{"current-checkout requirement", "current checkout"},
		{"escape hatch pointing at aiwfx-start-epic", "aiwfx-start-epic"},
	}
	for _, w := range wantContent {
		if !strings.Contains(preflight, w.marker) {
			t.Errorf("AC-2: preflight subsection must name %s (substring %q)", w.name, w.marker)
		}
	}
}

// TestAiwfxStartMilestone_PreflightExpectsACsPreExist_M0275 pins M-0275/AC-5:
// the preflight reframes ACs as expected to already exist — created at plan time
// by aiwfx-plan-milestones (M-0275/AC-4) — and retains on-the-spot `aiwf add ac`
// creation only as a recovery fallback for a hand-written spec, no longer the
// default path. Heading-scoped to the preflight subsection so the reframe lives
// where a reader starting the milestone actually looks.
func TestAiwfxStartMilestone_PreflightExpectsACsPreExist_M0275(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	preflight := findStartMilestonePreflightSection(body)
	if preflight == "" {
		t.Fatal("AC-5: `## Workflow` must contain a `### …preflight…` subsection (step 1)")
	}
	lower := strings.ToLower(preflight)

	// Reframe: ACs are expected to pre-exist because they were created at plan time.
	if !strings.Contains(lower, "plan time") {
		t.Error("AC-5: preflight must frame ACs as created at plan time (expected to pre-exist), not something to add here by default")
	}
	// The on-the-spot creation is explicitly a recovery fallback, not the default.
	if !strings.Contains(lower, "fallback") {
		t.Error("AC-5: preflight must frame on-the-spot `aiwf add ac` as a recovery fallback")
	}
	// The recovery command itself survives for the hand-written-spec case.
	if !strings.Contains(preflight, "aiwf add ac") {
		t.Error("AC-5: preflight must retain the `aiwf add ac` recovery command for a hand-written spec")
	}
}

// TestAiwfxStartMilestone_M0105_AC3_NoSilentFallthroughToParentCheckout
// pins M-0105/AC-3: the silent
// `git checkout -b epic/E-NNNN-<slug> origin/main # if missing`
// fallthrough that previously masked the missing-parent-branch case
// is removed from the workflow prose. The skill must not silently
// materialize the parent epic branch.
//
// Two-sided assertion. The forbidden pattern is absent under
// `## Workflow` (where the procedure lives); the anti-pattern
// section MAY still reference the old fallthrough as a known
// anti-pattern — that's documentation about what NOT to do, and
// counts as a positive signal.
//
// The assertion targets the workflow prose, not the anti-pattern
// section, so the documentation-of-anti-pattern usage is allowed.
func TestAiwfxStartMilestone_M0105_AC3_NoSilentFallthroughToParentCheckout(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("AC-3: body must contain a `## Workflow` section")
	}

	forbidden := []string{
		// The exact stale shell line the M-0105 spec calls out.
		"# if missing",
		// The shape of the old fallthrough's git invocation in the
		// workflow prose — the literal old skill body used
		// `git checkout -b epic/E-NNNN-<slug> origin/main`.
		"origin/main",
		// Structural shape of the fallthrough that catches
		// rephrased regressions (e.g. `# if absent`, no comment at
		// all): the skill body must NEVER prescribe creating the
		// parent epic branch — that's aiwfx-start-epic's job per
		// AC-2's tightened preflight. Reviewer feedback (M-0105
		// Cycle 2): narrow markers alone leak.
		"git checkout -b epic/",
	}
	for _, marker := range forbidden {
		if strings.Contains(workflow, marker) {
			t.Errorf("AC-3: `## Workflow` must not contain the silent fallthrough marker %q — removed per M-0105/AC-3", marker)
		}
	}
}

// TestAiwfxStartMilestone_M0105_AC4_WorkflowHeadingsInNewOrder pins
// M-0105/AC-4: the workflow headings, parsed structurally, appear
// in the new order — preflight → delegation prompt → sovereign
// promote → sovereign authorize → cut milestone branch →
// implementation → readiness check → hand off to wrap.
//
// Heading-content driven per CLAUDE.md §"Substring assertions are
// not structural assertions": each expected step asserts that the
// i-th `### N.` heading under `## Workflow` contains a distinctive
// lowercase token. The order is what's pinned; exact wording may
// evolve so long as the conceptual sequence holds.
func TestAiwfxStartMilestone_M0105_AC4_WorkflowHeadingsInNewOrder(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("AC-4: body must contain a `## Workflow` section")
	}

	stepHeading := regexp.MustCompile(`(?m)^### \d+\.\s+(.+)$`)
	matches := stepHeading.FindAllStringSubmatch(workflow, -1)
	gotHeadings := make([]string, 0, len(matches))
	for _, m := range matches {
		gotHeadings = append(gotHeadings, strings.ToLower(strings.TrimSpace(m[1])))
	}

	wantOrderTokens := []string{
		"preflight",          // step 1 — tightened
		"delegation",         // step 2 — new (moved earlier than the sovereign acts)
		"sovereign promot",   // step 3 — was step 2; now explicitly on parent
		"sovereign authoriz", // step 4 — new (only if delegating)
		"cut",                // step 5 — was buried in old step 3's branch-setup
		"implementation",     // step 6 — was step 4
		"readiness",          // step 7 — reframed from "self-review" per G-0271
		"hand off",           // step 8 — was step 6
	}
	if len(gotHeadings) != len(wantOrderTokens) {
		t.Fatalf("AC-4: expected %d workflow steps in the new ordering; got %d (headings: %q)",
			len(wantOrderTokens), len(gotHeadings), gotHeadings)
	}
	for i, tok := range wantOrderTokens {
		if !strings.Contains(gotHeadings[i], tok) {
			t.Errorf("AC-4: step %d heading %q does not contain expected token %q (full ordering: %q)",
				i+1, gotHeadings[i], tok, gotHeadings)
		}
	}
}

// TestAiwfxStartMilestone_M0105_AC5_SovereignActsNameOverride pins
// M-0105/AC-5: both sovereign acts (promote at step 3, authorize at
// step 4) name `--force --reason` as the override path. Mirrors
// M-0104/AC-5's pattern for aiwfx-start-epic — both acts on the
// parent epic branch are sovereign moments that need the operator
// to see the escape valve.
//
// Heading-scoped: the override must live INSIDE each step's
// subsection, not float in an unrelated section.
//
// The authorize section must additionally name the M-0105/AC-6
// carve-out's preconditions — ritual current branch + ritual
// --branch — so a reader who hits step 4 cold understands why the
// verb does not refuse despite the future-branch shape.
func TestAiwfxStartMilestone_M0105_AC5_SovereignActsNameOverride(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	// Step 3 — Sovereign promote. Locator: case-insensitive
	// "sovereign" + "promot" inside ## Workflow.
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("AC-5: body must contain a `## Workflow` section")
	}
	var promoteSection string
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		lower := strings.ToLower(text)
		if strings.Contains(lower, "sovereign") && strings.Contains(lower, "promot") {
			promoteSection = extractMarkdownSection(body, 3, text)
			break
		}
	}
	if promoteSection == "" {
		t.Fatal("AC-5: `## Workflow` must contain a `### …sovereign…promot…` subsection (step 3)")
	}
	wantPromote := []struct {
		name   string
		marker string
	}{
		{"the promote verb", "aiwf promote"},
		{"--force --reason override path", "--force --reason"},
	}
	for _, w := range wantPromote {
		if !strings.Contains(promoteSection, w.marker) {
			t.Errorf("AC-5: sovereign-promote subsection must name %s (substring %q)", w.name, w.marker)
		}
	}

	// Step 4 — Sovereign authorize.
	authorizeSection := findStartMilestoneAuthorizeSection(body)
	if authorizeSection == "" {
		t.Fatal("AC-5: `## Workflow` must contain a `### …sovereign…authoriz…` subsection (step 4)")
	}
	wantAuthorize := []struct {
		name   string
		marker string
	}{
		{"the authorize verb", "aiwf authorize"},
		{"--force --reason override path", "--force --reason"},
		{"--branch flag (the future-binding the carve-out permits)", "--branch"},
		{"future milestone branch shape", "milestone/M-NNNN"},
		{"parent epic branch context (the ritual current arm)", "epic/E-NNNN"},
	}
	for _, w := range wantAuthorize {
		if !strings.Contains(authorizeSection, w.marker) {
			t.Errorf("AC-5: sovereign-authorize subsection must name %s (substring %q)", w.name, w.marker)
		}
	}
}

// TestAiwfxStartMilestone_G0271_ReadinessNotReview pins G-0271
// defect #1: the pre-handoff pass is framed as a readiness check, not
// "self-review", and it forward-references the authoritative
// independent review that runs at aiwfx-wrap-milestone. The property:
// a reader of start-milestone alone comes away knowing the author
// pass is NOT the final review.
//
// Two assertions: (1) no `### N.` workflow heading is labelled
// "self-review" (the misleading label is gone); (2) the reframed
// step-7 subsection names the independent review at
// aiwfx-wrap-milestone as the authoritative check.
func TestAiwfxStartMilestone_G0271_ReadinessNotReview(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("G-0271: body must contain a `## Workflow` section")
	}

	// (1) No workflow step heading may reintroduce the "self-review"
	// label that let a reader believe review was already done.
	var readiness string
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "self-review") {
			t.Errorf("G-0271: no workflow step heading may be labelled self-review; got %q", strings.TrimSpace(line))
		}
		if strings.Contains(strings.ToLower(text), "readiness") {
			readiness = extractMarkdownSection(body, 3, text)
		}
	}

	// (2) The readiness step must forward-reference the authoritative
	// independent review at wrap.
	if readiness == "" {
		t.Fatal("G-0271: `## Workflow` must contain a `### …readiness…` subsection (the reframed step 7)")
	}
	wantForward := []struct {
		name   string
		marker string
	}{
		{"the review is independent", "independent"},
		{"the review runs at wrap", "aiwfx-wrap-milestone"},
	}
	for _, w := range wantForward {
		if !strings.Contains(readiness, w.marker) {
			t.Errorf("G-0271: readiness subsection must state %s (substring %q) so the author pass is not read as the final review", w.name, w.marker)
		}
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
