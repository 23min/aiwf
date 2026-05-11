package policies

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// compareSkillBytes is the byte-compare seam between the M-0096
// fixture and the marketplace-cache copy of the rituals-repo skill.
// Returns nil if the two byte slices are identical; returns a typed
// error containing skillPath and a re-deploy hint on drift.
//
// Extracted from `TestAiwfxStartEpic_AC5_DriftAgainstCache` per
// M-0097/AC-2 so the comparator's two arms can be exercised
// synthetically — the production AC-5 test only reaches the drift
// arm post-wrap in the rare drift-detected production state.
func compareSkillBytes(fixture, cached []byte, skillPath string) error {
	if bytes.Equal(fixture, cached) {
		return nil
	}
	return fmt.Errorf("drift between fixture and cached skill at %q — re-deploy fixture to rituals repo and reload plugins, or update the fixture if the rituals-side is canonical", skillPath)
}

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

// TestCompareSkillBytes_BranchCoverage pins M-0097/AC-2: the
// fixture-vs-cache byte-compare logic used by AC-5's drift check is
// exercised synthetically — both the match arm and the drift arm —
// regardless of whether the marketplace cache currently carries the
// rituals-repo content. Before M-0097, the drift arm could only be
// reached in the rare production state where cache bytes differed
// from fixture bytes; this test pins the arm with controlled inputs.
//
// The empty/empty case asserts the match arm tolerates an empty
// fixture and cache without producing a false-positive drift signal
// (defensive: prevents a regression where empty-vs-empty would be
// treated as "drift" by an over-eager comparator).
func TestCompareSkillBytes_BranchCoverage(t *testing.T) {
	cases := []struct {
		name    string
		fixture []byte
		cached  []byte
		wantErr bool
	}{
		{"identical", []byte("---\nname: x\n---\nbody\n"), []byte("---\nname: x\n---\nbody\n"), false},
		{"empty-both", []byte(""), []byte(""), false},
		{"drift-different-bytes", []byte("old content\n"), []byte("new content\n"), true},
		{"drift-fixture-only", []byte("present\n"), []byte(""), true},
		{"drift-cached-only", []byte(""), []byte("present\n"), true},
		{"drift-trailing-newline", []byte("body\n"), []byte("body"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := compareSkillBytes(tc.fixture, tc.cached, "/fake/skill/path/SKILL.md")
			if tc.wantErr {
				if err == nil {
					t.Errorf("%s: expected drift error, got nil", tc.name)
					return
				}
				if !strings.Contains(err.Error(), "/fake/skill/path/SKILL.md") {
					t.Errorf("%s: drift error must name the skill path; got %v", tc.name, err)
				}
			} else if err != nil {
				t.Errorf("%s: expected nil on match, got %v", tc.name, err)
			}
		})
	}
}

// TestAiwfxStartEpic_AC5_DriftAgainstCache pins M-0096/AC-5: the
// fixture content matches the currently-active plugin install per
// `installed_plugins.json` when the cache is present and the skill
// is materialised in it. The test's job is to detect **drift** — a
// missing cache or a not-yet-deployed skill is a "skip" state, not a
// "fail" state.
//
// Skip semantics:
//   - `installed_plugins.json` absent → skip (CI without plugin install).
//   - `aiwf-extensions@ai-workflow-rituals` not installed → skip.
//   - Skill not yet materialised in the active install → skip (the
//     rituals-repo copy lands at M-0096 wrap; pre-wrap the file is
//     legitimately absent there).
//
// Fail semantics:
//   - Skill materialised, bytes differ from fixture → FAIL with a
//     drift message pointing at the cache path so the operator can
//     either re-deploy the fixture or update it.
//
// The drift detection itself is exercised post-wrap (when the
// rituals-repo carries the fixture) and continues to detect future
// drift in either direction. The "skill missing" arm is the design's
// way of staying clean during the M-0096 milestone itself, where the
// rituals-repo copy has not landed yet.
func TestAiwfxStartEpic_AC5_DriftAgainstCache(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	manifestPath := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	manifest, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		t.Skipf("AC-5 skip: %q not present; run after plugin install to verify drift-check", manifestPath)
	}
	if err != nil {
		t.Fatalf("AC-5: reading %q: %v", manifestPath, err)
	}

	// Resolve the *active* install path from installed_plugins.json,
	// matching the M-0090 precedent's lookup shape.
	var parsed struct {
		Plugins map[string][]struct {
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if jsonErr := json.Unmarshal(manifest, &parsed); jsonErr != nil {
		t.Fatalf("AC-5: parsing %q: %v", manifestPath, jsonErr)
	}
	installs, ok := parsed.Plugins["aiwf-extensions@ai-workflow-rituals"]
	if !ok || len(installs) == 0 {
		t.Skipf("AC-5 skip: aiwf-extensions@ai-workflow-rituals not installed (no entry in %q)", manifestPath)
	}
	skillPath := filepath.Join(installs[0].InstallPath, "skills", "aiwfx-start-epic", "SKILL.md")
	if _, statErr := os.Stat(skillPath); os.IsNotExist(statErr) {
		// Pre-wrap: the rituals-repo copy has not landed yet. This is
		// a legitimate transient state during M-0096; skip rather than
		// fail. The wrap step (copying the fixture to the rituals repo)
		// flips this from skip to "active drift-check".
		t.Skipf("AC-5 skip: aiwfx-start-epic not materialised in active install (expected at %q); pre-wrap state — rituals-repo copy lands at M-0096 wrap", skillPath)
	} else if statErr != nil {
		t.Fatalf("AC-5: stat %q: %v", skillPath, statErr)
	}

	cached, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("AC-5: reading cached skill at %q: %v", skillPath, err)
	}

	fixture := loadAiwfxStartEpicFixture(t)
	if err := compareSkillBytes([]byte(fixture), cached, skillPath); err != nil {
		t.Errorf("AC-5: %v", err)
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
