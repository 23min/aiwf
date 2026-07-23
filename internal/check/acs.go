package check

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

// Finding codes emitted by this file. Typed per G-0129 so the
// compiler closes on rename / retire across emit sites and tests.
const (
	CodeACsShape                        = "acs-shape"
	CodeACsTitleProse                   = "acs-title-prose"
	CodeACsTDDAudit                     = "acs-tdd-audit"
	CodeMilestoneDoneIncompleteACs      = "milestone-done-incomplete-acs"
	CodeMilestoneDraftIncompleteACs     = "milestone-draft-incomplete-acs"
	CodeMilestoneCancelledIncompleteACs = "milestone-cancelled-incomplete-acs"
	CodeACsBodyCoherence                = "acs-body-coherence"
	CodeMilestoneDoneZeroACs            = "milestone-done-zero-acs"
	CodeACsEmptyBodyOnStart             = "acs-empty-body"
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
//   - tdd_phase (subcode: tdd-phase) — a present phase value not in
//     the closed set. Absence is always legal (G-0286) — "met requires
//     tdd_phase: done" is a separate concern enforced by acsTDDAudit.
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
		// M-0086: archive scoping per ADR-0004 §"Check shape rules".
		// acs-shape is in the shape-and-health group; archived
		// milestones' AC structure is out of scope for active linting.
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		// Validate the milestone's own tdd: policy when present.
		if e.TDD != "" && !entity.IsAllowedTDDPolicy(e.TDD) {
			findings = append(findings, Finding{
				Code:     CodeACsShape,
				Severity: SeverityError,
				Subcode:  "tdd-policy",
				Message: fmt.Sprintf("milestone tdd: %q is not allowed (allowed: %s)",
					e.TDD, strings.Join(entity.AllowedTDDPolicies(), ", ")),
				Path:     e.Path,
				EntityID: e.ID,
				Field:    "tdd",
			})
		}

		for i, ac := range e.ACs {
			compositeID := e.ID + "/" + ac.ID
			expectedID := fmt.Sprintf("AC-%d", i+1)

			// id: missing, malformed, or wrong position.
			switch {
			case ac.ID == "":
				findings = append(findings, Finding{
					Code:     CodeACsShape,
					Severity: SeverityError,
					Subcode:  "id",
					Message:  fmt.Sprintf("acs[%d] missing required field: id (expected %s)", i, expectedID),
					Path:     e.Path,
					EntityID: e.ID,
					Field:    "acs",
				})
			case !acIDPattern.MatchString(ac.ID):
				findings = append(findings, Finding{
					Code:     CodeACsShape,
					Severity: SeverityError,
					Subcode:  "id",
					Message:  fmt.Sprintf("acs[%d].id %q does not match the AC-N format", i, ac.ID),
					Path:     e.Path,
					EntityID: compositeID,
					Field:    "acs",
				})
			case ac.ID != expectedID:
				findings = append(findings, Finding{
					Code:     CodeACsShape,
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
					Code:     CodeACsShape,
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
					Code:     CodeACsShape,
					Severity: SeverityError,
					Subcode:  "status",
					Message:  fmt.Sprintf("%s missing required field: status", composeForMessage(e.ID, ac.ID, i)),
					Path:     e.Path,
					EntityID: composeIfValid(e.ID, ac.ID),
					Field:    "acs",
				})
			case !entity.IsAllowedACStatus(ac.Status):
				findings = append(findings, Finding{
					Code:     CodeACsShape,
					Severity: SeverityError,
					Subcode:  "status",
					Message: fmt.Sprintf("%s status %q is not allowed (allowed: %s)",
						composeForMessage(e.ID, ac.ID, i), ac.Status, strings.Join(entity.AllowedACStatuses(), ", ")),
					Path:     e.Path,
					EntityID: composeIfValid(e.ID, ac.ID),
					Field:    "acs",
				})
			}

			// tdd_phase: absent is always legal (an AC not yet
			// started, or one whose milestone never opted into TDD
			// tracking, has no honest phase to carry); when present,
			// it must be in the closed phase set. "met requires
			// tdd_phase: done" is a distinct concern enforced by
			// acsTDDAudit below, not here (G-0286).
			if ac.TDDPhase != "" && !entity.IsAllowedTDDPhase(ac.TDDPhase) {
				findings = append(findings, Finding{
					Code:     CodeACsShape,
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
				Code:     CodeACsTitleProse,
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
		// M-0086: archive scoping per ADR-0004 §"Check shape rules".
		// acs-tdd-audit is in the shape-and-health group; archived
		// milestones' TDD audit is out of scope for active linting.
		if entity.IsArchivedPath(e.Path) {
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
				Code:     CodeACsTDDAudit,
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
		// M-0086: archive scoping per ADR-0004 §"Check shape rules".
		// milestone-done-incomplete-acs is in the shape-and-health
		// group; archived done milestones whose ACs aren't all met
		// represent historical state, not active drift.
		if entity.IsArchivedPath(e.Path) {
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
			Code:     CodeMilestoneDoneIncompleteACs,
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

// milestoneDraftIncompleteACs fires (warning) when a non-archived draft
// milestone carries an incomplete AC contract, in two shapes: subcode zero-acs
// when acs[] is empty (M-0275/AC-1), and subcode empty-body when acs[] is
// populated but any AC's `### AC-N` body subsection carries no non-heading prose
// (M-0275/AC-2). draft is a legitimate mid-planning state, so both surface the
// missing-contract gap without blocking — the complement, at the draft rung, to
// the draft->in_progress contract guard (acsEmptyBodyOnStart / M-0268) that
// blocks one FSM stage later. Warning, never error, per D-0047 point 2 /
// G-0440. Archive-scoped per ADR-0004 §"Check shape rules": an archived draft
// milestone represents historical state, not active drift.
func milestoneDraftIncompleteACs(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if e.Status != entity.StatusDraft {
			continue
		}
		if len(e.ACs) == 0 {
			findings = append(findings, Finding{
				Code:     CodeMilestoneDraftIncompleteACs,
				Severity: SeverityWarning,
				Subcode:  "zero-acs",
				Message: fmt.Sprintf("draft milestone %s has zero acceptance criteria; add them at plan time (aiwf add ac) so the contract is visible before the milestone lands on main",
					e.ID),
				Path:     e.Path,
				EntityID: e.ID,
				Field:    "acs",
			})
			continue
		}
		// The milestone has ACs, but any AC whose `### AC-N` body subsection
		// carries no non-heading prose is an incomplete contract too. Surface
		// it one FSM stage earlier than acsEmptyBodyOnStart (which fires error
		// at in_progress/done), as a warning, so plan-time review catches the
		// empty body before the milestone lands on main. Same body-emptiness
		// mechanism and the same missing-heading / cancelled-AC carve-outs as
		// that rule (M-0275/AC-2).
		fullPath := filepath.Join(t.Root, e.Path)
		raw, err := os.ReadFile(fullPath)
		if err != nil {
			//coverage:ignore defensive: e.Path comes from the loaded tree, so the file is present; the loader's own load-error finding already covers a vanished file
			continue
		}
		_, body, ok := entity.Split(raw)
		if !ok {
			//coverage:ignore defensive: a file that round-tripped through the loader already has valid frontmatter delimiters
			continue
		}
		sections := entity.ParseACSections(body)
		for _, ac := range e.ACs {
			if ac.ID == "" || ac.Status == entity.StatusCancelled {
				continue
			}
			content, found := sections[ac.ID]
			if !found {
				continue
			}
			if !entity.ACSectionIsEmpty(content) {
				continue
			}
			compositeID := e.ID + "/" + ac.ID
			findings = append(findings, Finding{
				Code:     CodeMilestoneDraftIncompleteACs,
				Severity: SeverityWarning,
				Subcode:  "empty-body",
				Message: fmt.Sprintf("draft milestone %s has no body content under its `### %s` heading; fill the acceptance criterion at plan time (aiwf edit-body) so the contract is visible before the milestone lands on main",
					compositeID, ac.ID),
				Path:     e.Path,
				EntityID: compositeID,
				Field:    "acs",
			})
		}
	}
	return findings
}

// milestoneDoneZeroACs fires (warning) when a non-archived milestone
// has status: done and an empty acs[] (M-0268/AC-3, D-0039 point 2).
// Check-time only — there is no verb-time refusal at this transition;
// a milestone is allowed to reach `done` permanently AC-less, but the
// warning keeps the state visible rather than silent. Extends
// milestoneDoneIncompleteACs's own `status: done` scope with the
// complementary "zero ACs at all" case that rule's open-AC loop can
// never reach (a nil acs[] has no open entries to report).
func milestoneDoneZeroACs(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		// M-0086: archive scoping per ADR-0004 §"Check shape rules".
		// milestone-done-zero-acs is in the shape-and-health group;
		// archived done milestones with zero ACs represent historical
		// state, not active drift.
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if e.Status != entity.StatusDone {
			continue
		}
		if len(e.ACs) > 0 {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeMilestoneDoneZeroACs,
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("milestone %s is done with zero acceptance criteria", e.ID),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "acs",
		})
	}
	return findings
}

// milestoneCancelledIncompleteACs fires when a milestone has status:
// cancelled and at least one AC has status: open. Met, deferred, and
// cancelled are acceptable terminal AC states alongside each other for
// a cancelled milestone — only `open` blocks it, mirroring
// milestoneDoneIncompleteACs's own precondition for `done`.
//
// This runs on every aiwf check pass, not just on verb projection.
// `aiwf promote <milestone> cancelled` and `aiwf cancel <milestone>`
// both already refuse the transition while an AC is open (G-0335:
// MilestonePromoteNonTerminalACsError / MilestoneCancelNonTerminalACs-
// Error), so under normal use this finding never fires — it exists as
// the defense-in-depth backstop for state that bypassed the verb layer
// entirely (a hand-edit, a pre-fix binary), the same backstop role
// milestoneDoneIncompleteACs already plays for `done`.
func milestoneCancelledIncompleteACs(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		// M-0086: archive scoping per ADR-0004 §"Check shape rules".
		// milestone-cancelled-incomplete-acs is in the shape-and-health
		// group; archived cancelled milestones whose ACs aren't all
		// terminal represent historical state, not active drift.
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if e.Status != entity.StatusCancelled {
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
			Code:     CodeMilestoneCancelledIncompleteACs,
			Severity: SeverityError,
			Message: fmt.Sprintf("milestone %s is cancelled but %d AC(s) still open: %s",
				e.ID, len(openIDs), strings.Join(openIDs, ", ")),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "status",
		})
	}
	return findings
}

// acsEmptyBodyOnStart fires (error) when a non-archived milestone is
// in_progress or done and any non-cancelled AC's body subsection
// carries no non-heading prose (M-0268/AC-4, G-0216). Archive-scoped,
// forward-only per D-0039 point 3 — an archived milestone never fires
// this finding regardless of body state, matching every sibling rule
// in this file; there is no separate grandfather or timestamp
// mechanism.
//
// Deliberately does NOT use entityBodyEmpty's terminal-status
// lifecycle gate: that gate silences the AC subcode of
// entity-body-empty once a milestone reaches a terminal status, but
// this rule's own scope is exactly in_progress and done — the two
// statuses where a milestone AC is supposed to have a real contract.
//
// An AC with no `### AC-N` heading in the body at all is a different
// problem (a frontmatter/body desync) — acs-body-coherence/missing-
// heading's concern, not this one, matching the verb-time guard's own
// carve-out (M-0268/AC-2).
func acsEmptyBodyOnStart(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		// M-0086: archive scoping per ADR-0004 §"Check shape rules".
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if e.Status != entity.StatusInProgress && e.Status != entity.StatusDone {
			continue
		}
		fullPath := filepath.Join(t.Root, e.Path)
		raw, err := os.ReadFile(fullPath)
		if err != nil {
			//coverage:ignore defensive: e.Path comes from the loaded tree, so the file is present; the loader's own load-error finding already covers a vanished file
			continue
		}
		_, body, ok := entity.Split(raw)
		if !ok {
			//coverage:ignore defensive: a file that round-tripped through the loader already has valid frontmatter delimiters
			continue
		}
		sections := entity.ParseACSections(body)
		for _, ac := range e.ACs {
			if ac.ID == "" || ac.Status == entity.StatusCancelled {
				continue
			}
			content, found := sections[ac.ID]
			if !found {
				continue
			}
			if !entity.ACSectionIsEmpty(content) {
				continue
			}
			compositeID := e.ID + "/" + ac.ID
			findings = append(findings, Finding{
				Code:     CodeACsEmptyBodyOnStart,
				Severity: SeverityError,
				Message:  fmt.Sprintf("%s has no body content under its `### %s` heading", compositeID, ac.ID),
				Path:     e.Path,
				EntityID: compositeID,
				Field:    "acs",
			})
		}
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
// Three subcodes:
//   - missing-heading:   frontmatter has an AC the body has no heading for.
//   - orphan-heading:    body has a heading the frontmatter has no AC for.
//   - duplicate-heading: the `## Acceptance criteria` section repeats a
//     `### AC-N` heading. A duplicate of an id that is also in
//     frontmatter is neither missing nor orphan, so without this subcode
//     the set-collapse hid it entirely (G-0247). Scoped to the AC
//     section so the `## Work log` convention (which repeats
//     `### AC-N — <outcome>` headings) is not a false positive.
func acsBodyCoherence(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		// M-0086: archive scoping per ADR-0004 §"Check shape rules".
		// acs-body-coherence is in the shape-and-health group;
		// archived milestone body/frontmatter coherence is out of
		// scope for active linting.
		if entity.IsArchivedPath(e.Path) {
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
			if bodyIDs[ac.ID] == 0 {
				findings = append(findings, Finding{
					Code:     CodeACsBodyCoherence,
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
					Code:     CodeACsBodyCoherence,
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

		// A repeated `### AC-N` heading inside the `## Acceptance
		// criteria` section is neither missing nor orphan (the id is in
		// both body and frontmatter), so the present/absent checks above
		// let it pass — flag the count explicitly (G-0247). Scope to the
		// AC section: the `## Work log` convention legitimately repeats
		// `### AC-N — <outcome>` headings, so a whole-body count would
		// false-positive on every wrapped milestone.
		acSection := ""
		if secs := entity.ParseBodySections(body); secs != nil {
			acSection = secs["acceptance_criteria"]
		}
		for id, count := range scanACHeadings([]byte(acSection)) {
			if count <= 1 {
				continue
			}
			findings = append(findings, Finding{
				Code:     CodeACsBodyCoherence,
				Severity: SeverityWarning,
				Subcode:  "duplicate-heading",
				Message: fmt.Sprintf("the `## Acceptance criteria` section has %d `### %s` headings; keep exactly one",
					count, id),
				Path:     e.Path,
				EntityID: e.ID + "/" + id,
				Field:    "acs",
			})
		}
	}
	return findings
}

// acHeadingPattern matches `### AC-<N>` lines in milestone bodies.
// The separator (after the id) is permissive: em-dash, hyphen, colon,
// or absent. Title text (group 2) is captured for future use by
// aiwf show; the coherence check itself only consults the id.
var acHeadingPattern = regexp.MustCompile(`^### AC-(\d+)(?:\s*[—\-:]\s*(.+))?$`)

// scanACHeadings walks body bytes line by line and returns, per AC id,
// the number of `### AC-N` heading lines that carry it. The count (not
// a bare set) is what lets acsBodyCoherence flag a duplicated heading:
// two `### AC-2` lines for an id that is also in frontmatter are neither
// missing nor orphan, so a set would silently collapse them and the
// duplicate would pass clean (G-0247). An id absent from the body has
// no key (callers read a zero count as "no heading").
func scanACHeadings(body []byte) map[string]int {
	out := map[string]int{}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		m := acHeadingPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out["AC-"+m[1]]++
	}
	return out
}
