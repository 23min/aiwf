package policies

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
)

// sectionViolation builds this policy's violations from one literal, so the
// firing-fixture inventory sees a single policy id and every report carries a
// line.
func sectionViolation(rel string, line int, format string, args ...any) Violation {
	return Violation{
		Policy: "milestone-section-name-resolution",
		File:   rel,
		Line:   line,
		Detail: fmt.Sprintf(format, args...),
	}
}

// PolicyMilestoneSectionNameResolution asserts that every backticked
// `## Section` name written in the shipped ritual, agent-card and template tree
// resolves to a heading some entity template — or the wrap artefact's own
// scaffold — actually carries.
//
// The gap this addresses: the rules for a milestone spec's sections are restated
// across several shipped surfaces with no owner, and two of them already
// disagree (G-0636). Naming a section is the smallest piece of that with a
// machine shape: a surface instructing an author to fill `## Foo` makes a claim
// about an artefact, and the artefact is there to check it against.
//
// This is the evidence shape D-0070 leaves available over a shipped surface. A
// prose- or heading-presence assertion is retired there, because it pins a
// wording a legitimate rewrite breaks and nothing catches. A relationship
// between two artefacts is different: a ritual naming a section no artefact
// carries reports, and so does a heading renamed out of every template.
//
// What it catches, stated precisely, because the weaker claim is the true one:
// a name that resolves to *no* template and no wrap-artefact section. It does
// not catch a rename of a heading another template still carries — the universe
// is the union across all six templates, so renaming `## Goal` in the milestone
// template leaves it resolvable via the epic template. Per-target resolution
// (a reference naming which artefact it means, as the cross-skill citation
// policy already does for skills) is the stronger design and belongs with the
// section-ownership work G-0636 tracks, not here.
//
// Scope is every markdown file under the ritual authoring tree, derived rather
// than listed: a hand-maintained surface list fails silently, since adding a
// ritual that names sections reports nothing.
//
// The tree itself is still a chosen boundary, and other shipped trees do carry
// section names — the `aiwf-add` skill describes the entity templates at length,
// and the `aiwf-check` skill writes `## Section` as a placeholder while
// explaining a finding code. Widening to those would report the placeholder,
// which is a generic mention rather than a claim about an artefact, and
// exempting it is the per-subject cost this scoping avoids. So a rename can
// still leave a stale mention outside this tree, unreported. Resolving that
// needs a reference that names which artefact it means, which is the
// section-ownership work G-0636 tracks.
//
// One more limit worth knowing: mentions are matched per line, so a backtick
// span wrapped across a line break is invisible to this policy.
func PolicyMilestoneSectionNameResolution(root string) ([]Violation, error) {
	surfaces, err := sectionSurfaces(root)
	if err != nil {
		return []Violation{sectionViolation(filepath.ToSlash(sectionRitualsDir), 0, "the authoring tree is unwalkable, so no surface's section names are checked: %v", err)}, nil
	}

	universe, err := sectionNameUniverse(root)
	if err != nil {
		return []Violation{sectionViolation(filepath.ToSlash(filepath.Join(sectionRitualsDir, "templates")), 0, "the section-name universe is unreadable, so no surface's claim about a spec section can be checked against it: %v", err)}, nil
	}

	var vs []Violation
	for _, rel := range surfaces {
		full := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), filepath.FromSlash(rel))
		data, err := os.ReadFile(full) //nolint:gosec // path is a compile-time constant joined to the repo root
		if err != nil {
			vs = append(vs, sectionViolation(filepath.ToSlash(filepath.Join(sectionRitualsDir, rel)), 0, "declared surface is unreadable, so the section names it writes go unchecked: %v", err))
			continue
		}
		for _, m := range mentionedSectionNames(string(data)) {
			if universe[m.name] {
				continue
			}
			vs = append(vs, sectionViolation(filepath.ToSlash(filepath.Join(sectionRitualsDir, rel)), m.line, "this surface tells an author to use section %q, which no shipped template heading and no wrap-artefact section carries — so the instruction names a section that does not exist; rename it to the section the artefact ships, or add the section to the artefact.", "## "+m.name))
		}
	}
	return vs, nil
}

// sectionRitualsDir is the authoring tree holding the templates that define
// spec sections and the rituals that instruct authors to fill them.
const sectionRitualsDir = "internal/skills/embedded-rituals/plugins/aiwf-extensions"

// sectionSurfaces returns every markdown file under the authoring tree whose
// backticked section names are checked. Deriving the set is what keeps a newly
// added ritual in scope without anyone remembering to declare it.
func sectionSurfaces(root string) ([]string, error) {
	base := filepath.Join(root, filepath.FromSlash(sectionRitualsDir))
	var out []string
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		relToBase, relErr := filepath.Rel(base, path)
		if relErr != nil { //coverage:ignore defensive: WalkDir yields paths under base, so Rel against base cannot fail
			return relErr
		}
		out = append(out, filepath.ToSlash(relToBase))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking the authoring tree: %w", err)
	}
	sort.Strings(out)
	return out, nil
}

// sectionNameUniverse is every section name a declared surface may legitimately
// write: the headings of every shipped entity template, plus the sections of the
// wrap artefact, which is not a template but is scaffolded verbatim inside the
// wrap ritual.
func sectionNameUniverse(root string) (universe map[string]bool, err error) {
	dir := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "templates")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading templates dir: %w", err)
	}

	universe = map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // dir is a compile-time constant joined to the repo root
		if readErr != nil {
			return nil, fmt.Errorf("reading template %s: %w", e.Name(), readErr)
		}
		for _, h := range topLevelHeadings(string(data)) {
			universe[h] = true
		}
	}

	wrapSkill := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "skills", "aiwfx-wrap-epic", "SKILL.md")
	data, err := os.ReadFile(wrapSkill) //nolint:gosec // path is a compile-time constant joined to the repo root
	if err != nil {
		return nil, fmt.Errorf("reading wrap ritual: %w", err)
	}
	for _, h := range topLevelHeadings(scaffoldBlock(string(data))) {
		universe[h] = true
	}
	return universe, nil
}

// scaffoldBlock returns the body of the ```markdown fenced block in body that
// contains scaffoldMarker — the wrap artefact's own scaffold. Deriving the
// artefact's sections from the scaffold rather than listing them here is what
// lets a section added to the artefact reach this policy without a second edit.
//
// Matching on content rather than on being the first fence is what keeps an
// unrelated markdown example added earlier in the ritual from silently becoming
// the scaffold, which would report every real artefact section as unresolvable.
func scaffoldBlock(body string) string {
	const fence = "```markdown"
	rest := body
	for {
		_, after, found := strings.Cut(rest, fence)
		if !found {
			return ""
		}
		block, remainder, closed := strings.Cut(after, "\n```")
		if !closed {
			if opensWithScaffoldMarker(after) {
				return after
			}
			return ""
		}
		if opensWithScaffoldMarker(block) {
			return block
		}
		rest = remainder
	}
}

// scaffoldMarker is the heading the wrap artefact's scaffold opens with. It
// identifies that block among any other markdown examples the ritual carries.
const scaffoldMarker = "# Epic wrap"

// opensWithScaffoldMarker reports whether block's first non-blank line is the
// scaffold's opening heading. Matching the marker anywhere in the block would
// select a fence that merely discusses the artefact — which is what a ritual
// explaining its own output writes — and every real artefact section would then
// resolve nowhere.
func opensWithScaffoldMarker(block string) bool {
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		return strings.HasPrefix(strings.TrimSpace(line), scaffoldMarker)
	}
	return false
}

// topLevelHeadings returns the text of every `## ` heading in body, ignoring
// deeper levels.
func topLevelHeadings(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if name, ok := strings.CutPrefix(line, "## "); ok {
			out = append(out, strings.TrimSpace(name))
		}
	}
	return out
}

// sectionMentionRe matches a backticked top-level section reference such as
// `## Release note`. The name is captured without its heading marker.
var sectionMentionRe = regexp.MustCompile("`(## [A-Za-z][^`]*)`")

// sectionSpanSplit separates the names inside one backtick span. A single span
// legitimately lists several sections ("`## Goal / ## Scope`"); read whole it
// yields one name matching nothing and checks neither real one. The `##` is
// required, so a heading whose own name contains a slash stays intact.
var sectionSpanSplit = regexp.MustCompile(`\s*/\s*##\s+`)

type sectionMention struct {
	name string
	line int
}

// mentionedSectionNames returns every backticked section reference in body with
// the 1-indexed line it sits on.
func mentionedSectionNames(body string) []sectionMention {
	var out []sectionMention
	for i, line := range strings.Split(body, "\n") {
		for _, m := range sectionMentionRe.FindAllStringSubmatch(line, -1) {
			span := strings.TrimPrefix(m[1], "## ")
			for _, name := range sectionSpanSplit.Split(span, -1) {
				if name = strings.TrimSpace(name); name != "" {
					out = append(out, sectionMention{name: name, line: i + 1})
				}
			}
		}
	}
	return out
}

// PolicyReleaseNoteHeadingResolves asserts that the milestone-spec heading the
// kernel's release-note rule reads is a heading the shipped milestone template
// actually carries.
//
// Without it the two drift silently in the one direction that costs most. A
// coherent rename — the template heading and every ritual mention moved together
// — passes every other check here, including the sibling name-resolution policy,
// because both sides of the relationship it compares moved. The kernel rule's
// own name for the section does not move with them, so it looks for a heading no
// spec will ever carry again, `present` is never true, and the rule stops firing
// for good with nothing going red.
//
// This is a relationship between two artefacts rather than an assertion about
// either one's prose, which is the evidence D-0070 leaves available here: the
// constant is resolved against the template, so moving either end reports.
func PolicyReleaseNoteHeadingResolves(root string) ([]Violation, error) {
	rel := filepath.ToSlash(filepath.Join(sectionRitualsDir, "templates", "milestone-spec.md"))
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) //nolint:gosec // path is a compile-time constant joined to the repo root
	if err != nil {
		return []Violation{{
			Policy: "release-note-heading-resolves",
			File:   rel,
			Detail: fmt.Sprintf("the milestone template is unreadable, so the section the release-note rule reads cannot be resolved against it: %v", err),
		}}, nil
	}
	// Compare by slug, because that is how the rule resolves the section: an
	// exact comparison would report a template heading differing only in case or
	// punctuation as a drift the rule does not actually suffer.
	want := entity.SectionSlug(check.ReleaseNoteSectionHeading)
	for _, h := range topLevelHeadings(string(data)) {
		if entity.SectionSlug(h) == want {
			return nil, nil
		}
	}
	return []Violation{{
		Policy: "release-note-heading-resolves",
		File:   rel,
		Detail: fmt.Sprintf("the milestone template ships no %q heading, but the kernel rule milestone-done-empty-release-note reads exactly that section; a spec scaffolded from this template can never carry it, so the rule would stop firing with nothing else going red — rename the constant in internal/check to match the template, or restore the heading", "## "+check.ReleaseNoteSectionHeading),
	}}, nil
}
