// Package roadmap renders a markdown view of a tree's epics and the
// milestones nested under them. It is the read-side companion to the
// mutating verbs in package verb: every entity already lives in the
// tree, so this package only orders, groups, and prints.
package roadmap

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// Render produces the markdown roadmap document for t. Output is
// deterministic — entities are sorted by id within their group — so
// callers can byte-compare the result to detect drift.
//
// Layout:
//
//	# Roadmap
//
//	## E-01 — <title> (<status>)
//	| Milestone | Title | Status |
//	|---|---|---|
//	| M-001 | ... | done |
//
// Milestones whose `parent` does not resolve to an epic in t are
// surfaced under a final "Unparented milestones" section so they
// don't disappear from the report. (`aiwf check` is the place to
// fix the underlying reference; render just doesn't hide it.)
func Render(t *tree.Tree) []byte {
	epics := append([]*entity.Entity(nil), t.ByKind(entity.KindEpic)...)
	sort.SliceStable(epics, func(i, j int) bool { return epics[i].ID < epics[j].ID })

	byParent := map[string][]*entity.Entity{}
	for _, m := range t.ByKind(entity.KindMilestone) {
		byParent[m.Parent] = append(byParent[m.Parent], m)
	}
	for _, ms := range byParent {
		sort.SliceStable(ms, func(i, j int) bool { return ms[i].ID < ms[j].ID })
	}

	var buf bytes.Buffer
	buf.WriteString("# Roadmap\n\n")
	if len(epics) == 0 {
		buf.WriteString("_No epics yet._\n")
		return buf.Bytes()
	}

	// Index parents by canonical id so a tree mid-migration (some
	// narrow, some canonical) still groups milestones under their
	// epic correctly (AC-2 in M-081).
	knownEpic := make(map[string]bool, len(epics))
	for _, e := range epics {
		knownEpic[entity.Canonicalize(e.ID)] = true
	}

	for _, e := range epics {
		canonE := entity.Canonicalize(e.ID)
		fmt.Fprintf(&buf, "## %s — %s (%s)\n\n", canonE, escape(e.Title), e.Status)
		if goal := readEpicGoal(t.Root, e.Path); goal != nil {
			buf.WriteString("### Goal\n\n")
			buf.Write(goal)
			buf.WriteString("\n\n")
		}
		// byParent is keyed by the milestone's on-disk Parent; collect
		// milestones whose canonicalized parent matches this epic.
		var ms []*entity.Entity
		for _, m := range t.ByKind(entity.KindMilestone) {
			if entity.Canonicalize(m.Parent) == canonE {
				ms = append(ms, m)
			}
		}
		sort.SliceStable(ms, func(i, j int) bool { return ms[i].ID < ms[j].ID })
		if len(ms) == 0 {
			buf.WriteString("_No milestones yet._\n\n")
			continue
		}
		buf.WriteString("| Milestone | Title | Status |\n")
		buf.WriteString("|---|---|---|\n")
		for _, m := range ms {
			fmt.Fprintf(&buf, "| %s | %s | %s |\n", entity.Canonicalize(m.ID), escape(m.Title), m.Status)
		}
		buf.WriteString("\n")
	}

	var orphans []*entity.Entity
	for parent, ms := range byParent {
		if knownEpic[entity.Canonicalize(parent)] {
			continue
		}
		orphans = append(orphans, ms...)
	}
	sort.SliceStable(orphans, func(i, j int) bool { return orphans[i].ID < orphans[j].ID })
	if len(orphans) == 0 {
		return buf.Bytes()
	}

	buf.WriteString("## Unparented milestones\n\n")
	buf.WriteString("| Milestone | Title | Parent | Status |\n")
	buf.WriteString("|---|---|---|---|\n")
	for _, m := range orphans {
		fmt.Fprintf(&buf, "| %s | %s | %s | %s |\n",
			entity.Canonicalize(m.ID), escape(m.Title), escape(entity.Canonicalize(m.Parent)), m.Status)
	}
	buf.WriteString("\n")
	return buf.Bytes()
}

// readEpicGoal reads the epic file at root+relPath and returns the body
// of its `## Goal` section with leading and trailing whitespace
// trimmed. Returns nil if the file can't be read, has no frontmatter,
// has no `## Goal` heading, or the goal body is whitespace-only — so
// callers can skip emitting an empty goal block. Tests that build
// in-memory trees without on-disk files get nil here, which means the
// roadmap output reduces to its pre-Goal form for those callers.
func readEpicGoal(root, relPath string) []byte {
	if root == "" || relPath == "" {
		return nil
	}
	content, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		return nil
	}
	_, body, ok := entity.Split(content)
	if !ok {
		return nil
	}
	return extractSection(body, "Goal")
}

// extractSection returns the contents of the first second-level section
// in src whose heading text equals heading (matched after trimming
// trailing whitespace). The heading line itself is excluded; the
// section ends at the next `# ` or `## ` heading or EOF. Trailing and
// leading whitespace are stripped, and a whitespace-only body returns
// nil so callers can skip empty sections cleanly.
func extractSection(src []byte, heading string) []byte {
	target := []byte("## " + heading)
	lines := bytes.Split(src, []byte("\n"))
	start := -1
	for i, line := range lines {
		if bytes.Equal(bytes.TrimRight(line, " \t\r"), target) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil
	}
	end := len(lines)
	for j := start; j < len(lines); j++ {
		if bytes.HasPrefix(lines[j], []byte("## ")) || bytes.HasPrefix(lines[j], []byte("# ")) {
			end = j
			break
		}
	}
	body := bytes.TrimSpace(bytes.Join(lines[start:end], []byte("\n")))
	if len(body) == 0 {
		return nil
	}
	return body
}

// escape protects markdown table cells from `|` (which would split the
// cell) and from line breaks in titles.
func escape(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// candidateHeadings names the section titles aiwf treats as
// human-curated, free-form lists of unscheduled work. The section is
// preserved verbatim across `aiwf render roadmap --write` cycles.
// "Candidates" is canonical; "Backlog" is accepted as an alias for
// repos that prefer that wording.
var candidateHeadings = []string{"Candidates", "Backlog"}

// ExtractCandidates returns the bytes of the first recognized
// candidates-or-backlog section in src, including its `## ` heading
// and trailing newline, up to (but not including) the next `## `
// heading at the same level. Returns nil when no recognized section
// is present.
//
// Recognition is case-sensitive on the heading word and tolerates
// trailing whitespace. The function does not parse list items; the
// caller appends the bytes verbatim to a generated roadmap so
// hand-curated content survives a regenerate.
func ExtractCandidates(src []byte) []byte {
	lines := bytes.Split(src, []byte("\n"))
	start := -1
	for i, line := range lines {
		if !bytes.HasPrefix(line, []byte("## ")) {
			continue
		}
		title := strings.TrimSpace(string(line[3:]))
		for _, h := range candidateHeadings {
			if title == h {
				start = i
				break
			}
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return nil
	}
	end := len(lines)
	for j := start + 1; j < len(lines); j++ {
		if bytes.HasPrefix(lines[j], []byte("## ")) {
			end = j
			break
		}
	}
	section := bytes.Join(lines[start:end], []byte("\n"))
	// Ensure trailing newline so concatenation is well-formed.
	if !bytes.HasSuffix(section, []byte("\n")) {
		section = append(section, '\n')
	}
	return section
}

// AppendCandidates returns the concatenation of generated and
// candidates, with a single blank line between them. If candidates is
// empty, generated is returned unchanged.
func AppendCandidates(generated, candidates []byte) []byte {
	if len(candidates) == 0 {
		return generated
	}
	var buf bytes.Buffer
	buf.Write(generated)
	if !bytes.HasSuffix(generated, []byte("\n")) {
		buf.WriteByte('\n')
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("\n\n")) {
		buf.WriteByte('\n')
	}
	buf.Write(candidates)
	return buf.Bytes()
}
