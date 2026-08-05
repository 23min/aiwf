package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestM0292_SkillNamesTheRatificationPath pins the shipped half of
// M-0292's discoverability claim. The kernel-side half — every
// ratifiable finding's hint naming the remedy — is pinned in
// internal/check by TestHintFor_EveryRatifiableProvenanceCodeAdvertisesTheRemedy;
// this is the surface an operator or agent reads when working from the
// skill rather than from a check run.
//
// The assertion is scoped to the provenance section, per CLAUDE.md
// §"Substring assertions are not structural assertions": the skill
// names `aiwf acknowledge illegal` in several other rows already, so a
// document-wide grep would pass with the provenance section saying
// nothing at all.
func TestM0292_SkillNamesTheRatificationPath(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	path := filepath.Join(root, "internal", "skills", "embedded", "aiwf-check", "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	section := provenanceFindingsSection(string(raw))
	if section == "" {
		t.Fatal("could not locate the provenance findings section in the aiwf-check skill; " +
			"the locator below is stale and this test is asserting nothing")
	}
	// Guard the locator: a section that does not carry the codes it is
	// supposed to describe means the slice is wrong, and every assertion
	// below would be reading the wrong prose.
	for _, code := range []string{"provenance-force-non-human", "provenance-actor-malformed"} {
		if !strings.Contains(section, code) {
			t.Fatalf("the located section does not mention %s; the locator is matching the wrong region", code)
		}
	}

	if !strings.Contains(section, "acknowledge illegal") {
		t.Error("the aiwf-check skill's provenance section never names `aiwf acknowledge illegal`. " +
			"Every fix it lists assumes the commit can still be shaped; for one already in history " +
			"they are impossible or require rewriting history, which is what ratification exists to avoid")
	}
	// The blast radius, stated where the reader decides whether to run it.
	if !strings.Contains(section, "every provenance rule") {
		t.Error("the provenance section names the ratification command without its scope; an " +
			"acknowledgment clears the commit rather than the finding, and a reader who learns that " +
			"afterwards has already run it")
	}
	// The carve-out, at the point of the claim rather than left to be
	// discovered by running the blanket form and seeing nothing change.
	if !strings.Contains(section, "--for-entity") {
		t.Error("the provenance section claims the ratification covers every provenance rule without " +
			"naming the per-(commit, entity) exception, which the blanket form does not clear")
	}
}

// TestM0292_AcknowledgeSkillStatesTheExemptionScope pins the other
// shipped surface: the skill an operator reads *before* running the
// verb, where the scope of what they are about to silence belongs.
//
// The section describes the exemption in terms of SHA and entity, so
// its claim has to say both what the blanket form covers — now the
// whole provenance family — and the one rule it does not.
func TestM0292_AcknowledgeSkillStatesTheExemptionScope(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	path := filepath.Join(root, "internal", "skills", "embedded", "aiwf-acknowledge", "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	section := namedSection(string(raw), "exemption semantics")
	if section == "" {
		t.Fatal("could not locate the exemption-semantics section in the aiwf-acknowledge skill; " +
			"the locator is stale and this test is asserting nothing")
	}
	if !strings.Contains(section, "aiwf-force-for") {
		t.Fatal("the located section does not describe the force-for exemption; wrong region")
	}
	if !strings.Contains(section, "provenance-") {
		t.Error("the exemption-semantics section does not say the blanket ack now covers the " +
			"provenance family; an operator reading it before running the verb cannot tell what " +
			"they are about to silence")
	}
	if !strings.Contains(section, "--for-entity") {
		t.Error("the exemption-semantics section does not name the per-(commit, entity) rule the " +
			"blanket form leaves firing")
	}
}

// namedSection returns the body of the heading whose text contains
// name (case-insensitive), up to the next heading of the same or
// shallower level. Returns "" when not found.
func namedSection(doc, name string) string {
	lines := strings.Split(doc, "\n")
	start, depth := -1, 0
	for i, line := range lines {
		if !strings.HasPrefix(line, "#") {
			continue
		}
		hashes := len(line) - len(strings.TrimLeft(line, "#"))
		if start == -1 {
			if strings.Contains(strings.ToLower(line), name) {
				start, depth = i, hashes
			}
			continue
		}
		if hashes <= depth {
			return strings.Join(lines[start:i], "\n")
		}
	}
	if start == -1 {
		return ""
	}
	return strings.Join(lines[start:], "\n")
}

// provenanceFindingsSection returns the body of the skill's
// provenance-findings section: from the heading that introduces it up
// to the next heading of the same or shallower level. Returns "" when
// no such heading is found, which the caller treats as a stale locator
// rather than a pass.
func provenanceFindingsSection(doc string) string {
	lines := strings.Split(doc, "\n")
	start, depth := -1, 0
	for i, line := range lines {
		if !strings.HasPrefix(line, "#") {
			continue
		}
		hashes := len(line) - len(strings.TrimLeft(line, "#"))
		heading := strings.ToLower(line)
		if start == -1 {
			// Both words: the skill also has a `--since` subsection whose
			// heading names provenance, and matching that one would slice
			// a region carrying none of the codes under test.
			if strings.Contains(heading, "provenance") && strings.Contains(heading, "findings") {
				start, depth = i, hashes
			}
			continue
		}
		if hashes <= depth {
			return strings.Join(lines[start:i], "\n")
		}
	}
	if start == -1 {
		return ""
	}
	return strings.Join(lines[start:], "\n")
}
