package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// adr0007Path is the canonical relative path to the ADR-0007 file.
// Anchored as a constant so the tests fail loudly with a single
// rename target if the slug ever changes.
const adr0007Path = "docs/adr/ADR-0007-planning-conversation-skills-rituals-plugin-placement-pure-skill-first-kernel-verb-only-if-usage-demands-it.md"

// loadADR0007 reads ADR-0007 from disk relative to the repo root.
// The tests are seam-tests against the live document — they assert
// the doctrinal claims M-078's ACs require, scoped to the relevant
// markdown section. Per CLAUDE.md *Testing* §"Substring assertions
// are not structural assertions", section-scoped checks beat flat
// greps because the same word in the wrong section is still wrong.
func loadADR0007(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, adr0007Path))
	if err != nil {
		t.Fatalf("loading %s: %v", adr0007Path, err)
	}
	return string(data)
}

// extractMarkdownSection returns the body of a markdown section
// named heading at the requested level (2 for `##`, 3 for `###`),
// from the first matching heading to the next heading at the same
// or higher level. Returns "" if not found. The match is on a
// prefix of the heading text (the literal headings carry trailing
// rationale after `—`, so prefix match is enough).
//
// Fenced code blocks (triple-backtick spans) are skipped when
// scanning for the closing heading: a bash-comment line like
// `# Preview the planned moves` inside a ```` ```bash ```` block
// is content, not a level-1 heading. Without this guard the
// scanner truncates SKILL.md sections at the first hash-prefixed
// line inside an example block.
func extractMarkdownSection(body string, level int, headingPrefix string) string {
	if level < 1 || level > 6 {
		return ""
	}
	prefix := strings.Repeat("#", level) + " "
	lines := strings.Split(body, "\n")
	start := -1
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			rest := strings.TrimPrefix(line, prefix)
			if strings.HasPrefix(rest, headingPrefix) {
				start = i + 1
				break
			}
		}
	}
	if start == -1 {
		return ""
	}
	end := len(lines)
	inFence = false
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// Stop at any heading of equal or higher level (i.e.,
		// fewer or equal `#` characters before the space).
		hashes := 0
		for _, r := range lines[i] {
			if r == '#' {
				hashes++
			} else {
				break
			}
		}
		if hashes >= 1 && hashes <= level && hashes < len(lines[i]) && lines[i][hashes] == ' ' {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

// TestADR0007_AC1_Allocation asserts AC-1's structural drift check:
// the ADR is allocated under docs/adr/ with frontmatter `id: ADR-0007`
// matching the path. The status check was originally part of this
// test as M-078's premature-ratification chokepoint, but per the
// 2026-05-09 cleanup framing, ADR ratification timing is a planning
// concern, not a kernel-enforced gate; the FSM (proposed → accepted)
// is the only mechanical surface that should constrain status
// transitions, and `aiwf promote` already enforces it.
func TestADR0007_AC1_Allocation(t *testing.T) {
	body := loadADR0007(t)

	if !regexp.MustCompile(`(?m)^id:\s*ADR-0007\s*$`).MatchString(body) {
		t.Error("AC-1: ADR-0007 frontmatter must contain `id: ADR-0007`")
	}
}

// TestADR0007_AC2_PlacementClaims asserts AC-2: the §Placement
// subsection (under §Decision) articulates the rituals-plugin
// placement rule, names the verb-wrapper / planning-conversation
// distinction, and cites the existing-pattern skills the AC spec
// names verbatim.
func TestADR0007_AC2_PlacementClaims(t *testing.T) {
	body := loadADR0007(t)
	section := extractMarkdownSection(body, 3, "Placement")
	if section == "" {
		t.Fatal("AC-2: ADR-0007 must have an `### Placement` subsection under §Decision")
	}

	// Case-insensitive phrase checks: prose may capitalise these
	// at sentence starts (e.g., "Kernel-embedded skills are…"),
	// which is fine — the doctrinal claim is the same regardless.
	mustContain := []string{
		"rituals plugin",
		"kernel-embedded",
		"verb wrapper",          // singular or part of `verb wrappers`
		"planning conversation", // discriminator phrase
	}
	lower := strings.ToLower(section)
	for _, p := range mustContain {
		if !strings.Contains(lower, p) {
			t.Errorf("AC-2: §Placement must contain phrase %q (case-insensitive)", p)
		}
	}

	// Per AC-2 spec text: cite at minimum these plugin-side skills.
	requiredPluginSkills := []string{
		"aiwfx-plan-epic",
		"aiwfx-plan-milestones",
		"aiwfx-start-milestone",
		"aiwfx-wrap-epic",
	}
	for _, s := range requiredPluginSkills {
		if !strings.Contains(section, s) {
			t.Errorf("AC-2: §Placement must cite plugin-side skill %q", s)
		}
	}

	// Per AC-2 spec text: cite at minimum these kernel-embedded skills.
	requiredKernelSkills := []string{
		"aiwf-status",
		"aiwf-history",
	}
	for _, s := range requiredKernelSkills {
		if !strings.Contains(section, s) {
			t.Errorf("AC-2: §Placement must cite kernel-embedded skill %q", s)
		}
	}
}

// TestADR0007_AC3_TieringRule asserts AC-3: the §Tiering subsection
// articulates the pure-skill-first rule, names the deferred kernel
// verb (`aiwf whiteboard`, per the post-correction value), documents
// trigger conditions for promotion, and explicitly cites E-21
// success criterion #7.
func TestADR0007_AC3_TieringRule(t *testing.T) {
	body := loadADR0007(t)
	section := extractMarkdownSection(body, 3, "Tiering")
	if section == "" {
		t.Fatal("AC-3: ADR-0007 must have an `### Tiering` subsection under §Decision")
	}

	// The rule itself (paraphrased phrases that must appear).
	rulePhrases := []string{
		"pure skill",  // either "pure skill" or "pure-skill"; substring covers both via lower-case
		"re-deriving", // matches "re-deriving" or "re-derivation"
		"trigger",     // §Tiering must mention triggers
	}
	for _, p := range rulePhrases {
		if !strings.Contains(section, p) {
			t.Errorf("AC-3: §Tiering must contain phrase %q", p)
		}
	}

	// Deferred verb name (post-correction): aiwf whiteboard, NOT aiwf landscape.
	if !strings.Contains(section, "aiwf whiteboard") {
		t.Error("AC-3: §Tiering must name the deferred kernel verb `aiwf whiteboard`")
	}

	// E-21 success criterion #7 must be cited.
	if !regexp.MustCompile(`(?i)success criter(ion|ia)\s*#?\s*7`).MatchString(section) {
		t.Error("AC-3: §Tiering must cite E-21 success criterion #7")
	}
}

// TestADR0007_AC4_NameWorkedExample asserts AC-4: the §Name
// subsection records the whiteboard fit-rationale (three named
// dimensions) and rejects each named alternative with at least
// one-line rationale prose adjacent to the name.
func TestADR0007_AC4_NameWorkedExample(t *testing.T) {
	body := loadADR0007(t)
	section := extractMarkdownSection(body, 3, "Name")
	if section == "" {
		t.Fatal("AC-4: ADR-0007 must have an `### Name` subsection under §Decision")
	}

	// Whiteboard fit-rationale: three dimensions per AC-4 spec.
	fitDimensions := []string{
		"Ephemerality",
		"Surfacing-not-deciding",
		"Operator-at-the-board",
	}
	for _, dim := range fitDimensions {
		if !strings.Contains(section, dim) {
			t.Errorf("AC-4: §Name must document whiteboard fit-rationale dimension %q", dim)
		}
	}

	// Each rejected name from the AC-4 spec text appears with
	// adjacent rationale (heuristic: the name in backticks within
	// a bullet-list line that has at least 30 chars after the
	// closing backtick).
	rejectedNames := []string{
		"recommend-sequence",
		"landscape",
		"paths",
		"focus",
		"next",
		"survey",
		"synthesise-open-work",
	}
	for _, name := range rejectedNames {
		// Look for `<name>` followed by some rationale text on
		// the same line or the next.
		needle := "`" + name + "`"
		idx := strings.Index(section, needle)
		if idx == -1 {
			t.Errorf("AC-4: §Name must reject candidate %q (in backticks)", name)
			continue
		}
		// Rationale check: the segment after the closing backtick
		// of this token, up to the next bullet or blank line, must
		// be at least 30 chars (a one-line rationale).
		after := section[idx+len(needle):]
		// Rationale ends at the next "\n- " (next bullet) or
		// double newline.
		nextBullet := strings.Index(after, "\n- ")
		nextBlank := strings.Index(after, "\n\n")
		end := len(after)
		if nextBullet > 0 && nextBullet < end {
			end = nextBullet
		}
		if nextBlank > 0 && nextBlank < end {
			end = nextBlank
		}
		rationale := strings.TrimSpace(after[:end])
		if len(rationale) < 30 {
			t.Errorf("AC-4: §Name rejection of %q lacks adjacent rationale (got %d chars: %q)", name, len(rationale), rationale)
		}
	}
}

// TestADR0007_AC5_CrossReferences asserts AC-5: the ADR cites
// ADR-0006 (M-074's skills judgment ADR) and both CLAUDE.md
// principles named in the AC spec text. The complementary-not-
// overlapping framing must appear (somewhere in the body — that
// claim is doctrinal and not section-scoped in the spec).
func TestADR0007_AC5_CrossReferences(t *testing.T) {
	body := loadADR0007(t)

	// ADR-0006 must be cited.
	if !strings.Contains(body, "ADR-0006") {
		t.Error("AC-5: ADR-0007 must cite `ADR-0006` (M-074's skills judgment ADR)")
	}

	// Complementary-not-overlapping framing.
	if !regexp.MustCompile(`(?i)complementary`).MatchString(body) {
		t.Error("AC-5: ADR-0007 must frame its scope as complementary to ADR-0006")
	}
	// The within-a-topic vs across-kernel-plugin distinction.
	if !strings.Contains(body, "within a topic") {
		t.Error("AC-5: ADR-0007 must articulate that ADR-0006 covers granularity *within a topic* (this ADR covers placement/tiering across kernel/plugin)")
	}

	// CLAUDE.md principles cited.
	principle1 := "Kernel functionality must be AI-discoverable"
	principle2 := "framework's correctness must not depend on the LLM"
	if !strings.Contains(body, principle1) {
		t.Errorf("AC-5: ADR-0007 must cite CLAUDE.md principle %q", principle1)
	}
	if !strings.Contains(body, principle2) {
		t.Errorf("AC-5: ADR-0007 must cite CLAUDE.md principle %q (or equivalent phrasing capturing the same principle)", principle2)
	}
}
