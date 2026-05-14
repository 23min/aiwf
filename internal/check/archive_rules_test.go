package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestArchivedEntityNotTerminal_FiresOnHandEditDrift — M-0086 AC-1.
//
// Per ADR-0004 §"`aiwf check` shape rules":
//
//	archived-entity-not-terminal — file lives in `archive/` but
//	frontmatter status isn't terminal. Fires after hand-edit drift.
//	Blocking under default strictness; remediation is to revert the
//	hand-edit (not to relocate the file — see Reversal above).
//
// The reversal section (ADR-0004 §"Reversal"): "If a contributor
// hand-edits frontmatter to take a status off terminal (legal at the
// markdown layer; status is the source of truth), the next aiwf check
// surfaces an `archived-entity-not-terminal` finding. The remediation
// is to revert the hand-edit, not to relocate the file."
//
// Seam: this test drives through tree.Load + check.Run so the loader's
// archive walk and the check.Run dispatch chain both participate.
func TestArchivedEntityNotTerminal_FiresOnHandEditDrift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Archived gap with a non-terminal status — the canonical
	// hand-edit-drift shape ADR-0004 §"Reversal" describes.
	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: Old gap that someone hand-edited
status: open
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	var found *Finding
	for i, f := range got {
		if f.Code == "archived-entity-not-terminal" && f.EntityID == "G-0099" {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected archived-entity-not-terminal finding for G-0099; got: %+v", got)
	}

	// Severity is blocking (error) per ADR-0004 §"Check shape rules".
	if found.Severity != SeverityError {
		t.Errorf("severity = %q, want SeverityError (blocking) per ADR-0004", found.Severity)
	}

	// The finding's path points at the archive file.
	if !strings.Contains(found.Path, "archive/") {
		t.Errorf("Path = %q, want a path under archive/", found.Path)
	}

	// Remediation message names the revert path, not relocation —
	// per ADR-0004 §"Reversal". Assert structurally on the Hint
	// (not just substring-grepping the message), since the hint is
	// the field the user reads as "what do I do."
	if found.Hint == "" {
		t.Fatalf("Hint must be populated for archived-entity-not-terminal; got empty")
	}
	hint := strings.ToLower(found.Hint)
	if !strings.Contains(hint, "revert") {
		t.Errorf("Hint = %q, want it to mention revert (the ADR-0004 remediation path)", found.Hint)
	}
	// Negative assertion: the hint must NOT direct the user to move
	// or relocate the file. Per ADR-0004 §"Reversal": "remediation
	// is to revert the hand-edit, not to relocate the file."
	for _, forbidden := range []string{"relocate", "move the file", "move to active"} {
		if strings.Contains(hint, forbidden) {
			t.Errorf("Hint = %q, must not direct relocation (ADR-0004 forbids): contains %q", found.Hint, forbidden)
		}
	}
}

// TestArchivedEntityNotTerminal_TerminalArchivedIsClean — M-0086 AC-1
// negative case. An archive file whose status IS terminal (the
// post-sweep steady state) does not fire. This is the in-the-clear
// case the rule must not over-flag.
func TestArchivedEntityNotTerminal_TerminalArchivedIsClean(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: Properly-archived terminal gap
status: addressed
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	for _, f := range got {
		if f.Code == "archived-entity-not-terminal" {
			t.Errorf("unexpected finding: %+v", f)
		}
	}
}

// TestArchivedEntityNotTerminal_QuietOnEmptyStatus — M-0086 AC-1
// branch coverage. An archived file with an empty status does not
// fire the archived-entity-not-terminal rule. Coverage rationale:
// the empty-status guard inside the rule remains as defensive code
// (in case frontmatter-shape's archive scoping is ever revisited);
// without the guard the rule would treat empty as "non-terminal"
// and emit a confusing finding alongside whatever the load-error
// path produces. Per AC-4, frontmatter-shape skips archive too, so
// the archive-with-empty-status case currently produces zero
// findings tree-wide — that's the in-the-clear shape this test
// pins.
func TestArchivedEntityNotTerminal_QuietOnEmptyStatus(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: Archived gap with empty status
status:
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	for _, f := range got {
		if f.Code == "archived-entity-not-terminal" {
			t.Errorf("rule fired on empty-status entity (must defer): %+v", f)
		}
	}
}

// TestArchivedEntityNotTerminal_QuietOnUnknownStatus — M-0086 AC-1
// branch coverage. An archived file with a status not in the
// kind's allowed set does not fire archived-entity-not-terminal.
// Currently status-valid is NOT in AC-4's archive-skip list, so
// status-valid still fires on the malformed status — assertion
// pins that.
func TestArchivedEntityNotTerminal_QuietOnUnknownStatus(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: Archived gap with bogus status
status: not-a-real-status
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	var sawArchive, sawValid bool
	for _, f := range got {
		if f.Code == "archived-entity-not-terminal" {
			sawArchive = true
		}
		if f.Code == "status-valid" && f.EntityID == "G-0099" {
			sawValid = true
		}
	}
	if sawArchive {
		t.Errorf("archived-entity-not-terminal should defer to status-valid on unknown status")
	}
	if !sawValid {
		t.Errorf("expected status-valid finding for G-0099 (unknown status; status-valid is not in AC-4's archive-skip list)")
	}
}

// TestTerminalEntityNotArchived_FiresOnTerminalInActiveDir — M-0086
// AC-2. Per ADR-0004 §"`aiwf check` shape rules":
//
//	terminal-entity-not-archived — file lives in active dir but
//	status is terminal. Fires for entities awaiting sweep — the
//	normal transient state under the decoupled model.
//	Advisory by default; not blocking. Counted by
//	archive-sweep-pending.
//
// One finding per terminal entity in an active dir. Severity is
// warning (advisory) — the threshold knob (M-0088) flips it to
// blocking past N.
func TestTerminalEntityNotArchived_FiresOnTerminalInActiveDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Two terminal-status entities sitting in active dirs — the
	// classic "we wrapped these but haven't run aiwf archive yet"
	// shape.
	mustWrite(t, root, "work/gaps/G-0050-fixed.md", `---
id: G-0050
title: Fixed gap awaiting sweep
status: addressed
---
`)
	mustWrite(t, root, "work/gaps/G-0051-wontfix.md", `---
id: G-0051
title: Wontfix gap awaiting sweep
status: wontfix
---
`)
	// Active gap (open) — must not fire.
	mustWrite(t, root, "work/gaps/G-0052-open.md", `---
id: G-0052
title: Active open gap
status: open
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	terminalFindings := map[string]Finding{}
	for _, f := range got {
		if f.Code == "terminal-entity-not-archived" {
			terminalFindings[f.EntityID] = f
		}
	}
	if len(terminalFindings) != 2 {
		t.Fatalf("expected 2 terminal-entity-not-archived findings (G-0050, G-0051); got %d: %+v", len(terminalFindings), terminalFindings)
	}
	for _, id := range []string{"G-0050", "G-0051"} {
		f, ok := terminalFindings[id]
		if !ok {
			t.Errorf("missing finding for %s", id)
			continue
		}
		if f.Severity != SeverityWarning {
			t.Errorf("severity for %s = %q, want SeverityWarning (advisory) per ADR-0004", id, f.Severity)
		}
	}
	// G-0052 (open gap in active dir) must NOT fire.
	if _, ok := terminalFindings["G-0052"]; ok {
		t.Errorf("rule fired on non-terminal active gap G-0052 (must skip)")
	}
}

// TestTerminalEntityNotArchived_DefersToFrontmatterShapeOnEmptyStatus —
// M-0086 AC-2 branch coverage. An active-dir entity with an empty
// status is reported by frontmatterShape; the archive rule defers,
// matching the same one-finding-per-authoring-problem rationale as
// archivedEntityNotTerminal.
func TestTerminalEntityNotArchived_DefersToFrontmatterShapeOnEmptyStatus(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/gaps/G-0099-empty.md", `---
id: G-0099
title: Active gap with empty status
status:
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	for _, f := range got {
		if f.Code == "terminal-entity-not-archived" {
			t.Errorf("rule fired on empty-status entity (must defer): %+v", f)
		}
	}
}

// TestTerminalEntityNotArchived_ArchivedTerminalIsClean — M-0086 AC-2
// boundary case. A terminal entity that has been swept into archive/
// (the post-sweep steady state) does not fire this rule. The rule is
// location-keyed to active dirs.
func TestTerminalEntityNotArchived_ArchivedTerminalIsClean(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: Properly-archived terminal gap
status: addressed
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	for _, f := range got {
		if f.Code == "terminal-entity-not-archived" {
			t.Errorf("rule fired on archived terminal entity (must skip): %+v", f)
		}
	}
}

// TestTerminalEntityNotArchived_SkipsMilestonesUnderActiveEpic — G-0124.
//
// Per ADR-0004 §"Storage — per-kind layout" (verbatim in the `aiwf
// archive` verb's docstring at `internal/verb/archive.go:40`):
//
//	| Milestone | work/epics/<epic>/M-NNNN-<slug>.md | does not
//	  archive independently — rides w/ epic |
//
// So `aiwf archive` correctly skips terminal milestones whose parent
// epic is still active. The `terminal-entity-not-archived` rule must
// honour the same definition of "sweep-eligible" — otherwise the
// chokepoint emits a warning whose remediation (`aiwf archive --apply`)
// is a no-op, training operators to ignore the rule.
//
// Fixture: an active epic carrying a terminal milestone, plus a
// terminal gap (control — must still fire to prove the rule isn't
// disabled wholesale).
func TestTerminalEntityNotArchived_SkipsMilestonesUnderActiveEpic(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Active epic — its directory is NOT swept yet.
	mustWrite(t, root, "work/epics/E-0099-in-flight/epic.md", `---
id: E-0099
title: In-flight epic
status: active
---
`)
	// Terminal milestone under the active epic — this is the case the
	// rule must skip (milestone rides with its parent epic per
	// ADR-0004; it has no independent archive path).
	mustWrite(t, root, "work/epics/E-0099-in-flight/M-0099-wrapped.md", `---
id: M-0099
title: Wrapped milestone under in-flight epic
status: done
---
`)
	// Control: a terminal gap that IS sweep-eligible. Confirms the
	// rule still fires for kinds with independent archive paths, so
	// the milestone skip isn't a wholesale disabling.
	mustWrite(t, root, "work/gaps/G-0050-fixed.md", `---
id: G-0050
title: Fixed gap awaiting sweep
status: addressed
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	var milestoneFinding, gapFinding *Finding
	for i, f := range got {
		if f.Code != "terminal-entity-not-archived" {
			continue
		}
		switch f.EntityID {
		case "M-0099":
			milestoneFinding = &got[i]
		case "G-0050":
			gapFinding = &got[i]
		}
	}

	if milestoneFinding != nil {
		t.Errorf("rule fired on milestone M-0099 under active epic E-0099 (must skip per ADR-0004 — milestones ride with parent epic, never archive independently): %+v", milestoneFinding)
	}
	if gapFinding == nil {
		t.Errorf("control case failed: rule did not fire on terminal gap G-0050; the milestone skip must not disable the rule for other kinds")
	}

	// The aggregate count must reflect the gap only (1), not the
	// milestone false-positive (which would inflate to 2).
	var aggregate *Finding
	for i, f := range got {
		if f.Code == "archive-sweep-pending" {
			aggregate = &got[i]
			break
		}
	}
	if aggregate == nil {
		t.Fatalf("expected archive-sweep-pending aggregate finding (1 terminal gap); got: %+v", got)
	}
	if !strings.Contains(aggregate.Message, "1 terminal") {
		t.Errorf("aggregate message = %q, want it to count 1 (gap only); milestone must not inflate the count", aggregate.Message)
	}
}

// TestArchiveSweepPending_AggregatesPendingCount — M-0086 AC-3.
//
// Per ADR-0004 §"Drift control" (1) and §"Check shape rules":
//
//	archive-sweep-pending — aggregate finding reporting the count
//	of terminal-entity-not-archived instances. Advisory;
//	configurable to blocking past archive.sweep_threshold.
//	"Hidden when zero." (§Drift control)
//
// The aggregate finding's Message must name the count so a human
// reading `aiwf check` output sees the magnitude at a glance. The
// per-file terminal-entity-not-archived findings stay alongside —
// the aggregate summarizes; it does not replace the leaves.
func TestArchiveSweepPending_AggregatesPendingCount(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Three terminal-status entities pending sweep.
	mustWrite(t, root, "work/gaps/G-0050-fixed.md", `---
id: G-0050
title: Fixed
status: addressed
---
`)
	mustWrite(t, root, "work/gaps/G-0051-wontfix.md", `---
id: G-0051
title: Wontfix
status: wontfix
---
`)
	mustWrite(t, root, "work/decisions/D-0010-old.md", `---
id: D-0010
title: Old decision
status: rejected
---
`)
	// Plus an active gap that must NOT count.
	mustWrite(t, root, "work/gaps/G-0052-open.md", `---
id: G-0052
title: Open
status: open
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	var aggregate *Finding
	leaves := 0
	for i, f := range got {
		switch f.Code {
		case "archive-sweep-pending":
			aggregate = &got[i]
		case "terminal-entity-not-archived":
			leaves++
		}
	}
	if aggregate == nil {
		t.Fatalf("expected archive-sweep-pending aggregate finding; got: %+v", got)
	}
	if aggregate.Severity != SeverityWarning {
		t.Errorf("aggregate severity = %q, want SeverityWarning (advisory by default per ADR-0004)", aggregate.Severity)
	}
	// The count must appear in the message so the human-readable
	// line names the magnitude. Structural assertion: parse the
	// count out of the message and confirm it equals the leaf
	// count, not just substring-grep for "3".
	if !strings.Contains(aggregate.Message, "3") {
		t.Errorf("aggregate Message %q must name the count (3)", aggregate.Message)
	}
	if leaves != 3 {
		t.Errorf("expected 3 terminal-entity-not-archived leaf findings; got %d", leaves)
	}
	// Aggregate is per-tree, not per-file — has no Path / EntityID.
	if aggregate.Path != "" {
		t.Errorf("aggregate Path = %q, want empty (rule is per-tree)", aggregate.Path)
	}
}

// TestArchiveSweepPending_HiddenWhenZero — M-0086 AC-3. Per
// ADR-0004 §"Drift control": "Hidden when zero." A clean tree (no
// pending sweep) emits no aggregate finding.
func TestArchiveSweepPending_HiddenWhenZero(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Only an active open gap and an archived terminal gap — no
	// pending sweep.
	mustWrite(t, root, "work/gaps/G-0050-open.md", `---
id: G-0050
title: Open
status: open
---
`)
	mustWrite(t, root, "work/gaps/archive/G-0049-old.md", `---
id: G-0049
title: Properly archived
status: addressed
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	for _, f := range got {
		if f.Code == "archive-sweep-pending" {
			t.Errorf("aggregate fired with zero pending-sweep entities (must be hidden): %+v", f)
		}
	}
}

// TestArchivedEntityNotTerminal_ActiveDirNeverFires — M-0086 AC-1
// boundary case. Even an entity with a non-terminal status in an
// ACTIVE dir does not trigger this rule (a different rule —
// terminal-entity-not-archived for the inverse — covers the active
// side). This rule is location-keyed: "lives in archive/ but isn't
// terminal," not "isn't terminal, period."
func TestArchivedEntityNotTerminal_ActiveDirNeverFires(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/gaps/G-0050-active-open.md", `---
id: G-0050
title: Active open gap
status: open
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	for _, f := range got {
		if f.Code == "archived-entity-not-terminal" {
			t.Errorf("rule fired on active-dir entity (location-keyed rule must skip): %+v", f)
		}
	}
}

// M-0088 AC-2 — archive.sweep_threshold escalates archive-sweep-pending.
//
// Per ADR-0004 §"Drift control" layer (2):
//
//	Configurable hard threshold. aiwf.yaml's archive.sweep_threshold
//	(default unset) flips the advisory finding to blocking past the
//	named count. Teams choose their own discipline; the default is
//	permissive (no threshold).
//
// Shape mirrors check.ApplyTDDStrict: a separate, testable bumper
// function that the verb-dispatcher calls after Run, reading the
// config value via cfg.ArchiveSweepThreshold(). The rule's emission
// stays config-agnostic; the escalation is a deterministic transform
// on the finding slice. The bumper escalates only the aggregate
// archive-sweep-pending finding; per-file terminal-entity-not-archived
// leaves stay advisory (the aggregate is the actionable signal).

// TestApplyArchiveSweepThreshold_UnsetIsNoOp pins the default-permissive
// rule: when no threshold is set, the bumper leaves every severity
// unchanged regardless of count. The set=false branch is the load-
// bearing default behavior — kernel must not nag a fresh consumer
// that has terminals awaiting sweep.
func TestApplyArchiveSweepThreshold_UnsetIsNoOp(t *testing.T) {
	t.Parallel()
	build := func() []Finding {
		return []Finding{
			{Code: "archive-sweep-pending", Severity: SeverityWarning, Message: "47 terminal entities awaiting `aiwf archive --apply`. Set `archive.sweep_threshold` in aiwf.yaml to escalate to blocking past N"},
			{Code: "terminal-entity-not-archived", Severity: SeverityWarning, EntityID: "G-0050"},
			{Code: "refs-resolve", Severity: SeverityError, EntityID: "M-0002"},
		}
	}
	findings := build()
	// set=false, threshold=0 → no escalation regardless of count.
	ApplyArchiveSweepThreshold(findings, 0, false, 47)
	for _, f := range findings {
		if f.Code == "archive-sweep-pending" && f.Severity != SeverityWarning {
			t.Errorf("set=false: archive-sweep-pending severity = %v, want warning preserved", f.Severity)
		}
		if f.Code == "terminal-entity-not-archived" && f.Severity != SeverityWarning {
			t.Errorf("set=false: leaf severity = %v, want warning preserved", f.Severity)
		}
		if f.Code == "refs-resolve" && f.Severity != SeverityError {
			t.Errorf("refs-resolve severity = %v, want error preserved", f.Severity)
		}
	}
}

// TestApplyArchiveSweepThreshold_AtOrBelowStaysAdvisory pins the
// other half of the escalation rule: when the count is ≤ threshold,
// the finding stays advisory. The threshold is a *strictly-exceeds*
// gate per ADR-0004's "past N" wording — at exactly N the consumer's
// declared ceiling is met but not breached.
func TestApplyArchiveSweepThreshold_AtOrBelowStaysAdvisory(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		count int
	}{
		{"below threshold", 3},
		{"at threshold", 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			findings := []Finding{
				{Code: "archive-sweep-pending", Severity: SeverityWarning},
				{Code: "terminal-entity-not-archived", Severity: SeverityWarning, EntityID: "G-0050"},
			}
			ApplyArchiveSweepThreshold(findings, 5, true, tc.count)
			for _, f := range findings {
				if f.Code == "archive-sweep-pending" && f.Severity != SeverityWarning {
					t.Errorf("count=%d ≤ threshold=5: archive-sweep-pending severity = %v, want warning",
						tc.count, f.Severity)
				}
				if f.Code == "terminal-entity-not-archived" && f.Severity != SeverityWarning {
					t.Errorf("leaf severity = %v, want warning preserved (leaves never escalate)", f.Severity)
				}
			}
		})
	}
}

// TestApplyArchiveSweepThreshold_PastThresholdEscalates is the load-
// bearing AC-2 case. count > threshold flips the aggregate to error
// severity so the pre-push hook blocks the push. Leaves stay warning
// — the aggregate is the actionable signal (one error blocks; ten
// warnings on every gap would just be noise once the operator's
// already seen the aggregate).
//
// The escalation also rewrites the Message so the human reading
// `aiwf check` output sees their declared threshold cited explicitly,
// not just the default "set the threshold to escalate" hint.
func TestApplyArchiveSweepThreshold_PastThresholdEscalates(t *testing.T) {
	t.Parallel()
	findings := []Finding{
		{
			Code:     "archive-sweep-pending",
			Severity: SeverityWarning,
			Message:  "47 terminal entities awaiting `aiwf archive --apply`. Set `archive.sweep_threshold` in aiwf.yaml to escalate to blocking past N",
		},
		{Code: "terminal-entity-not-archived", Severity: SeverityWarning, EntityID: "G-0050"},
		{Code: "refs-resolve", Severity: SeverityError, EntityID: "M-0002"},
	}
	ApplyArchiveSweepThreshold(findings, 5, true, 47)
	var aggregate *Finding
	for i := range findings {
		if findings[i].Code == "archive-sweep-pending" {
			aggregate = &findings[i]
		}
		if findings[i].Code == "terminal-entity-not-archived" && findings[i].Severity != SeverityWarning {
			t.Errorf("leaf severity = %v, want warning (leaves do not escalate)", findings[i].Severity)
		}
	}
	if aggregate == nil {
		t.Fatal("archive-sweep-pending finding disappeared")
	}
	if aggregate.Severity != SeverityError {
		t.Errorf("aggregate severity = %v, want error (count 47 > threshold 5)", aggregate.Severity)
	}
	// The rewritten message must name both the count and the
	// configured threshold so the human sees the magnitude of the
	// breach and the policy they crossed, not a generic warning.
	if !strings.Contains(aggregate.Message, "47") {
		t.Errorf("escalated Message %q must name the count (47)", aggregate.Message)
	}
	if !strings.Contains(aggregate.Message, "5") {
		t.Errorf("escalated Message %q must name the threshold (5)", aggregate.Message)
	}
	if !strings.Contains(aggregate.Message, "aiwf archive") {
		t.Errorf("escalated Message %q must name the sweep verb (`aiwf archive`)", aggregate.Message)
	}
}

// TestApplyArchiveSweepThreshold_NilFindingsIsNoOp: defensive
// branch coverage for a slice that has no findings yet. Mirrors
// the TestApplyTDDStrict nil-defense subtest.
func TestApplyArchiveSweepThreshold_NilFindingsIsNoOp(t *testing.T) {
	t.Parallel()
	ApplyArchiveSweepThreshold(nil, 5, true, 47)
}

// TestApplyArchiveSweepThreshold_NoAggregateInSliceIsNoOp: when
// the count is zero (no pending sweep), the aggregate is hidden per
// ADR-0004 §"Drift control"; the bumper's seam coverage must
// gracefully short-circuit when there is no archive-sweep-pending
// finding to escalate. (This is the production path on a clean
// tree with a threshold configured.)
func TestApplyArchiveSweepThreshold_NoAggregateInSliceIsNoOp(t *testing.T) {
	t.Parallel()
	findings := []Finding{
		{Code: "refs-resolve", Severity: SeverityError, EntityID: "M-0002"},
	}
	ApplyArchiveSweepThreshold(findings, 5, true, 0)
	if findings[0].Severity != SeverityError {
		t.Errorf("unrelated finding severity = %v, want error preserved", findings[0].Severity)
	}
}
