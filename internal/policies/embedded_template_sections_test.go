package policies

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/skills"
)

// TestEmbeddedTemplateCarriesRequiredSectionsAtTopLevel pins that every prose
// template aiwf ships carries its kind's required sections as `## ` headings,
// so a body drafted by filling the template in satisfies the same rule as one
// `aiwf add` scaffolds.
//
// Containment, not equality: the templates are a superset by design, carrying
// commentary and optional sections a scaffold has no business writing. Only
// the required set is asserted present.
//
// Heading level is part of containment. entity.ParseBodySections — the same
// production parser behind `aiwf show --format=json` and the entity-body-empty
// rule — matches `## ` alone, so a required section nested at `### ` is
// invisible to it and fails here. That is not a stylistic preference: a body
// following such a template yields no key for the section on any read path.
//
// Both sides derive: the required set is read from entity.RequiredSections,
// and the kind is resolved from the template's own placeholder id through the
// kernel's id-prefix table. Adding a section to a kind's set fails this test
// until the template follows, with no second list to maintain here.
func TestEmbeddedTemplateCarriesRequiredSectionsAtTopLevel(t *testing.T) {
	t.Parallel()
	templates, err := skills.ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("no embedded templates found; expected the shipped entity templates")
	}
	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()
			e, err := entity.Parse(tmpl.Name, tmpl.Content)
			if err != nil {
				t.Fatalf("template %s does not parse: %v", tmpl.Name, err)
			}
			kind, ok := kindForTemplatePlaceholderID(e.ID)
			if !ok {
				t.Fatalf("template %s carries placeholder id %q matching no kind's id prefix", tmpl.Name, e.ID)
			}
			required := entity.RequiredSections(kind)
			if len(required) == 0 {
				t.Fatalf("template %s resolves to kind %q, which names no required sections", tmpl.Name, kind)
			}
			_, body, split := entity.Split(tmpl.Content)
			if !split {
				t.Fatalf("template %s carries no frontmatter delimiter", tmpl.Name)
			}
			topLevel := entity.ParseBodySections(body)
			var missing []string
			for _, want := range required {
				if _, present := topLevel[entity.SectionSlug(want)]; !present {
					missing = append(missing, want)
				}
			}
			if len(missing) > 0 {
				t.Errorf("template %s (kind %s) omits required section(s) %v at `## ` level; a body filled from it yields no key for them on any read path",
					tmpl.Name, kind, missing)
			}
		})
	}
}

// TestEmbeddedTemplateDoesNotRequireTitleHeading pins that no shipped template
// presents the opening `# <id> — <title>` heading as mandatory. A template
// satisfies the rule either way: by omitting the heading, or by carrying it with
// an optionality marking above the first section.
//
// The kernel already treats the heading as optional and `aiwf retitle` implements
// that — it syncs a canonical heading when present, no-ops when absent, and leaves
// an operator-shaped one alone. A template that opens with the heading and says
// nothing tells its reader the opposite.
//
// One rule rather than a per-kind list, so which side a template satisfies stays
// an editorial choice this test does not encode. The choices made here follow the
// tree: no epic or milestone carries the heading, so those templates omit it; a
// quarter of ADRs and decisions carry a canonical one, so those keep it and mark it.
//
// The search is scoped to the region between the heading and the first `## `,
// which is where a marking about the heading belongs. A file-wide search for the
// word would pass on any template mentioning optionality anywhere.
func TestEmbeddedTemplateDoesNotRequireTitleHeading(t *testing.T) {
	t.Parallel()
	templates, err := skills.ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("no embedded templates found; expected the shipped entity templates")
	}
	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()
			_, body, ok := entity.Split(tmpl.Content)
			if !ok {
				t.Fatalf("template %s carries no frontmatter delimiter", tmpl.Name)
			}
			preamble, hasHeading := titleHeadingPreamble(body)
			if !hasHeading {
				return // omitting the heading satisfies the rule outright
			}
			if !strings.Contains(strings.ToLower(preamble), "optional") {
				t.Errorf("template %s opens with a title heading and never says it is optional; a reader filling it in cannot tell the heading is theirs to skip, though the kernel and aiwf retitle both treat it as optional", tmpl.Name)
			}
		})
	}
}

// TestTitleHeadingPreamble pins the region the marking assertion searches. The
// bound at the first `## ` is what makes the assertion structural: an optionality
// note anywhere below it describes a section, not the title heading.
func TestTitleHeadingPreamble(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, body, want string
		wantHeading      bool
	}{
		{"heading then section", "# T\nnote\n\n## Goal\nprose", "note\n", true},
		{"no heading", "## Goal\nprose", "", false},
		{"heading with no section", "# T\nnote", "note", true},
		{"section text below the bound is outside the region", "# T\n\n## Goal\noptional", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, hasHeading := titleHeadingPreamble([]byte(c.body))
			if got != c.want || hasHeading != c.wantHeading {
				t.Errorf("titleHeadingPreamble(%q) = (%q, %v), want (%q, %v)", c.body, got, hasHeading, c.want, c.wantHeading)
			}
		})
	}
}

// titleHeadingPreamble returns the text between a body's `# ` title heading and
// its first `## ` section, and whether such a heading is present at all.
func titleHeadingPreamble(body []byte) (string, bool) {
	var preamble []string
	seen := false
	for line := range strings.SplitSeq(string(body), "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			if seen {
				return strings.Join(preamble, "\n"), true
			}
		case strings.HasPrefix(line, "# "):
			seen = true
		default:
			if seen {
				preamble = append(preamble, line)
			}
		}
	}
	return strings.Join(preamble, "\n"), seen
}

// kindForTemplatePlaceholderID resolves a shipped template's placeholder id
// (`E-NNNN`, `ADR-NNNN`, …) to its kind through the kernel's id-prefix table.
//
// entity.KindFromID does not serve here: it matches the full id pattern, which
// requires digits, so every template placeholder resolves to no kind. Prefix
// matching is what a placeholder supports.
//
// First match wins: the six prefixes are pairwise non-shadowing, and
// entity.TestIDPrefix_AllKinds pins each literal, so a change introducing a
// shadowing prefix fails there first.
func kindForTemplatePlaceholderID(id string) (entity.Kind, bool) {
	for _, k := range entity.AllKinds() {
		if prefix := entity.IDPrefix(k); prefix != "" && strings.HasPrefix(id, prefix) {
			return k, true
		}
	}
	return "", false
}

// TestKindForTemplatePlaceholderID pins the resolution the section assertion
// above depends on. A placeholder resolving to the wrong kind would compare a
// template against another kind's section set and could pass vacuously, so the
// mapping is asserted directly rather than inferred from the templates that
// happen to ship.
func TestKindForTemplatePlaceholderID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id     string
		want   entity.Kind
		wantOK bool
	}{
		{"E-NNNN", entity.KindEpic, true},
		{"M-NNNN", entity.KindMilestone, true},
		{"ADR-NNNN", entity.KindADR, true},
		{"G-NNNN", entity.KindGap, true},
		{"D-NNNN", entity.KindDecision, true},
		{"C-NNNN", entity.KindContract, true},
		{"Q-NNNN", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			t.Parallel()
			got, ok := kindForTemplatePlaceholderID(c.id)
			if got != c.want || ok != c.wantOK {
				t.Errorf("kindForTemplatePlaceholderID(%q) = (%q, %v), want (%q, %v)", c.id, got, ok, c.want, c.wantOK)
			}
		})
	}
}
