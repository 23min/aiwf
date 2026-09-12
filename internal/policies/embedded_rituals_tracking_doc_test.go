package policies

import (
	"path/filepath"
	"strconv"
	"testing"
)

// TestPolicy_EmbeddedRitualsNoRetiredTrackingDoc runs the ban against the live
// tree, which is what the retirement asserts: no surface aiwf ships instructs
// the v1 separate tracking-doc convention.
func TestPolicy_EmbeddedRitualsNoRetiredTrackingDoc(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyEmbeddedRitualsNoRetiredTrackingDoc)
}

// trackingDocReports runs the ban over a synthetic skills tree and renders each
// violation as `file:line`, so a test can assert where it fired and not only
// that it did. The walk it rides is pinned in shipped_surface_ban_test.go;
// what these tests pin is which lines this ban's own rules catch.
func trackingDocReports(t *testing.T, files map[string]string) []string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		mustWrite(t, filepath.Join(root, rel), content)
	}
	vs, err := PolicyEmbeddedRitualsNoRetiredTrackingDoc(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	got := make([]string, 0, len(vs))
	for _, v := range vs {
		// The two bans are mirror files, so the id is exactly what a copy
		// between them would carry over unchanged; checked on every route
		// rather than once, since a wrong id is invisible in a location.
		if v.Policy != "embedded-rituals-no-retired-tracking-doc" {
			t.Errorf("violation stamped with policy id %q", v.Policy)
		}
		got = append(got, v.File+":"+strconv.Itoa(v.Line)+":"+ruleTag(t, trackingDocBans, v))
	}
	return got
}

// TestEmbeddedRitualsNoRetiredTrackingDoc_ReintroductionRoutes drives each
// shape that restores the retired convention, one line at a time. The verb
// skill is the route the ban could not see while it read the ritual snapshot
// alone: an instruction there is followed exactly as one in a ritual is, since
// which convention an agent adopts depends only on which artifact it opened.
func TestEmbeddedRitualsNoRetiredTrackingDoc_ReintroductionRoutes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		rel  string
		line string
		rule string
	}{
		{
			name: "retired directory named in a ritual",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md",
			line: "Finalize the entry under work/tracking/M-NNNN.md before wrapping.",
			rule: "r0",
		},
		{
			name: "retired directory named in a verb skill",
			rel:  "internal/skills/embedded/aiwf-show/SKILL.md",
			line: "The spec's companion lives at work/tracking/.",
			rule: "r0",
		},
		{
			name: "instruction-shaped mention in a ritual",
			rel:  "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md",
			line: "Finalize the tracking doc before you merge.",
			rule: "r1",
		},
		{
			name: "hyphenated mention in an agent card",
			rel:  "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/builder.md",
			line: "- Append progress to the tracking-doc as each step lands.",
			rule: "r1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := trackingDocReports(t, map[string]string{tc.rel: "# S\n\n" + tc.line + "\n"})
			want := tc.rel + ":3:" + tc.rule
			if len(got) != 1 || got[0] != want {
				t.Errorf("route did not report at %s; got %v", want, got)
			}
		})
	}
}

// TestEmbeddedRitualsNoRetiredTrackingDoc_DirectoryRuleIsCaseSensitive pins the
// one rule here that does not ignore case. It matches a path an agent copies
// verbatim, and the retired directory is lower-case; the phrase rule beside it
// matches prose, where a heading or a sentence start varies the case freely.
// Without this the asymmetry reads as an oversight and tidying it away would
// go unnoticed.
func TestEmbeddedRitualsNoRetiredTrackingDoc_DirectoryRuleIsCaseSensitive(t *testing.T) {
	t.Parallel()
	rel := "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/builder.md"
	line := "An upper-cased WORK/TRACKING/ names no directory anyone creates."
	if got := trackingDocReports(t, map[string]string{rel: "# S\n\n" + line + "\n"}); len(got) != 0 {
		t.Errorf("the directory rule matched a differently-cased path: %v", got)
	}
}

// TestEmbeddedRitualsNoRetiredTrackingDoc_V1ContextPasses pins the one escape:
// naming the convention as history is how the retirement gets stated at all,
// so a line carrying "v1" is describing what is gone rather than instructing
// it. Without the escape the ban would fire on the retirement statements, which
// is how a ban gets switched off.
func TestEmbeddedRitualsNoRetiredTrackingDoc_V1ContextPasses(t *testing.T) {
	t.Parallel()
	rel := "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md"
	line := "The v1 separate tracking doc is gone; AC progress lives in the spec's frontmatter."
	if got := trackingDocReports(t, map[string]string{rel: "# S\n\n" + line + "\n"}); len(got) != 0 {
		t.Errorf("a mention in explicit v1-historical context reported: %v", got)
	}
}

// TestEmbeddedRitualsNoRetiredTrackingDoc_DirectoryShapeIgnoresTheEscape pins
// that naming the retired directory fires whatever else the line says. The
// escape licenses describing the convention, not pointing an agent at a path
// it would then recreate in the consumer repo.
func TestEmbeddedRitualsNoRetiredTrackingDoc_DirectoryShapeIgnoresTheEscape(t *testing.T) {
	t.Parallel()
	rel := "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md"
	line := "The v1 tracking doc lived at work/tracking/M-NNNN.md."
	if got := trackingDocReports(t, map[string]string{rel: "# S\n\n" + line + "\n"}); len(got) != 1 {
		t.Errorf("the directory shape did not fire through the v1 escape; got %v", got)
	}
}
