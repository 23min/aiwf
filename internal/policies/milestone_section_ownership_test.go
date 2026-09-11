package policies

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_MilestoneSectionOwnership holds the live tree to the ownership
// rule: every section a spec gains after it is authored has one owning ritual,
// every surface recording that assignment agrees, and the builder card claims
// none of the wrap ritual's sections.
func TestPolicy_MilestoneSectionOwnership(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyMilestoneSectionOwnership)
}

// ownershipTemplate is a milestone template the policy reports clean: a map
// assigning every below-map heading to one of two rituals, an authoring-time
// heading above it that needs no owner, and — above the map — a prose mention of
// a ritual naming a section that is no heading. That last one becomes an
// assignment if the parser stops requiring an owner line's em-dash, and would
// then report a section the template does not carry, so the clean verdict is
// what pins the em-dash rule.
const ownershipTemplate = "# T\n\n" + // 1
	"## Goal\n\n" + // 3
	"## Closes\n\n" + // 5
	"<!-- `aiwfx-start-milestone` states what belongs in `## Not a heading`. -->\n\n" + // 7
	"---\n\n" + // 9
	"<!-- Owners:\n" + // 11
	"       `aiwfx-start-milestone` — `## Closes` (above), `## Field notes`,\n" + // 12
	"                                 `## Deferrals`\n" + // 13
	"       `aiwfx-wrap-milestone`  — `## Release note`, `## Validation`\n\n" + // 14
	"     Prose after the map is not an assignment. -->\n\n" + // 16
	"## Release note\n\n" + // 18
	"## Field notes\n\n" + // 20
	"## Deferrals\n\n" + // 22
	"## Validation\n" // 24

// ownershipRitual is a ritual carrying the same assignment as ownershipTemplate.
// Its owner lines sit on lines 5 and 6.
const ownershipRitual = "# R\n\n" + // 1
	"Owners:\n\n" + // 3
	"- `aiwfx-start-milestone` — `## Closes`, `## Field notes`, `## Deferrals`\n" + // 5
	"- `aiwfx-wrap-milestone` — `## Release note`, `## Validation`\n\n" + // 6
	"Prose below the map.\n" // 8

// ownershipInvertedAssignment swaps `## Validation` and `## Deferrals` between
// the two owners. Applied to every surface at once it is a coherent, if
// different, assignment — which is what tells a card check reading the map apart
// from one recognising a section by name.
var ownershipInvertedAssignment = strings.NewReplacer(
	"`aiwfx-start-milestone` — `## Closes` (above), `## Field notes`,\n"+
		"                                 `## Deferrals`",
	"`aiwfx-start-milestone` — `## Closes` (above), `## Field notes`,\n"+
		"                                 `## Validation`",
	"`aiwfx-wrap-milestone`  — `## Release note`, `## Validation`",
	"`aiwfx-wrap-milestone`  — `## Release note`, `## Deferrals`",
	"`aiwfx-start-milestone` — `## Closes`, `## Field notes`, `## Deferrals`",
	"`aiwfx-start-milestone` — `## Closes`, `## Field notes`, `## Validation`",
	"`aiwfx-wrap-milestone` — `## Release note`, `## Validation`",
	"`aiwfx-wrap-milestone` — `## Release note`, `## Deferrals`",
)

// ownershipFixtureRoot writes a shipped tree the policy reports clean, then
// applies overrides. An override with empty content deletes the file, which is
// how the unreadable cases are built.
func ownershipFixtureRoot(t *testing.T, overrides map[string]string) string {
	t.Helper()
	root := t.TempDir()

	base := map[string]string{
		"templates/milestone-spec.md":           ownershipTemplate,
		"skills/aiwfx-start-milestone/SKILL.md": ownershipRitual,
		"skills/aiwfx-wrap-milestone/SKILL.md":  ownershipRitual,
		"agents/builder.md":                     "# Builder\n\n- Responsibilities.\n- Fill `## Field notes`.\n",
	}
	maps.Copy(base, overrides)

	for rel, content := range base {
		if content == "" {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

// ownershipReports runs the policy over a fixture and renders each violation as
// "file:line — detail", so a case asserts where a report pointed as well as what
// it said. A violation that loses its location still reads as a report, which is
// why the location rides in every expectation rather than in a test of its own.
func ownershipReports(t *testing.T, overrides map[string]string) []string {
	t.Helper()
	vs, err := PolicyMilestoneSectionOwnership(ownershipFixtureRoot(t, overrides))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		if v.Policy != "milestone-section-ownership" {
			t.Errorf("violation carries policy %q, want milestone-section-ownership", v.Policy)
		}
		out = append(out, fmt.Sprintf("%s:%d — %s", v.File, v.Line, v.Detail))
	}
	return out
}

// Where a violation about each surface must point.
var (
	ownershipTemplateRel = filepath.ToSlash(ownershipTemplatePath)
	ownershipBuilderRel  = filepath.ToSlash(ownershipBuilderPath)
	ownershipStartRel    = filepath.ToSlash(ownershipStartPath)
	ownershipWrapRel     = filepath.ToSlash(ownershipWrapPath)
)

func TestPolicyMilestoneSectionOwnership_CleanFixtureReportsNothing(t *testing.T) {
	t.Parallel()
	if got := ownershipReports(t, nil); len(got) != 0 {
		t.Errorf("clean fixture reported %d violations: %v", len(got), got)
	}
}

// TestPolicyMilestoneSectionOwnership_InvertedMapClearsTheCard pins the card
// check to the map rather than to a section name. Every surface assigns
// `## Validation` to the implementing ritual here, so the builder card naming it
// is correct and nothing reports. A check recognising the section by name would
// report anyway.
func TestPolicyMilestoneSectionOwnership_InvertedMapClearsTheCard(t *testing.T) {
	t.Parallel()
	got := ownershipReports(t, map[string]string{
		"templates/milestone-spec.md":           ownershipInvertedAssignment.Replace(ownershipTemplate),
		"skills/aiwfx-start-milestone/SKILL.md": ownershipInvertedAssignment.Replace(ownershipRitual),
		"skills/aiwfx-wrap-milestone/SKILL.md":  ownershipInvertedAssignment.Replace(ownershipRitual),
		"agents/builder.md":                     "# Builder\n\n- Responsibilities.\n- Fill `## Validation` as you work.\n",
	})
	if len(got) != 0 {
		t.Errorf("a coherent inverted assignment reported: %v", got)
	}
}

func TestPolicyMilestoneSectionOwnership_Fires(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		overrides map[string]string
		want      string
	}{
		{
			// The defect G-0636 measured: a section the wrap ritual owns, named
			// in the card the implementing agent reads. The mention sits on line
			// 4, so the report has to carry the line and not the file alone.
			name:      "builder card claims a wrap-owned section",
			overrides: map[string]string{"agents/builder.md": "# Builder\n\n- Responsibilities.\n- Maintain `## Validation` as you work.\n"},
			want:      ownershipBuilderRel + `:4 — the builder card names "## Validation", which the template's map assigns to "aiwfx-wrap-milestone"`,
		},
		{
			// Reported by owner rather than by section name; the inverted-map
			// test above is the other half of this pair.
			name:      "builder card claims a different wrap-owned section",
			overrides: map[string]string{"agents/builder.md": "# Builder\n\n- Write `## Release note` while you work.\n"},
			want:      ownershipBuilderRel + `:3 — the builder card names "## Release note", which the template's map assigns to "aiwfx-wrap-milestone"`,
		},
		{
			// A reassignment in one surface only. This is what the check exists
			// for: nothing else notices a section changing hands.
			name: "a ritual reassigns a section the template assigns elsewhere",
			overrides: map[string]string{
				"skills/aiwfx-wrap-milestone/SKILL.md": strings.Replace(ownershipRitual,
					"- `aiwfx-wrap-milestone` — `## Release note`, `## Validation`",
					"- `aiwfx-wrap-milestone` — `## Validation`", 1),
			},
			want: ownershipWrapRel + `:6 — the milestone template's map assigns "## Release note" to "aiwfx-wrap-milestone", but this ritual's copy omits it`,
		},
		{
			name: "a ritual gives a section a different owner",
			overrides: map[string]string{
				"skills/aiwfx-start-milestone/SKILL.md": strings.Replace(ownershipRitual,
					"- `aiwfx-start-milestone` — `## Closes`, `## Field notes`, `## Deferrals`",
					"- `aiwfx-start-milestone` — `## Closes`, `## Field notes`, `## Deferrals`, `## Validation`", 1),
			},
			want: ownershipStartRel + `:5 — this ritual assigns "## Validation" to "aiwfx-start-milestone" while the milestone template's map assigns it to "aiwfx-wrap-milestone"`,
		},
		{
			// A hyphen where the em-dash belongs stops the line opening an
			// entry, so its sections fold into the owner above it.
			name: "an owner line that assigns nothing reports",
			overrides: map[string]string{
				"skills/aiwfx-wrap-milestone/SKILL.md": strings.Replace(ownershipRitual,
					"- `aiwfx-wrap-milestone` — ", "- `aiwfx-wrap-milestone` - ", 1),
			},
			want: ownershipWrapRel + `:6 — this line names "aiwfx-wrap-milestone" inside the ownership map but assigns nothing`,
		},
		{
			name: "a section below the map has no owner",
			overrides: map[string]string{
				"templates/milestone-spec.md": ownershipTemplate + "\n## Reviewer notes\n",
			},
			want: ownershipTemplateRel + `:26 — section "## Reviewer notes" is filled after the spec is authored but the map assigns it no owner`,
		},
		{
			// The map ends on its last owner line, so a heading on the line
			// right after the comment closes is already below it.
			name: "an unowned heading immediately below the map reports",
			overrides: map[string]string{
				"templates/milestone-spec.md": strings.Replace(ownershipTemplate,
					"     Prose after the map is not an assignment. -->\n\n## Release note",
					"     Prose after the map is not an assignment. -->\n## Reviewer notes\n\n## Release note", 1),
			},
			want: ownershipTemplateRel + `:17 — section "## Reviewer notes" is filled after the spec is authored but the map assigns it no owner`,
		},
		{
			name: "the map claims a heading the template dropped",
			overrides: map[string]string{
				"templates/milestone-spec.md": strings.Replace(ownershipTemplate, "## Deferrals\n\n## Validation", "## Validation", 1),
			},
			want: ownershipTemplateRel + `:13 — the map assigns "## Deferrals" to "aiwfx-start-milestone", but the template carries no such heading`,
		},
		{
			name: "both rituals claim the same section",
			overrides: map[string]string{
				"templates/milestone-spec.md": strings.Replace(ownershipTemplate,
					"`aiwfx-wrap-milestone`  — `## Release note`, `## Validation`",
					"`aiwfx-wrap-milestone`  — `## Release note`, `## Validation`, `## Field notes`", 1),
			},
			want: ownershipTemplateRel + `:14 — section "## Field notes" is claimed by both "aiwfx-start-milestone" and "aiwfx-wrap-milestone"`,
		},
		{
			name:      "the map names a ritual that ships no skill",
			overrides: map[string]string{"skills/aiwfx-wrap-milestone/SKILL.md": ""},
			want:      ownershipTemplateRel + `:14 — the map assigns sections to "aiwfx-wrap-milestone", which ships no skill directory`,
		},
		{
			name:      "the template is unreadable",
			overrides: map[string]string{"templates/milestone-spec.md": ""},
			want:      ownershipTemplateRel + ":0 — the milestone template is unreadable",
		},
		{
			name:      "the template carries no map",
			overrides: map[string]string{"templates/milestone-spec.md": "# T\n\n## Goal\n"},
			want:      ownershipTemplateRel + ":0 — the milestone template carries no ownership map",
		},
		{
			name:      "a ritual carries no map",
			overrides: map[string]string{"skills/aiwfx-start-milestone/SKILL.md": "# R\n\nNo map here.\n"},
			want:      ownershipStartRel + ":0 — the aiwfx-start-milestone ritual carries no ownership map",
		},
		{
			name:      "the builder card is unreadable",
			overrides: map[string]string{"agents/builder.md": ""},
			want:      ownershipBuilderRel + ":0 — the builder card is unreadable",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ownershipReports(t, tc.overrides)
			matched := 0
			for _, r := range got {
				if strings.Contains(r, tc.want) {
					matched++
				}
			}
			if matched != 1 {
				t.Errorf("want exactly one report containing %q, got %d of %v", tc.want, matched, got)
			}
		})
	}
}

// TestPolicyMilestoneSectionOwnership_MapEndsAtProse pins the map's boundary: a
// section named in prose below the map is not read as an assignment, so it
// cannot silently supply the owner a real heading is missing.
func TestPolicyMilestoneSectionOwnership_MapEndsAtProse(t *testing.T) {
	t.Parallel()
	tmpl := strings.Replace(ownershipTemplate,
		"     Prose after the map is not an assignment. -->",
		"     Prose naming `## Reviewer notes` is not an assignment. -->", 1) + "\n## Reviewer notes\n"
	got := ownershipReports(t, map[string]string{"templates/milestone-spec.md": tmpl})
	for _, r := range got {
		if strings.Contains(r, `section "## Reviewer notes" is filled after the spec is authored but the map assigns it no owner`) {
			return
		}
	}
	t.Errorf("prose mention was read as an assignment; got %v", got)
}

// TestPolicyMilestoneSectionOwnership_OwnerNamedTwiceKeepsBothLines covers a map
// splitting one ritual's sections across two owner lines. The second line
// extends the entry rather than replacing it, or the first line's sections lose
// their owner and report as unowned.
func TestPolicyMilestoneSectionOwnership_OwnerNamedTwiceKeepsBothLines(t *testing.T) {
	t.Parallel()
	tmpl := strings.Replace(ownershipTemplate,
		"       `aiwfx-start-milestone` — `## Closes` (above), `## Field notes`,\n"+
			"                                 `## Deferrals`\n",
		"       `aiwfx-start-milestone` — `## Closes` (above), `## Field notes`\n"+
			"       `aiwfx-start-milestone` — `## Deferrals`\n", 1)
	if got := ownershipReports(t, map[string]string{"templates/milestone-spec.md": tmpl}); len(got) != 0 {
		t.Errorf("a ritual named on two owner lines lost one line's sections: %v", got)
	}
}

// TestPolicyMilestoneSectionOwnership_ProseAboveTheMapOpensNothing pins the
// requirement that an owner line assign a section. A ritual handing off to
// another names it with the same dash, and reading that as the map makes every
// section below look missing from a copy sitting untouched a few lines away.
func TestPolicyMilestoneSectionOwnership_ProseAboveTheMapOpensNothing(t *testing.T) {
	t.Parallel()
	for _, prose := range []string{
		"Hand off to `aiwfx-wrap-milestone` — it runs the review.\n\n",
		"```\nSee `aiwfx-wrap-milestone` — the wrap ritual.\n```\n\n",
	} {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			ritual := strings.Replace(ownershipRitual, "Owners:\n\n", prose+"Owners:\n\n", 1)
			if got := ownershipReports(t, map[string]string{"skills/aiwfx-start-milestone/SKILL.md": ritual}); len(got) != 0 {
				t.Errorf("prose above the map was read as an owner line: %v", got)
			}
		})
	}
}

// TestPolicyMilestoneSectionOwnership_SecondMapReports covers a surface carrying
// two maps. Only the first is read, so the second is compared against nothing
// and can disagree with it in silence.
func TestPolicyMilestoneSectionOwnership_SecondMapReports(t *testing.T) {
	t.Parallel()
	second := "- `aiwfx-start-milestone` — `## Validation`\n- `aiwfx-wrap-milestone` — `## Field notes`\n"
	ritual := ownershipRitual + "\n" + second
	got := ownershipReports(t, map[string]string{"skills/aiwfx-wrap-milestone/SKILL.md": ritual})
	requireReport(t, got, ownershipWrapRel+":10 — the ownership map appears more than once in this surface")
}

// TestMilestoneSectionOwnership_LiveMapAssignsValidationToTheWrapRitual keeps
// the clean verdict on the live tree from being vacuous, and pins the assignment
// the gap this policy serves was opened over.
//
// `## Validation` was claimed as an in-flight section by the builder card and as
// a wrap-time one by the template, so a spec was filled differently depending on
// which surface an agent had loaded. The answer is read out of the live map
// rather than asserted about its wording: rewording keeps this green, and
// reassigning the section reddens it.
func TestMilestoneSectionOwnership_LiveMapAssignsValidationToTheWrapRitual(t *testing.T) {
	t.Parallel()
	m := parseOwnershipMap(liveFile(t, ownershipTemplatePath))
	if m.end == 0 {
		t.Fatal("the live template's ownership map did not parse, so the clean verdict proves nothing")
	}
	got := map[string]string{}
	for name, entry := range m.owners {
		for _, s := range entry.sections {
			got[s.name] = name
		}
	}
	if got["Validation"] != "aiwfx-wrap-milestone" {
		t.Errorf("the live map assigns %q to %q, want aiwfx-wrap-milestone — it is pasted at wrap, and nothing during implementation writes it", "## Validation", got["Validation"])
	}
}

// TestMilestoneSectionOwnership_LiveSurfacesCatchTheirDefects measures, against
// the live surfaces rather than synthetic ones, that the two defects this policy
// was built for report.
func TestMilestoneSectionOwnership_LiveSurfacesCatchTheirDefects(t *testing.T) {
	t.Parallel()
	live := map[string]string{
		"templates/milestone-spec.md":           liveFile(t, ownershipTemplatePath),
		"skills/aiwfx-start-milestone/SKILL.md": liveFile(t, ownershipStartPath),
		"skills/aiwfx-wrap-milestone/SKILL.md":  liveFile(t, ownershipWrapPath),
		"agents/builder.md":                     liveFile(t, ownershipBuilderPath),
	}

	t.Run("builder card claiming a wrap-owned section", func(t *testing.T) {
		t.Parallel()
		o := maps.Clone(live)
		o["agents/builder.md"] = "# Builder\n\n- Maintain the spec's in-flight sections — `## Validation`.\n"
		requireReport(t, ownershipReports(t, o), ownershipBuilderRel+`:3 — the builder card names "## Validation"`)
	})

	// A wrap-owned section moved out of the template's map, and nowhere else.
	// Before the rituals carried their own copy this passed in silence, since
	// nothing else recorded where the section belonged.
	t.Run("a wrap-owned section dropped from the template's map alone", func(t *testing.T) {
		t.Parallel()
		o := maps.Clone(live)
		mutated := strings.Replace(live["templates/milestone-spec.md"],
			"`## Release note`, `## Validation`", "`## Validation`", 1)
		if mutated == live["templates/milestone-spec.md"] {
			t.Fatal("the live map no longer carries the owner line this mutation edits")
		}
		o["templates/milestone-spec.md"] = mutated
		reports := ownershipReports(t, o)
		requireReport(t, reports, `this ritual assigns "## Release note" to "aiwfx-wrap-milestone", but the milestone template's map does not carry that section at all`)
		requireReport(t, reports, ownershipWrapRel)
	})
}

// requireReport fails unless some report contains want.
func requireReport(t *testing.T, got []string, want string) {
	t.Helper()
	for _, r := range got {
		if strings.Contains(r, want) {
			return
		}
	}
	t.Errorf("no report contained %q; got %v", want, got)
}

// liveFile reads a shipped surface, so a test can put the real bytes in front of
// the policy instead of a synthetic stand-in.
func liveFile(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel))) //nolint:gosec // path is a compile-time constant joined to the repo root
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}
