package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// aiwfxWhiteboardFixturePath is the canonical authoring location
// for the `aiwfx-whiteboard` skill body during M-079, per CLAUDE.md
// §"Cross-repo plugin testing". At wrap, the fixture content is
// copied to the rituals plugin repo (`plugins/aiwf-extensions/
// skills/aiwfx-whiteboard/SKILL.md` there); a drift-check test
// guards the long-term coupling.
const aiwfxWhiteboardFixturePath = "internal/policies/testdata/aiwfx-whiteboard/SKILL.md"

// loadAiwfxWhiteboardFixture reads the fixture relative to repo
// root. The tests under this file are seam-tests against the
// authored skill body — they assert the doctrinal content M-079's
// ACs require, scoped to the relevant markdown section per
// CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions".
func loadAiwfxWhiteboardFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxWhiteboardFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxWhiteboardFixturePath, err)
	}
	return string(data)
}

// frontmatterField extracts a single-line frontmatter value (the
// pattern aiwfx-* skills use; block-scalar `|` form is not yet
// produced by this skill). Returns "" if not found.
func frontmatterField(body, key string) string {
	// Frontmatter ends at the second `---`.
	if !strings.HasPrefix(body, "---\n") {
		return ""
	}
	end := strings.Index(body[4:], "\n---")
	if end == -1 {
		return ""
	}
	front := body[4 : 4+end]
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:\s*(.+?)\s*$`)
	m := re.FindStringSubmatch(front)
	if m == nil {
		return ""
	}
	return m[1]
}

// TestFrontmatterField_BranchCoverage exercises every reachable
// branch of frontmatterField against synthetic inputs. The helper
// is only ever called from this test file today, but each branch
// is reachable via real inputs (a body missing the frontmatter
// fence, an unterminated frontmatter, a frontmatter that doesn't
// carry the queried key) and the project's branch-coverage rule
// applies even to test-package helpers.
func TestFrontmatterField_BranchCoverage(t *testing.T) {
	cases := []struct {
		name string
		body string
		key  string
		want string
	}{
		{"no leading fence", "no frontmatter here\n", "name", ""},
		{"unterminated frontmatter", "---\nname: x\n", "name", ""},
		{"key not present", "---\ndescription: x\n---\n", "name", ""},
		{"key present", "---\nname: aiwfx-x\n---\nbody", "name", "aiwfx-x"},
		{"key present with surrounding whitespace", "---\nname:   aiwfx-x   \n---\n", "name", "aiwfx-x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := frontmatterField(tc.body, tc.key)
			if got != tc.want {
				t.Errorf("frontmatterField(%q) = %q; want %q", tc.body, got, tc.want)
			}
		})
	}
}

// TestAiwfxWhiteboard_AC1_SkillScaffolded asserts AC-1: the skill
// fixture exists with frontmatter declaring `name: aiwfx-whiteboard`
// (matching the directory) and a non-empty `description:`. This is
// the v1 single-SKILL.md layout (no template subdirs).
func TestAiwfxWhiteboard_AC1_SkillScaffolded(t *testing.T) {
	body := loadAiwfxWhiteboardFixture(t)

	name := frontmatterField(body, "name")
	if name != "aiwfx-whiteboard" {
		t.Errorf("AC-1: frontmatter `name:` must be `aiwfx-whiteboard` (got %q)", name)
	}

	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Error("AC-1: frontmatter `description:` must be non-empty")
	}
}

// TestAiwfxWhiteboard_AC3_TierRubric asserts AC-3: the body has a
// `## Tier classification rubric` section that names all five
// tiers (Tier 1..Tier 5), assigns each a criterion phrase, and
// cites examples drawn from `critical-path.md` (which is in-tree
// at the time of authoring; deletion is M-080's act).
func TestAiwfxWhiteboard_AC3_TierRubric(t *testing.T) {
	body := loadAiwfxWhiteboardFixture(t)
	section := extractMarkdownSection(body, 2, "Tier classification rubric")
	if section == "" {
		t.Fatal("AC-3: SKILL.md must have a `## Tier classification rubric` section")
	}

	// Five tiers with their descriptive labels per critical-path.md
	// and the spec's at-minimum list. Match the digit + the label
	// keyword so a future reorder catches the test.
	tierLabels := map[string]string{
		"Tier 1": "compounding",
		"Tier 2": "foundational",
		"Tier 3": "ritual",
		"Tier 4": "debris",
		"Tier 5": "defer",
	}
	lower := strings.ToLower(section)
	for tier, keyword := range tierLabels {
		if !strings.Contains(section, tier) {
			t.Errorf("AC-3: §Tier classification rubric must name %q", tier)
		}
		if !strings.Contains(lower, keyword) {
			t.Errorf("AC-3: §Tier classification rubric must use the descriptive keyword %q (for %s)", keyword, tier)
		}
	}

	// Spec: the rubric must cite examples drawn from
	// critical-path.md's actual tier placements. Pick one
	// representative from each tier and assert the id appears
	// in the rubric body. (Per critical-path.md: Tier 1 = G-071,
	// Tier 2 = ADR-0001, Tier 3 = G-059, Tier 4 = G-056, Tier 5
	// = G-070.)
	exemplars := []string{"G-071", "ADR-0001", "G-059", "G-056", "G-070"}
	for _, id := range exemplars {
		if !strings.Contains(section, id) {
			t.Errorf("AC-3: §Tier classification rubric must cite exemplar %q from critical-path.md", id)
		}
	}
}

// TestAiwfxWhiteboard_AC4_OutputTemplate asserts AC-4: the body
// has an `## Output template` section that names the four output
// blocks the skill emits — tiered landscape, recommended sequence,
// first-decision fork, pending-decisions list — with column /
// ordering shape spelled out.
func TestAiwfxWhiteboard_AC4_OutputTemplate(t *testing.T) {
	body := loadAiwfxWhiteboardFixture(t)
	section := extractMarkdownSection(body, 2, "Output template")
	if section == "" {
		t.Fatal("AC-4: SKILL.md must have a `## Output template` section")
	}
	lower := strings.ToLower(section)

	// The four named output blocks per AC-4 spec text.
	required := []string{
		"tiered landscape",     // (a) per spec
		"recommended sequence", // (b)
		"first-decision",       // (c) — "first-decision fork"
		"pending decision",     // (d) — "pending-decisions list"; match singular for table-row case
	}
	for _, r := range required {
		if !strings.Contains(lower, r) {
			t.Errorf("AC-4: §Output template must name the %q output block", r)
		}
	}

	// Spec: landscape table specifies columns; the named columns
	// per AC-4 are kind, cost-estimate, what-it-unblocks. Match
	// case-insensitively because column headers may capitalise.
	requiredColumns := []string{"kind", "cost", "unblock"}
	for _, c := range requiredColumns {
		if !strings.Contains(lower, c) {
			t.Errorf("AC-4: §Output template must name the %q column for the landscape table", c)
		}
	}

	// Spec: sequence prose uses explicit before/after/parallel framing.
	for _, term := range []string{"before", "after", "parallel"} {
		if !strings.Contains(lower, term) {
			t.Errorf("AC-4: §Output template must use the %q ordering frame for the sequence section", term)
		}
	}
}

// TestAiwfxWhiteboard_AC5_QAGate asserts AC-5: the body has a
// `## Q&A gate` section carrying the canonical gate prompt and the
// one-at-a-time framing per CLAUDE.md *Working with the user*
// §Q&A format.
func TestAiwfxWhiteboard_AC5_QAGate(t *testing.T) {
	body := loadAiwfxWhiteboardFixture(t)
	section := extractMarkdownSection(body, 2, "Q&A gate")
	if section == "" {
		t.Fatal("AC-5: SKILL.md must have a `## Q&A gate` section")
	}
	lower := strings.ToLower(section)

	// The exact gate-text template per spec (AC-5 quotes the
	// phrasing verbatim).
	gatePhrase := "walk through the pending decisions one at a time"
	if !strings.Contains(lower, gatePhrase) {
		t.Errorf("AC-5: §Q&A gate must include the canonical gate prompt %q", gatePhrase)
	}

	// One-at-a-time discipline (decline path → exit, opt-in →
	// walk one at a time).
	if !strings.Contains(lower, "one at a time") {
		t.Error("AC-5: §Q&A gate must enforce one-at-a-time framing")
	}

	// Reference to CLAUDE.md's Q&A convention or equivalent.
	if !regexp.MustCompile(`(?i)claude\.md|working with the user|q&a format`).MatchString(section) {
		t.Error("AC-5: §Q&A gate must reference the CLAUDE.md Q&A convention")
	}
}

// TestAiwfxWhiteboard_AC6_AntiPatterns asserts AC-6: the body has
// an `## Anti-patterns` section listing the four spec-named
// anti-patterns (no operator override, no verb invention, no
// persisted artefact, scope locked to direction-synthesis).
func TestAiwfxWhiteboard_AC6_AntiPatterns(t *testing.T) {
	body := loadAiwfxWhiteboardFixture(t)
	section := extractMarkdownSection(body, 2, "Anti-patterns")
	if section == "" {
		t.Fatal("AC-6: SKILL.md must have an `## Anti-patterns` section")
	}
	lower := strings.ToLower(section)

	// AC-6 names four anti-patterns; assert each is named by a
	// distinguishing phrase.
	requiredPhrases := map[string]string{
		"no-operator-override":  "operator",  // skill surfaces and gates, doesn't override operator judgement
		"no-verb-invention":     "verb",      // doesn't invent verbs that don't exist
		"no-persisted-artefact": "persist",   // doesn't write its output to a file (matches "persist", "persisted")
		"scope-locked":          "direction", // scope is locked to direction-synthesis
	}
	for label, term := range requiredPhrases {
		if !strings.Contains(lower, term) {
			t.Errorf("AC-6: §Anti-patterns must name %s (keyword %q)", label, term)
		}
	}
}

// TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent asserts
// AC-7: the skill conforms to the same invariants the kernel's
// PolicySkillCoverageMatchesVerbs (M-074) enforces for embedded
// skills. The kernel policy walks `internal/skills/embedded/`
// only — it does not walk plugin paths today — so this test
// applies the equivalent invariants directly to the fixture.
//
// This is the AC-7 "plugin equivalent" path the spec sanctions
// when M-074's policy is kernel-only. The follow-up gap to expand
// the kernel policy to plugin skills is captured under the
// milestone's *Deferrals* section.
func TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent(t *testing.T) {
	body := loadAiwfxWhiteboardFixture(t)

	// Frontmatter shape: name matches dir, description non-empty,
	// name follows aiwfx-<topic> convention.
	name := frontmatterField(body, "name")
	if name != "aiwfx-whiteboard" {
		t.Errorf("AC-7: skill name must equal dir basename `aiwfx-whiteboard` (got %q)", name)
	}
	if !strings.HasPrefix(name, "aiwfx-") {
		t.Errorf("AC-7: skill name %q must follow aiwfx-<topic> convention", name)
	}
	if frontmatterField(body, "description") == "" {
		t.Error("AC-7: description must be non-empty")
	}

	// No-verb-invention: every backticked `aiwf <verb>` mention in
	// the body resolves to a real top-level Cobra verb. This is
	// the load-bearing AC-7 assertion — it catches the same
	// failure mode that would have fired G-061's repro on a
	// kernel-side skill.
	verbs, err := findTopLevelVerbs(repoRoot(t))
	if err != nil {
		t.Fatalf("findTopLevelVerbs: %v", err)
	}
	mentions := backtickedAiwfMentions(body)
	for _, m := range mentions {
		if _, ok := verbs[m.verb]; !ok {
			t.Errorf("AC-7: skill body mentions `aiwf %s` but no such top-level verb is registered", m.verb)
		}
	}
}

// TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck asserts AC-8:
// the skill is materialised by the marketplace install (the
// "distribution path" the AC names) and the cached copy matches
// the fixture authored in this repo. Implements the drift-check
// pattern from CLAUDE.md §"Cross-repo plugin testing".
//
// Skip semantics:
//   - If the marketplace cache for ai-workflow-rituals is absent
//     entirely (no plugin install on this machine), the test
//     skips cleanly. CI without a plugin install therefore
//     doesn't fail; the AC is verified locally where the cache
//     lives.
//   - If the plugin is installed but the aiwfx-whiteboard skill
//     is missing from the cache, the test FAILS — that's the
//     "not materialised" condition AC-8 forbids.
//   - If the skill is present but content differs from the
//     fixture, the test FAILS — that's the drift condition
//     CLAUDE.md's pattern is designed to catch.
func TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	cacheRoot := filepath.Join(home, ".claude", "plugins", "cache", "ai-workflow-rituals")
	if _, err := os.Stat(cacheRoot); os.IsNotExist(err) {
		t.Skipf("AC-8 skip: marketplace cache %q not present; run after plugin install to verify materialisation", cacheRoot)
	}

	// Walk down to find aiwfx-whiteboard inside the cached
	// aiwf-extensions plugin. The cache layout is
	// `.../ai-workflow-rituals/aiwf-extensions/<sha-prefix>/skills/aiwfx-whiteboard/SKILL.md`.
	pluginRoot := filepath.Join(cacheRoot, "aiwf-extensions")
	entries, err := os.ReadDir(pluginRoot)
	if err != nil {
		t.Skipf("AC-8 skip: aiwf-extensions plugin not cached at %q: %v", pluginRoot, err)
	}

	var skillPath string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(pluginRoot, e.Name(), "skills", "aiwfx-whiteboard", "SKILL.md")
		if _, err := os.Stat(candidate); err == nil {
			skillPath = candidate
			break
		}
	}
	if skillPath == "" {
		t.Errorf("AC-8: aiwfx-whiteboard not materialised in plugin cache (looked under %q)", pluginRoot)
		return
	}

	cached, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("AC-8: reading cached skill at %q: %v", skillPath, err)
	}

	fixture := loadAiwfxWhiteboardFixture(t)
	if string(cached) != fixture {
		t.Errorf("AC-8: drift between fixture and cached skill at %q — re-deploy fixture to rituals repo and reload plugins, or update the fixture if the rituals-side is canonical", skillPath)
	}
}

// TestAiwfxWhiteboard_AC2_DescriptionPhrasings asserts AC-2: the
// frontmatter `description:` carries at minimum five of the named
// natural-language query phrasings the user might type to a
// description-match-routing host. Per the spec, the skill name
// (`whiteboard`) is metaphor-shaped not query-shaped, so
// description-density does the routing work.
func TestAiwfxWhiteboard_AC2_DescriptionPhrasings(t *testing.T) {
	body := loadAiwfxWhiteboardFixture(t)
	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Fatal("AC-2: frontmatter description is empty (AC-1 should have caught this)")
	}

	// Spec-listed phrasings. AC-2 requires at least 5 of these
	// (the spec offers 6 + an "or equivalent metaphor-anchored
	// phrasing" for the last). Match case-insensitively because
	// the description may quote them with different capitalisation.
	candidates := []string{
		"what should i work on next",
		"give me the landscape",
		"where should we focus",
		"what's the critical path",
		"synthesise the open work",
		"draw the whiteboard",
	}
	lower := strings.ToLower(desc)
	hits := 0
	missing := []string{}
	for _, p := range candidates {
		if strings.Contains(lower, p) {
			hits++
		} else {
			missing = append(missing, p)
		}
	}
	if hits < 5 {
		t.Errorf("AC-2: description must carry ≥5 spec-listed phrasings (got %d; missing: %v)", hits, missing)
	}
}
