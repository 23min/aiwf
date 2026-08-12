package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/skills"
)

// showEnvelope models only the fields this test asserts on. The parse fails if
// the binary emits anything but a well-formed envelope carrying a body map.
type showEnvelope struct {
	Result struct {
		Body map[string]string `json:"body"`
	} `json:"result"`
}

// templateDraft describes one kind's route from shipped template bytes to a
// real entity file. The add invocation genuinely differs per kind — a milestone
// needs a parent epic and a TDD policy, the born-complete kinds refuse a bare
// scaffold and so must be created with --body-file — so the per-kind arguments
// are setup, not a restatement of any section set.
type templateDraft struct {
	id       string
	kind     entity.Kind
	template string
	addArgs  []string
}

// TestTemplateDraftedEntity_BodyKeysNameSections pins what a consumer reads back
// after drafting an entity from a shipped prose template: the body keys in
// `aiwf show --format=json` name the template's sections, and nothing else.
//
// Two claims, both end-to-end. Every section the kind's owned set names is
// present as a key — the property AC-1's flattening produces, asserted here over
// a real file through `aiwf add`, the loader, and the envelope rather than over
// template bytes in memory. And the full key set equals the template's section
// names, which is what fails while a heading carries a parenthetical.
//
// SectionSlug folds a heading's whole text into its key, so `## Risks (optional)`
// yields `risks_optional`. That parenthetical is authoring guidance — whether to
// keep the section, which tool the vocabulary belongs to — and not part of the
// section's name. A key carrying it names the guidance, and two entities of one
// kind disagree on their key set according to whether each author deleted the
// parenthetical. The expected set below therefore derives from each heading's
// name with any parenthetical qualifier removed, so a template that still ships
// one fails here.
//
// Scope is deliberately the shipped templates, not the tree. An author writing
// `## Risks (to weigh at the ADR)` on a gap has written a heading, not a marker;
// live entities carry such headings legitimately and this test never sees them.
func TestTemplateDraftedEntity_BodyKeysNameSections(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	bodies := templateBodiesByName(t)
	root := newTemplateDraftRepo(t)
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	// Ordered: the milestone's parent epic must exist before it is added.
	drafts := []templateDraft{
		{"E-0001", entity.KindEpic, "epic-spec.md", []string{"add", "epic", "--title", "Platform"}},
		{"M-0001", entity.KindMilestone, "milestone-spec.md", []string{"add", "milestone", "--title", "Cache layer", "--epic", "E-0001", "--tdd", "none"}},
		{"ADR-0001", entity.KindADR, "adr.md", []string{"add", "adr", "--title", "Use one queue"}},
		{"D-0001", entity.KindDecision, "decision.md", []string{"add", "decision", "--title", "Pick a runner"}},
	}

	for _, d := range drafts {
		body, ok := bodies[d.template]
		if !ok {
			t.Fatalf("shipped templates carry no %s", d.template)
		}
		bodyPath := filepath.Join(root, d.template)
		if err := os.WriteFile(bodyPath, body, 0o644); err != nil {
			t.Fatalf("writing %s body: %v", d.template, err)
		}
		args := append(append([]string{}, d.addArgs...), "--body-file", bodyPath)
		if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("aiwf %v: %v\n%s", args, err, out)
		}
	}

	for _, d := range drafts {
		t.Run(string(d.kind), func(t *testing.T) {
			t.Parallel()
			out, err := testutil.RunBin(t, root, binDir, nil, "show", d.id, "--format=json")
			if err != nil {
				t.Fatalf("aiwf show %s: %v\n%s", d.id, err, out)
			}
			var env showEnvelope
			if err := json.Unmarshal([]byte(out), &env); err != nil {
				t.Fatalf("parsing show envelope for %s: %v\n%s", d.id, err, out)
			}

			for _, want := range entity.RequiredSections(d.kind) {
				if _, present := env.Result.Body[entity.SectionSlug(want)]; !present {
					t.Errorf("%s drafted from %s: show JSON carries no key for required section %q",
						d.id, d.template, want)
				}
			}

			wantKeys := sectionNameSlugs(bodies[d.template])
			gotKeys := sortedKeys(env.Result.Body)
			if strings.Join(wantKeys, ",") != strings.Join(gotKeys, ",") {
				t.Errorf("%s drafted from %s: body keys do not name the template's sections\n got: %v\nwant: %v",
					d.id, d.template, gotKeys, wantKeys)
			}
		})
	}
}

// templateBodiesByName returns each shipped template's body bytes (frontmatter
// stripped), keyed by template filename.
func templateBodiesByName(t *testing.T) map[string][]byte {
	t.Helper()
	templates, err := skills.ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	out := make(map[string][]byte, len(templates))
	for _, tmpl := range templates {
		_, body, ok := entity.Split(tmpl.Content)
		if !ok {
			t.Fatalf("template %s carries no frontmatter delimiter", tmpl.Name)
		}
		out[tmpl.Name] = body
	}
	return out
}

// sectionNameSlugs returns the sorted, deduplicated slugs of a template body's
// `## ` section names — each heading with any parenthetical qualifier removed,
// since a qualifier is guidance to the author rather than part of the section's
// name.
//
// Deduplicated because the body map it is compared against is keyed by slug and
// collapses duplicates. Two headings that slug alike would otherwise differ in
// length alone and fail with a message about naming the wrong sections.
func sectionNameSlugs(body []byte) []string {
	seen := map[string]bool{}
	var slugs []string
	for line := range strings.SplitSeq(string(body), "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		slug := entity.SectionSlug(stripParenthetical(strings.TrimPrefix(line, "## ")))
		if seen[slug] {
			continue
		}
		seen[slug] = true
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

// TestStripParenthetical pins the expected-set derivation the assertion above
// rests on. Once the templates carry no qualifier the stripping arm stops being
// reached by the drafted entities, yet it is the whole guard: neutered, a
// re-added `(optional)` makes the expected and actual key sets agree on the
// leaked key and the check passes silently.
func TestStripParenthetical(t *testing.T) {
	t.Parallel()
	cases := []struct{ heading, want string }{
		{"Risks (optional)", "Risks"},
		{"Status vocabulary (aiwf)", "Status vocabulary"},
		{"Risks", "Risks"},
		{"Coverage notes  ", "Coverage notes"},
		// Not trailing, so not a qualifier — the heading keeps its parentheses.
		{"Already shipped (the quick patch) today", "Already shipped (the quick patch) today"},
	}
	for _, c := range cases {
		t.Run(c.heading, func(t *testing.T) {
			t.Parallel()
			if got := stripParenthetical(c.heading); got != c.want {
				t.Errorf("stripParenthetical(%q) = %q, want %q", c.heading, got, c.want)
			}
		})
	}
}

// stripParenthetical removes a trailing "(...)" qualifier from a heading.
func stripParenthetical(heading string) string {
	if open := strings.LastIndex(heading, "("); open >= 0 && strings.HasSuffix(strings.TrimSpace(heading), ")") {
		return strings.TrimSpace(heading[:open])
	}
	return strings.TrimSpace(heading)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// newTemplateDraftRepo inits a fresh repo with aiwf initialized and returns its
// root.
func newTemplateDraftRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	binDir := filepath.Dir(testutil.AiwfBinary(t))
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	return root
}
