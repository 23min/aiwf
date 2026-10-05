package policies

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/projectguidance"
)

// E-0092's inventory test. M-0349's inventory gives every guidance rule one
// home and one form, and the later milestones carry its decisions out; this
// file holds the table to its own rules while they do. E-0092's wrap deletes
// the file once M-0338 records its last run, since the table is history
// from then on.

const (
	// inventoryEntity is the milestone whose body holds the inventory.
	inventoryEntity = "M-0349"
	// inventoryFreeze is the commit that froze the guidance the inventory
	// was taken from.
	inventoryFreeze = "3fcc69611b42b1912261ab10973bf96e2b9e701c"
	// inventoryFragment is the shipped fragment, written F in a home.
	inventoryFragment = "internal/skills/embedded-guidance/aiwf-guidance.md"
	// inventoryFragmentDir holds the fragment and the host inserts spliced
	// into it.
	inventoryFragmentDir = "internal/skills/embedded-guidance/"
)

// inventoryHostInserts are the host inserts spliced into the fragment's
// opening; each is part of the fragment its host renders.
var inventoryHostInserts = []string{
	inventoryFragmentDir + "claude/skill-invocation.md",
	inventoryFragmentDir + "codex/skill-invocation.md",
}

// invRow is one row of the inventory table.
type invRow struct {
	Label, Rule string
	Homes       []invHome
	Words       int
	Audience    string
	Kind        string
	Placement   string
	Disposition string
}

// invHome is one place a rule is stated: a file and, below it, a heading
// path or a fragment rule's lead-in. An empty path is the file's preamble.
type invHome struct {
	File string
	Path []string
}

// inventoryColumns is the table's header, in order.
var inventoryColumns = []string{"Label", "Rule", "Current homes", "Words", "Audience", "Kind", "Placement", "Disposition", "Notes"}

// parseInventory reads the rows of the table under a body's Inventory
// heading. A row that cannot be read is a finding, not a parse failure.
func parseInventory(body string) (rows []invRow, findings []string) {
	section := extractMarkdownSection(body, 2, "Inventory")
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "| Label |") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), " | ")
		if len(cells) != len(inventoryColumns) {
			findings = append(findings, fmt.Sprintf("row %q has %d cells, want %d", line, len(cells), len(inventoryColumns)))
			continue
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		words, err := strconv.Atoi(cells[3])
		if err != nil || words < 1 {
			findings = append(findings, fmt.Sprintf("%s: words %q is not a positive count", cells[0], cells[3]))
		}
		rows = append(rows, invRow{
			Label: cells[0], Rule: cells[1], Homes: parseHomes(cells[2]), Words: words,
			Audience: cells[4], Kind: cells[5], Placement: cells[6], Disposition: cells[7],
		})
	}
	return rows, findings
}

// parseHomes reads a Current homes cell: backticked homes separated by
// semicolons, each a file and an optional path joined by ›.
func parseHomes(cell string) []invHome {
	var out []invHome
	for _, part := range strings.Split(cell, ";") {
		parts := strings.Split(strings.Trim(strings.TrimSpace(part), "`"), " › ")
		file := parts[0]
		if file == "F" {
			file = inventoryFragment
		}
		out = append(out, invHome{File: file, Path: parts[1:]})
	}
	return out
}

// invDisposition is a parsed disposition: its kind and, for the kinds that
// carry one, its target.
type invDisposition struct {
	Kind, Target string
}

// The disposition forms the inventory accepts.
var dispositionForms = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"keep", regexp.MustCompile("^keep$")},
	{"tighten", regexp.MustCompile("^tighten$")},
	{"conflict", regexp.MustCompile("^conflict$")},
	{"merge", regexp.MustCompile("^merge into ([a-z0-9-]+)$")},
	{"on demand", regexp.MustCompile("^move on demand to `([^`]+)`$")},
	{"fragment", regexp.MustCompile("^move to the shipped fragment$")},
	{"skill", regexp.MustCompile("^move to the shipped skill `([^`]+)`$")},
	{"repository", regexp.MustCompile("^move to repository guidance `([^`]+)`$")},
	{"pointer", regexp.MustCompile("^pointer to `([^`]+)`$")},
	{"copy", regexp.MustCompile("^delete as copy of `([^`]+)`$")},
}

// parseDisposition reads a disposition, reporting false for one outside
// the closed set.
func parseDisposition(s string) (invDisposition, bool) {
	for _, f := range dispositionForms {
		if m := f.re.FindStringSubmatch(s); m != nil {
			d := invDisposition{Kind: f.kind}
			if len(m) > 1 {
				d.Target = m[1]
			}
			return d, true
		}
	}
	return invDisposition{}, false
}

// inventoryDocuments returns the on-demand documents the body's router
// draft links to, as repository paths.
func inventoryDocuments(body string) []string {
	var out []string
	for _, m := range regexp.MustCompile(`\]\(([^)\s]+)\)`).FindAllStringSubmatch(extractMarkdownSection(body, 2, "On-demand documents"), -1) {
		if p, ok := projectguidance.ResolveReference(projectguidance.RouterFile, m[1]); ok {
			out = append(out, p)
		}
	}
	return out
}

// homeClass is what a final home is: who it serves and when it loads.
type homeClass struct{ Audience, Placement string }

// classifyHome says what a final home is, or reports false for a path
// that is no guidance home.
func classifyHome(p string, documents []string) (homeClass, bool) {
	switch {
	case p == projectguidance.ClaudeFile, p == projectguidance.AgentsFile, p == projectguidance.RouterFile:
		return homeClass{"development", "primed"}, true
	case slices.Contains(documents, p), strings.HasPrefix(p, ".guidance/packs/"):
		return homeClass{"development", "on demand"}, true
	case p == inventoryFragment, slices.Contains(inventoryHostInserts, p):
		return homeClass{"operating", "primed"}, true
	case strings.HasPrefix(p, "internal/skills/embedded/"), strings.HasPrefix(p, "internal/skills/embedded-rituals/"):
		return homeClass{"operating", "on demand"}, true
	}
	return homeClass{}, false
}

// inventoryRowFindings holds each row to the inventory's rules: one label
// per row, the closed sets, the dispositions a fragment rule may take, a
// final home that matches the row's audience and placement, and
// destinations that exist. exists reports whether a repository path is
// present.
func inventoryRowFindings(rows []invRow, documents []string, exists func(string) bool) []string {
	var out []string
	byLabel := map[string]invRow{}
	for i := range rows {
		r := &rows[i]
		if _, dup := byLabel[r.Label]; dup {
			out = append(out, fmt.Sprintf("%s: label used by more than one row", r.Label))
		}
		byLabel[r.Label] = *r
	}
	for i := range rows {
		r := &rows[i]
		for _, c := range []struct{ field, value string }{{"audience", r.Audience}, {"kind", r.Kind}, {"placement", r.Placement}} {
			if !slices.Contains(map[string][]string{
				"audience":  {"development", "operating"},
				"kind":      {"judgment", "check-backed"},
				"placement": {"primed", "on demand"},
			}[c.field], c.value) {
				out = append(out, fmt.Sprintf("%s: %s %q is outside its closed set", r.Label, c.field, c.value))
			}
		}
		d, ok := parseDisposition(r.Disposition)
		if !ok {
			out = append(out, fmt.Sprintf("%s: disposition %q is outside the closed set", r.Label, r.Disposition))
			continue
		}
		if d.Kind == "conflict" {
			out = append(out, fmt.Sprintf("%s: an undecided conflict; decide it with the maintainer and record the outcome", r.Label))
			continue
		}
		if strings.HasPrefix(r.Homes[0].File, inventoryFragmentDir) && d.Kind != "keep" && d.Kind != "repository" {
			out = append(out, fmt.Sprintf("%s: a rule in the shipped fragment takes only keep or a move to repository guidance, not %q", r.Label, r.Disposition))
		}
		final, finding := finalHome(*r, d, byLabel, documents, exists)
		if finding != "" {
			out = append(out, r.Label+": "+finding)
			continue
		}
		class, ok := classifyHome(final, documents)
		if !ok {
			out = append(out, fmt.Sprintf("%s: final home %s is no guidance home", r.Label, final))
		} else if class.Audience != r.Audience || class.Placement != r.Placement {
			out = append(out, fmt.Sprintf("%s: final home %s serves %s guidance %s, but the row is %s and %s", r.Label, final, class.Audience, class.Placement, r.Audience, r.Placement))
		}
	}
	return out
}

// finalHome is where a row's rule ends: its own home when it stays, the
// survivor's when it merges, the destination when it moves or is a copy.
func finalHome(r invRow, d invDisposition, byLabel map[string]invRow, documents []string, exists func(string) bool) (final, finding string) {
	switch d.Kind {
	case "merge":
		target, ok := byLabel[d.Target]
		if !ok || target.Label == r.Label {
			return "", fmt.Sprintf("merges into %q, which names no other row", d.Target)
		}
		td, _ := parseDisposition(target.Disposition)
		if td.Kind == "merge" {
			return "", fmt.Sprintf("merges into %s, which itself merges; name the survivor", d.Target)
		}
		return finalHome(target, td, byLabel, documents, exists)
	case "on demand", "repository":
		if !slices.Contains(documents, d.Target) && (d.Kind != "repository" || d.Target != projectguidance.ClaudeFile) {
			return "", fmt.Sprintf("destination %s is not a named on-demand document", d.Target)
		}
		return d.Target, ""
	case "fragment":
		return inventoryFragment, ""
	case "skill", "copy":
		if !exists(d.Target) {
			return "", fmt.Sprintf("destination %s does not exist", d.Target)
		}
		return d.Target, ""
	}
	return r.Homes[0].File, ""
}

// guidanceSection is a part of a guidance document a home can cite: a
// heading path, a fragment rule's lead-in, or the preamble (an empty path).
// A required section must be cited by some row.
type guidanceSection struct {
	Path     []string
	Required bool
	Lead     bool
}

// guidanceSections returns the sections of a document. Its ## and ###
// headings and a preamble with text are required; #### headings may be
// cited but need not be. In the shipped fragment, each top-level rule's
// bold lead-in is required too.
func guidanceSections(content string, fragment bool) []guidanceSection {
	var out []guidanceSection
	var heads [3]string
	preamble, inFence := false, false
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
		}
		level := len(line) - len(strings.TrimLeft(line, "#"))
		switch {
		case inFence:
		case level >= 2 && level <= 4 && strings.HasPrefix(line[level:], " "):
			heads[level-2] = line[level+1:]
			for j := level - 1; j < len(heads); j++ {
				heads[j] = ""
			}
			out = append(out, guidanceSection{Path: slices.Clone(heads[:level-1]), Required: level <= 3})
		case fragment && strings.HasPrefix(line, "- **"):
			out = append(out, guidanceSection{Path: []string{bulletLead(lines[i:])}, Required: true, Lead: true})
		case heads[0] == "" && level == 0 && !strings.HasPrefix(line, " ") && strings.TrimSpace(line) != "":
			preamble = true
		}
	}
	if preamble {
		out = append(out, guidanceSection{Required: true})
	}
	return out
}

// bulletLead is the bold lead-in of the bullet starting at lines[0], read
// across its continuation lines, with the closing full stop dropped.
func bulletLead(lines []string) string {
	text := lines[0]
	for _, l := range lines[1:] {
		if !strings.HasPrefix(l, "  ") {
			break
		}
		text += " " + strings.TrimSpace(l)
	}
	lead, _, _ := strings.Cut(strings.TrimPrefix(text, "- **"), "**")
	return strings.TrimSuffix(lead, ".")
}

// citeNames reports whether a cited path names a section: the same path,
// or, for a fragment rule, the start of its lead-in.
func citeNames(cite []string, section guidanceSection) bool {
	if section.Lead {
		return len(cite) == 1 && strings.HasPrefix(section.Path[0], cite[0])
	}
	return slices.Equal(cite, section.Path)
}

// citeCovers reports whether a cited path covers a required section: it
// names the section or a section beneath it. Only a bare cite covers a
// preamble.
func citeCovers(cite []string, section guidanceSection) bool {
	if len(section.Path) == 0 || section.Lead {
		return citeNames(cite, section)
	}
	return len(cite) >= len(section.Path) && slices.Equal(cite[:len(section.Path)], section.Path)
}

// inventoryCoverageFindings reports each required section of the guidance
// documents no row cites, directly or through a section beneath it, and
// each home a row cites that names no section of its file. documents maps
// each inventoried file to its content.
func inventoryCoverageFindings(rows []invRow, documents map[string]string) []string {
	var out []string
	sections := map[string][]guidanceSection{}
	for file, content := range documents {
		sections[file] = guidanceSections(content, file == inventoryFragment)
	}
	for _, file := range slices.Sorted(maps.Keys(sections)) {
		for _, s := range sections[file] {
			if !s.Required {
				continue
			}
			cited := false
			for i := range rows {
				r := &rows[i]
				for _, h := range r.Homes {
					cited = cited || (h.File == file && citeCovers(h.Path, s))
				}
			}
			if !cited {
				out = append(out, fmt.Sprintf("%s › %s is cited by no row", file, strings.Join(s.Path, " › ")))
			}
		}
	}
	for i := range rows {
		r := &rows[i]
		for _, h := range r.Homes {
			s, inventoried := sections[h.File]
			if inventoried && !slices.ContainsFunc(s, func(sec guidanceSection) bool { return citeNames(h.Path, sec) }) {
				out = append(out, fmt.Sprintf("%s: home %s › %s names no section of the file", r.Label, h.File, strings.Join(h.Path, " › ")))
			}
		}
	}
	return out
}

// TestE0092Inventory holds M-0349's inventory to its rules against the
// tree (M-0349 AC-1, AC-2, AC-4): every section of the guidance as frozen
// is cited by a row, and every row is well formed and ends where its
// audience and placement say.
func TestE0092Inventory(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)
	e := tr.ByID(inventoryEntity)
	if e == nil {
		t.Fatalf("%s not found in the tree", inventoryEntity)
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	rows, findings := parseInventory(body)
	if len(rows) == 0 {
		t.Fatalf("%s's inventory has no rows", inventoryEntity)
	}
	exists := func(p string) bool {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		return err == nil
	}
	findings = append(findings, inventoryRowFindings(rows, inventoryDocuments(body), exists)...)

	frozen := func(p string) string {
		out, err := exec.Command("git", "-C", root, "show", inventoryFreeze+":"+p).Output()
		if err != nil {
			t.Fatalf("reading %s at the freeze: %v", p, err)
		}
		return string(out)
	}
	router := frozen(projectguidance.RouterFile)
	if routed := projectguidance.RoutedDocuments(router); len(routed) != 0 {
		t.Fatalf("the frozen router links %v; inventory them too", routed)
	}
	findings = append(findings, inventoryCoverageFindings(rows, map[string]string{
		projectguidance.ClaudeFile: projectguidance.HandwrittenText(frozen(projectguidance.ClaudeFile)),
		projectguidance.AgentsFile: projectguidance.HandwrittenText(frozen(projectguidance.AgentsFile)),
		projectguidance.RouterFile: router,
		inventoryFragment:          frozen(inventoryFragment),
		inventoryHostInserts[0]:    frozen(inventoryHostInserts[0]),
		inventoryHostInserts[1]:    frozen(inventoryHostInserts[1]),
	})...)
	for _, f := range findings {
		t.Error(f)
	}
}

// inventoryFixture is a milestone body with one on-demand document and the
// given inventory rows.
func inventoryFixture(rows ...string) string {
	return "## On-demand documents\n\n```markdown\n- [Testing](project/testing.md) — tests.\n```\n\n## Inventory\n\n" +
		"| Label | Rule | Current homes | Words | Audience | Kind | Placement | Disposition | Notes |\n|---|---|---|---|---|---|---|---|---|\n" +
		strings.Join(rows, "\n") + "\n"
}

// TestInventoryRowFindings pins each row rule, one case per rule, against
// a valid row that reports nothing.
func TestInventoryRowFindings(t *testing.T) {
	t.Parallel()
	const moved = "| moved | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` | |"
	exists := func(p string) bool {
		return p == "go.mod" || p == ".guidance/packs/go/cobra/guide.md" || p == "internal/skills/embedded/aiwf-add/SKILL.md"
	}
	cases := []struct {
		name string
		rows []string
		want string
	}{
		{"a valid table reports nothing", []string{
			moved,
			"| kept | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | keep | |",
			"| merged | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | merge into kept | |",
			"| copied | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` | |",
			"| skilled | A rule | `CLAUDE.md › Testing` | 5 | operating | judgment | on demand | move to the shipped skill `internal/skills/embedded/aiwf-add/SKILL.md` | |",
			"| shipped | A rule | `CLAUDE.md › Testing` | 5 | operating | judgment | primed | move to the shipped fragment | |",
			"| fragment-kept | A rule | `F › A rule` | 5 | operating | judgment | primed | keep | |",
			"| fragment-out | A rule | `F › A rule` | 5 | development | judgment | primed | move to repository guidance `CLAUDE.md` | |",
		}, ""},
		{"a row with the wrong number of cells", []string{"| short | A rule | 5 |"}, "cells, want"},
		{"a word count that is not a count", []string{strings.Replace(moved, "| 5 |", "| five |", 1)}, "not a positive count"},
		{"a label used twice", []string{moved, moved}, "more than one row"},
		{"a value outside a closed set", []string{strings.Replace(moved, "| judgment |", "| opinion |", 1)}, "outside its closed set"},
		{"a disposition outside the closed set", []string{strings.Replace(moved, "move on demand to", "relocate to", 1)}, "outside the closed set"},
		{"an undecided conflict", []string{"| torn | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | conflict | |"}, "undecided conflict"},
		{"a fragment rule taking a disposition its home does not allow", []string{"| frag | A rule | `F › A rule` | 5 | operating | judgment | primed | tighten | |"}, "takes only keep"},
		{"a merge into no other row", []string{"| merged | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | merge into absent | |"}, "names no other row"},
		{"a merge into a row that merges", []string{
			"| a | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | merge into b | |",
			"| b | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | merge into c | |",
			"| c | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | keep | |",
		}, "itself merges"},
		{"an on-demand destination the router draft does not name", []string{strings.Replace(moved, "testing.md", "other.md", 1)}, "not a named on-demand document"},
		{"a destination that does not exist", []string{"| copied | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | delete as copy of `absent.md` | |"}, "does not exist"},
		{"a final home that is no guidance home", []string{"| copied | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | delete as copy of `go.mod` | |"}, "no guidance home"},
		{"a final home that disagrees with the row", []string{strings.Replace(moved, "| on demand |", "| primed |", 1)}, "but the row is"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := inventoryFixture(tc.rows...)
			rows, findings := parseInventory(body)
			findings = append(findings, inventoryRowFindings(rows, inventoryDocuments(body), exists)...)
			if tc.want == "" {
				if len(findings) != 0 {
					t.Errorf("findings = %q, want none", findings)
				}
				return
			}
			if len(findings) != 1 || !strings.Contains(findings[0], tc.want) {
				t.Errorf("findings = %q, want one containing %q", findings, tc.want)
			}
		})
	}
}

// TestInventoryCoverageFindings pins the coverage rules: every ## and ###
// heading, a preamble with text, and every top-level rule of the fragment
// is cited; a cite beneath a heading covers it; a #### heading need not be
// cited; and a cite must name a section of its file.
func TestInventoryCoverageFindings(t *testing.T) {
	t.Parallel()
	claude := "# Title\n\nPreamble.\n\n## Testing\n\n### Rules\n\n#### Detail\n\nText.\n\n```bash\n## not a heading\n```\n\n## Other\n\nText.\n"
	fragment := "# Guidance\n\n- **A rule about\n  things.** Body.\n- **Another rule.** Body.\n"
	row := func(homes string) string {
		return "| r | A rule | " + homes + " | 5 | development | judgment | primed | keep | |"
	}
	all := row("`CLAUDE.md`; `CLAUDE.md › Testing › Rules › Detail`; `CLAUDE.md › Other`; `F › A rule about things`; `F › Another rule`")
	cases := []struct {
		name string
		row  string
		want string
	}{
		{"every required section cited, one through a section beneath it", all, ""},
		{"an uncited heading", strings.Replace(all, "; `CLAUDE.md › Other`", "", 1), "Other is cited by no row"},
		{"a preamble cited only through a section", strings.Replace(all, "`CLAUDE.md`; ", "", 1), "CLAUDE.md ›  is cited by no row"},
		{"an uncited fragment rule", strings.Replace(all, "; `F › Another rule`", "", 1), "Another rule is cited by no row"},
		{"a cite naming no section", strings.Replace(all, "`F › Another rule`", "`F › Another rule`; `CLAUDE.md › Testing › Missing`", 1), "names no section"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows, _ := parseInventory(inventoryFixture(tc.row))
			findings := inventoryCoverageFindings(rows, map[string]string{"CLAUDE.md": claude, inventoryFragment: fragment})
			if tc.want == "" {
				if len(findings) != 0 {
					t.Errorf("findings = %q, want none", findings)
				}
				return
			}
			if len(findings) != 1 || !strings.Contains(findings[0], tc.want) {
				t.Errorf("findings = %q, want one containing %q", findings, tc.want)
			}
		})
	}
}
