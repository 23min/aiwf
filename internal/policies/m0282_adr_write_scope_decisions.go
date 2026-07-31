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
// pins. Resolved through the loader rather than by path literal, so
// the assertion survives an archive sweep (ADR-0004) and a retitle.
const adrWriteScopeID = "ADR-0038"

// writeScopeDecision names one `### ` subsection required under the
// ADR's `## Decision` heading, together with the substantive marker its
// prose must carry. The marker is what distinguishes "the heading
// exists" from "the decision was actually recorded" — a subsection
// present but silent on its own verdict fails.
//
// AnyOf holds alternative spellings of the recorded verdict: the AC
// requires that a choice be recorded, not which choice, so a later
// amendment that flips a verdict keeps passing while a subsection that
// records no verdict at all fails.
//
// Requires holds markers that must ALL appear, for the criteria that
// oblige a decision to carry its reasoning rather than only its verdict.
// It is also what keeps the assertion from passing vacuously: a marker
// short enough to have several spellings also matches prose that decides
// nothing — "does not apply" contains "apply", "nonetheless" contains
// "none" — so the distinctive term is required rather than offered as
// one alternative among many.
type writeScopeDecision struct {
	AC       string
	Heading  string
	AnyOf    []string
	Requires []string
	WhatFor  string
}

// writeScopeDecisions is the agenda M-0282's acceptance criteria pin,
// one entry per AC. Order matches the ADR's own subsection order.
var writeScopeDecisions = []writeScopeDecision{
	{
		AC:       "AC-1",
		Heading:  "Seam",
		Requires: []string{"verb.Apply"},
		WhatFor:  "where the precondition runs relative to the same-state comparison",
	},
	{
		AC:      "AC-2",
		Heading: "Path scope",
		AnyOf:   []string{"committed path set", "committed-path"},
		// The AC obliges the nested case to be addressed in this
		// decision's own text, not merely listed elsewhere.
		Requires: []string{"nested"},
		WhatFor:  "whether the guard is entity-scoped or committed-path-scoped",
	},
	{
		AC:      "AC-5",
		Heading: "Field scope",
		AnyOf:   []string{"whole-file", "frontmatter-only"},
		WhatFor: "whether the guard compares frontmatter only or the whole file",
	},
	{
		AC:      "AC-3",
		Heading: "Verdict",
		AnyOf:   []string{"refuses", "refuse"},
		// The AC obliges the verdict to be weighed against the
		// illegal-transition escape, not only against misattribution.
		Requires: []string{"illegal-transition"},
		WhatFor:  "refuse or warn",
	},
	{
		AC:       "AC-4",
		Heading:  "Escape hatch",
		Requires: []string{"--force"},
		WhatFor:  "whether an escape hatch exists and what it costs",
	},
}

// PolicyM0282ADRWriteScopeDecisions asserts that ADR-0038 records each
// of the five write-scope decisions M-0282 exists to settle, each in
// its own `### ` subsection under `## Decision`, and that two of them
// carry their consequence in `## Consequences`.
//
// The assertions are structural, not substring greps over the file: a
// marker is required inside the *named* subsection, so prose that
// happens to mention "refuse" somewhere else in the document does not
// satisfy the verdict decision. This is the bar CLAUDE.md sets for
// doc-shaped acceptance criteria — a grep proves a literal exists
// somewhere, not that it exists in the right place.
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

	for _, d := range writeScopeDecisions {
		body, ok := extractMarkdownSubsection(doc, "Decision", d.Heading)
		if !ok || strings.TrimSpace(body) == "" {
			report(fmt.Sprintf(
				"%s: `## Decision` has no non-empty `### %s` subsection recording %s",
				d.AC, d.Heading, d.WhatFor))
			continue
		}
		if len(d.AnyOf) > 0 && !containsAny(body, d.AnyOf) {
			report(fmt.Sprintf(
				"%s: `### %s` records no verdict on %s (expected one of: %s)",
				d.AC, d.Heading, d.WhatFor, strings.Join(d.AnyOf, ", ")))
		}
		for _, req := range d.Requires {
			if !containsAny(body, []string{req}) {
				report(fmt.Sprintf(
					"%s: `### %s` does not carry %q — the criterion obliges the decision to record its reasoning, not only its verdict",
					d.AC, d.Heading, req))
			}
		}
	}

	// AC-1 and AC-5 additionally require their consequence to be stated,
	// not just the verdict: which misbehaviors the chosen seam reaches,
	// and what the chosen field scope does to the bless workflow. A
	// decision recorded without its cost is the shape the AC bodies
	// call out as "the cost was not weighed".
	// ParseBodySections keys by slug, not by display heading — the same
	// slugification `aiwf show --format=json` exposes as body keys.
	consequences, ok := entity.ParseBodySections([]byte(doc))[entity.SectionSlug("Consequences")]
	switch {
	case !ok || strings.TrimSpace(consequences) == "":
		report("AC-1/AC-5: `## Consequences` is missing or empty; the seam's reach and the field scope's effect on the bless workflow are both recorded there")
	default:
		if !containsAny(consequences, []string{"empty-diff", "false"}) {
			report("AC-1: `## Consequences` does not state which of the two measured misbehaviors the chosen seam reaches (the false no-change claim, the empty-diff commit)")
		}
		if !containsAny(consequences, []string{"bless"}) {
			report("AC-5: `## Consequences` does not address what the chosen field scope does to the bless workflow")
		}
	}

	return vs, nil
}

// containsAny reports whether hay contains any of the needles,
// case-insensitively.
func containsAny(hay string, needles []string) bool {
	low := strings.ToLower(hay)
	for _, n := range needles {
		if strings.Contains(low, strings.ToLower(n)) {
			return true
		}
	}
	return false
}
