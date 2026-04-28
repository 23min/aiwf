// Package roadmap renders a markdown view of a tree's epics and the
// milestones nested under them. It is the read-side companion to the
// mutating verbs in package verb: every entity already lives in the
// tree, so this package only orders, groups, and prints.
package roadmap

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
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

	knownEpic := make(map[string]bool, len(epics))
	for _, e := range epics {
		knownEpic[e.ID] = true
	}

	for _, e := range epics {
		fmt.Fprintf(&buf, "## %s — %s (%s)\n\n", e.ID, escape(e.Title), e.Status)
		ms := byParent[e.ID]
		if len(ms) == 0 {
			buf.WriteString("_No milestones yet._\n\n")
			continue
		}
		buf.WriteString("| Milestone | Title | Status |\n")
		buf.WriteString("|---|---|---|\n")
		for _, m := range ms {
			fmt.Fprintf(&buf, "| %s | %s | %s |\n", m.ID, escape(m.Title), m.Status)
		}
		buf.WriteString("\n")
	}

	var orphans []*entity.Entity
	for parent, ms := range byParent {
		if knownEpic[parent] {
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
		fmt.Fprintf(&buf, "| %s | %s | %s | %s |\n", m.ID, escape(m.Title), escape(m.Parent), m.Status)
	}
	buf.WriteString("\n")
	return buf.Bytes()
}

// escape protects markdown table cells from `|` (which would split the
// cell) and from line breaks in titles.
func escape(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
