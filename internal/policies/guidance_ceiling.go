package policies

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/projectguidance"
	"github.com/23min/aiwf/internal/skills"
)

// PolicyGuidanceCeiling holds each host's handwritten primed load at or
// under its ceiling. The primed load is what a host reads before any
// task: its entry point outside aiwf's managed blocks, plus every
// document a required read reaches. The managed blocks — the aiwf
// fragment and the routing block — are measured as a separate figure and
// not held here; the fragment is reduced only by its own milestone.
//
// The measure is an explicit model of this repository's routing, not an
// inspection of a model's context. Claude loads an `@` import on its own,
// so an import is a required read without further declaration. Every
// other reference from primed text is classified in guidanceReadTable as
// a required or a conditional read, and a reference the table does not
// classify is reported rather than silently counted as either: a new
// "read X in full" cannot enter the primed load unseen.
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
	{Name: "claude-code", Entry: fenceClaudeMD, Ceiling: 9489},
	{Name: "codex", Entry: fenceAgentsMD, Ceiling: 9618},
}

// guidanceReadTable classifies every reference primed text makes today.
// AGENTS.md's preamble requires reading CLAUDE.md in full, so for Codex
// that read is primed; every reference CLAUDE.md makes is read when a
// task needs it.
var guidanceReadTable = map[guidanceRef]readKind{
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
	for _, h := range hosts {
		load, vs := measureGuidanceLoad(read, h.Entry, table)
		out = append(out, vs...)
		if load.Handwritten > h.Ceiling {
			out = append(out, Violation{
				Policy: "guidance-ceiling",
				File:   h.Entry,
				Detail: fmt.Sprintf("%s's handwritten primed load is %d words, above its ceiling of %d (aiwf-generated load %d, reported separately); move text on demand behind the project router or cut it, rather than raising the ceiling.", h.Name, load.Handwritten, h.Ceiling, load.Generated),
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
	load.Generated = generatedWords(read, entryContent)

	seen := map[string]bool{entry: true}
	queue := []string{entry}
	for len(queue) > 0 {
		doc := queue[0]
		queue = queue[1:]
		content, _ := read(doc)
		text := content
		if doc == fenceClaudeMD || doc == fenceAgentsMD {
			text = handwrittenText(content)
		}
		load.Handwritten += len(strings.Fields(withoutImports(text)))
		for _, ref := range guidanceReferences(doc, text) {
			kind := ref.kind
			if kind == 0 {
				kind = table[guidanceRef{From: doc, To: ref.to}]
			}
			if _, exists := read(ref.to); !exists {
				out = append(out, Violation{Policy: "guidance-ceiling", File: ref.to, Detail: fmt.Sprintf("%s references %s, which does not exist; routing that resolves nowhere is reported rather than left out of the measure.", doc, ref.to)})
				continue
			}
			switch kind {
			case readRequired:
				if !seen[ref.to] {
					seen[ref.to] = true
					load.RequiredReads = append(load.RequiredReads, ref.to)
					queue = append(queue, ref.to)
				}
			case readConditional:
			default:
				out = append(out, Violation{Policy: "guidance-ceiling", File: ref.to, Detail: fmt.Sprintf("%s references %s from primed text, and guidanceReadTable does not classify it; add it as a required read (primed) or a conditional one (on demand).", doc, ref.to)})
			}
		}
	}
	return load, out
}

// guidanceReference is one reference found in text. kind is readRequired
// for an import, which a host loads without being told to, and zero for
// a link, which the table classifies.
type guidanceReference struct {
	to   string
	kind readKind
}

// guidanceReferences returns the imports and markdown links in text,
// resolved against the directory of the document making them. A link
// with a scheme, a bare anchor, and an import from outside the
// repository — personal or global material — are not references into
// the repository and are left out.
func guidanceReferences(doc, text string) []guidanceReference {
	var out []guidanceReference
	for _, target := range importTargets(text) {
		if strings.HasPrefix(target, "~") || strings.HasPrefix(target, "/") {
			continue
		}
		out = append(out, guidanceReference{to: path.Clean(path.Join(path.Dir(doc), target)), kind: readRequired})
	}
	for _, m := range fenceRouterLink.FindAllStringSubmatch(text, -1) {
		target, _, _ := strings.Cut(m[1], "#")
		if target == "" || strings.Contains(target, ":") {
			continue
		}
		out = append(out, guidanceReference{to: path.Clean(path.Join(path.Dir(doc), target))})
	}
	return out
}

// importTargets returns the paths of the `@path` import lines in text.
func importTargets(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if target, ok := strings.CutPrefix(strings.TrimSpace(line), "@"); ok && target != "" && !strings.ContainsAny(target, " \t") {
			out = append(out, target)
		}
	}
	return out
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
// blocks, expanding the imports they carry: the Claude block holds only
// an import of the materialized fragment, the Codex block the fragment
// itself.
func generatedWords(read func(string) (string, bool), content string) int {
	n := 0
	for _, markers := range [][3]string{fenceMarkers(initrepo.GuidanceMarkers), fenceMarkers(projectguidance.RouteMarkers)} {
		start, end, err := pathutil.ManagedBlockSpan(content, markers[0], markers[1], markers[2])
		if err != nil || start < 0 {
			continue
		}
		block := strings.TrimSuffix(strings.TrimPrefix(content[start:end], markers[0]), markers[1])
		n += len(strings.Fields(withoutImports(block)))
		for _, target := range importTargets(block) {
			if imported, ok := read(target); ok {
				n += len(strings.Fields(imported))
			}
		}
	}
	return n
}

// repoGuidanceReader reads this repository's files from disk. The
// materialized Claude fragment is gitignored, so absent in CI; it reads as
// the embedded source rendered, which is what the import loads.
func repoGuidanceReader(root string) func(string) (string, bool) {
	return func(p string) (string, bool) {
		if p == skills.GuidanceFile {
			rendered, err := skills.RenderGuidance("")
			return string(rendered), err == nil
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			return "", false
		}
		return string(content), true
	}
}
