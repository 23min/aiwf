package policies

import (
	"os"
	"path/filepath"
	"runtime"
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

// TestSkillCoverageAllowlist_AllEntriesHaveRationale enforces M-074
// AC-6's whole-allowlist invariant: every entry carries a non-empty
// rationale string. Per the spec, the allowlist's purpose is making
// each absence visible — an empty value defeats that.
func TestSkillCoverageAllowlist_AllEntriesHaveRationale(t *testing.T) {
	if len(skillCoverageAllowlist) == 0 {
		t.Fatal("skillCoverageAllowlist is empty; expected at least the standard ops/trivial verbs")
	}
	for verb, rationale := range skillCoverageAllowlist {
		if strings.TrimSpace(rationale) == "" {
			t.Errorf("verb %q has empty allowlist rationale (AC-6)", verb)
		}
	}
}

// --- Negative-case tests for AC-2 through AC-5 -------------------------
//
// These exercise the per-check helpers with synthetic inputs that
// should fire specific violations. Without these, the policy's
// "passes against the live tree" test is parsing-coverage, not
// enforcement-coverage (CLAUDE.md §"Substring assertions are not
// structural assertions" / §"Test untested code paths").

// TestCheckSkillFrontmatter_FiresOnEmptyName is M-074 AC-2 negative
// case: a skill with empty `name:` frontmatter must fire a violation
// naming the missing field.
func TestCheckSkillFrontmatter_FiresOnEmptyName(t *testing.T) {
	skills := []embeddedSkillEntry{
		{
			relPath:         "internal/skills/embedded/aiwf-broken/SKILL.md",
			dirName:         "aiwf-broken",
			frontmatterName: "",
			description:     "valid description",
		},
	}
	got := checkSkillFrontmatter(skills)
	mustHaveViolation(t, got, "missing a `name:` frontmatter")
}

// TestCheckSkillFrontmatter_FiresOnEmptyDescription is M-074 AC-2
// negative case: a skill with empty `description:` fires a violation.
func TestCheckSkillFrontmatter_FiresOnEmptyDescription(t *testing.T) {
	skills := []embeddedSkillEntry{
		{
			relPath:         "internal/skills/embedded/aiwf-empty-desc/SKILL.md",
			dirName:         "aiwf-empty-desc",
			frontmatterName: "aiwf-empty-desc",
			description:     "   \t\n   ",
		},
	}
	got := checkSkillFrontmatter(skills)
	mustHaveViolation(t, got, "missing a `description:` frontmatter")
}

// TestCheckSkillFrontmatter_FiresOnNameDirMismatch is M-074 AC-3
// negative case: a skill whose `name:` differs from its directory
// fires a violation citing both values.
func TestCheckSkillFrontmatter_FiresOnNameDirMismatch(t *testing.T) {
	skills := []embeddedSkillEntry{
		{
			relPath:         "internal/skills/embedded/aiwf-foo/SKILL.md",
			dirName:         "aiwf-foo",
			frontmatterName: "aiwf-bar",
			description:     "x",
		},
	}
	got := checkSkillFrontmatter(skills)
	mustHaveViolation(t, got, "does not match its directory")
}

// TestCheckSkillFrontmatter_FiresOnAiwfPrefixMissing is M-074 AC-3
// negative case: a name that doesn't follow the `aiwf-<topic>`
// convention fires a violation. (The dir-mismatch branch fires first
// when both apply, so we set name == dirName to isolate this branch.)
func TestCheckSkillFrontmatter_FiresOnAiwfPrefixMissing(t *testing.T) {
	skills := []embeddedSkillEntry{
		{
			relPath:         "internal/skills/embedded/foo/SKILL.md",
			dirName:         "foo",
			frontmatterName: "foo",
			description:     "x",
		},
	}
	got := checkSkillFrontmatter(skills)
	mustHaveViolation(t, got, "does not match the `aiwf-<topic>` convention")
}

// TestCheckSkillFrontmatter_FiresOnAiwfPrefixOnly is M-074 AC-3 edge
// case: name == "aiwf-" with no topic suffix is also a convention
// violation (the topic carries discovery semantics).
func TestCheckSkillFrontmatter_FiresOnAiwfPrefixOnly(t *testing.T) {
	skills := []embeddedSkillEntry{
		{
			relPath:         "internal/skills/embedded/aiwf-/SKILL.md",
			dirName:         "aiwf-",
			frontmatterName: "aiwf-",
			description:     "x",
		},
	}
	got := checkSkillFrontmatter(skills)
	mustHaveViolation(t, got, "does not match the `aiwf-<topic>` convention")
}

// TestCheckSkillFrontmatter_NoFalsePositive: a fully valid skill
// produces no violations. The negative-case tests above confirm the
// policy fires; this one confirms it doesn't fire too eagerly.
func TestCheckSkillFrontmatter_NoFalsePositive(t *testing.T) {
	skills := []embeddedSkillEntry{
		{
			relPath:         "internal/skills/embedded/aiwf-list/SKILL.md",
			dirName:         "aiwf-list",
			frontmatterName: "aiwf-list",
			description:     "use to filter the planning tree",
		},
	}
	got := checkSkillFrontmatter(skills)
	if len(got) != 0 {
		t.Errorf("valid skill should produce no violations; got: %+v", got)
	}
}

// TestCheckVerbCoverage_FiresOnUncoveredVerb is M-074 AC-4 negative
// case: a top-level verb without a same-named skill and without an
// allowlist entry fires a violation.
func TestCheckVerbCoverage_FiresOnUncoveredVerb(t *testing.T) {
	skills := []embeddedSkillEntry{
		// Only `aiwf-list` ships; `widget` is registered as a verb but
		// has no skill and no allowlist entry.
		{frontmatterName: "aiwf-list"},
	}
	verbs := map[string]string{
		"list":   "newListCmd",
		"widget": "newWidgetCmd",
	}
	allowlist := map[string]string{} // intentionally empty

	got := checkVerbCoverage(skills, verbs, allowlist)
	mustHaveViolation(t, got, "\"widget\" has no embedded skill")
}

// TestCheckVerbCoverage_AllowlistRescuesUncoveredVerb: the same
// scenario, but with the verb in the allowlist, must NOT fire. This
// pins the allowlist's entire purpose.
func TestCheckVerbCoverage_AllowlistRescuesUncoveredVerb(t *testing.T) {
	skills := []embeddedSkillEntry{{frontmatterName: "aiwf-list"}}
	verbs := map[string]string{
		"list":   "newListCmd",
		"widget": "newWidgetCmd",
	}
	allowlist := map[string]string{
		"widget": "ops verb; --help suffices",
	}

	got := checkVerbCoverage(skills, verbs, allowlist)
	if len(got) != 0 {
		t.Errorf("allowlisted verb should not fire; got: %+v", got)
	}
}

// TestCheckVerbCoverage_SkillCoverageRescuesVerb: a verb with a
// same-named skill produces no violation.
func TestCheckVerbCoverage_SkillCoverageRescuesVerb(t *testing.T) {
	skills := []embeddedSkillEntry{{frontmatterName: "aiwf-list"}}
	verbs := map[string]string{"list": "newListCmd"}
	got := checkVerbCoverage(skills, verbs, map[string]string{})
	if len(got) != 0 {
		t.Errorf("skill-covered verb should not fire; got: %+v", got)
	}
}

// TestCheckSkillBodyMentionsResolve_FiresOnUnknownVerb is M-074 AC-5
// negative case: a skill body referencing a non-existent verb fires
// a violation citing both the offending mention and the skill path.
func TestCheckSkillBodyMentionsResolve_FiresOnUnknownVerb(t *testing.T) {
	skills := []embeddedSkillEntry{
		{
			relPath: "internal/skills/embedded/aiwf-fake/SKILL.md",
			body:    "Run `aiwf nosuchverb` to do the thing.",
		},
	}
	verbs := map[string]string{"list": "newListCmd"}

	got := checkSkillBodyMentionsResolve(skills, verbs)
	mustHaveViolation(t, got, "`aiwf nosuchverb`")
}

// TestCheckSkillBodyMentionsResolve_HelpAndCompletionResolve: Cobra
// auto-adds `help` and `completion` at the root. They aren't in
// cmd/aiwf source, so the verb set built by findTopLevelVerbs won't
// have them. The policy adds them at check time; this test pins
// that.
func TestCheckSkillBodyMentionsResolve_HelpAndCompletionResolve(t *testing.T) {
	skills := []embeddedSkillEntry{
		{body: "Try `aiwf help` for the verb list, or `aiwf completion bash` for shell wiring."},
	}
	verbs := map[string]string{} // deliberately no help/completion in registered set

	got := checkSkillBodyMentionsResolve(skills, verbs)
	if len(got) != 0 {
		t.Errorf("`aiwf help` and `aiwf completion` should always resolve; got: %+v", got)
	}
}

// TestCheckSkillBodyMentionsResolve_FencedAndInlineBothChecked: the
// G-061 repro shape was a fenced code block in `aiwf-contract/SKILL.md`
// referencing a non-existent verb. The policy must catch both inline-
// and fenced-code mentions.
func TestCheckSkillBodyMentionsResolve_FencedAndInlineBothChecked(t *testing.T) {
	skills := []embeddedSkillEntry{
		{
			relPath: "skill-with-fenced.md",
			body:    "Inline: `aiwf inlinebogus`. Fenced:\n```\naiwf fencedbogus\n```\n",
		},
	}
	verbs := map[string]string{"list": "newListCmd"}

	got := checkSkillBodyMentionsResolve(skills, verbs)
	if len(got) != 2 {
		t.Fatalf("expected 2 violations (inline + fenced), got %d: %+v", len(got), got)
	}
	mustHaveViolation(t, got, "`aiwf inlinebogus`")
	mustHaveViolation(t, got, "`aiwf fencedbogus`")
}

// TestRunSkillCoverageChecks_FullDriftFiresAllAxes assembles a
// composite synthetic input that triggers every check at once and
// asserts the whole policy fires. This is the integration-test
// counterpart to the per-check tests above — it proves the parts
// compose correctly.
func TestRunSkillCoverageChecks_FullDriftFiresAllAxes(t *testing.T) {
	skills := []embeddedSkillEntry{
		// AC-2: empty name + empty description.
		{
			relPath:         "internal/skills/embedded/aiwf-empty/SKILL.md",
			dirName:         "aiwf-empty",
			frontmatterName: "",
			description:     "",
			body:            "valid body, no aiwf mentions",
		},
		// AC-3: name/dir mismatch.
		{
			relPath:         "internal/skills/embedded/aiwf-foo/SKILL.md",
			dirName:         "aiwf-foo",
			frontmatterName: "aiwf-bar",
			description:     "fine",
			body:            "no mentions",
		},
		// AC-5: body references a non-existent verb.
		{
			relPath:         "internal/skills/embedded/aiwf-list/SKILL.md",
			dirName:         "aiwf-list",
			frontmatterName: "aiwf-list",
			description:     "fine",
			body:            "Run `aiwf phantom` for the thing.",
		},
	}
	// AC-4: `widget` is a registered verb without skill or allowlist.
	verbs := map[string]string{
		"list":   "newListCmd",
		"widget": "newWidgetCmd",
	}
	allowlist := map[string]string{}

	got := runSkillCoverageChecks(skills, verbs, allowlist)
	mustHaveViolation(t, got, "missing a `name:` frontmatter")        // AC-2 (name)
	mustHaveViolation(t, got, "missing a `description:` frontmatter") // AC-2 (description)
	mustHaveViolation(t, got, "does not match its directory")         // AC-3
	mustHaveViolation(t, got, "\"widget\" has no embedded skill")     // AC-4
	mustHaveViolation(t, got, "`aiwf phantom`")                       // AC-5
}

// mustHaveViolation asserts that vs contains at least one Violation
// whose Detail contains needle. Reports the full violation set on
// failure so a regression's diff is human-readable.
func mustHaveViolation(t *testing.T, vs []Violation, needle string) {
	t.Helper()
	for _, v := range vs {
		if strings.Contains(v.Detail, needle) {
			return
		}
	}
	t.Errorf("expected a violation containing %q; got %d violations:\n%+v", needle, len(vs), vs)
}

// TestNoReintroducedDeadVerbForms_ContractsAndSkill is M-072 AC-8's
// future-drift guard. The skill-coverage policy resolves only the
// *first word* after `aiwf` (per its header godoc); `aiwf list
// contracts` would no longer trip it because `list` is now a real
// verb. But the second word `contracts` is still a dead positional —
// the original G-061 drift shape — and the spec for M-072 AC-8 named
// docs/pocv3/plans/contracts-plan.md and aiwf-contract/SKILL.md
// specifically. This test pins the fix.
//
// G-086 extended the watched set to docs/pocv3/contracts.md (a third
// drift-class file M-072 AC-8 didn't reach). The flag-form mentions
// (`aiwf list contracts --drifted` etc.) were deleted outright in the
// same sweep — they were speculative future axes, not today's V1
// surface (M-072 ships `--kind`, `--status`, `--parent`, `--archived`,
// `--format`, `--pretty`).
func TestNoReintroducedDeadVerbForms_ContractsAndSkill(t *testing.T) {
	root := repoRootForFile(t)

	type deadForm struct {
		needle  string
		because string
	}
	dead := []deadForm{
		{
			needle:  "aiwf list contracts",
			because: "G-061: kind is a flag, not a positional. Use `aiwf list --kind contract`.",
		},
	}

	sites := []string{
		"docs/pocv3/plans/contracts-plan.md",
		"internal/skills/embedded/aiwf-contract/SKILL.md",
		"docs/pocv3/contracts.md",
	}

	for _, site := range sites {
		path := filepath.Join(root, site)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", site, err)
			continue
		}
		content := string(data)
		for _, d := range dead {
			if strings.Contains(content, d.needle) {
				t.Errorf("%s contains forbidden form %q (M-072 AC-8 / G-061): %s",
					site, d.needle, d.because)
			}
		}
	}
}

// repoRootForFile returns the repo root from this test file's
// location, mirroring repoRoot() in policies_test.go but local to
// this file so it doesn't depend on the other test's helper.
func repoRootForFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller returned ok=false")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}
