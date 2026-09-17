package policies

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// A ritual that spells the gap-filing verb is the surface in hand when a
// defect surfaces mid-wrap or mid-planning, so it is what decides whether the
// gap-authoring ritual is reached. Spelling the verb without naming the ritual
// routes the reader past it — not by contradicting it, but by being the
// instruction they are already following.
//
// Scope is the line. A reader acts on the instruction in front of them, and
// the line carrying the verb is the smallest unit that reliably contains it;
// anything wider admits a routing clause that sits too far away to be read as
// part of the same instruction.
//
// A fenced example is a line like any other, so a ritual showing the bare
// command in a code block routes nobody and fails here. Name the ritual in the
// sentence that introduces the block and put the command inline, or carry it in
// a comment on the command line.
//
// The population is rituals, not every shipped surface that spells the verb. A
// ritual instructs a reader to file; the verb's own skill documents the verb,
// spelling it six times across examples and a flag table, and owing a pointer
// once rather than on every line. Widening this to that surface would demand the
// ritual's name beside each example, which is the check dictating prose.
//
// This is a ban, not a mandate. It obliges no ritual to spell the verb, so a
// ritual that routes in prose alone is outside the population and correct to
// be there. That is also the limit worth stating: an instruction phrased
// without the verb ("or open a gap") is invisible here, and keeping those
// routed is a review obligation this check does not carry.
const (
	ritualSkillsRoot = "internal/skills/embedded-rituals"
	gapFilingVerb    = "aiwf add gap"
	gapRitualDirName = "aiwfx-record-gap"
)

// gapRitualName reads the gap-authoring ritual's declared name from its own
// frontmatter, so a rename turns every stale handoff red rather than leaving
// this check matching a name nothing answers to.
func gapRitualName(t *testing.T, root string) string {
	t.Helper()
	skillFile := filepath.Join(root, ritualSkillsRoot, "plugins", "aiwf-extensions", "skills", gapRitualDirName, "SKILL.md")
	body, err := os.ReadFile(skillFile) //nolint:gosec // a repo-relative shipped source
	if err != nil {
		t.Fatalf("reading %s/SKILL.md for the ritual's declared name: %v", gapRitualDirName, err)
	}
	name := frontmatterField(string(body), "name")
	if name == "" {
		t.Fatalf("%s/SKILL.md declares no `name:`; nothing can route to it", gapRitualDirName)
	}
	return name
}

// TestRitualsInstructingGapFilingRouteToTheRitual fails when a ritual spells
// the gap-filing verb on a line that does not name the gap-authoring ritual.
// The ritual's own SKILL.md is excluded: it spells the verb because it runs
// it, and it cannot route to itself.
func TestRitualsInstructingGapFilingRouteToTheRitual(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	ritual := gapRitualName(t, root)

	err := walkShippedMarkdown(root, []string{ritualSkillsRoot}, func(rel, content string) {
		if path.Base(rel) != "SKILL.md" || path.Base(path.Dir(rel)) == gapRitualDirName {
			return
		}
		for n, line := range strings.Split(content, "\n") {
			if !strings.Contains(line, gapFilingVerb) || strings.Contains(line, ritual) {
				continue
			}
			t.Errorf("%s:%d spells %q without naming %q on the same line, so a reader acting on that "+
				"instruction files a gap without the ritual that opens the template and reproduces the claim",
				rel, n+1, gapFilingVerb, ritual)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}
