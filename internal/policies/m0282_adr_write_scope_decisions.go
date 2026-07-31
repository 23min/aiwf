package policies

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// adrWriteScopeID is the ADR whose Decision subsections this policy
// pins. Resolved through the loader rather than by path literal, so the
// assertion survives an archive sweep (ADR-0004) and a retitle.
const adrWriteScopeID = "ADR-0038"

// writeScopeDecisionHeadings names the `### ` subsections that must exist
// under the ADR's `## Decision` heading, one per acceptance criterion on
// M-0282. Order matches the ADR's own subsection order.
var writeScopeDecisionHeadings = []struct {
	AC      string
	Heading string
	WhatFor string
}{
	{"AC-1", "Seam", "where the precondition runs relative to the same-state comparison"},
	{"AC-2", "Path scope", "which paths the guard compares"},
	{"AC-5", "Field scope", "which parts of each file the guard compares"},
	{"AC-3", "Verdict", "refuse or warn"},
	{"AC-4", "Escape hatch", "whether an escape hatch exists and what it costs"},
}

// PolicyM0282ADRWriteScopeDecisions asserts that ADR-0038 has the shape
// M-0282's acceptance criteria require: it is accepted, it carries a
// named `### ` subsection per decision under `## Decision`, each with
// prose under it, and it carries a non-empty `## Consequences`.
//
// What this asserts is placement and presence, and the assertion is
// structural rather than a substring grep: a heading must sit under
// `## Decision` specifically, so the same words elsewhere in the document
// do not satisfy it.
//
// What it deliberately does NOT assert is that a subsection records a
// real decision. That was attempted with keyword matching and does not
// work, for a reason no amount of tightening fixes: a deferral names
// every term a decision would name ("whether it refuses or warns is
// deferred" contains "refuse"). The distinction is meaning, not
// vocabulary, so it is a review judgment and belongs at the human gate —
// which is where CLAUDE.md puts judgment classes no check can cover.
// Overstating this policy's reach is worse than its narrowness: it would
// be a chokepoint that reads as enforcing and does not.
//
// Pins M-0282/AC-1 through AC-5.
func PolicyM0282ADRWriteScopeDecisions(root string) ([]Violation, error) {
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil { //coverage:ignore tree.Load errors only on a filesystem fault (an unstattable walk root); a malformed entity surfaces as a LoadError, not an error here, so the arm needs a permissions fault the harness cannot stage portably
		return nil, fmt.Errorf("loading tree: %w", err)
	}
	e := tr.ByID(adrWriteScopeID)
	if e == nil {
		return []Violation{{
			Policy: "m0282-adr-write-scope-decisions",
			File:   adrWriteScopeID,
			Detail: "not found in the entity tree; M-0282's acceptance criteria require an ADR recording the five write-scope decisions",
		}}, nil
	}

	relPath := e.Path
	// The loader parses frontmatter only, so the body has to be read
	// here. It read this same file moments ago to produce the entity, so
	// a failure now means the file vanished or lost permissions mid-run:
	// an infrastructure fault rather than a policy violation, and
	// reported as an error so it cannot be mistaken for a clean pass.
	raw, readErr := os.ReadFile(filepath.Join(root, relPath))
	if readErr != nil { //coverage:ignore unreachable in practice: the loader already read this path successfully to build the entity
		return nil, fmt.Errorf("reading %s: %w", relPath, readErr)
	}
	doc := string(raw)

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0282-adr-write-scope-decisions",
			File:   relPath,
			Detail: detail,
		})
	}

	// A rejected or superseded ADR does not record a decision in force,
	// so the criteria it backs would be met by a document the project has
	// disowned.
	if e.Status != entity.StatusAccepted {
		report(fmt.Sprintf(
			"status is %q; the acceptance criteria require a decision in force, so only %q satisfies them",
			e.Status, entity.StatusAccepted))
	}

	for _, d := range writeScopeDecisionHeadings {
		body, ok := extractMarkdownSubsection(doc, "Decision", d.Heading)
		if !ok || strings.TrimSpace(body) == "" {
			report(fmt.Sprintf(
				"%s: `## Decision` has no non-empty `### %s` subsection recording %s",
				d.AC, d.Heading, d.WhatFor))
		}
	}

	// ParseBodySections keys by slug, not by display heading — the same
	// slugification `aiwf show --format=json` exposes as body keys.
	consequences, ok := entity.ParseBodySections([]byte(doc))[entity.SectionSlug("Consequences")]
	if !ok || strings.TrimSpace(consequences) == "" {
		report("`## Consequences` is missing or empty; each decision's cost is recorded there")
	}

	return vs, nil
}
