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
)

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
// Scope is every markdown file under the authoring tree, derived rather than
// listed. A hand-maintained surface list fails silently — add a ritual that
// names sections and nothing reports it — and measured, the exemptions a derived
// scan was supposed to require do not exist: the generic-prose mentions that
// motivated a list (`wf-doc-lint` describing a `## Contents` pattern in consumer
// docs, the `aiwf-check` skill writing `## Section` as a placeholder) both live
// outside this tree and were never in scope to begin with.
func PolicyMilestoneSectionNameResolution(root string) ([]Violation, error) {
	surfaces, err := sectionSurfaces(root)
	if err != nil {
		return []Violation{{
			Policy: "milestone-section-name-resolution",
			File:   filepath.ToSlash(sectionRitualsDir),
			Detail: fmt.Sprintf("the authoring tree is unwalkable, so no surface's section names are checked: %v", err),
		}}, nil
	}

	universe, wrapRitual, err := sectionNameUniverse(root)
	if err != nil {
		return []Violation{{
			Policy: "milestone-section-name-resolution",
			File:   filepath.ToSlash(filepath.Join(sectionRitualsDir, "templates")),
			Detail: fmt.Sprintf("the section-name universe is unreadable, so no surface's claim about a spec section can be checked against it: %v", err),
		}}, nil
	}

	var vs []Violation
	for _, name := range unscaffoldedSectionsWithoutACreationSite(wrapRitual) {
		vs = append(vs, Violation{
			Policy: "milestone-section-name-resolution",
			File:   filepath.ToSlash(filepath.Join(sectionRitualsDir, "skills", "aiwfx-wrap-epic", "SKILL.md")),
			Detail: fmt.Sprintf("%q is listed as a wrap-artefact section created outside the scaffold, but the wrap ritual never creates it; an entry with no creation site is an exemption that silently suppresses real findings — remove it, or add the step that creates the section", "## "+name),
		})
	}

	for _, rel := range surfaces {
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

// wrapArtefactUnscaffoldedSections names wrap.md sections that the ritual
// creates outside its step-1 scaffold, so parsing the scaffold alone does not
// find them. `## Doc findings` is appended by the doc-lint sweep step.
//
// Every entry must have a creation site in the wrap ritual, and the policy
// checks that rather than trusting it — otherwise the slice is an exemption
// list, and adding a name to it silently widens the universe so real findings
// stop being reported.
var wrapArtefactUnscaffoldedSections = []string{"Doc findings"}

// sectionNameUniverse is every section name a declared surface may legitimately
// write: the headings of every shipped entity template, plus the sections of the
// wrap artefact, which is not a template but is scaffolded verbatim inside the
// wrap ritual.
func sectionNameUniverse(root string) (universe map[string]bool, wrapRitual string, err error) {
	dir := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "templates")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", fmt.Errorf("reading templates dir: %w", err)
	}

	universe = map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // dir is a compile-time constant joined to the repo root
		if readErr != nil {
			return nil, "", fmt.Errorf("reading template %s: %w", e.Name(), readErr)
		}
		for _, h := range topLevelHeadings(string(data)) {
			universe[h] = true
		}
	}

	wrapSkill := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "skills", "aiwfx-wrap-epic", "SKILL.md")
	data, err := os.ReadFile(wrapSkill) //nolint:gosec // path is a compile-time constant joined to the repo root
	if err != nil {
		return nil, "", fmt.Errorf("reading wrap ritual: %w", err)
	}
	wrapRitual = string(data)
	for _, h := range topLevelHeadings(scaffoldBlock(string(data))) {
		universe[h] = true
	}
	for _, h := range wrapArtefactUnscaffoldedSections {
		universe[h] = true
	}
	return universe, wrapRitual, nil
}

// unscaffoldedSectionsWithoutACreationSite returns the entries of
// wrapArtefactUnscaffoldedSections the wrap ritual never names. An entry with no
// creation site is an exemption rather than a derived fact: it suppresses real
// findings for a section no artefact carries.
//
// It takes the ritual body the universe read already produced, so there is no
// second read and no error path that only a vanished file could reach.
func unscaffoldedSectionsWithoutACreationSite(wrapRitual string) []string {
	var missing []string
	for _, h := range wrapArtefactUnscaffoldedSections {
		if !strings.Contains(wrapRitual, "`## "+h+"`") && !strings.Contains(wrapRitual, "\n## "+h+"\n") {
			missing = append(missing, h)
		}
	}
	return missing
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
			if strings.Contains(after, scaffoldMarker) {
				return after
			}
			return ""
		}
		if strings.Contains(block, scaffoldMarker) {
			return block
		}
		rest = remainder
	}
}

// scaffoldMarker is the heading the wrap artefact's scaffold opens with. It
// identifies that block among any other markdown examples the ritual carries.
const scaffoldMarker = "# Epic wrap"

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
// yields one name matching nothing and checks neither real one.
var sectionSpanSplit = regexp.MustCompile(`\s*/\s*(?:##\s+)?`)

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
			span := strings.TrimPrefix(strings.TrimSpace(m[1]), "## ")
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
	for _, h := range topLevelHeadings(string(data)) {
		if h == check.ReleaseNoteSectionHeading {
			return nil, nil
		}
	}
	return []Violation{{
		Policy: "release-note-heading-resolves",
		File:   rel,
		Detail: fmt.Sprintf("the milestone template ships no %q heading, but the kernel rule milestone-done-empty-release-note reads exactly that section; a spec scaffolded from this template can never carry it, so the rule would stop firing with nothing else going red — rename the constant in internal/check to match the template, or restore the heading", "## "+check.ReleaseNoteSectionHeading),
	}}, nil
}
