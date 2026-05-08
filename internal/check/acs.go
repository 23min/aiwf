package check

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// acsShape validates the structure of every milestone's acs[] list and
// the milestone's own tdd: policy field. Five concerns, each surfaced
// with its own subcode so callers can filter:
//
//   - id (subcode: id)        — id missing, malformed, or wrong position.
//     AC ids are position-stable: acs[i].id must equal "AC-{i+1}",
//     including when earlier ACs are cancelled (status flip, not
//     deletion). The allocator picks max+1 over the full list.
//   - title (subcode: title)  — title missing on an AC.
//   - status (subcode: status)
//   - tdd_phase (subcode: tdd-phase) — phase value not in the closed
//     set, or absent when the parent milestone is tdd: required.
//   - tdd policy (subcode: tdd-policy) — milestone's own tdd: value not
//     in {required, advisory, none}.
//
// Findings on AC-scoped problems use the composite id (M-NNN/AC-N) as
// EntityID so they're filterable by composite id; milestone-scoped
// problems (the tdd policy itself) carry the bare milestone id. Path
// is always the milestone's file path — ACs live inside that file.
func acsShape(t *tree.Tree) []Finding {
	var findings []Finding
	acIDPattern := regexp.MustCompile(`^AC-\d+$`)

	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		// Validate the milestone's own tdd: policy when present.
		if e.TDD != "" && !entity.IsAllowedTDDPolicy(e.TDD) {
			findings = append(findings, Finding{
				Code:     "acs-shape",
				Severity: SeverityError,
				Subcode:  "tdd-policy",
				Message: fmt.Sprintf("milestone tdd: %q is not allowed (allowed: %s)",
					e.TDD, strings.Join(entity.AllowedTDDPolicies(), ", ")),
				Path:     e.Path,
				EntityID: e.ID,
				Field:    "tdd",
			})
		}

		tddRequired := e.TDD == "required"
		for i, ac := range e.ACs {
			compositeID := e.ID + "/" + ac.ID
			expectedID := fmt.Sprintf("AC-%d", i+1)

			// id: missing, malformed, or wrong position.
			switch {
			case ac.ID == "":
				findings = append(findings, Finding{
					Code:     "acs-shape",
					Severity: SeverityError,
					Subcode:  "id",
					Message:  fmt.Sprintf("acs[%d] missing required field: id (expected %s)", i, expectedID),
					Path:     e.Path,
					EntityID: e.ID,
					Field:    "acs",
				})
			case !acIDPattern.MatchString(ac.ID):
				findings = append(findings, Finding{
					Code:     "acs-shape",
					Severity: SeverityError,
					Subcode:  "id",
					Message:  fmt.Sprintf("acs[%d].id %q does not match the AC-N format", i, ac.ID),
					Path:     e.Path,
					EntityID: compositeID,
					Field:    "acs",
				})
			case ac.ID != expectedID:
				findings = append(findings, Finding{
					Code:     "acs-shape",
					Severity: SeverityError,
					Subcode:  "id",
					Message: fmt.Sprintf("acs[%d].id %q is at the wrong position; expected %s (position-based, cancelled entries count toward position)",
						i, ac.ID, expectedID),
					Path:     e.Path,
					EntityID: compositeID,
					Field:    "acs",
				})
			}

			// title: required.
			if strings.TrimSpace(ac.Title) == "" {
				findings = append(findings, Finding{
					Code:     "acs-shape",
					Severity: SeverityError,
					Subcode:  "title",
					Message:  fmt.Sprintf("%s missing required field: title", composeForMessage(e.ID, ac.ID, i)),
					Path:     e.Path,
					EntityID: composeIfValid(e.ID, ac.ID),
					Field:    "acs",
				})
			}

			// status: required and in the closed set.
			switch {
			case ac.Status == "":
				findings = append(findings, Finding{
					Code:     "acs-shape",
					Severity: SeverityError,
					Subcode:  "status",
					Message:  fmt.Sprintf("%s missing required field: status", composeForMessage(e.ID, ac.ID, i)),
					Path:     e.Path,
					EntityID: composeIfValid(e.ID, ac.ID),
					Field:    "acs",
				})
			case !entity.IsAllowedACStatus(ac.Status):
				findings = append(findings, Finding{
					Code:     "acs-shape",
					Severity: SeverityError,
					Subcode:  "status",
					Message: fmt.Sprintf("%s status %q is not allowed (allowed: %s)",
						composeForMessage(e.ID, ac.ID, i), ac.Status, strings.Join(entity.AllowedACStatuses(), ", ")),
					Path:     e.Path,
					EntityID: composeIfValid(e.ID, ac.ID),
					Field:    "acs",
				})
			}

			// tdd_phase: required when milestone is tdd: required;
			// when present, must be in the closed phase set.
			switch {
			case ac.TDDPhase == "" && tddRequired:
				findings = append(findings, Finding{
					Code:     "acs-shape",
					Severity: SeverityError,
					Subcode:  "tdd-phase",
					Message: fmt.Sprintf("%s missing required field: tdd_phase (milestone is tdd: required)",
						composeForMessage(e.ID, ac.ID, i)),
					Path:     e.Path,
					EntityID: composeIfValid(e.ID, ac.ID),
					Field:    "acs",
				})
			case ac.TDDPhase != "" && !entity.IsAllowedTDDPhase(ac.TDDPhase):
				findings = append(findings, Finding{
					Code:     "acs-shape",
					Severity: SeverityError,
					Subcode:  "tdd-phase",
					Message: fmt.Sprintf("%s tdd_phase %q is not allowed (allowed: %s)",
						composeForMessage(e.ID, ac.ID, i), ac.TDDPhase, strings.Join(entity.AllowedTDDPhases(), ", ")),
					Path:     e.Path,
					EntityID: composeIfValid(e.ID, ac.ID),
					Field:    "acs",
				})
			}
		}
	}
	return findings
}

// composeForMessage returns a stable user-facing label for an AC under
// a milestone, suitable for finding messages. When the AC's id is
// empty or malformed, falls back to "acs[i]" so the position is still
// visible.
func composeForMessage(milestoneID, acID string, position int) string {
	if acID != "" {
		return milestoneID + "/" + acID
	}
	return fmt.Sprintf("%s acs[%d]", milestoneID, position)
}

// composeIfValid returns the composite id only when the AC's id is
// non-empty; otherwise returns the bare milestone id. EntityID on a
// finding should be a real id (queryable via aiwf history), not a
// position placeholder.
func composeIfValid(milestoneID, acID string) string {
	if acID != "" {
		return milestoneID + "/" + acID
	}
	return milestoneID
}

// acsTitleProse (warning) fires when an AC's title looks like prose
// rather than a short label. Long, markdown-formatted, or multi-
// sentence titles render as one giant `### AC-N — <title>` heading
// in the milestone body, which is the bug G20 was filed against.
//
// The verb path (`aiwf add ac`) refuses prose-y titles up front; this
// check is the standing-tree counterpart, catching titles that landed
// via hand-edits or pre-G20 tooling.
func acsTitleProse(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		for _, ac := range e.ACs {
			if !entity.IsProseyTitle(ac.Title) {
				continue
			}
			compositeID := e.ID + "/" + ac.ID
			findings = append(findings, Finding{
				Code:     "acs-title-prose",
				Severity: SeverityWarning,
				Message: fmt.Sprintf("%s title looks like prose (long / multi-sentence / contains markdown); shorten the title and move detail prose into the body section under `### %s`",
					compositeID, ac.ID),
				Path:     e.Path,
				EntityID: compositeID,
				Field:    "acs",
			})
		}
	}
	return findings
}

// acsTDDAudit fires when a milestone has tdd: required (error) or
// tdd: advisory (warning) and any AC has status: met without
// tdd_phase: done. The kernel guards the *outcome* (met implies done);
// the rituals plugin's wf-tdd-cycle drives the flow. A human or any
// AI can satisfy the kernel without that skill installed.
//
// Skipped silently when tdd: none (the default) — that policy opts
// out of the audit entirely.
func acsTDDAudit(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		var sev Severity
		switch e.TDD {
		case "required":
			sev = SeverityError
		case "advisory":
			sev = SeverityWarning
		default:
			// "none" or absent: audit doesn't run.
			continue
		}
		for _, ac := range e.ACs {
			if ac.Status != entity.StatusMet {
				continue
			}
			if ac.TDDPhase == entity.TDDPhaseDone {
				continue
			}
			compositeID := e.ID + "/" + ac.ID
			phase := ac.TDDPhase
			if phase == "" {
				phase = "(absent)"
			}
			findings = append(findings, Finding{
				Code:     "acs-tdd-audit",
				Severity: sev,
				Message: fmt.Sprintf("%s status: met under tdd: %s but tdd_phase is %s (expected done)",
					compositeID, e.TDD, phase),
				Path:     e.Path,
				EntityID: compositeID,
				Field:    "acs",
			})
		}
	}
	return findings
}

// milestoneDoneIncompleteACs fires when a milestone has status: done
// and at least one AC has status: open. Cancelled and deferred are
// acceptable terminal AC states for a done milestone — only `open`
// blocks the milestone-done state.
//
// This runs on every aiwf check pass, not just on verb projection,
// so a milestone that became `done` via --force --reason while ACs
// were still open keeps surfacing the inconsistency until the ACs
// reach a terminal state. The companion verb-time guard lives in
// the promote verb.
func milestoneDoneIncompleteACs(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		if e.Status != entity.StatusDone {
			continue
		}
		var openIDs []string
		for _, ac := range e.ACs {
			if ac.Status == entity.StatusOpen {
				openIDs = append(openIDs, ac.ID)
			}
		}
		if len(openIDs) == 0 {
			continue
		}
		findings = append(findings, Finding{
			Code:     "milestone-done-incomplete-acs",
			Severity: SeverityError,
			Message: fmt.Sprintf("milestone %s is done but %d AC(s) still open: %s",
				e.ID, len(openIDs), strings.Join(openIDs, ", ")),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "status",
		})
	}
	return findings
}

// acsBodyCoherence pairs the frontmatter acs[] list with the body's
// `### AC-<N>` headings and warns when the two disagree. Pairing is
// by id only — the body heading's title text is prose and remains
// kernel-blind, consistent with the design's "prose is not parsed"
// principle.
//
// The body parser is a minimal heading walker. It runs a single regex
// over each line: `^### AC-(\d+)(?:\s*[—\-:]\s*(.+))?$` — accepting
// em-dash, hyphen, colon, or id-only forms. The regex is permissive
// on purpose; aiwf add ac scaffolds em-dash by default, but a hand-
// typed hyphen should not produce a noisy warning.
//
// Two subcodes:
//   - missing-heading: frontmatter has an AC the body has no heading for.
//   - orphan-heading:  body has a heading the frontmatter has no AC for.
func acsBodyCoherence(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		// Read the body once per milestone; failures (missing file,
		// I/O error) silently produce zero findings — the load-error
		// path already covers the file-can't-be-read case.
		fullPath := filepath.Join(t.Root, e.Path)
		raw, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		_, body, ok := entity.Split(raw)
		if !ok {
			continue
		}
		bodyIDs := scanACHeadings(body)

		fmIDs := make(map[string]bool, len(e.ACs))
		for _, ac := range e.ACs {
			if ac.ID != "" {
				fmIDs[ac.ID] = true
			}
		}

		// Frontmatter has it, body doesn't.
		for _, ac := range e.ACs {
			if ac.ID == "" {
				continue
			}
			if !bodyIDs[ac.ID] {
				findings = append(findings, Finding{
					Code:     "acs-body-coherence",
					Severity: SeverityWarning,
					Subcode:  "missing-heading",
					Message: fmt.Sprintf("%s/%s has no `### %s` heading in the milestone body",
						e.ID, ac.ID, ac.ID),
					Path:     e.Path,
					EntityID: e.ID + "/" + ac.ID,
					Field:    "acs",
				})
			}
		}

		// Body has a heading, frontmatter doesn't.
		for id := range bodyIDs {
			if !fmIDs[id] {
				findings = append(findings, Finding{
					Code:     "acs-body-coherence",
					Severity: SeverityWarning,
					Subcode:  "orphan-heading",
					Message: fmt.Sprintf("milestone body has `### %s` heading but acs[] has no matching entry",
						id),
					Path:     e.Path,
					EntityID: e.ID,
					Field:    "acs",
				})
			}
		}
	}
	return findings
}

// acHeadingPattern matches `### AC-<N>` lines in milestone bodies.
// The separator (after the id) is permissive: em-dash, hyphen, colon,
// or absent. Title text (group 2) is captured for future use by
// aiwf show; the coherence check itself only consults the id.
var acHeadingPattern = regexp.MustCompile(`^### AC-(\d+)(?:\s*[—\-:]\s*(.+))?$`)

// scanACHeadings walks body bytes line by line and returns the set
// of AC ids that appear as a heading. Duplicate headings collapse
// (the set holds one entry per id); duplicates are not flagged here
// because acsShape's position rule already disallows a duplicate id
// in the frontmatter, and an extra body heading without a frontmatter
// match shows up as an orphan-heading warning.
func scanACHeadings(body []byte) map[string]bool {
	out := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		m := acHeadingPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out["AC-"+m[1]] = true
	}
	return out
}
