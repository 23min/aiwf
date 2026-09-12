package policies

import (
	"path/filepath"
	"strconv"
	"testing"
)

// TestPolicy_EmbeddedNoWorkLogSection runs the ban against the live tree, which
// is what the retirement asserts: no surface aiwf ships names the section.
func TestPolicy_EmbeddedNoWorkLogSection(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyEmbeddedNoWorkLogSection)
}

// workLogReports runs the ban over a synthetic skills tree and renders each
// violation as `file:line`, so a test can assert where it fired and not only
// that it did. The walk it rides is pinned in shipped_surface_ban_test.go;
// what these tests pin is which lines this ban's own rules catch.
func workLogReports(t *testing.T, files map[string]string) []string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		mustWrite(t, filepath.Join(root, rel), content)
	}
	vs, err := PolicyEmbeddedNoWorkLogSection(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	got := make([]string, 0, len(vs))
	for _, v := range vs {
		// The two bans are mirror files, so the id is exactly what a copy
		// between them would carry over unchanged; checked on every route
		// rather than once, since a wrong id is invisible in a location.
		if v.Policy != "embedded-no-work-log-section" {
			t.Errorf("violation stamped with policy id %q", v.Policy)
		}
		got = append(got, v.File+":"+strconv.Itoa(v.Line)+":"+ruleTag(t, workLogBans, v))
	}
	return got
}

// TestEmbeddedNoWorkLogSection_ReintroductionRoutes drives every single-edit
// route measured to restore the retired section while the section-ownership
// policies stay silent. Each is one line in one shipped tree, and each must
// report on its own — a route that only fails in combination with another is a
// route a reintroduction takes.
func TestEmbeddedNoWorkLogSection_ReintroductionRoutes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		rel  string
		line string
		rule string
	}{
		{
			name: "heading above the template's ownership map",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/milestone-spec.md",
			line: "## Work log",
			rule: "r0",
		},
		{
			name: "unbackticked instruction in a milestone ritual",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md",
			line: "- Append a Work log entry to the milestone spec.",
			rule: "r1",
		},
		{
			name: "backticked mention in the engineering-skill tree",
			rel:  "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md",
			line: "Write the outcome into the spec's `## Work log`.",
			rule: "r0",
		},
		{
			// The escape is what makes this row pin the body-key rule rather
			// than the mention rule: it suppresses the second, so only the
			// first can report, and dropping `work_log` goes silent.
			name: "body key in a verb skill",
			rel:  "internal/skills/embedded/aiwf-show/SKILL.md",
			line: "If the project reads JSON: aiwf show M-NNNN --format=json | jq '.result.body.work_log'",
			rule: "r0",
		},
		{
			name: "hyphenated mention in an agent card",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/reviewer.md",
			line: "- Specs, with their work-log sections, stay readable.",
			rule: "r1",
		},
		{
			name: "mention in the always-on guidance",
			rel:  "internal/skills/embedded-guidance/aiwf-guidance.md",
			line: "Keep the Work log current as you go.",
			rule: "r1",
		},
		{
			name: "instruction naming a project without the conditional",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md",
			line: "Append a work log entry to the project's milestone spec, one per criterion.",
			rule: "r1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := workLogReports(t, map[string]string{tc.rel: "# S\n\n" + tc.line + "\n"})
			want := tc.rel + ":3:" + tc.rule
			if len(got) != 1 || got[0] != want {
				t.Errorf("route did not report at %s; got %v", want, got)
			}
		})
	}
}

// TestEmbeddedNoWorkLogSection_ConsumerProjectConditionalPasses pins the one
// escape: a skill asking whether the reader's own project keeps a work log is
// about that project's habit, not about aiwf's retired section. Without the
// escape the ban would fire on the engineering skills, which is how a ban gets
// switched off.
func TestEmbeddedNoWorkLogSection_ConsumerProjectConditionalPasses(t *testing.T) {
	t.Parallel()
	rel := "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-review-code/SKILL.md"
	line := "- If the project keeps a work log or change log alongside the diff, the entry is present."
	if got := workLogReports(t, map[string]string{rel: "# S\n\n" + line + "\n"}); len(got) != 0 {
		t.Errorf("a mention conditional on the reader's own project reported: %v", got)
	}
}

// TestEmbeddedNoWorkLogSection_SectionShapeIgnoresTheEscape pins that naming the
// section as a section fires whatever else the line says. The escape is scoped
// to a sentence about a consumer's own habit, and "the project's `## Work log`"
// is not one — it is the reintroduction, wearing the escape.
func TestEmbeddedNoWorkLogSection_SectionShapeIgnoresTheEscape(t *testing.T) {
	t.Parallel()
	rel := "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md"
	line := "If the project uses aiwf, confirm its `## Work log` carries one entry per criterion."
	if got := workLogReports(t, map[string]string{rel: "# S\n\n" + line + "\n"}); len(got) != 1 {
		t.Errorf("the section shape did not fire through the consumer-project escape; got %v", got)
	}
}
