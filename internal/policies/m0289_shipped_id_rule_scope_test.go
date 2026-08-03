package policies

// M-0289 AC-5: the surfaces a consumer reads say which id rule governs which
// files.
//
// Three of the four id-shape rules are live in a consumer repo — body-prose-id
// over their entity files, doc-id-width and doc-id-slug over their README. Two
// of them disagree about the same token by design: a canonical letter-N
// placeholder is the defect in an entity body and the correct form in a doc.
// A consumer who meets both without being told sees aiwf contradicting itself.
//
// The always-on guidance predates the doc rules and spoke of "committed
// prose", which now reads across both corpora and tells a consumer to strip
// exactly the placeholders the doc rule asks for. Scoping that sentence is the
// correction; the comparison table is what stops the next reader deriving the
// split from four rule-file headers.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// idRuleScopeSection is the heading under which the aiwf-check skill states
// the corpus-to-rule mapping. Named as a constant because the assertion is
// that this section exists, not merely that the words appear somewhere.
const idRuleScopeSection = "Which id rule applies where"

// TestM0289_AC5_GuidanceScopesTheIDBullet pins that the shipped guidance no
// longer states the entity-body rule as though it governed every file.
func TestM0289_AC5_GuidanceScopesTheIDBullet(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "internal", "skills",
		"embedded-guidance", "aiwf-guidance.md"))
	if err != nil {
		t.Fatalf("reading shipped guidance: %v", err)
	}
	body := string(raw)
	// Scope to the operating rules, excluding the code-health appendix, so a
	// stray mention there cannot satisfy this.
	section := extractMarkdownSection(body, 1, "aiwf — standing guidance")
	if !strings.Contains(section, "## Code-health priming") {
		t.Fatalf("the code-health anchor is missing — the section scope would widen to the whole file")
	}
	section = strings.SplitN(section, "## Code-health priming", 2)[0]

	// Narrow to the one bullet under test. Scoping only to the section would
	// let a neighbouring bullet satisfy a general word like "entity", so
	// deleting the scope from THIS bullet would leave the assertion green —
	// measured, not hypothetical.
	bullet := ""
	for _, b := range strings.Split(section, "\n- ") {
		if strings.Contains(b, "fake id-shaped token") {
			bullet = b
			break
		}
	}
	if bullet == "" {
		t.Fatalf("the id-shape bullet is gone from the shipped guidance entirely")
	}

	// Both live corpora and both rule families, named inside the one bullet a
	// consumer reads when deciding what to write.
	for _, phrase := range []string{"body-prose-id", "doc-id-width", "entity files", "README.md", "docs.paths"} {
		if !strings.Contains(bullet, phrase) {
			t.Errorf("the id-shape bullet does not name %q — without it the instruction "+
				"reads across every committed file, and a consumer strips the very "+
				"placeholders the doc rule asks for", phrase)
		}
	}
}

// TestM0289_AC5_CheckSkillMapsCorpusToRule pins the comparison table. The
// skill documents each finding code in isolation, which is the shape that
// leaves a consumer deriving the disagreement themselves.
func TestM0289_AC5_CheckSkillMapsCorpusToRule(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "internal", "skills",
		"embedded", "aiwf-check", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading aiwf-check skill: %v", err)
	}
	section := extractMarkdownSection(string(raw), 2, idRuleScopeSection)
	if strings.TrimSpace(section) == "" {
		t.Fatalf("the aiwf-check skill has no %q section — each code is documented alone, "+
			"so a consumer meeting two of them sees aiwf contradicting itself", idRuleScopeSection)
	}
	// Every rule a consumer can actually hit, named in the one place that
	// compares them.
	for _, code := range []string{"body-prose-id", "doc-id-width", "doc-id-slug"} {
		if !strings.Contains(section, code) {
			t.Errorf("section %q omits %q, which is live in a consumer repo", idRuleScopeSection, code)
		}
	}
}
