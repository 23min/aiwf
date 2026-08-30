package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PolicyMilestoneSectionNameResolution asserts that every backticked
// `## Section` name written in the surfaces that instruct an author about
// milestone-spec and wrap-artefact sections resolves to a heading some shipped
// template — or the wrap artefact's own scaffold — actually carries.
//
// The gap this addresses: the rules for a milestone spec's sections are
// restated across several shipped surfaces with no owner, and two of them
// already disagree (G-0636). Naming a section is the smallest piece of that
// which has a machine shape: a surface that instructs an author to fill
// `## Foo` is making a claim about an artefact, and the artefact is right
// there to check it against.
//
// This is the evidence shape D-0070 leaves available over a shipped surface.
// A prose- or heading-presence assertion is retired there, because it pins a
// wording that a legitimate rewrite breaks and nothing catches. A relationship
// between two artefacts is different: renaming a heading in a template reddens
// this policy, and so does a ritual naming a section no template ships, so
// either side moving is caught while neither side's phrasing is frozen.
//
// Scope is a declared set of surfaces rather than the whole shipped tree, and
// that is the load-bearing design choice. Shipped prose legitimately discusses
// section syntax generically — `wf-doc-lint` describes a `## Contents` pattern
// found in arbitrary consumer docs, and the `aiwf-check` skill writes
// `## Section` as a placeholder while explaining a finding code. Neither names
// a milestone-spec section, and a whole-tree scan would report both. Carrying
// exemptions for them would make this a rule paid per subject forever, growing
// every time someone writes a generic section name in shipped prose, and paid
// by people who broke nothing. A declared surface set is paid once: it changes
// when a ritual is added, and its failure mode is a missed inconsistency in an
// unscanned file rather than a false alarm in a correct one.
func PolicyMilestoneSectionNameResolution(root string) ([]Violation, error) {
	universe, err := sectionNameUniverse(root)
	if err != nil {
		return []Violation{{
			Policy: "milestone-section-name-resolution",
			File:   filepath.ToSlash(filepath.Join(sectionRitualsDir, "templates")),
			Detail: fmt.Sprintf("the section-name universe is unreadable, so no surface's claim about a spec section can be checked against it: %v", err),
		}}, nil
	}

	var vs []Violation
	for _, rel := range milestoneSectionSurfaces {
		full := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), filepath.FromSlash(rel))
		data, err := os.ReadFile(full) //nolint:gosec // path is a compile-time constant joined to the repo root
		if err != nil {
			vs = append(vs, Violation{
				Policy: "milestone-section-name-resolution",
				File:   filepath.ToSlash(filepath.Join(sectionRitualsDir, rel)),
				Detail: fmt.Sprintf("declared surface is unreadable, so the section names it writes go unchecked: %v", err),
			})
			continue
		}
		for _, m := range mentionedSectionNames(string(data)) {
			if universe[m.name] {
				continue
			}
			vs = append(vs, Violation{
				Policy: "milestone-section-name-resolution",
				File:   filepath.ToSlash(filepath.Join(sectionRitualsDir, rel)),
				Line:   m.line,
				Detail: fmt.Sprintf("this surface tells an author to use section %q, which no shipped template heading and no wrap-artefact section carries — so the instruction names a section that does not exist; rename it to the section the artefact ships, or add the section to the artefact.", "## "+m.name),
			})
		}
	}
	return vs, nil
}

// sectionRitualsDir is the authoring tree holding the templates that define
// spec sections and the rituals that instruct authors to fill them.
const sectionRitualsDir = "internal/skills/embedded-rituals/plugins/aiwf-extensions"

// milestoneSectionSurfaces is the declared set of surfaces whose backticked
// section names are checked. A surface belongs here when it instructs an author
// about a milestone-spec or wrap-artefact section; one that merely discusses
// section syntax in general does not, which is what keeps this policy free of
// per-subject exemptions. Paths are relative to sectionRitualsDir.
var milestoneSectionSurfaces = []string{
	"skills/aiwfx-start-milestone/SKILL.md",
	"skills/aiwfx-wrap-milestone/SKILL.md",
	"skills/aiwfx-wrap-epic/SKILL.md",
	"agents/builder.md",
	"agents/reviewer.md",
	"templates/epic-spec.md",
	"templates/milestone-spec.md",
}

// wrapArtefactUnscaffoldedSections names wrap.md sections that the ritual
// creates outside its step-1 scaffold, so parsing the scaffold alone does not
// find them. `## Doc findings` is appended by the doc-lint sweep step. The set
// is closed and each entry has a creation site in aiwfx-wrap-epic; it is not an
// exemption list for names that resolve nowhere.
var wrapArtefactUnscaffoldedSections = []string{"Doc findings"}

// sectionNameUniverse is every section name a declared surface may legitimately
// write: the headings of every shipped entity template, plus the sections of the
// wrap artefact, which is not a template but is scaffolded verbatim inside the
// wrap ritual.
func sectionNameUniverse(root string) (map[string]bool, error) {
	dir := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "templates")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading templates dir: %w", err)
	}

	universe := map[string]bool{}
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
	for _, h := range wrapArtefactUnscaffoldedSections {
		universe[h] = true
	}
	return universe, nil
}

// scaffoldBlock returns the body of the first ```markdown fenced block in body,
// which in the wrap ritual is the wrap.md scaffold. Deriving the artefact's
// sections from the scaffold rather than listing them here is what makes a
// section added to the artefact reach this policy without a second edit.
func scaffoldBlock(body string) string {
	const fence = "```markdown"
	_, rest, found := strings.Cut(body, fence)
	if !found {
		return ""
	}
	block, _, found := strings.Cut(rest, "\n```")
	if !found {
		return rest
	}
	return block
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
var sectionMentionRe = regexp.MustCompile("`## ([A-Za-z][^`]*)`")

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
			out = append(out, sectionMention{name: strings.TrimSpace(m[1]), line: i + 1})
		}
	}
	return out
}
