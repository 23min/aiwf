package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ownershipViolation builds this policy's violations from one literal, so the
// firing-fixture inventory sees a single policy id and every report carries a
// line.
func ownershipViolation(rel string, line int, format string, args ...any) Violation {
	return Violation{
		Policy: "milestone-section-ownership",
		File:   rel,
		Line:   line,
		Detail: fmt.Sprintf(format, args...),
	}
}

// Where the ownership map is written. The template is the one every reader of a
// scaffolded spec meets; each ritual carries it so an agent working inside one
// knows which sections are its own without opening another file.
var (
	ownershipTemplatePath = filepath.Join(sectionRitualsDir, "templates", "milestone-spec.md")
	ownershipStartPath    = filepath.Join(sectionRitualsDir, "skills", "aiwfx-start-milestone", "SKILL.md")
	ownershipWrapPath     = filepath.Join(sectionRitualsDir, "skills", "aiwfx-wrap-milestone", "SKILL.md")
	ownershipBuilderPath  = filepath.Join(sectionRitualsDir, "agents", "builder.md")
)

// PolicyMilestoneSectionOwnership asserts that every section a milestone spec
// gains after it is authored has exactly one owning ritual, and that all four
// surfaces recording that assignment agree.
//
// Each such section has one owning surface: the ritual where the section is
// first written states when it is filled and what it holds, and every other
// surface names the section without restating its rule.
//
// The assignment is written three times — in the template and in each ritual —
// so a reader inside any one of them is self-sufficient. Three copies drift
// unless something compares them, which is what this does. Everything is derived
// from the documents; a list kept here would be a fourth copy, and the one no
// reader ever sees.
//
// What it reports:
//
//   - A section heading the template carries below its map with no owner, or a
//     section the map names that the template no longer carries.
//   - A section claimed by both rituals, or an owner naming a ritual that ships
//     no skill.
//   - A ritual whose own copy of the map disagrees with the template's — a
//     section assigned to a different owner, or present in one copy only. This
//     is what makes a reassignment report rather than pass silently.
//   - A wrap-owned section named in the builder agent card, which is the
//     implementing agent's: naming one there says it is filled during
//     implementation.
//   - A line inside a map that names a ritual without assigning anything, which
//     is what a mistyped owner line looks like — its sections would otherwise
//     fold silently into the owner above it.
//   - A second map in the same surface, which would be compared against nothing.
//
// What it does not reach is whether a rule's prose is stated twice. A ritual
// re-specifying a section it does not own reads as ordinary instruction, and
// pinning it would pin a wording D-0070 leaves unpinnable. That half is carried
// at review.
func PolicyMilestoneSectionOwnership(root string) ([]Violation, error) {
	tmplRel := filepath.ToSlash(ownershipTemplatePath)
	tmpl, vs := readOwnershipMap(root, ownershipTemplatePath)
	if tmpl == nil {
		return vs, nil
	}

	owned := map[string]string{} // section name -> owning ritual
	for _, o := range sortedOwnerNames(tmpl.owners) {
		skillDir := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "skills", o)
		if _, err := os.Stat(skillDir); err != nil {
			vs = append(vs, ownershipViolation(tmplRel, tmpl.owners[o].line, "the map assigns sections to %q, which ships no skill directory — a section whose owner does not exist has no surface stating its rule; name a ritual that ships, or move the sections to one that does", o))
		}
		for _, s := range tmpl.owners[o].sections {
			if prev, dup := owned[s.name]; dup {
				vs = append(vs, ownershipViolation(tmplRel, s.line, "section %q is claimed by both %q and %q; one owner states a section's rule and every other surface points at it, so two claims put the rule back in two places", "## "+s.name, prev, o))
				continue
			}
			owned[s.name] = o
		}
	}

	headings := map[string]bool{}
	for _, h := range topLevelHeadings(tmpl.body) {
		headings[h] = true
	}
	for _, o := range sortedOwnerNames(tmpl.owners) {
		for _, s := range tmpl.owners[o].sections {
			if !headings[s.name] {
				vs = append(vs, ownershipViolation(tmplRel, s.line, "the map assigns %q to %q, but the template carries no such heading — a retired section left in the map keeps an owner for a section no spec carries; drop the entry, or restore the heading", "## "+s.name, o))
			}
		}
	}

	// Every section below the map is filled after the spec is authored, so it
	// needs an owner. Sections above it are the author's own.
	for _, h := range headingsAfter(tmpl.body, tmpl.end) {
		if _, ok := owned[h.name]; !ok {
			vs = append(vs, ownershipViolation(tmplRel, h.line, "section %q is filled after the spec is authored but the map assigns it no owner, so no surface states when it is written or what it holds; add it to the ritual that writes it first", "## "+h.name))
		}
	}

	vs = append(vs, tmpl.strayViolations(tmplRel)...)
	for _, ritual := range []string{ownershipStartPath, ownershipWrapPath} {
		vs = append(vs, ritualMapAgreementViolations(root, ritual, owned)...)
	}
	vs = append(vs, builderCardOwnershipViolations(root, owned)...)
	return vs, nil
}

// readOwnershipMap parses one surface's copy of the map, reporting rather than
// failing when the surface is unreadable or carries no map at all.
func readOwnershipMap(root, rel string) (*ownershipMap, []Violation) {
	slash := filepath.ToSlash(rel)
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) //nolint:gosec // path is a compile-time constant joined to the repo root
	if err != nil {
		return nil, []Violation{ownershipViolation(slash, 0, "%s is unreadable, so the section owners it records cannot be compared: %v", ownershipSurfaceName(rel), err)}
	}
	m := parseOwnershipMap(string(data))
	if len(m.owners) == 0 {
		return nil, []Violation{ownershipViolation(slash, 0, "%s carries no ownership map, so nothing there records which ritual states each section's rule; restore it as owner lines of the form \"`aiwfx-<ritual>` — `## Section`, `## Section`\"", ownershipSurfaceName(rel))}
	}
	return m, nil
}

// ownershipSurfaceName names a map-carrying surface the way a report should
// refer to it. The builder card carries no map and is reported by name where it
// is read.
func ownershipSurfaceName(rel string) string {
	if rel == ownershipTemplatePath {
		return "the milestone template"
	}
	return "the " + filepath.Base(filepath.Dir(rel)) + " ritual"
}

// ritualMapAgreementViolations reports where a ritual's own copy of the map
// disagrees with the template's. This is what catches a reassignment: moving a
// section between owners in one surface leaves the others saying otherwise.
func ritualMapAgreementViolations(root, rel string, owned map[string]string) []Violation {
	slash := filepath.ToSlash(rel)
	m, vs := readOwnershipMap(root, rel)
	if m == nil {
		return vs
	}
	vs = append(vs, m.strayViolations(slash)...)

	seen := map[string]bool{}
	for _, o := range sortedOwnerNames(m.owners) {
		for _, s := range m.owners[o].sections {
			seen[s.name] = true
			switch want := owned[s.name]; {
			case want == "":
				vs = append(vs, ownershipViolation(slash, s.line, "this ritual assigns %q to %q, but the milestone template's map does not carry that section at all; the copies have to agree, so add it there or drop it here", "## "+s.name, o))
			case want != o:
				vs = append(vs, ownershipViolation(slash, s.line, "this ritual assigns %q to %q while the milestone template's map assigns it to %q; a section has one owner, so one of the two copies is a reassignment the other never received", "## "+s.name, o, want))
			}
		}
	}
	for _, name := range sortedSectionNames(owned) {
		if !seen[name] {
			vs = append(vs, ownershipViolation(slash, m.end, "the milestone template's map assigns %q to %q, but this ritual's copy omits it; a reader working from this copy would not know the section has an owner", "## "+name, owned[name]))
		}
	}
	return vs
}

// builderCardOwnershipViolations reports a wrap-owned section named in the
// builder card. The card lists what the implementing agent fills, so naming a
// section the wrap ritual owns asserts it is filled during implementation.
func builderCardOwnershipViolations(root string, owned map[string]string) []Violation {
	const wrapRitual = "aiwfx-wrap-milestone"
	rel := filepath.ToSlash(ownershipBuilderPath)
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ownershipBuilderPath))) //nolint:gosec // path is a compile-time constant joined to the repo root
	if err != nil {
		return []Violation{ownershipViolation(rel, 0, "the builder card is unreadable, so the sections it claims go unchecked against the map: %v", err)}
	}
	var vs []Violation
	for _, m := range mentionedSectionNames(string(data)) {
		if owned[m.name] == wrapRitual {
			vs = append(vs, ownershipViolation(rel, m.line, "the builder card names %q, which the template's map assigns to %q; the card lists what the implementing agent fills, so naming a wrap-owned section here says it is filled during implementation and contradicts the owner", "## "+m.name, wrapRitual))
		}
	}
	return vs
}

// ownerLineRe matches an ownership-map owner line: a backticked ritual name
// followed by an em-dash. The em-dash is what separates an assignment from an
// ordinary prose reference to the same ritual elsewhere in the surface.
var ownerLineRe = regexp.MustCompile("`(aiwfx-[a-z0-9-]+)`\\s*—")

// ritualMentionRe matches any backticked ritual name, so a line inside the map
// that names one without assigning anything can be told apart from prose.
var ritualMentionRe = regexp.MustCompile("`(aiwfx-[a-z0-9-]+)`")

type ownedSection struct {
	name string
	line int
}

type ownerEntry struct {
	line     int
	sections []ownedSection
}

type strayLine struct {
	ritual string
	line   int
}

// ownershipMap is one surface's copy of the assignment.
type ownershipMap struct {
	body   string
	owners map[string]*ownerEntry
	end    int
	strays []strayLine
	extras []int
}

// strayViolations reports a line inside the map naming a ritual without opening
// an entry. That is what a mistyped owner line looks like — a hyphen where the
// em-dash belongs — and its sections would otherwise be read as a continuation
// of the entry above, silently changing their owner.
func (m *ownershipMap) strayViolations(rel string) []Violation {
	var vs []Violation
	for _, s := range m.strays {
		vs = append(vs, ownershipViolation(rel, s.line, "this line names %q inside the ownership map but assigns nothing, so the sections on it are read as belonging to the owner above; an owner line separates the ritual from its sections with an em-dash", s.ritual))
	}
	for _, line := range m.extras {
		vs = append(vs, ownershipViolation(rel, line, "the ownership map appears more than once in this surface; only the first is read, so this one is compared against nothing and can disagree with it silently — keep one map per surface"))
	}
	return vs
}

// parseOwnershipMap reads a surface's ownership map: the first run of lines that
// assign sections to rituals.
//
// A line opens the map when it names a ritual, separates it with an em-dash, and
// assigns at least one section. Requiring a section is what keeps ordinary prose
// out — a sentence handing off to another ritual has the same ritual-and-dash
// shape, and reading one as an owner line makes every section below it look
// missing from a copy sitting untouched a few lines away. A following line
// carrying section names continues the map, which is how one owner's assignment
// wraps; the first line carrying none ends it.
//
// Every later run of the same shape is recorded as an extra, because a surface
// carrying two maps has two answers and this reads only the first.
func parseOwnershipMap(body string) *ownershipMap {
	m := &ownershipMap{body: body, owners: map[string]*ownerEntry{}}
	var current *ownerEntry
	started, done, inExtra := false, false, false

	for i, line := range strings.Split(body, "\n") {
		lineNo := i + 1
		mentions := mentionedSectionNames(line)
		om := ownerLineRe.FindStringSubmatch(line)
		opens := om != nil && len(mentions) > 0

		// Past the first map, note where another begins and read nothing from
		// it — merging a second map into the first would hide the disagreement
		// that makes it worth reporting.
		if done {
			switch {
			case inExtra && len(mentions) == 0:
				inExtra = false
			case !inExtra && opens:
				m.extras = append(m.extras, lineNo)
				inExtra = true
			}
			continue
		}

		switch {
		case opens:
			started = true
			entry, seen := m.owners[om[1]]
			if !seen {
				entry = &ownerEntry{line: lineNo}
				m.owners[om[1]] = entry
			}
			current = entry
			current.sections = append(current.sections, namedAt(mentions, lineNo)...)
			m.end = lineNo
		case started && len(mentions) > 0:
			if rm := ritualMentionRe.FindStringSubmatch(line); rm != nil {
				m.strays = append(m.strays, strayLine{ritual: rm[1], line: lineNo})
			}
			current.sections = append(current.sections, namedAt(mentions, lineNo)...)
			m.end = lineNo
		case started:
			done = true
		}
	}
	return m
}

// namedAt converts the section mentions found on one line into owned sections.
func namedAt(mentions []sectionMention, line int) []ownedSection {
	out := make([]ownedSection, 0, len(mentions))
	for _, m := range mentions {
		out = append(out, ownedSection{name: m.name, line: line})
	}
	return out
}

// sortedOwnerNames returns a map's ritual names in a stable order, so a run
// reports the same violations in the same sequence.
func sortedOwnerNames(owners map[string]*ownerEntry) []string {
	out := make([]string, 0, len(owners))
	for name := range owners {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// sortedSectionNames returns an assignment's section names in a stable order.
func sortedSectionNames(owned map[string]string) []string {
	out := make([]string, 0, len(owned))
	for name := range owned {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// headingsAfter returns the surface's top-level headings below the given line.
func headingsAfter(body string, after int) []ownedSection {
	var out []ownedSection
	for i, line := range strings.Split(body, "\n") {
		lineNo := i + 1
		if lineNo <= after {
			continue
		}
		if name, ok := strings.CutPrefix(line, "## "); ok {
			out = append(out, ownedSection{name: strings.TrimSpace(name), line: lineNo})
		}
	}
	return out
}
