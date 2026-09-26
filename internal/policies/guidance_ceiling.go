package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/projectguidance"
	"github.com/23min/aiwf/internal/skills"
)

// PolicyGuidanceCeiling holds each host's handwritten primed load at or
// under its ceiling. The primed load is what a host reads before any
// task: its entry point outside aiwf's managed blocks, plus every
// document a required read reaches — the project router the routing block
// sends it to first among them. The managed blocks and the pack index are
// aiwf's output and are measured as a separate figure, not held here; the
// fragment is reduced only by its own milestone.
//
// The measure is an explicit model of this repository's routing, not an
// inspection of a model's context. A reference is a markdown link, in any
// CommonMark form, or an `@` import in prose; a path named any other way —
// in backticks, or in a sentence — is not one, and a "read X in full"
// written that way is outside the model. Claude Code expands an import in
// its memory — its entry point and the files that imports, recursively — so
// there an import is a required read without further declaration; anywhere
// else, and for Codex, the line is text. A
// link from a host entry point's handwritten text must be classified in
// guidanceReadTable as a required or a conditional read, and one the table
// does not classify is reported rather than silently counted as either. A
// link from the router is conditional unless the table says otherwise,
// since routing on demand is its job. Every target must exist.
//
// The ceiling is internal to this repository; consumers get none
// (ADR-0053).
func PolicyGuidanceCeiling(root string) ([]Violation, error) {
	return guidanceCeilingViolations(repoGuidanceReader(root), guidanceHosts, guidanceReadTable), nil
}

// readKind classifies a reference from primed text.
type readKind int

const (
	// readRequired is a reference the host must read in full before any
	// task, so its target joins the primed load.
	readRequired readKind = iota + 1
	// readConditional is a reference read only when a task needs it.
	readConditional
	// readGenerated is a required read of a document aiwf generates, so
	// its words join the aiwf-generated figure rather than the ceiling.
	readGenerated
)

// guidanceRef is one reference, from the document that makes it to the
// repository path it names.
type guidanceRef struct{ From, To string }

// guidanceHost is one host's entry point and its handwritten ceiling.
type guidanceHost struct {
	Name, Entry string
	Ceiling     int
}

// guidanceLoad is a host's measured primed load, in whitespace-separated
// words.
type guidanceLoad struct {
	Handwritten, Generated int
	RequiredReads          []string
}

// guidanceHosts are this repository's two hosts. Each ceiling is the
// host's measured handwritten primed load; E-0092's later milestones
// lower them.
var guidanceHosts = []guidanceHost{
	{Name: "claude-code", Entry: fenceClaudeMD, Ceiling: 9558},
	{Name: "codex", Entry: fenceAgentsMD, Ceiling: 9687},
}

// guidanceReadTable classifies every reference primed text makes today.
// Both routing blocks send the host to the project router first and then
// the pack index before any task, so the router is primed and the index,
// which aiwf generates, joins the generated figure. AGENTS.md's preamble
// requires reading CLAUDE.md in full, so for Codex that read is primed;
// every reference CLAUDE.md makes is read when a task needs it.
var guidanceReadTable = map[guidanceRef]readKind{
	{From: fenceClaudeMD, To: fenceRouter}:                                                                                 readRequired,
	{From: fenceAgentsMD, To: fenceRouter}:                                                                                 readRequired,
	{From: fenceClaudeMD, To: ".guidance/index.md"}:                                                                        readGenerated,
	{From: fenceAgentsMD, To: ".guidance/index.md"}:                                                                        readGenerated,
	{From: fenceAgentsMD, To: fenceClaudeMD}:                                                                               readRequired,
	{From: fenceClaudeMD, To: ".claude/hooks/validate-agent-isolation.sh"}:                                                 readConditional,
	{From: fenceClaudeMD, To: ".devcontainer/README.md"}:                                                                   readConditional,
	{From: fenceClaudeMD, To: ".devcontainer/devcontainer.json"}:                                                           readConditional,
	{From: fenceClaudeMD, To: ".devcontainer/initialize.sh"}:                                                               readConditional,
	{From: fenceClaudeMD, To: ".github/workflows/changelog-check.yml"}:                                                     readConditional,
	{From: fenceClaudeMD, To: "CHANGELOG.md"}:                                                                              readConditional,
	{From: fenceClaudeMD, To: "CONTRIBUTING.md"}:                                                                           readConditional,
	{From: fenceClaudeMD, To: "README.md"}:                                                                                 readConditional,
	{From: fenceClaudeMD, To: "docs/adr/ADR-0006-skills-policy-per-verb-default-or-help-only.md"}:                          readConditional,
	{From: fenceClaudeMD, To: "docs/adr/ADR-0015-settings-json-edits-require-explicit-per-invocation-consent.md"}:          readConditional,
	{From: fenceClaudeMD, To: "docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md"}: readConditional,
	{From: fenceClaudeMD, To: "docs/adr/ADR-0036-same-status-fsm-transitions-converge-to-noop-not-refusal.md"}:             readConditional,
	{From: fenceClaudeMD, To: "docs/adr/ADR-0037-retitle-re-derives-the-slug-only-while-it-tracks-the-title.md"}:           readConditional,
	{From: fenceClaudeMD, To: "docs/adr/ADR-0038-refuse-verb-writes-over-head-divergent-entity-content.md"}:                readConditional,
	{From: fenceClaudeMD, To: "docs/archive/migration/from-prior-systems.md"}:                                              readConditional,
	{From: fenceClaudeMD, To: "docs/design/design-decisions.md"}:                                                           readConditional,
	{From: fenceClaudeMD, To: "docs/design/design-lessons.md"}:                                                             readConditional,
	{From: fenceClaudeMD, To: "docs/design/provenance-model.md"}:                                                           readConditional,
	{From: fenceClaudeMD, To: "docs/explorations/loom/loom-by-example.md"}:                                                 readConditional,
	{From: fenceClaudeMD, To: "docs/explorations/loom/loom-light-plan.md"}:                                                 readConditional,
	{From: fenceClaudeMD, To: "scripts/sign-and-run.sh"}:                                                                   readConditional,
}

// guidanceCeilingViolations measures each host and reports its routing
// findings and a handwritten load above its ceiling.
func guidanceCeilingViolations(read func(string) (string, bool), hosts []guidanceHost, table map[guidanceRef]readKind) []Violation {
	var out []Violation
	// A routing finding in a document both hosts read is one finding.
	reported := map[Violation]bool{}
	for _, h := range hosts {
		load, vs := measureGuidanceLoad(read, h.Entry, table)
		for _, v := range vs {
			if !reported[v] {
				reported[v] = true
				out = append(out, v)
			}
		}
		if load.Handwritten > h.Ceiling {
			out = append(out, Violation{
				Policy: "guidance-ceiling",
				File:   h.Entry,
				Detail: fmt.Sprintf("%s's handwritten primed load is %d words, above its ceiling of %d (aiwf-generated load %d, reported separately); move text into an on-demand document the project router links to, or cut it, rather than raising the ceiling.", h.Name, load.Handwritten, h.Ceiling, load.Generated),
			})
		}
	}
	return out
}

// measureGuidanceLoad walks a host's primed load from its entry point.
// Each document is counted once, however many required reads reach it,
// so shared targets and cycles are both safe.
func measureGuidanceLoad(read func(string) (string, bool), entry string, table map[guidanceRef]readKind) (guidanceLoad, []Violation) {
	var load guidanceLoad
	var out []Violation
	entryContent, ok := read(entry)
	if !ok {
		return load, []Violation{{Policy: "guidance-ceiling", File: entry, Detail: "the host's entry point is missing, so its primed load cannot be measured."}}
	}
	// Claude Code expands an import only in its memory: its entry point and
	// the files that imports, recursively.
	memory := map[string]bool{}
	if entry == fenceClaudeMD {
		memory[entry] = true
	}
	load.Generated = generatedWords(read, entryContent, memory[entry])

	seen := map[string]bool{entry: true}
	queue := []string{entry}
	follow := func(doc, to string, kind readKind) {
		// Routing on demand is the router's job, so a link from it is
		// conditional unless the table declares otherwise.
		if kind == 0 && doc == fenceRouter {
			kind = readConditional
		}
		content, exists := read(to)
		switch {
		case !exists:
			out = append(out, Violation{Policy: "guidance-ceiling", File: to, Detail: fmt.Sprintf("%s references %s, which does not exist; routing that resolves nowhere is reported rather than left out of the measure.", doc, to)})
		case kind == 0:
			out = append(out, Violation{Policy: "guidance-ceiling", File: to, Detail: fmt.Sprintf("%s references %s from primed text, and guidanceReadTable does not classify it; add it as a required read (primed) or a conditional one (on demand).", doc, to)})
		case kind == readConditional || seen[to]:
		default:
			seen[to] = true
			load.RequiredReads = append(load.RequiredReads, to)
			if kind == readGenerated {
				load.Generated += len(strings.Fields(content))
			} else {
				queue = append(queue, to)
			}
		}
	}
	// In Claude's memory an import is a required read, since the host loads
	// it without being told to; it is a filesystem path, so one under the
	// home directory or absolute is personal or global material and is left
	// out. A link is classified by the table.
	followText := func(doc, text string) {
		for _, target := range markdownImports(text) {
			if p, ok := resolveReference(doc, target); ok && memory[doc] && !strings.HasPrefix(target, "~") && !strings.HasPrefix(target, "/") {
				memory[p] = true
				follow(doc, p, readRequired)
			}
		}
		for _, target := range markdownLinks(text) {
			if p, ok := resolveReference(doc, target); ok {
				follow(doc, p, table[guidanceRef{From: doc, To: p}])
			}
		}
	}
	// The routing block is aiwf's, but where it sends the host is read
	// before the task: its links are classified like handwritten ones.
	followText(entry, blockText(entryContent, projectguidance.RouteMarkers))
	for len(queue) > 0 {
		doc := queue[0]
		queue = queue[1:]
		text, _ := read(doc)
		if doc == fenceClaudeMD || doc == fenceAgentsMD {
			text = handwrittenText(text)
		}
		counted := text
		if memory[doc] {
			counted = withoutImports(text)
		}
		load.Handwritten += len(strings.Fields(counted))
		followText(doc, text)
	}
	return load, out
}

// blockText returns the text inside one of a host entry point's managed
// blocks, or "" when it has none.
func blockText(content string, markers func() (start, end, prefix string)) string {
	start, end, prefix := markers()
	from, to, err := pathutil.ManagedBlockSpan(content, start, end, prefix)
	if err != nil || from < 0 {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(content[from:to], start), end)
}

// withoutImports drops import lines, which are directives rather than
// text and count no words themselves.
func withoutImports(text string) string {
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "@") && !strings.ContainsAny(trimmed, " \t") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// generatedWords counts the words inside a host entry point's managed
// blocks. Where the host expands imports, an import counts as the file it
// loads: the Claude block holds only an import of the materialized
// fragment. The Codex block holds the fragment itself, and an `@` line
// there is text.
func generatedWords(read func(string) (string, bool), content string, imports bool) int {
	n := 0
	for _, markers := range managedBlockMarkers {
		block := blockText(content, markers)
		if !imports {
			n += len(strings.Fields(block))
			continue
		}
		n += len(strings.Fields(withoutImports(block)))
		for _, target := range markdownImports(block) {
			imported, _ := read(target)
			n += len(strings.Fields(imported))
		}
	}
	return n
}

// repoGuidanceReader reads this repository's files from disk. The
// materialized Claude fragment is gitignored, so absent in CI; it reads as
// the embedded source rendered, which is what the import loads. A
// directory exists and holds no text of its own.
func repoGuidanceReader(root string) func(string) (string, bool) {
	return func(p string) (string, bool) {
		if p == skills.GuidanceFile {
			rendered, err := skills.RenderGuidance("")
			return string(rendered), err == nil
		}
		full := filepath.Join(root, filepath.FromSlash(p))
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			return "", true
		}
		content, err := os.ReadFile(full)
		if err != nil {
			return "", false
		}
		return string(content), true
	}
}
