package policies

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestParseSkillMarkdown_FrontmatterShapes pins the parser against
// the variations actually seen in shipped skills: single-line `name:`
// and `description:` values, multi-line wrapped descriptions, and
// missing frontmatter.
func TestParseSkillMarkdown_FrontmatterShapes(t *testing.T) {
	t.Run("single-line fields", func(t *testing.T) {
		got := parseSkillMarkdown([]byte("---\nname: aiwf-list\ndescription: filter the planning tree\n---\n\n# body\n"))
		if got.frontmatterName != "aiwf-list" {
			t.Errorf("name = %q, want aiwf-list", got.frontmatterName)
		}
		if got.description != "filter the planning tree" {
			t.Errorf("description = %q, want %q", got.description, "filter the planning tree")
		}
		if !strings.Contains(got.body, "# body") {
			t.Errorf("body lost the # body heading: %q", got.body)
		}
	})

	t.Run("missing frontmatter", func(t *testing.T) {
		got := parseSkillMarkdown([]byte("# body only\nno frontmatter at all\n"))
		if got.frontmatterName != "" {
			t.Errorf("name = %q, want empty", got.frontmatterName)
		}
		if got.description != "" {
			t.Errorf("description = %q, want empty", got.description)
		}
	})

	t.Run("description ending without trailing space carries closing words", func(t *testing.T) {
		// Regression-style: an early version of the wrapping accumulator
		// dropped the last word when the wrap had no leading space.
		got := parseSkillMarkdown([]byte("---\nname: aiwf-x\ndescription: line one then line two\n---\n"))
		if got.description != "line one then line two" {
			t.Errorf("description = %q", got.description)
		}
	})
}

// TestBacktickedAiwfMentions_Extraction covers the load-bearing shapes:
// inline-code `aiwf X`, multi-token `aiwf X Y`, flag-shaped `aiwf --v`,
// and the non-aiwf prose that must not match.
func TestBacktickedAiwfMentions_Extraction(t *testing.T) {
	body := "Run `aiwf list` and then `aiwf show <id>`. Also `aiwf list --kind contract`. Don't pick up `aiwf --version` (flag, no verb). Bare `aiwf` alone has no verb.\n\nFenced too:\n```\naiwf history E-01\n```\nAnd a non-aiwf mention `git status` must be skipped."

	got := backtickedAiwfMentions(body)
	gotVerbs := make([]string, len(got))
	for i, m := range got {
		gotVerbs[i] = m.verb
	}

	// Order matches order in body. `aiwf --version` does not match (the
	// regex requires the next token to start with [a-z], so `--` is
	// rejected). Bare `aiwf` (no following word) does not match.
	want := []string{"list", "show", "list", "history"}
	if diff := cmp.Diff(want, gotVerbs); diff != "" {
		t.Errorf("backtickedAiwfMentions mismatch (-want +got):\n%s", diff)
	}
}

// TestBacktickedAiwfMentions_FlagOnlyDoesNotMatch is a focused negative
// case: a backticked `aiwf --flag` reference must not be reported as a
// verb. Otherwise the policy fires false positives on legitimate
// flag-shaped help text.
func TestBacktickedAiwfMentions_FlagOnlyDoesNotMatch(t *testing.T) {
	for _, body := range []string{
		"`aiwf --version`",
		"`aiwf --help`",
		"`aiwf -v`",
	} {
		got := backtickedAiwfMentions(body)
		if len(got) != 0 {
			t.Errorf("body %q should yield no mentions; got %+v", body, got)
		}
	}
}

// TestSkillCoverageAllowlist_HasShowEntry guards the deferred-skill
// invariant: AC-7 of M-074 files a follow-up gap for the absent
// aiwf-show skill, and AC-6 demands the allowlist's `show` entry
// reference that gap by id. A future change that drops the entry or
// the rationale fails here.
func TestSkillCoverageAllowlist_HasShowEntry(t *testing.T) {
	rationale, ok := skillCoverageAllowlist["show"]
	if !ok {
		t.Fatal("skillCoverageAllowlist missing entry for `show` (AC-6)")
	}
	if !strings.Contains(strings.ToLower(rationale), "deferred") {
		t.Errorf("show rationale must mark as deferred; got %q", rationale)
	}
	if !strings.Contains(rationale, "G-") {
		t.Errorf("show rationale must reference the follow-up gap by id; got %q", rationale)
	}
}
