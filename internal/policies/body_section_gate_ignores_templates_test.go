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
// sections its kind does not require holds exactly what a template edit would
// add or remove, so dropping those must be free; dropping a declared one must
// fire. Both expectations are derived, so moving a section between the template
// and entity.RequiredSections moves it between the two halves of this test on
// its own, with no list to maintain here.
//
// Not every template carries extras — some ship exactly their declared set, and
// for those the first half has nothing to drop. That is a fact about the
// template rather than a gap in the gate, so it is skipped per template and
// asserted across the corpus: if no template anywhere carried a section beyond
// its declaration, the half would be silently inert and the test would pass
// without discriminating.
//
// Driven through WalkDroppedBodySections over real commits, not through the
// scan alone: what the gate enforces is the claim, and the scan is only how it
// gets there.

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

// commitBody writes body at relPath and commits it, returning the new SHA and
// the one it replaced.
func (f *gateFixture) commitBody(relPath, body, msg string) (sha, parent string) {
	f.t.Helper()
	parent = f.git("rev-parse", "HEAD")
	abs := filepath.Join(f.root, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		f.t.Fatal(err)
	}
	f.git("add", "-A")
	f.git("commit", "-q", "-m", msg)
	return f.git("rev-parse", "HEAD"), parent
}

// entityPathFor returns a real on-disk path of the shape PathKind recognizes
// for k, so the gate resolves the fixture the way it resolves a live tree.
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

// withoutSections returns body with each named `## ` section removed, heading
// and content, up to the next top-level heading.
func withoutSections(body string, drop []string) string {
	want := map[string]bool{}
	for _, d := range drop {
		want[d] = true
	}
	var out []string
	skipping := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			skipping = want[strings.TrimSpace(strings.TrimPrefix(line, "## "))]
		}
		if !skipping {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
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

	if n := templatesCarryingExtras(t, templates); n == 0 {
		t.Fatal("no shipped template carries a section beyond its kind's declared set, " +
			"so the template-only half below never runs and the test cannot tell a gate " +
			"reading the declaration from one reading the template")
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
				t.Fatalf("template %s carries placeholder id %q matching no kind", tmpl.Name, e.ID)
			}
			relPath, ok := entityPathFor(kind)
			if !ok {
				t.Fatalf("no fixture path shape for kind %q", kind)
			}
			_, body, split := entity.Split(tmpl.Content)
			if !split {
				t.Fatalf("template %s carries no frontmatter delimiter", tmpl.Name)
			}

			required := entity.RequiredSections(kind)
			declared := map[string]bool{}
			for _, r := range required {
				declared[r] = true
			}
			var templateOnly []string
			for _, line := range strings.Split(string(body), "\n") {
				if !strings.HasPrefix(line, "## ") {
					continue
				}
				h := strings.TrimSpace(strings.TrimPrefix(line, "## "))
				if !declared[h] {
					templateOnly = append(templateOnly, h)
				}
			}
			f := newGateFixture(t)
			full := "---\nid: " + e.ID + "\n---\n" + string(body)
			f.commitBody(relPath, full, "seed from the shipped template")

			// Dropping every section the template adds beyond the declaration
			// must be free — that is the edit AC-5 says cannot reach the gate.
			if len(templateOnly) > 0 {
				tmplSHA, tmplParent := f.commitBody(relPath,
					withoutSections(full, templateOnly), "drop the template-only sections")
				if got := WalkSections(t, f.root, tmplSHA, tmplParent, relPath); len(got) != 0 {
					t.Errorf("dropping template-only sections %v fired %v; the gate is reading "+
						"the template rather than entity.RequiredSections", templateOnly, got)
				}
			}

			// Dropping the declared ones must fire, each of them.
			reqSHA, reqParent := f.commitBody(relPath,
				withoutSections(full, append(append([]string{}, templateOnly...), required...)),
				"drop the declared sections too")
			if diff := cmp.Diff(required, WalkSections(t, f.root, reqSHA, reqParent, relPath)); diff != "" {
				t.Errorf("dropping %s's declared sections (-want +got):\n%s", kind, diff)
			}
		})
	}
}

// WalkSections runs the gate over one commit and returns the section names it
// reported, in the order the gate emits them.
func WalkSections(t *testing.T, root, sha, parent, relPath string) []string {
	t.Helper()
	var out []string
	for _, d := range check.WalkDroppedBodySections(context.Background(), root, []check.UntrailedCommit{
		{SHA: sha, ParentSHAs: []string{parent}, Paths: []string{relPath}},
	}) {
		out = append(out, d.Section)
	}
	return out
}

// templatesCarryingExtras counts the shipped templates that carry at least one
// `## ` section their kind does not declare. It is the corpus-level guard on
// the per-template half that only runs for such a template.
func templatesCarryingExtras(t *testing.T, templates []skills.Skill) int {
	t.Helper()
	n := 0
	for _, tmpl := range templates {
		e, err := entity.Parse(tmpl.Name, tmpl.Content)
		if err != nil {
			continue
		}
		kind, ok := kindForTemplatePlaceholderID(e.ID)
		if !ok {
			continue
		}
		declared := map[string]bool{}
		for _, r := range entity.RequiredSections(kind) {
			declared[r] = true
		}
		_, body, split := entity.Split(tmpl.Content)
		if !split {
			continue
		}
		for _, line := range strings.Split(string(body), "\n") {
			if !strings.HasPrefix(line, "## ") {
				continue
			}
			if !declared[strings.TrimSpace(strings.TrimPrefix(line, "## "))] {
				n++
				break
			}
		}
	}
	return n
}
