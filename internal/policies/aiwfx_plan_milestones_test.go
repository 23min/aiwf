package policies

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfxPlanMilestonesFixturePath is the canonical authoring location
// for the `aiwfx-plan-milestones` skill body during the G-0079 patch,
// per CLAUDE.md §"Cross-repo plugin testing". The fixture content is
// copied to the rituals plugin repo (`plugins/aiwf-extensions/skills/
// aiwfx-plan-milestones/SKILL.md` there) in a separate commit; the
// drift-check below guards the long-term coupling.
const aiwfxPlanMilestonesFixturePath = "internal/policies/testdata/aiwfx-plan-milestones/SKILL.md"

// loadAiwfxPlanMilestonesFixture reads the fixture relative to repo
// root. The tests under this file are seam-tests against the authored
// skill body — they assert the doctrinal content G-0079 requires,
// scoped to the relevant markdown section per CLAUDE.md *Testing*
// §"Substring assertions are not structural assertions".
func loadAiwfxPlanMilestonesFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxPlanMilestonesFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxPlanMilestonesFixturePath, err)
	}
	return string(data)
}

// findDependsOnStep locates the milestone-dependency-declaration step
// inside `## Workflow`. The `aiwfx-plan-milestones` skill uses a flat
// numbered-list workflow (`1.`, `2.`, … under `## Workflow`) rather
// than `### N.` subheadings, so the locator walks numbered top-level
// list items and returns the body of the one whose first line names
// "depend" (case-insensitive). Content-driven rather than number-driven
// so a future reshuffle that moves the step does not silently break
// the structural check.
//
// A "step" runs from its `N. ` line up to (but not including) the next
// line that matches `M. ` at column 0 (the next step) or the next
// `## ` heading (end of workflow). Fenced code blocks are skipped so a
// code-comment line like `# Some note` inside a ```bash block doesn't
// terminate the step body prematurely.
//
// Returns the section body, or "" if no numbered step under
// `## Workflow` names "depend".
func findDependsOnStep(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	lines := strings.Split(workflow, "\n")
	stepStart := -1
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// Match a top-level numbered list item at column 0: digits
		// followed by `. ` and at least one more char.
		if !isNumberedStepStart(line) {
			continue
		}
		// Read past the leading `N. ` to inspect the heading text,
		// then scope the match to the bolded title (`**...**`) at
		// the start of the step body. This matters because a step
		// like `3. Sequence them. … parallelized (no dependency
		// between them).` mentions "dependency" in prose without
		// being the dependency-declaration step — the locator
		// would otherwise return it wrongly.
		text := strings.TrimLeft(line, "0123456789")
		text = strings.TrimPrefix(text, ". ")
		boldTitle := extractBoldedHeading(text)
		if strings.Contains(strings.ToLower(boldTitle), "depend") {
			stepStart = i
			break
		}
	}
	if stepStart == -1 {
		return ""
	}
	// Find the end of this step: the next numbered step at column 0
	// (skipping fence content).
	end := len(lines)
	inFence = false
	for i := stepStart + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if isNumberedStepStart(lines[i]) {
			end = i
			break
		}
	}
	return strings.Join(lines[stepStart:end], "\n")
}

// extractBoldedHeading returns the text inside the first `**...**`
// span of s, or s itself if no bold span is present. The plan-milestones
// skill writes each step as `N. **Title.** body` — the bolded span is
// the step's effective heading, and matching against it (rather than
// the full step prose) avoids false hits when an unrelated step's
// prose happens to mention the same word.
func extractBoldedHeading(s string) string {
	start := strings.Index(s, "**")
	if start == -1 {
		return s
	}
	rest := s[start+2:]
	end := strings.Index(rest, "**")
	if end == -1 {
		return s
	}
	return rest[:end]
}

// TestExtractBoldedHeading_BranchCoverage covers every reachable
// branch of extractBoldedHeading: no bold span, an unterminated
// span, and a well-formed span.
func TestExtractBoldedHeading_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"no bold here", "no bold here"},
		{"unterminated **bold start", "unterminated **bold start"},
		{"**heading.** body prose", "heading."},
		{"prefix **heading** suffix", "heading"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := extractBoldedHeading(tc.in); got != tc.want {
				t.Errorf("extractBoldedHeading(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

// isNumberedStepStart reports whether the line is a top-level
// numbered-list step start (e.g. `6. **Declare…`). Top-level means
// no leading whitespace; sub-items in a step (`   - foo`) and prose
// continuation lines are excluded. The line must also carry at least
// one non-space character after `N. ` so a trailing-space-only line
// is rejected.
func isNumberedStepStart(line string) bool {
	if line == "" {
		return false
	}
	// Must start with a digit.
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 {
		return false
	}
	// Followed by ". " and at least one more character.
	if i+1 >= len(line) || line[i] != '.' || line[i+1] != ' ' {
		return false
	}
	// Reject trailing-space-only lines (e.g. `"1. "` with no content).
	if i+2 >= len(line) {
		return false
	}
	return true
}

// TestFindDependsOnStep_BranchCoverage exercises the defensive return
// arms of findDependsOnStep that the happy-path fixture test does not
// reach. Cheap insurance per CLAUDE.md §"Test untested code paths
// before declaring code paths done" — every reachable branch has a
// test.
func TestFindDependsOnStep_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		body     string
		wantHas  string // substring that must appear in the returned step
		wantNone bool   // when true, expect empty return
	}{
		{
			name:     "missing-workflow",
			body:     "no headings here, just prose",
			wantNone: true,
		},
		{
			name:     "workflow-without-depends-step",
			body:     "## Workflow\n\n1. Some other step\n2. Another step\n",
			wantNone: true,
		},
		{
			name: "fenced-code-with-numbered-comments-not-confused-for-step",
			body: "## Workflow\n\n1. First step\n\n   ```bash\n   1. not-a-step\n   ```\n\n2. Declare depends\n\n   body here\n",
			// The depends step should be reached, not pre-empted by the
			// `1. not-a-step` line inside the fence on step 1.
			wantHas: "Declare depends",
		},
		{
			name: "happy-path",
			body: "## Workflow\n\n6. Declare depends\n\n   inside step body\n\n7. Next step\n",
			// Step 7 must terminate the depends step; "Next step" must
			// not appear inside the returned slice.
			wantHas: "Declare depends",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findDependsOnStep(tc.body)
			if tc.wantNone {
				if got != "" {
					t.Errorf("findDependsOnStep(%q) = %q; want empty", tc.name, got)
				}
				return
			}
			if !strings.Contains(got, tc.wantHas) {
				t.Errorf("findDependsOnStep(%q) = %q; want it to contain %q", tc.name, got, tc.wantHas)
			}
			// Confirm step terminates correctly: the canary "Next step"
			// from the happy-path body must not leak in.
			if tc.name == "happy-path" && strings.Contains(got, "Next step") {
				t.Errorf("findDependsOnStep(%q): step body leaked into next step (got %q)", tc.name, got)
			}
		})
	}
}

// TestIsNumberedStepStart_BranchCoverage exercises every reachable
// branch of isNumberedStepStart against synthetic inputs.
func TestIsNumberedStepStart_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		line string
		want bool
	}{
		{"", false}, // empty line
		{"prose without digits", false},
		{"1", false},         // digit but no `. ` suffix
		{"1.", false},        // digit + dot but no space
		{"1. ", false},       // digit + `. ` but no content after
		{"1. content", true}, // canonical step start
		{"42. multi-digit content", true},
		{"   1. content", false}, // indented (sub-item), not top-level
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			if got := isNumberedStepStart(tc.line); got != tc.want {
				t.Errorf("isNumberedStepStart(%q) = %v; want %v", tc.line, got, tc.want)
			}
		})
	}
}

// TestAiwfxPlanMilestones_FixtureFrontmatter pins the skill's
// frontmatter: `name: aiwfx-plan-milestones` plus a non-empty
// `description:` so the plugin loader (and AI-assistant discovery)
// continues to surface the skill correctly. Acts as a guard against
// accidental frontmatter edits during the dependency-doc update.
func TestAiwfxPlanMilestones_FixtureFrontmatter(t *testing.T) {
	t.Parallel()
	body := loadAiwfxPlanMilestonesFixture(t)

	if name := frontmatterField(body, "name"); name != "aiwfx-plan-milestones" {
		t.Errorf("frontmatter `name:` must be `aiwfx-plan-milestones` (got %q)", name)
	}
	if desc := frontmatterField(body, "description"); desc == "" {
		t.Error("frontmatter `description:` must be non-empty")
	}
}

// TestAiwfxPlanMilestones_DependsOnUsesVerb_ClosesG0079 pins G-0079's
// substantive doc claim: the `## Workflow` step that covers milestone
// dependency declaration teaches the verb-based path
// (`aiwf add milestone --depends-on` + `aiwf milestone depends-on …`),
// not the hand-edit-frontmatter path that M-0076's writers obviate.
//
// Heading-scoped per CLAUDE.md §"Substring assertions are not
// structural assertions"; the verb names could plausibly appear in
// unrelated sections (e.g. the allocation step references
// `aiwf add milestone`), so the locator scopes the claim to the
// step that exists *because* of dependency declaration.
//
// The hand-edit anti-pattern check fires on the literal YAML
// `depends_on: [M-` snippet — the unambiguous shape the old guidance
// used to teach. Prose may legitimately discuss `depends_on` in
// rationale form ("don't hand-edit `depends_on:` in frontmatter"); the
// bracketed-array example is what the anti-pattern test forbids.
func TestAiwfxPlanMilestones_DependsOnUsesVerb_ClosesG0079(t *testing.T) {
	t.Parallel()
	body := loadAiwfxPlanMilestonesFixture(t)

	section := findDependsOnStep(body)
	if section == "" {
		t.Fatal("G-0079: `## Workflow` must contain a `### …depends…` step that documents milestone dependency declaration")
	}

	// Positive content: both writer verbs must be named in the step,
	// plus the --clear variant from M-0076 AC-3 and a reference to
	// M-0076 so the doc grounds itself in the shipped surface.
	wantContent := []struct {
		name   string
		marker string
	}{
		{"allocation-time flag verb", "aiwf add milestone"},
		{"--depends-on flag", "--depends-on"},
		{"post-allocation dedicated verb", "aiwf milestone depends-on"},
		{"--on flag", "--on"},
		{"--clear flag", "--clear"},
		{"M-0076 reference grounds the surface", "M-0076"},
	}
	for _, w := range wantContent {
		if !strings.Contains(section, w.marker) {
			t.Errorf("G-0079: depends-step must name the %s (substring %q)", w.name, w.marker)
		}
	}

	// Anti-pattern guard: the old "edit M-NNN's frontmatter" snippet
	// (a bracketed `depends_on: [M-NNNN]` YAML example) must not
	// appear anywhere in the body. The literal bracket-form is the
	// load-bearing signal the old guidance taught the hand-edit; the
	// new guidance teaches the verb invocation. Prose references like
	// "do not hand-edit `depends_on:` in frontmatter" remain fine
	// because they don't carry the bracketed array.
	if strings.Contains(body, "depends_on: [M-") {
		t.Error("G-0079: skill must not teach hand-edit (`depends_on: [M-…]` YAML snippet) — use the writer verbs instead")
	}

	// Doctrinal claim: the skill should explicitly steer readers away
	// from hand-editing. Phrased loosely (case-insensitive substring)
	// so wording can drift without breaking the test, but the steer
	// must exist somewhere in the body.
	if !strings.Contains(strings.ToLower(body), "hand-edit") {
		t.Error("G-0079: skill must explicitly warn against hand-editing `depends_on:` so the verb-based path stays the default")
	}
}

// TestAiwfxPlanMilestones_DriftAgainstCache mirrors the M-0096
// drift-check pattern: the fixture's bytes must match the rituals-repo
// copy in the active marketplace cache once the upstream side of this
// patch has landed. Pre-deploy (the rituals-repo edit hasn't shipped
// yet) the test legitimately *skips* — the cached file still holds
// the old hand-edit guidance, which the fixture deliberately replaces.
//
// Skip semantics:
//   - `installed_plugins.json` absent → skip (CI without plugin install).
//   - `aiwf-extensions@ai-workflow-rituals` not installed → skip.
//   - Skill not yet materialised in the active install → skip.
//
// Fail semantics:
//   - Skill materialised, bytes differ from fixture → FAIL with a
//     drift message pointing at the cache path so the operator can
//     either re-deploy the fixture or update it.
//
// Until the rituals-repo upstream edit lands and the operator reloads
// plugins, this test skips with an explanatory message rather than
// failing on the (intentional) pre-deploy drift.
func TestAiwfxPlanMilestones_DriftAgainstCache(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	manifestPath := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	manifest, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		t.Skipf("drift-check skip: %q not present; run after plugin install to verify drift-check", manifestPath)
	}
	if err != nil {
		t.Fatalf("reading %q: %v", manifestPath, err)
	}

	var parsed struct {
		Plugins map[string][]struct {
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if jsonErr := json.Unmarshal(manifest, &parsed); jsonErr != nil {
		t.Fatalf("parsing %q: %v", manifestPath, jsonErr)
	}
	installs, ok := parsed.Plugins["aiwf-extensions@ai-workflow-rituals"]
	if !ok || len(installs) == 0 {
		t.Skipf("drift-check skip: aiwf-extensions@ai-workflow-rituals not installed (no entry in %q)", manifestPath)
	}
	skillPath := filepath.Join(installs[0].InstallPath, "skills", "aiwfx-plan-milestones", "SKILL.md")
	if _, statErr := os.Stat(skillPath); os.IsNotExist(statErr) {
		t.Skipf("drift-check skip: aiwfx-plan-milestones not materialised in active install (expected at %q)", skillPath)
	} else if statErr != nil {
		t.Fatalf("stat %q: %v", skillPath, statErr)
	}

	cached, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading cached skill at %q: %v", skillPath, err)
	}

	fixture := loadAiwfxPlanMilestonesFixture(t)
	if err := compareSkillBytes([]byte(fixture), cached, skillPath); err != nil {
		// Pre-deploy state: the rituals-repo upstream edit has not
		// landed (or the operator has not reloaded plugins), so the
		// cached copy still carries the old hand-edit guidance the
		// fixture replaces. Skip rather than fail until the upstream
		// commit lands — at which point this becomes an active
		// drift-detector in both directions.
		if strings.Contains(string(cached), "depends_on: [M-") {
			t.Skipf("drift-check skip: cached skill still carries the pre-G-0079 hand-edit guidance (`depends_on: [M-…]`); rituals-repo upstream edit not yet deployed")
		}
		t.Errorf("drift-check: %v", err)
	}
}
