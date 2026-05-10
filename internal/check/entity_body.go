package check

// This file holds the M-066 entity-body-empty rule.
//
// Each entity kind has a hardcoded list of load-bearing body sections
// that must contain non-empty prose. The rule walks the body, locates
// each named section by heading, and emits a finding when the section
// is empty between its heading and the next heading (or EOF).
//
// Per-kind dispatch:
//
//	epic        — `## Goal`, `## Scope`, `## Out of scope`
//	milestone   — `## Goal`, `## Approach`, `## Acceptance criteria`
//	gap         — `## What's missing`, `## Why it matters`
//	adr         — `## Context`, `## Decision`, `## Consequences`
//	decision    — `## Question`, `## Decision`, `## Reasoning`
//	contract    — `## Purpose`, `## Stability`
//	AC body     — under each `### AC-N — <title>` heading inside its
//	              parent milestone
//
// Definition of empty: between the section heading and the next
// heading (or EOF), no non-whitespace content other than headings
// themselves. Top-level (`## Section`) bodies treat sub-headings as
// content (a milestone's `## Acceptance criteria` is non-empty if it
// contains AC sub-headings, even with no parent-level prose). AC
// bodies (`### AC-N`) require true prose, since AC bodies are the
// leaf prose containers.
//
// HTML comments are stripped before the emptiness check so a bare
// `<!-- TODO: write this -->` does not satisfy the rule (M-066/AC-4).

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// requiredSectionsByKind lists the load-bearing top-level body
// sections for each entity kind. Order is the canonical render order
// in the kind's spec template. Sub-element kinds (AC) handled
// separately because their heading level is `###` and their parent
// is a milestone, not a standalone file.
var requiredSectionsByKind = map[entity.Kind][]string{
	entity.KindEpic:      {"Goal", "Scope", "Out of scope"},
	entity.KindMilestone: {"Goal", "Approach", "Acceptance criteria"},
	entity.KindGap:       {"What's missing", "Why it matters"},
	entity.KindADR:       {"Context", "Decision", "Consequences"},
	entity.KindDecision:  {"Question", "Decision", "Reasoning"},
	entity.KindContract:  {"Purpose", "Stability"},
}

// htmlCommentPattern matches a single HTML comment block,
// possibly multi-line. Used to strip operator-deferred placeholders
// before the emptiness check (M-066/AC-4).
var htmlCommentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)

// h2Heading matches a `## <name>` line. Captured group is the
// heading text (trimmed by the caller).
var h2Heading = regexp.MustCompile(`^##\s+(.+?)\s*$`)

// h3ACHeading matches a `### AC-N — <title>` line (separator may be
// em-dash, hyphen, or colon — same permissive shape as
// acsBodyCoherence's locator). Capture: AC id integer.
var h3ACHeading = regexp.MustCompile(`^###\s+AC-(\d+)(?:\s*[—\-:]\s*(.+))?$`)

// ApplyTDDStrict bumps every entity-body-empty finding's severity
// from warning to error when strict=true (M-066/AC-2). Mutates the
// findings slice in place. The function is the single source of
// truth for which codes are covered by `aiwf.yaml: tdd.strict` —
// today only entity-body-empty; M-065's `milestone-tdd-undeclared`
// will be added to the same bumper when its rule lands. The bumper
// is intentionally narrow: codes outside this set pass through
// unchanged regardless of the flag.
//
// Callers run this AFTER `Run` (or after appending the rule's
// findings to their own slice) so the rule's emission stays
// config-agnostic and the strictness bump is a separate, testable
// transformation.
func ApplyTDDStrict(findings []Finding, strict bool) {
	if !strict {
		return
	}
	for i := range findings {
		if findings[i].Code == "entity-body-empty" {
			findings[i].Severity = SeverityError
		}
	}
}

// entityBodyEmpty fires for any entity whose load-bearing body
// section is empty. Warning severity by default; severity escalation
// to error under aiwf.yaml tdd.strict is applied separately via
// ApplyTDDStrict (M-066/AC-2).
func entityBodyEmpty(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		fullPath := filepath.Join(t.Root, e.Path)
		raw, err := os.ReadFile(fullPath)
		if err != nil {
			// Missing-file path is already covered by the loader's
			// load-error finding; the body check stays silent.
			continue
		}
		_, body, ok := entity.Split(raw)
		if !ok {
			continue
		}
		stripped := stripHTMLComments(body)

		// Lifecycle gate (M-075/AC-2, closes G-071 case 2): terminal-
		// status entities are preserved historical artifacts; warning
		// about empty body sections perpetually after the entity has
		// reached `done`/`cancelled`/`superseded`/`rejected`/`addressed`/
		// `wontfix`/`retired` is noise. The rule was scoped to catch
		// active drafting; the predicate keeps it scoped to live entities.
		if entity.IsTerminal(e.Kind, e.Status) {
			continue
		}

		// Top-level body sections.
		// coverage:ignore-on-miss — `requiredSectionsByKind` covers
		// every top-level entity kind; the `has=false` arm only fires
		// for synthetic/unknown Kind values that the tree loader does
		// not produce. Documented unreachable in production.
		if sections, has := requiredSectionsByKind[e.Kind]; has {
			present := scanH2Sections(stripped)
			for _, name := range sections {
				content, found := present[name]
				if !found {
					continue
				}
				if isAllWhitespaceOrHeadings(content, false) {
					findings = append(findings, Finding{
						Code:     "entity-body-empty",
						Severity: SeverityWarning,
						Subcode:  string(e.Kind),
						Message: fmt.Sprintf("%s body section `## %s` is empty",
							e.ID, name),
						Path:     e.Path,
						EntityID: e.ID,
						Field:    "body",
					})
				}
			}
		}

		// AC sub-element bodies (under a milestone parent).
		// Lifecycle gate (M-075/AC-3, closes G-071 case 1): when the
		// parent milestone is `draft`, freshly-allocated ACs have
		// empty bodies by design — `aiwfx-plan-milestones` ships shape
		// first, prose lands as TDD work begins. Warning before the
		// milestone promotes to `in_progress` is noise.
		if e.Kind == entity.KindMilestone && e.Status != entity.StatusDraft {
			acBodies := scanACBodies(stripped)
			for _, ac := range e.ACs {
				if ac.ID == "" || ac.Status == entity.StatusCancelled {
					continue
				}
				content, found := acBodies[ac.ID]
				if !found {
					continue
				}
				if isAllWhitespaceOrHeadings(content, true) {
					findings = append(findings, Finding{
						Code:     "entity-body-empty",
						Severity: SeverityWarning,
						Subcode:  "ac",
						Message: fmt.Sprintf("%s/%s body under `### %s` is empty",
							e.ID, ac.ID, ac.ID),
						Path:     e.Path,
						EntityID: e.ID + "/" + ac.ID,
						Field:    "acs",
					})
				}
			}
		}
	}
	return findings
}

// stripHTMLComments removes HTML comment blocks from body bytes.
// Operator-deferred placeholders (`<!-- TODO: write this -->`) do
// not satisfy the non-empty requirement.
func stripHTMLComments(body []byte) []byte {
	return htmlCommentPattern.ReplaceAll(body, nil)
}

// scanH2Sections walks body bytes line by line and returns a map of
// section heading → content bytes between that heading and the next
// `## ` heading (or EOF). Sub-headings (`###`, `####`, …) are
// included verbatim in the content; the caller decides how to count
// them.
func scanH2Sections(body []byte) map[string][]byte {
	out := map[string][]byte{}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var (
		currentName    string
		currentContent []byte
	)
	flush := func() {
		if currentName != "" {
			out[currentName] = currentContent
		}
	}
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if m := h2Heading.FindStringSubmatch(line); m != nil {
			flush()
			currentName = strings.TrimSpace(m[1])
			currentContent = nil
			continue
		}
		if currentName == "" {
			continue
		}
		currentContent = append(currentContent, []byte(line+"\n")...)
	}
	flush()
	return out
}

// scanACBodies walks body bytes line by line and returns a map of
// AC id → content bytes between that `### AC-N` heading and the next
// `###` (or `## `) heading or EOF. Used for AC body emptiness checks.
func scanACBodies(body []byte) map[string][]byte {
	out := map[string][]byte{}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var (
		currentID      string
		currentContent []byte
	)
	flush := func() {
		if currentID != "" {
			out[currentID] = currentContent
		}
	}
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if m := h3ACHeading.FindStringSubmatch(line); m != nil {
			flush()
			currentID = "AC-" + m[1]
			currentContent = nil
			continue
		}
		// A `## ` heading or any other `### `/`#### ` heading ends
		// the current AC body region.
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			flush()
			currentID = ""
			currentContent = nil
			continue
		}
		if currentID == "" {
			continue
		}
		currentContent = append(currentContent, []byte(line+"\n")...)
	}
	flush()
	return out
}

// isAllWhitespaceOrHeadings reports whether content is empty in the
// rule's sense.
//
//	leafLevel=true  (AC body)   — only non-heading non-whitespace
//	                              content counts; sub-headings of any
//	                              level are also "empty".
//	leafLevel=false (top-level) — any non-whitespace content counts,
//	                              including sub-headings.
//
// Whitespace and blank lines never count.
func isAllWhitespaceOrHeadings(content []byte, leafLevel bool) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))
		if line == "" {
			continue
		}
		if leafLevel && strings.HasPrefix(line, "#") {
			continue
		}
		return false
	}
	return true
}
