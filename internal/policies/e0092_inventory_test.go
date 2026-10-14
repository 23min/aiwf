package policies

import (
	"context"
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
	"unicode"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/config"
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
	// was taken from. Its router links no document, so the guidance set is
	// the two host files and the router.
	inventoryFreeze = "3fcc69611b42b1912261ab10973bf96e2b9e701c"
	// inventoryReconcileBase is the commit that promoted M-0336 to
	// in_progress. Disposition blocks after it name the row they carry out
	// (M-0336 AC-5); those before it predate the label.
	inventoryReconcileBase = "aba8c273231fb494ed47e75106c15572e66f585e"
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
	Label       string
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

// inventoryLabel is the shape of a row label, which a merge or a removal
// commit names.
var inventoryLabel = regexp.MustCompile(`^[a-z0-9-]+$`)

// parseInventory reads the rows of the table under a body's Inventory
// heading. A header or row that cannot be read is a finding, not a parse
// failure.
func parseInventory(body string) (rows []invRow, findings []string) {
	section := extractMarkdownSection(body, 2, "Inventory")
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), " | ")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if cells[0] == inventoryColumns[0] {
			if !slices.Equal(cells, inventoryColumns) {
				findings = append(findings, fmt.Sprintf("header %q, want %q", cells, inventoryColumns))
			}
			continue
		}
		if len(cells) != len(inventoryColumns) {
			findings = append(findings, fmt.Sprintf("row %q has %d cells, want %d", line, len(cells), len(inventoryColumns)))
			continue
		}
		words, err := strconv.Atoi(cells[3])
		if err != nil || words < 1 {
			findings = append(findings, fmt.Sprintf("%s: words %q is not a positive count", cells[0], cells[3]))
		}
		homes := parseHomes(cells[2])
		if len(homes) == 0 {
			findings = append(findings, fmt.Sprintf("%s: names no home", cells[0]))
			continue
		}
		rows = append(rows, invRow{
			Label: cells[0], Homes: homes, Words: words,
			Audience: cells[4], Kind: cells[5], Placement: cells[6], Disposition: cells[7],
		})
	}
	return rows, findings
}

// inventoryHomeSpan is one backticked home in a Current homes cell.
var inventoryHomeSpan = regexp.MustCompile("`([^`]*)`")

// parseHomes reads a Current homes cell: backticked homes, each a file and
// an optional path joined by ›. A lead-in may hold a semicolon, so homes
// are read by their backticks rather than split on the separator.
func parseHomes(cell string) []invHome {
	var out []invHome
	for _, m := range inventoryHomeSpan.FindAllStringSubmatch(cell, -1) {
		parts := strings.Split(m[1], " › ")
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

// inventoryRouterDraft is the router text the body's On-demand documents
// section drafts, inside its fenced block.
func inventoryRouterDraft(body string) string {
	section := extractMarkdownSection(body, 2, "On-demand documents")
	_, draft, _ := strings.Cut(section, "```markdown\n")
	draft, _, _ = strings.Cut(draft, "```")
	return draft
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
// per row, in the label shape; homes in files that exist; the closed sets;
// the dispositions a fragment rule may take, judged on the row's first
// home; a final home that matches the row's audience and placement; and
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
		if !inventoryLabel.MatchString(r.Label) {
			out = append(out, fmt.Sprintf("%q: a label is lower-case letters, digits and hyphens", r.Label))
		}
		for _, h := range r.Homes {
			if h.File == "" || !exists(h.File) {
				out = append(out, fmt.Sprintf("%s: home %q names no file in the repository", r.Label, h.File))
			}
		}
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

// finalHome is where a row's rule ends: its own first home when it stays,
// the survivor's when it merges, the destination when it moves or is a
// copy.
func finalHome(r invRow, d invDisposition, byLabel map[string]invRow, documents []string, exists func(string) bool) (final, finding string) {
	switch d.Kind {
	case "merge":
		target, ok := byLabel[d.Target]
		if !ok {
			return "", fmt.Sprintf("merges into %q, which names no row", d.Target)
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
// heading path, a fragment rule's lead-in, or the preamble (an empty
// path). Text and Words are the section's own text, leaving out headings,
// the H1 and --- separators; a fragment's rules are not text of the
// heading above them, and a lead-in section holds no text.
type guidanceSection struct {
	Path     []string
	Required bool
	Lead     bool
	Words    int
	Text     string
}

// guidanceSections returns the sections of a document. Its ## and ###
// headings and a preamble with text are required; #### headings may be
// cited but need not be. In the shipped fragment, each top-level rule's
// bold lead-in is required too.
func guidanceSections(content string, fragment bool) []guidanceSection {
	out := []guidanceSection{{Required: true}}
	var heads [3]string
	current, inFence, inBullet := 0, false, false
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		fence := strings.HasPrefix(strings.TrimSpace(line), "```")
		level := len(line) - len(strings.TrimLeft(line, "#"))
		switch {
		case !inFence && level >= 2 && level <= 4 && strings.HasPrefix(line[level:], " "):
			heads[level-2] = line[level+1:]
			for j := level - 1; j < len(heads); j++ {
				heads[j] = ""
			}
			out = append(out, guidanceSection{Path: slices.Clone(heads[:level-1]), Required: level <= 3})
			current, inBullet = len(out)-1, false
		case !inFence && fragment && strings.HasPrefix(line, "- **"):
			out = append(out, guidanceSection{Path: []string{bulletLead(lines[i:])}, Required: true, Lead: true})
			inBullet = true
		case fragment && inBullet && strings.HasPrefix(line, "  "):
		case !inFence && (strings.HasPrefix(line, "# ") || strings.TrimSpace(line) == "---"):
		default:
			inBullet = false
			out[current].Words += len(strings.Fields(line))
			out[current].Text += line + "\n"
		}
		if fence {
			inFence = !inFence
		}
	}
	if out[0].Words == 0 {
		out = out[1:]
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

// citeCovers reports whether a cited path covers a required section. A
// section with text of its own is covered only by a cite that names it; a
// heading that holds only the sections beneath it is covered by a cite of
// any of them.
func citeCovers(cite []string, section guidanceSection) bool {
	if section.Words > 0 || len(section.Path) == 0 || section.Lead {
		return citeNames(cite, section)
	}
	return len(cite) > len(section.Path) && slices.Equal(cite[:len(section.Path)], section.Path)
}

// inventoryCoverageFindings reports each required section of the guidance
// documents no row covers, and each home a row cites in an inventoried
// file that names no section, or more than one. documents maps each
// inventoried file to its content.
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
				for _, h := range rows[i].Homes {
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
			if !inventoried {
				continue
			}
			switch n := len(slices.DeleteFunc(slices.Clone(s), func(sec guidanceSection) bool { return !citeNames(h.Path, sec) })); {
			case n == 0:
				out = append(out, fmt.Sprintf("%s: home %s › %s names no section of the file", r.Label, h.File, strings.Join(h.Path, " › ")))
			case n > 1:
				out = append(out, fmt.Sprintf("%s: home %s › %s names more than one section of the file", r.Label, h.File, strings.Join(h.Path, " › ")))
			}
		}
	}
	return out
}

// inventoryWordFindings reports each section of the development guidance
// whose measured words differ from the words of the rows citing it first,
// so a rule dropped from the table, or a row whose count is wrong, shows
// below the section level. documents maps each development guidance file
// to its handwritten text.
func inventoryWordFindings(rows []invRow, documents map[string]string) []string {
	var out []string
	for _, file := range slices.Sorted(maps.Keys(documents)) {
		for _, s := range guidanceSections(documents[file], false) {
			sum := 0
			for i := range rows {
				if h := rows[i].Homes[0]; h.File == file && slices.Equal(h.Path, s.Path) {
					sum += rows[i].Words
				}
			}
			if sum != s.Words {
				out = append(out, fmt.Sprintf("%s › %s has %d words; the rows citing it first count %d", file, strings.Join(s.Path, " › "), s.Words, sum))
			}
		}
	}
	return out
}

// inventoryCeilingTargets derives each host's handwritten primed target
// from the table (M-0349 AC-5): the words of every row whose final home is
// a host's primed document, a tightened rule counted at its current size
// and a merged rule counted beside its survivor; the router at its drafted
// size; and the H1 and the headings of the sections those rules sit in,
// read from the frozen documents. Codex reads CLAUDE.md as well as
// AGENTS.md, so its target carries both.
func inventoryCeilingTargets(rows []invRow, routerDraft string, frozen map[string]string, exists func(string) bool) (claude, codex int) {
	byLabel := map[string]invRow{}
	for i := range rows {
		byLabel[rows[i].Label] = rows[i]
	}
	headings := map[string]map[string]bool{projectguidance.ClaudeFile: {}, projectguidance.AgentsFile: {}}
	words := map[string]int{}
	for i := range rows {
		r := &rows[i]
		d, _ := parseDisposition(r.Disposition)
		final, _ := finalHome(*r, d, byLabel, nil, exists)
		if headings[final] == nil {
			continue
		}
		words[final] += r.Words
		if d.Kind == "merge" {
			continue
		}
		for n := range r.Homes[0].Path {
			headings[final][strings.Repeat("#", n+2)+" "+r.Homes[0].Path[n]] = true
		}
	}
	for file, set := range headings {
		for _, line := range strings.Split(frozen[file], "\n") {
			if strings.HasPrefix(line, "# ") || set[line] {
				words[file] += len(strings.Fields(line))
			}
		}
	}
	claude = words[projectguidance.ClaudeFile] + len(strings.Fields(routerDraft))
	return claude, claude + words[projectguidance.AgentsFile]
}

// removalBlock is a disposition block from a commit after
// inventoryReconcileBase.
type removalBlock struct {
	SHA string
	check.DispositionBlock
}

// decidedDisposition is the block disposition that carries out a row's
// decision, or false for a decision that removes nothing.
func decidedDisposition(d invDisposition) (string, bool) {
	switch d.Kind {
	case "copy", "skill", "repository":
		return "copy of " + d.Target, true
	case "fragment":
		return "copy of " + inventoryFragment, true
	case "on demand":
		return "relocated to " + d.Target, true
	case "pointer":
		return "pointer to " + d.Target, true
	case "merge", "tighten":
		return "reworded", true
	}
	return "", false
}

// isIDEdit reports whether a word is an id-shaped token the development
// guidance may not carry, as AC-2's scan classifies it. The word is the
// token itself: one opening a quoted passage is the passage's first word.
func isIDEdit(word string) bool {
	return !strings.HasPrefix(word, `"`) && len(check.ScanPlainTextIDs(word, "")) > 0
}

// isEntityID reports whether a disposition's destination names an entity
// rather than a path: a path holding an id in its file name is a path.
func isEntityID(target string) bool {
	return isIDEdit(target) && !strings.Contains(target, "/")
}

// notIDRune separates the words of a row's text into the tokens an id edit
// may name, so a shorter id never matches inside a longer one.
func notIDRune(r rune) bool {
	return r != '-' && r != '/' && !unicode.IsDigit(r) && !unicode.IsLetter(r)
}

// inventoryReconcileFindings holds each removal block to the inventory
// (M-0336 AC-5, AC-6). A block names its row as the first word of its
// Removed: value. One naming an id second is an edit AC-2 requires: the id
// must be in the row's text at the freeze, and the block is held to no
// decision. Every other block must carry its row's decision. A destination
// path a block names must exist and an entity it names must resolve, while
// a pointer to a check is not looked up. A block removing a rule the
// inventory moves to the shipped fragment, a shipped skill or repository
// guidance must come after a commit since the base changed its new home.
// rowText returns a row's first home as frozen; resolves reports whether
// an entity id resolves; changedBefore whether a commit after the base and
// before sha changed a path.
func inventoryReconcileFindings(blocks []removalBlock, rows []invRow, rowText func(invRow) string, exists, resolves func(string) bool, changedBefore func(sha, path string) bool) []string {
	var out []string
	byLabel := map[string]invRow{}
	for i := range rows {
		byLabel[rows[i].Label] = rows[i]
	}
	for _, b := range blocks {
		at := fmt.Sprintf("commit %.9s, block %q", b.SHA, b.Removed)
		words := strings.Fields(b.Removed)
		if len(words) == 0 {
			out = append(out, at+": names no inventory row")
			continue
		}
		row, ok := byLabel[words[0]]
		if !ok {
			out = append(out, at+": its first word names no inventory row")
			continue
		}
		for _, form := range []string{"copy of ", "relocated to ", "pointer to "} {
			target, named := strings.CutPrefix(b.Disposition, form)
			switch {
			case !named:
			case isEntityID(target):
				if !resolves(target) {
					out = append(out, fmt.Sprintf("%s: names %s, which resolves to no entity", at, target))
				}
			case form != "pointer to " && !exists(target):
				out = append(out, fmt.Sprintf("%s: names %s, which does not exist", at, target))
			}
		}
		if len(words) > 1 && isIDEdit(words[1]) {
			if !slices.Contains(strings.FieldsFunc(rowText(row), notIDRune), words[1]) {
				out = append(out, fmt.Sprintf("%s: names %s, which row %s's text at the freeze does not carry", at, words[1], row.Label))
			}
			continue
		}
		d, _ := parseDisposition(row.Disposition)
		want, removes := decidedDisposition(d)
		switch {
		case !removes:
			out = append(out, fmt.Sprintf("%s: row %s decides %q, which removes nothing", at, row.Label, row.Disposition))
		case b.Disposition != want:
			out = append(out, fmt.Sprintf("%s: disposition %q, but row %s decides %q", at, b.Disposition, row.Label, want))
		case d.Kind == "fragment" || d.Kind == "skill" || d.Kind == "repository":
			home := strings.TrimPrefix(want, "copy of ")
			if !changedBefore(b.SHA, home) {
				out = append(out, fmt.Sprintf("%s: removes a rule the inventory moves to %s before any commit adds it there", at, home))
			}
		}
	}
	return out
}

// fragmentLossFindings reports each part of the shipped fragment whose
// words at the freeze no longer all survive, in order, at HEAD (M-0336
// AC-5). Every fragment row is kept, so the epic may add to the fragment
// but never take from it; the fence does not judge the fragment, so this
// is what holds a removal there.
func fragmentLossFindings(frozen, current map[string]string) []string {
	var out []string
	for _, p := range slices.Sorted(maps.Keys(frozen)) {
		if !check.WordsSurviveInOrder(frozen[p], current[p]) {
			out = append(out, fmt.Sprintf("%s: words the fragment held at the freeze are gone; every fragment row is kept, so restore them", p))
		}
	}
	return out
}

// repoRemovalBlocks reads the disposition blocks of every commit after
// inventoryReconcileBase that is not on trunk, and reports each malformed
// one, since a block the parser drops would otherwise escape the check.
func repoRemovalBlocks(t *testing.T, root string) (blocks []removalBlock, malformed []string) {
	t.Helper()
	cfg, _ := config.Load(root)
	trunk, _ := cfg.AllocateTrunkRef()
	commits, err := check.WalkInstructionCommits(context.Background(), root, inventoryReconcileBase+"..HEAD", trunk)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range commits {
		good, bad := check.ParseDispositionBlocks(c.Body)
		for _, b := range good {
			blocks = append(blocks, removalBlock{c.SHA, b})
		}
		for _, m := range bad {
			malformed = append(malformed, fmt.Sprintf("commit %.9s: malformed disposition block %q", c.SHA, m))
		}
	}
	return blocks, malformed
}

// ceilingTargetFindings reports each host whose ceiling sits above the
// target M-0349's inventory sets for it (M-0337 AC-5). The ceiling policy
// holds each host's load at or under its ceiling, so this holds the load
// at or under the target.
func ceilingTargetFindings(hosts []guidanceHost, targets map[string]int) []string {
	var out []string
	for _, h := range hosts {
		target, ok := targets[h.Name]
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("%s has no target in M-0349's inventory", h.Name))
		case h.Ceiling > target:
			out = append(out, fmt.Sprintf("%s's ceiling %d is above M-0349's target %d", h.Name, h.Ceiling, target))
		}
	}
	return out
}

// TestE0092Inventory holds M-0349's inventory to its rules against the
// tree (M-0349 AC-1, AC-2, AC-4): every section of the guidance as frozen
// is cited by a row, and each development section's words are its rows';
// every row is well formed and ends where its audience and placement say.
// Every removal since M-0336's start carries out its row (M-0336 AC-5,
// AC-6). It logs M-0349 AC-5's targets and holds each host's ceiling at or
// under its target (M-0337 AC-5).
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
	exists := func(p string) bool {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		return err == nil
	}
	findings = append(findings, inventoryRowFindings(rows, projectguidance.RoutedDocuments(inventoryRouterDraft(body)), exists)...)

	frozen := func(p string) string {
		out, err := exec.Command("git", "-C", root, "show", inventoryFreeze+":"+p).Output()
		if err != nil {
			t.Fatalf("reading %s at the freeze: %v", p, err)
		}
		return string(out)
	}
	development := map[string]string{
		projectguidance.ClaudeFile: projectguidance.HandwrittenText(frozen(projectguidance.ClaudeFile)),
		projectguidance.AgentsFile: projectguidance.HandwrittenText(frozen(projectguidance.AgentsFile)),
		projectguidance.RouterFile: frozen(projectguidance.RouterFile),
	}
	documents := maps.Clone(development)
	documents[inventoryFragment] = frozen(inventoryFragment)
	for _, p := range inventoryHostInserts {
		documents[p] = frozen(p)
	}
	frozenFragment, currentFragment := map[string]string{}, map[string]string{}
	for _, p := range append([]string{inventoryFragment}, inventoryHostInserts...) {
		frozenFragment[p] = documents[p]
		current, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		currentFragment[p] = string(current)
	}
	findings = append(findings, inventoryCoverageFindings(rows, documents)...)
	findings = append(findings, inventoryWordFindings(rows, development)...)
	rowText := func(r invRow) string {
		h := r.Homes[0]
		for _, s := range guidanceSections(documents[h.File], h.File == inventoryFragment) {
			if citeNames(h.Path, s) {
				return s.Text
			}
		}
		return ""
	}
	changedBefore := func(sha, path string) bool {
		out, err := exec.Command("git", "-C", root, "log", "--format=%H", inventoryReconcileBase+".."+sha+"^", "--", path).Output()
		if err != nil {
			t.Fatalf("reading the commits that change %s: %v", path, err)
		}
		return strings.TrimSpace(string(out)) != ""
	}
	blocks, malformed := repoRemovalBlocks(t, root)
	findings = append(findings, malformed...)
	findings = append(findings, inventoryReconcileFindings(blocks, rows, rowText, exists,
		func(id string) bool { return check.EntityResolves(tr, id) }, changedBefore)...)
	findings = append(findings, fragmentLossFindings(frozenFragment, currentFragment)...)
	claude, codex := inventoryCeilingTargets(rows, inventoryRouterDraft(body), development, exists)
	t.Logf("M-0349 AC-5 handwritten primed targets: claude-code %d, codex %d", claude, codex)
	findings = append(findings, ceilingTargetFindings(guidanceHosts, map[string]int{"claude-code": claude, "codex": codex})...)
	for _, f := range findings {
		t.Error(f)
	}
}

// inventoryFixture is a milestone body with one on-demand document and the
// given inventory rows under the given header.
func inventoryFixture(header string, rows ...string) string {
	return "## On-demand documents\n\n```markdown\n- [Testing](project/testing.md) — tests.\n```\n\n## Inventory\n\n" +
		header + "\n|---|---|---|---|---|---|---|---|---|\n" + strings.Join(rows, "\n") + "\n"
}

// inventoryHeader is the table's header line.
var inventoryHeader = "| " + strings.Join(inventoryColumns, " | ") + " |"

// TestInventoryRowFindings pins each row rule, one case per rule, against
// a valid table that reports nothing.
func TestInventoryRowFindings(t *testing.T) {
	t.Parallel()
	const moved = "| moved | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` | |"
	files := []string{
		"CLAUDE.md", "AGENTS.md", ".guidance/project.md", inventoryFragment, inventoryHostInserts[1], "go.mod",
		".guidance/packs/go/cobra/guide.md", "internal/skills/embedded/aiwf-add/SKILL.md",
		"internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/adr.md",
	}
	// The empty path exists, as the repository root does on disk, so the
	// empty-home rule is what reports an empty home.
	exists := func(p string) bool { return p == "" || slices.Contains(files, p) }
	cases := []struct {
		name   string
		header string
		rows   []string
		want   string
	}{
		{"a valid table reports nothing", inventoryHeader, []string{
			moved,
			"| kept | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | keep | |",
			"| merged | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | merge into moved | |",
			"| pointed | A rule | `CLAUDE.md › Testing` | 5 | development | check-backed | primed | pointer to `some-check` | |",
			"| copied | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` | |",
			"| skilled | A rule | `CLAUDE.md › Testing` | 5 | operating | judgment | on demand | move to the shipped skill `internal/skills/embedded/aiwf-add/SKILL.md` | |",
			"| shipped | A rule | `CLAUDE.md › Testing` | 5 | operating | judgment | primed | move to the shipped fragment | |",
			"| fragment-kept | A rule | `F › A rule` | 5 | operating | judgment | primed | keep | |",
			"| codex-kept | A rule | `AGENTS.md` | 5 | development | judgment | primed | keep | |",
			"| router-kept | A rule | `.guidance/project.md` | 5 | development | judgment | primed | tighten | |",
			"| templated | A rule | `CLAUDE.md › Testing` | 5 | operating | judgment | on demand | delete as copy of `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/adr.md` | |",
			"| fragment-out | A rule | `F › A rule` | 5 | development | judgment | primed | move to repository guidance `CLAUDE.md` | |",
		}, ""},
		{"a header out of order", strings.Replace(inventoryHeader, "Audience | Kind", "Kind | Audience", 1), []string{moved}, "header"},
		{"a row with the wrong number of cells", inventoryHeader, []string{"| short | A rule | 5 |"}, "cells, want"},
		{"a word count that is not a count", inventoryHeader, []string{strings.Replace(moved, "| 5 |", "| 0 |", 1)}, "not a positive count"},
		{"a label used twice", inventoryHeader, []string{moved, moved}, "more than one row"},
		{"a label outside the label shape", inventoryHeader, []string{strings.Replace(moved, "| moved |", "| Moved_Rule |", 1)}, "a label is"},
		{"a home in no file", inventoryHeader, []string{strings.Replace(moved, "`CLAUDE.md › Testing`", "`absent.md › Testing`", 1)}, "names no file"},
		{"an empty home", inventoryHeader, []string{strings.Replace(moved, "`CLAUDE.md › Testing`", "``", 1)}, "names no file"},
		{"a row naming no home", inventoryHeader, []string{strings.Replace(moved, "`CLAUDE.md › Testing`", "none", 1)}, "names no home"},
		{"a value outside a closed set", inventoryHeader, []string{strings.Replace(moved, "| judgment |", "| opinion |", 1)}, "outside its closed set"},
		{"a disposition outside the closed set", inventoryHeader, []string{strings.Replace(moved, "move on demand to", "relocate to", 1)}, "outside the closed set"},
		{"an undecided conflict", inventoryHeader, []string{"| torn | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | conflict | |"}, "undecided conflict"},
		{"a fragment rule taking a disposition its home does not allow", inventoryHeader, []string{"| frag | A rule | `F › A rule` | 5 | operating | judgment | primed | tighten | |"}, "takes only keep"},
		{"a host insert taking a disposition its home does not allow", inventoryHeader, []string{"| ins | A rule | `" + inventoryHostInserts[1] + "` | 5 | operating | judgment | primed | tighten | |"}, "takes only keep"},
		{"a merge into no row", inventoryHeader, []string{"| merged | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | merge into absent | |"}, "names no row"},
		{"a merge into a row that merges", inventoryHeader, []string{
			"| a | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | merge into b | |",
			"| b | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | merge into c | |",
			"| c | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | keep | |",
		}, "itself merges"},
		{"an on-demand destination the router draft does not name", inventoryHeader, []string{strings.Replace(moved, "testing.md", "other.md", 1)}, "not a named on-demand document"},
		{"a destination that does not exist", inventoryHeader, []string{"| copied | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | delete as copy of `absent.md` | |"}, "does not exist"},
		{"a final home that is no guidance home", inventoryHeader, []string{"| copied | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | delete as copy of `go.mod` | |"}, "no guidance home"},
		{"a final home that disagrees with the row", inventoryHeader, []string{strings.Replace(moved, "| on demand |", "| primed |", 1)}, "but the row is"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := inventoryFixture(tc.header, tc.rows...)
			rows, findings := parseInventory(body)
			findings = append(findings, inventoryRowFindings(rows, projectguidance.RoutedDocuments(inventoryRouterDraft(body)), exists)...)
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

// TestInventoryCoverageFindings pins the coverage rules: a ## or ###
// heading with text of its own needs a cite that names it; one holding
// only the sections beneath it is covered by a cite of one of them; a ####
// heading need not be cited; a preamble needs a bare cite; every top-level
// rule of the fragment is cited; and a cite names exactly one section of
// its file.
func TestInventoryCoverageFindings(t *testing.T) {
	t.Parallel()
	claude := "# Title\n\nPreamble.\n\n---\n\n## Testing\n\nIntro.\n\n### Rules\n\nText.\n\n#### Detail\n\nText.\n\n```bash\n## not a heading\n```\n\n## Group\n\n### Inner\n\nText.\n"
	fragment := "# Guidance\n\n- **A rule about\n  things.** Body.\n- **Another rule; with more.** Body.\n"
	row := func(homes string) string {
		return "| r | A rule | " + homes + " | 5 | development | judgment | primed | keep | |"
	}
	all := row("`CLAUDE.md`; `CLAUDE.md › Testing`; `CLAUDE.md › Testing › Rules`; `CLAUDE.md › Group › Inner`; `F › A rule about things`; `F › Another rule`")
	cases := []struct {
		name string
		row  string
		want string
	}{
		{"every required section cited, a text-less heading through its child, no #### cite", all, ""},
		{"a section with text cited only through a subsection", strings.Replace(all, " `CLAUDE.md › Testing`;", "", 1), "Testing is cited by no row"},
		{"an uncited ### heading", strings.Replace(all, " `CLAUDE.md › Testing › Rules`;", "", 1), "Testing › Rules is cited by no row"},
		{"a preamble cited only through a section", strings.Replace(all, "`CLAUDE.md`; ", "", 1), "CLAUDE.md ›  is cited by no row"},
		{"an uncited fragment rule", strings.Replace(all, "; `F › Another rule`", "", 1), "Another rule; with more is cited by no row"},
		{"a cite naming no section", strings.Replace(all, "`F › Another rule`", "`F › Another rule`; `CLAUDE.md › Testing › Missing`", 1), "names no section"},
		{"a lead-in cite with a path beneath it", strings.Replace(all, "`F › Another rule`", "`F › Another rule`; `F › Another rule › More`", 1), "names no section"},
		{"a cite naming more than one section", strings.Replace(all, "`F › Another rule`", "`F › Another rule`; `F › A`", 1), "more than one section"},
		{"a lead-in holding a semicolon", strings.Replace(all, "`F › Another rule`", "`F › Another rule; with more`", 1), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows, _ := parseInventory(inventoryFixture(inventoryHeader, tc.row))
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

// TestInventoryWordFindings pins the word rule: each section's own words,
// without its heading, the H1 or --- separators but with what a fence
// holds, equal the words of the rows citing it first.
func TestInventoryWordFindings(t *testing.T) {
	t.Parallel()
	claude := "# Title words\n\nOne two three.\n\n---\n\n## Section\n\nFour five.\n\n```\n---\n```\n"
	for _, tc := range []struct {
		name, sectionWords, want string
	}{
		{"words that match, a fenced line counted", "5", ""},
		{"words that differ", "6", "has 5 words; the rows citing it first count 6"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows, _ := parseInventory(inventoryFixture(inventoryHeader,
				"| a | A rule | `CLAUDE.md` | 3 | development | judgment | primed | keep | |",
				"| b | A rule | `CLAUDE.md › Section`; `CLAUDE.md` | "+tc.sectionWords+" | development | judgment | primed | keep | |",
			))
			findings := inventoryWordFindings(rows, map[string]string{"CLAUDE.md": claude})
			if tc.want == "" && len(findings) != 0 || tc.want != "" && (len(findings) != 1 || !strings.Contains(findings[0], tc.want)) {
				t.Errorf("findings = %q, want %q", findings, tc.want)
			}
		})
	}
}

// TestInventoryCeilingTargets pins how a target is derived: a kept,
// tightened or merged development rule ending in a primed host document
// counts at its words, with its section's headings and the H1; a moved
// rule does not count; the router counts at its drafted size; and Codex
// adds AGENTS.md to CLAUDE.md.
func TestInventoryCeilingTargets(t *testing.T) {
	t.Parallel()
	rows, _ := parseInventory(inventoryFixture(inventoryHeader,
		"| kept | A rule | `CLAUDE.md › Rules` | 10 | development | judgment | primed | keep | |",
		"| tight | A rule | `CLAUDE.md › Rules` | 20 | development | judgment | primed | tighten | |",
		"| merged | A rule | `CLAUDE.md › Other` | 5 | development | judgment | primed | merge into kept | |",
		"| moved | A rule | `CLAUDE.md › Other` | 40 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` | |",
		"| codex | A rule | `AGENTS.md` | 7 | development | judgment | primed | keep | |",
	))
	frozen := map[string]string{
		"CLAUDE.md": "# Repo rules\n\n## Rules\n\ntext\n\n## Other\n\ntext\n",
		"AGENTS.md": "# Codex\n\ntext\n",
	}
	claude, codex := inventoryCeilingTargets(rows, "three draft words", frozen, func(string) bool { return true })
	// CLAUDE.md: 10 + 20 + 5 rule words, 3 for "# Repo rules", 2 for "## Rules"; the router 3.
	if claude != 43 || codex != 43+7+2 {
		t.Errorf("targets = %d, %d; want 43, 52", claude, codex)
	}
}

// TestInventoryReconcileFindings pins each reconciliation rule (M-0336
// AC-5, AC-6), one case per rule, against a valid set that reports nothing.
func TestInventoryReconcileFindings(t *testing.T) {
	t.Parallel()
	rows, _ := parseInventory(inventoryFixture(inventoryHeader,
		"| copied | A rule | `CLAUDE.md › Go` | 5 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` | |",
		"| moved | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` | |",
		"| missing | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | on demand | move on demand to `.guidance/project/missing.md` | |",
		"| merged | A rule | `CLAUDE.md › Testing` | 5 | development | judgment | primed | merge into kept | |",
		"| shipped | A rule | `CLAUDE.md › Gates` | 5 | operating | judgment | primed | move to the shipped fragment | |",
		"| kept | A rule | `CLAUDE.md › Kept` | 5 | development | judgment | primed | keep | |",
		"| tightened | A rule | `CLAUDE.md › Kept` | 5 | development | judgment | primed | tighten | |",
		"| pointed | A rule | `CLAUDE.md › Kept` | 5 | development | check-backed | primed | pointer to `missing-check` | |",
		"| skilled | A rule | `CLAUDE.md › Gates` | 5 | operating | judgment | on demand | move to the shipped skill `internal/skills/embedded/aiwf-add/SKILL.md` | |",
		"| repoed | A rule | `F › A rule` | 5 | development | judgment | primed | move to repository guidance `CLAUDE.md` | |",
	))
	rowText := func(r invRow) string {
		if r.Label == "kept" {
			return "A rule, as G-0491 records for M-0336/AC-1.\n"
		}
		return ""
	}
	exists := func(p string) bool { return !strings.Contains(p, "missing") }
	resolves := func(id string) bool { return id == "G-0491" }
	// The fragment changes in commit "add", ahead of "late", which has no
	// such commit before it.
	moved := []string{inventoryFragment, "internal/skills/embedded/aiwf-add/SKILL.md", "CLAUDE.md"}
	changedBefore := func(sha, path string) bool { return sha != "late" && slices.Contains(moved, path) }
	block := func(sha, removed, disposition string) removalBlock {
		return removalBlock{sha, check.DispositionBlock{Removed: removed, Disposition: disposition}}
	}
	valid := []removalBlock{
		block("rm", `copied "x"`, "copy of .guidance/packs/go/cobra/guide.md"),
		block("rm", `moved "x"`, "relocated to .guidance/project/testing.md"),
		block("rm", `merged "x"`, "reworded"),
		block("rm", `shipped "x"`, "copy of "+inventoryFragment),
		block("rm", "kept G-0491", "copy of G-0491"),
		block("rm", `tightened "x"`, "reworded"),
		block("rm", `pointed "x"`, "pointer to missing-check"),
		block("rm", `skilled "x"`, "copy of internal/skills/embedded/aiwf-add/SKILL.md"),
		block("rm", `repoed "x"`, "copy of CLAUDE.md"),
	}
	cases := []struct {
		name  string
		block removalBlock
		want  string
	}{
		{"a valid set reports nothing", removalBlock{}, ""},
		{"a block naming no row", block("rm", `absent "x"`, "deleted"), "names no inventory row"},
		{"a block whose Removed: is empty", block("rm", "", "deleted"), "names no inventory row"},
		{"a block naming only its row carries the row's decision", block("rm", "moved", "deleted"), "but row moved decides"},
		{"an id later in a quoted passage is not an id edit", block("rm", `moved "cites G-0491"`, "deleted"), "but row moved decides"},
		{"an id edit naming an id the row does not carry", block("rm", "kept G-0999", "reworded"), "does not carry"},
		{"an id edit naming a shorter id inside one the row carries", block("rm", "kept G-049", "reworded"), "does not carry"},
		{"an id edit naming an id the row carries only inside a composite", block("rm", "kept M-0336", "reworded"), "does not carry"},
		{"a quoted passage opening with an id is not an id edit", block("rm", `kept "G-0491 records"`, "deleted"), "removes nothing"},
		{"an id edit naming an entity that does not resolve", block("rm", "kept G-0491", "copy of G-0999"), "resolves to no entity"},
		{"a pointer naming an entity that does not resolve", block("rm", "kept G-0491", "pointer to G-0999"), "resolves to no entity"},
		{"a destination that does not exist", block("rm", `missing "x"`, "relocated to .guidance/project/missing.md"), "does not exist"},
		{"a destination path holding an id is judged as a path", block("rm", "kept G-0491", "copy of docs/adr/ADR-0004-missing.md"), "does not exist"},
		{"a block disagreeing with its row", block("rm", `moved "x"`, "deleted"), "but row moved decides"},
		{"a block under a row that removes nothing", block("rm", `kept "x"`, "deleted"), "removes nothing"},
		{"a move removed before its new home changed", block("late", `shipped "x"`, "copy of "+inventoryFragment), "before any commit adds it"},
		{"a skill move removed before its new home changed", block("late", `skilled "x"`, "copy of internal/skills/embedded/aiwf-add/SKILL.md"), "before any commit adds it"},
		{"a repository move removed before its new home changed", block("late", `repoed "x"`, "copy of CLAUDE.md"), "before any commit adds it"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			blocks := slices.Clone(valid)
			if tc.want != "" {
				blocks = append(blocks, tc.block)
			}
			findings := inventoryReconcileFindings(blocks, rows, rowText, exists, resolves, changedBefore)
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

// TestFragmentLossFindings pins the fragment rule (M-0336 AC-5): words
// added among the fragment's frozen text pass, and a frozen word gone
// fails.
func TestFragmentLossFindings(t *testing.T) {
	t.Parallel()
	frozen := map[string]string{inventoryFragment: "- **Decide one thing at a time.** Each: the options.\n"}
	for _, tc := range []struct{ name, current, want string }{
		{"a clause added among the frozen words passes", "- **Decide one thing at a time.** The list ends with more. Each: the options.\n", ""},
		{"a frozen word gone fails", "- **Decide one thing.** Each: the options.\n", "words the fragment held at the freeze are gone"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			findings := fragmentLossFindings(frozen, map[string]string{inventoryFragment: tc.current})
			if tc.want == "" && len(findings) != 0 || tc.want != "" && (len(findings) != 1 || !strings.Contains(findings[0], tc.want)) {
				t.Errorf("findings = %q, want %q", findings, tc.want)
			}
		})
	}
}

// M-0337's audit. Its table holds a row, with the check's message and a
// verdict, for every policy and finding code the development guidance
// named when the milestone started (M-0337 AC-1). These tests go with the
// file at E-0092's wrap, and their guidance-reader entry with them.
const (
	// auditEntity is the milestone whose body holds the audit table.
	auditEntity = "M-0337"
	// auditBase is the commit that promoted M-0337 to in_progress; the
	// guidance as it stood there is what the audit covers.
	auditBase = "65d8462221205a42a3f76a4acd3caf7ee80470e7"
)

// auditColumns is the audit table's header, in order.
var auditColumns = []string{"Check", "Kind", "Message", "Verdict"}

// auditVerdicts is the closed set a row's verdict takes.
var auditVerdicts = []string{"states the fix", "does not"}

// policyFunc is a policy's entry-point declaration.
var policyFunc = regexp.MustCompile(`(?m)^func (Policy[A-Za-z0-9]+)\(`)

// namedChecks returns each policy id and finding code the documents name,
// mapped to its kind. A policy is named by its id, by the file that
// constructs its violations, or by a Policy function declared in that
// file; byFile maps a policy file to the ids it constructs, byFunc a
// Policy function to its file.
func namedChecks(docs map[string]string, byFile map[string][]string, byFunc map[string]string, codes map[string]struct{}) map[string]string {
	policies := map[string]bool{}
	for _, ids := range byFile {
		for _, id := range ids {
			policies[id] = true
		}
	}
	out := map[string]string{}
	for _, text := range docs {
		for _, m := range guidanceSpan.FindAllStringSubmatch(text, -1) {
			span := m[1]
			if file, ok := byFunc[span]; ok {
				span = file
			}
			switch _, isCode := codes[span]; {
			case policies[span]:
				out[span] = "policy"
			case isCode:
				out[span] = "finding"
			default:
				for _, id := range byFile[span] {
					out[id] = "policy"
				}
			}
		}
	}
	return out
}

// parseAudit reads the rows of the table under a body's Audit heading,
// keyed by check. A header or row that cannot be read is a finding.
func parseAudit(body string) (verdicts map[string]string, findings []string) {
	verdicts = map[string]string{}
	for _, line := range strings.Split(extractMarkdownSection(body, 2, "Audit"), "\n") {
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), " | ")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		switch {
		case cells[0] == auditColumns[0]:
			if !slices.Equal(cells, auditColumns) {
				findings = append(findings, fmt.Sprintf("header %q, want %q", cells, auditColumns))
			}
		case len(cells) != len(auditColumns):
			findings = append(findings, fmt.Sprintf("row %q has %d cells, want %d", line, len(cells), len(auditColumns)))
		default:
			verdicts[strings.Trim(cells[0], "`")] = cells[3]
		}
	}
	return verdicts, findings
}

// auditCoverageFindings reports each named check the table has no row
// for, and each row whose verdict is outside the closed set.
func auditCoverageFindings(named, verdicts map[string]string) []string {
	var out []string
	for _, c := range slices.Sorted(maps.Keys(named)) {
		if _, ok := verdicts[c]; !ok {
			out = append(out, fmt.Sprintf("%s %s is named by the development guidance and has no audit row", named[c], c))
		}
	}
	for _, c := range slices.Sorted(maps.Keys(verdicts)) {
		if !slices.Contains(auditVerdicts, verdicts[c]) {
			out = append(out, fmt.Sprintf("%s: verdict %q, want one of %q", c, verdicts[c], auditVerdicts))
		}
	}
	return out
}

// policyFiles maps each policy source file to the ids it constructs and
// each Policy function to the file declaring it.
func policyFiles(t *testing.T, root string) (byFile map[string][]string, byFunc map[string]string) {
	t.Helper()
	sites, err := constructionSites(root)
	if err != nil {
		t.Fatal(err)
	}
	byFile, byFunc = map[string][]string{}, map[string]string{}
	for _, s := range sites {
		if !slices.Contains(byFile[s.file], s.id) {
			byFile[s.file] = append(byFile[s.file], s.id)
		}
	}
	for file := range byFile {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range policyFunc.FindAllStringSubmatch(string(data), -1) {
			byFunc[m[1]] = file
		}
	}
	return byFile, byFunc
}

// TestM0337Audit holds M-0337's audit table to the development guidance
// as it stood at the milestone's start (M-0337 AC-1): every policy and
// finding code it named has a row with a verdict.
func TestM0337Audit(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)
	e := tr.ByID(auditEntity)
	if e == nil {
		t.Fatalf("%s not found in the tree", auditEntity)
	}
	body, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatal(err)
	}
	if out, gitErr := exec.Command("git", "-C", root, "cat-file", "-e", auditBase+"^{commit}").CombinedOutput(); gitErr != nil {
		t.Fatalf("reading the guidance at %s needs that commit: %v\n%s", auditBase, gitErr, out)
	}
	docs := developmentGuidance(func(p string) (string, bool) {
		out, gitErr := exec.Command("git", "-C", root, "show", auditBase+":"+p).Output()
		return string(out), gitErr == nil
	})
	files, err := WalkGoFiles(root, true)
	if err != nil {
		t.Fatal(err)
	}
	byFile, byFunc := policyFiles(t, root)
	named := namedChecks(docs, byFile, byFunc, allCheckCodes(files))
	if len(named) == 0 {
		t.Fatalf("the guidance at %s names no policy or finding code; reading it failed", auditBase)
	}
	verdicts, findings := parseAudit(string(body))
	findings = append(findings, auditCoverageFindings(named, verdicts)...)
	for _, f := range findings {
		t.Error(f)
	}
}

// TestNamedChecks pins how the guidance names a check (M-0337 AC-1): by
// policy id, by policy file, by Policy function, or by finding code; any
// other span names none.
func TestNamedChecks(t *testing.T) {
	t.Parallel()
	byFile := map[string][]string{
		"internal/policies/a.go": {"policy-a"},
		"internal/policies/b.go": {"policy-b"},
		"internal/policies/c.go": {"policy-c"},
	}
	byFunc := map[string]string{"PolicyCee": "internal/policies/c.go"}
	codes := map[string]struct{}{"code-x": {}}
	got := namedChecks(map[string]string{
		"CLAUDE.md":              "Enforced by `policy-a`; see `internal/policies/b.go`.",
		".guidance/project/x.md": "Chokepoint: `PolicyCee`. Reported as `code-x`. Run `make ci` or `internal/policies/`.",
	}, byFile, byFunc, codes)
	want := map[string]string{"policy-a": "policy", "policy-b": "policy", "policy-c": "policy", "code-x": "finding"}
	if !maps.Equal(got, want) {
		t.Errorf("namedChecks = %v, want %v", got, want)
	}
}

// TestAuditCoverageFindings pins the audit table's rules (M-0337 AC-1): a
// named check needs a row, a row's verdict is from the closed set, and a
// header or row that cannot be read is reported.
func TestAuditCoverageFindings(t *testing.T) {
	t.Parallel()
	body := "## Audit\n\n| Check | Kind | Message | Verdict |\n|---|---|---|---|\n" +
		"| `policy-a` | policy | run X | states the fix |\n" +
		"| `code-x` | finding | bad | maybe |\n" +
		"| `short` | policy |\n\n## Next\n\n| Check | Kind | Message | Verdict |\n| `outside` | policy | m | does not |\n"
	verdicts, findings := parseAudit(body)
	findings = append(findings, auditCoverageFindings(map[string]string{"policy-a": "policy", "code-x": "finding", "policy-b": "policy"}, verdicts)...)
	want := []string{
		"has 2 cells",
		"policy policy-b is named by the development guidance and has no audit row",
		`code-x: verdict "maybe"`,
	}
	if len(findings) != len(want) {
		t.Fatalf("findings = %q, want %d", findings, len(want))
	}
	for i := range want {
		if !strings.Contains(findings[i], want[i]) {
			t.Errorf("finding %d = %q, want it to contain %q", i, findings[i], want[i])
		}
	}
	if _, f := parseAudit("## Audit\n\n| Check | Message | Verdict |\n"); len(f) != 1 || !strings.Contains(f[0], "header") {
		t.Errorf("a wrong header: findings = %q, want one header finding", f)
	}
}

// TestCeilingTargetFindings pins M-0337 AC-5's rule: a host's ceiling at
// or under its target passes, and one above it, or a host with no target,
// is reported.
func TestCeilingTargetFindings(t *testing.T) {
	t.Parallel()
	hosts := []guidanceHost{{Name: "claude-code", Ceiling: 100}, {Name: "codex", Ceiling: 201}, {Name: "other", Ceiling: 1}}
	got := ceilingTargetFindings(hosts, map[string]int{"claude-code": 100, "codex": 200})
	want := []string{"codex's ceiling 201 is above M-0349's target 200", "other has no target in M-0349's inventory"}
	if !slices.Equal(got, want) {
		t.Errorf("findings = %q, want %q", got, want)
	}
}
