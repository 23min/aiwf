package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skillSection reads a shipped skill and returns the body of one
// heading, failing the test when either the file or the heading is
// missing — a locator that silently returns "" would make every
// assertion below pass on an empty string.
//
// The exact-heading match comes from markdownSection, which this
// package already carries; a substring match on the heading would also
// select the skill's `--since` subsection, whose heading names
// provenance too.
func skillSection(t *testing.T, skill, heading string) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), "internal", "skills", "embedded", skill, "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	section := markdownSection(string(raw), heading)
	if section == "" {
		t.Fatalf("%s has no %q section; the locator is stale and every assertion against it "+
			"would be reading an empty string", path, heading)
	}
	return section
}

// TestM0292_SkillNamesTheRatificationPath pins the shipped half of
// M-0292's discoverability claim. The kernel-side half — every
// ratifiable finding's hint naming the remedy — is pinned in
// internal/check by TestRatifiableByAcknowledgment_MatchesWhatRunProvenanceEmits
// and its siblings; this is the surface read by someone working from
// the skill rather than from a check run.
//
// Scoped to the section, per CLAUDE.md §"Substring assertions are not
// structural assertions": the skill names `aiwf acknowledge illegal` in
// five other rows already, so a document-wide grep would pass with this
// section saying nothing.
func TestM0292_SkillNamesTheRatificationPath(t *testing.T) {
	t.Parallel()
	section := skillSection(t, "aiwf-check", "## Provenance findings (errors)")

	// Guard the locator: a region not carrying the codes it should
	// describe means the heading moved and the assertions below are
	// reading the wrong prose.
	for _, code := range []string{"provenance-force-non-human", "provenance-actor-malformed"} {
		if !strings.Contains(section, code) {
			t.Fatalf("the located section does not mention %s; wrong region", code)
		}
	}

	if !strings.Contains(section, "acknowledge illegal") {
		t.Error("the provenance section never names `aiwf acknowledge illegal`. Every fix it lists " +
			"assumes the commit can still be shaped; for one already in history they are impossible " +
			"or require rewriting history, which is what ratification exists to avoid")
	}
	// The scope, stated where the reader decides whether to run it. The
	// phrase has to be one only the positive claim can supply — an
	// assertion the carve-out sentence alone satisfies would pass with
	// the section describing the wrong scope entirely.
	if !strings.Contains(section, "every rule in this table") {
		t.Error("the provenance section names the ratification command without its scope; an " +
			"acknowledgment clears the commit rather than the finding, and a reader who learns " +
			"that afterwards has already run it")
	}
	if !strings.Contains(section, "--for-entity") {
		t.Error("the provenance section does not name the rule the blanket form leaves firing")
	}
}

// TestM0292_AcknowledgeSkillStatesTheExemptionScope pins the surface an
// operator reads *before* running the verb, where the scope of what
// they are about to silence belongs.
func TestM0292_AcknowledgeSkillStatesTheExemptionScope(t *testing.T) {
	t.Parallel()
	section := skillSection(t, "aiwf-acknowledge", "### Exemption semantics")

	if !strings.Contains(section, "aiwf-force-for") {
		t.Fatal("the located section does not describe the force-for exemption; wrong region")
	}
	// Only the widened claim supplies this. Asserting on "provenance-"
	// alone would be satisfied by the carve-out sentence below it, so
	// the section could revert to its pre-M-0292 per-entity-only scope
	// and still pass.
	if !strings.Contains(section, "every rule the provenance history audit raises") {
		t.Error("the exemption-semantics section does not say the blanket ack covers the rules " +
			"the provenance history audit raises; an operator reading it before running the verb " +
			"cannot tell what they are about to silence")
	}
	if !strings.Contains(section, "retires the commit's others too") {
		t.Error("the section states the breadth without its consequence — that a reason written " +
			"about one finding clears the commit's other findings as well")
	}
	if !strings.Contains(section, "--for-entity") {
		t.Error("the section does not name the per-(commit, entity) rule the blanket form leaves firing")
	}
}
