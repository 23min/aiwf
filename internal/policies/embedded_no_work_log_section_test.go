package policies

import (
	"os"
	"path/filepath"
	"slices"
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
// that it did.
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
		got = append(got, v.File+":"+strconv.Itoa(v.Line))
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
	}{
		{
			name: "heading above the template's ownership map",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/milestone-spec.md",
			line: "## Work log",
		},
		{
			name: "unbackticked instruction in a milestone ritual",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md",
			line: "- Append a Work log entry to the milestone spec.",
		},
		{
			name: "backticked mention in the engineering-skill tree",
			rel:  "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md",
			line: "Write the outcome into the spec's `## Work log`.",
		},
		{
			// The escape is what makes this row pin the body-key rule rather
			// than the mention rule: it suppresses the second, so only the
			// first can report, and dropping `work_log` goes silent.
			name: "body key in a verb skill",
			rel:  "internal/skills/embedded/aiwf-show/SKILL.md",
			line: "If the project reads JSON: aiwf show M-NNNN --format=json | jq '.result.body.work_log'",
		},
		{
			name: "hyphenated mention in an agent card",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/reviewer.md",
			line: "- Specs, with their work-log sections, stay readable.",
		},
		{
			name: "mention in the always-on guidance",
			rel:  "internal/skills/embedded-guidance/aiwf-guidance.md",
			line: "Keep the Work log current as you go.",
		},
		{
			name: "instruction naming a project without the conditional",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md",
			line: "Append a work log entry to the project's milestone spec, one per criterion.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := workLogReports(t, map[string]string{tc.rel: "# S\n\n" + tc.line + "\n"})
			want := tc.rel + ":3"
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

// TestEmbeddedNoWorkLogSection_ScopeIsTheEmbeddedTrees pins that the walk covers
// every embedded tree and nothing else under internal/skills/. A sibling package
// there is aiwf's own source rather than a surface a consumer reads, so a mention
// in it is not a shipped instruction.
func TestEmbeddedNoWorkLogSection_ScopeIsTheEmbeddedTrees(t *testing.T) {
	t.Parallel()
	got := workLogReports(t, map[string]string{
		"internal/skills/materialize.go":                 "package skills\n\n// the ## Work log section\n",
		"internal/skills/materialize_test.go":            "package skills\n\n// work log\n",
		"internal/skills/testdata/golden/spec.md":        "## Work log\n",
		"internal/skills/embedded-statusline/status.sh":  "# renders the ## Work log\n",
		"internal/policies/embedded/x.md":                "## Work log\n",
		"internal/skills/embedded/aiwf-history/SKILL.md": "## Work log\n",
	})
	want := []string{
		"internal/skills/embedded-statusline/status.sh:1",
		"internal/skills/embedded/aiwf-history/SKILL.md:1",
	}
	if len(got) != len(want) {
		t.Fatalf("scope mismatch: got %v, want %v", got, want)
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("expected a report at %s; got %v", w, got)
		}
	}
}

// TestEmbeddedNoWorkLogSection_MissingSkillsTreeIsAnError pins that a root with
// no internal/skills/ reports the walk failure rather than a clean verdict. A
// ban that passes when it read nothing is the failure mode this policy exists to
// prevent, one level up.
func TestEmbeddedNoWorkLogSection_MissingSkillsTreeIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := PolicyEmbeddedNoWorkLogSection(t.TempDir()); err == nil {
		t.Error("a root carrying no internal/skills/ tree returned no error")
	}
}

// TestEmbeddedNoWorkLogSection_UnreadableFileIsAnError pins that a file the walk
// cannot read fails the policy rather than passing it. A ban that reports clean
// on the bytes it never saw is worse than no ban, because the clean verdict is
// what stops the next reader looking.
func TestEmbeddedNoWorkLogSection_UnreadableFileIsAnError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rel := filepath.Join("internal", "skills", "embedded", "aiwf-show", "SKILL.md")
	mustWrite(t, filepath.Join(root, rel), "# S\n")
	// A dangling symlink reads as a non-directory entry the walk yields and
	// every uid fails to open, so the arm is reachable without file modes.
	if err := os.Remove(filepath.Join(root, rel)); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "no-such-target"), filepath.Join(root, rel)); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := PolicyEmbeddedNoWorkLogSection(root); err == nil {
		t.Error("an unreadable shipped surface returned no error")
	}
}
