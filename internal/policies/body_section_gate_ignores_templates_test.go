package policies

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/skills"
)

// body_section_gate_ignores_templates_test.go — M-0331/AC-5. The push-seam
// membership gate reads the section set the kernel declares and nothing else,
// so editing a prose template cannot change what it enforces.
//
// The two artefacts are compared rather than described. A template carrying
// sections its kind does not declare holds exactly what a template edit would
// add or remove, so dropping those must be free; dropping a declared one must
// fire. Both expectations are derived, so moving a section between the template
// and entity.RequiredSections moves it between the two halves of this test on
// its own. Headings are matched by slug, the key every body reader uses.
//
// Whether each template carries every declared section is a separate property,
// pinned by TestEmbeddedTemplateCarriesRequiredSectionsAtTopLevel; here the
// expectation covers the declared sections a template actually carries, so a
// template missing one fails that test and not this one.
//
// Not every template carries extras — some ship exactly their declared set, and
// for those the first half has nothing to drop. That is skipped per template and
// asserted across the corpus: if no template carried a section beyond its
// declaration, the half would be silently inert.

// gateFixture is a throwaway git repo the gate can be run against.
type gateFixture struct {
	t    *testing.T
	root string
}

func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	f := &gateFixture{t: t, root: t.TempDir()}
	f.git("init", "-q", "-b", "main")
	f.git("config", "user.email", "test@example.com")
	f.git("config", "user.name", "aiwf-test")
	f.git("commit", "-q", "--allow-empty", "-m", "seed")
	return f
}

func (f *gateFixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = f.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit writes body at relPath and commits it, returning the new HEAD.
func (f *gateFixture) commit(relPath, body, msg string) string {
	f.t.Helper()
	abs := filepath.Join(f.root, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		f.t.Fatal(err)
	}
	f.git("add", "-A")
	f.git("commit", "-q", "-m", msg)
	return f.git("rev-parse", "HEAD")
}

// entityPathFor returns a real path of the shape PathKind recognizes for k, so
// the gate resolves the fixture the way it resolves a live tree.
func entityPathFor(k entity.Kind) (string, bool) {
	switch k {
	case entity.KindEpic:
		return "work/epics/E-0001-fixture/epic.md", true
	case entity.KindMilestone:
		return "work/epics/E-0001-fixture/M-0001-fixture.md", true
	case entity.KindGap:
		return "work/gaps/G-0001-fixture.md", true
	case entity.KindDecision:
		return "work/decisions/D-0001-fixture.md", true
	case entity.KindADR:
		return "docs/adr/ADR-0001-fixture.md", true
	case entity.KindContract:
		return "work/contracts/C-0001-fixture/contract.md", true
	}
	return "", false
}

// templateSections reads a template body's top-level sections by slug. It
// returns the declared sections the template carries, as the declaration names
// them and in its order, and the sections the template adds beyond them.
func templateSections(kind entity.Kind, body []byte) (carried, extraSlugs []string) {
	present := map[string]bool{}
	declared := map[string]bool{}
	for _, r := range entity.RequiredSections(kind) {
		declared[entity.SectionSlug(r)] = true
	}
	for _, s := range entity.ParseBodySectionsOrdered(body) {
		present[s.Slug] = true
		if !declared[s.Slug] {
			extraSlugs = append(extraSlugs, s.Slug)
		}
	}
	for _, r := range entity.RequiredSections(kind) {
		if present[entity.SectionSlug(r)] {
			carried = append(carried, r)
		}
	}
	return carried, extraSlugs
}

// withoutSections returns body with every `## ` section whose slug is in
// dropSlugs removed, heading and content, up to the next top-level heading.
func withoutSections(body string, dropSlugs []string) string {
	drop := map[string]bool{}
	for _, d := range dropSlugs {
		drop[d] = true
	}
	var out []string
	skipping := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			skipping = drop[entity.SectionSlug(strings.TrimSpace(strings.TrimPrefix(line, "## ")))]
		}
		if !skipping {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func slugsOf(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = entity.SectionSlug(n)
	}
	return out
}

func TestPolicy_BodySectionGateReadsTheDeclarationNotTheTemplate(t *testing.T) {
	t.Parallel()
	templates, err := skills.ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("no embedded templates found; the comparison has nothing to run against")
	}

	withExtras := 0
	for _, tmpl := range templates {
		e, err := entity.Parse(tmpl.Name, tmpl.Content)
		if err != nil {
			t.Fatalf("template %s does not parse: %v", tmpl.Name, err)
		}
		kind, ok := kindForTemplatePlaceholderID(e.ID)
		if !ok {
			t.Fatalf("template %s carries placeholder id %q matching no kind", tmpl.Name, e.ID)
		}
		_, body, split := entity.Split(tmpl.Content)
		if !split {
			t.Fatalf("template %s carries no frontmatter delimiter", tmpl.Name)
		}
		if _, extra := templateSections(kind, body); len(extra) > 0 {
			withExtras++
		}

		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()
			relPath, ok := entityPathFor(kind)
			if !ok {
				t.Fatalf("no fixture path shape for kind %q", kind)
			}
			carried, extra := templateSections(kind, body)
			f := newGateFixture(t)
			full := "---\nid: " + e.ID + "\n---\n" + string(body)
			base := f.commit(relPath, full, "seed from the shipped template")

			// Dropping every section the template adds beyond the declaration
			// must be free — that is the edit AC-5 says cannot reach the gate.
			if len(extra) > 0 {
				f.commit(relPath, withoutSections(full, extra), "drop the template-only sections")
				if got := gateSections(f.root, base); len(got) != 0 {
					t.Errorf("dropping template-only sections %v fired %v; the gate is reading "+
						"the template rather than entity.RequiredSections", extra, got)
				}
			}

			// Dropping the declared ones the template carries must fire, each.
			f.commit(relPath, withoutSections(full, append(extra, slugsOf(carried)...)), "drop the declared sections too")
			if diff := cmp.Diff(carried, gateSections(f.root, base)); diff != "" {
				t.Errorf("dropping %s's declared sections (-want +got):\n%s", kind, diff)
			}
		})
	}
	if withExtras == 0 {
		t.Fatal("no shipped template carries a section beyond its kind's declared set, so no " +
			"subtest can tell a gate reading the declaration from one reading the template")
	}
}

// gateSections runs the gate from base to HEAD and returns the section names it
// reported, in the order it emits them.
func gateSections(root, base string) []string {
	var out []string
	for _, d := range check.WalkDroppedBodySections(context.Background(), root, base) {
		out = append(out, d.Section)
	}
	return out
}
