// Package verb implements aiwf's mutating verbs: add, promote, cancel,
// rename, reallocate, and friends.
//
// Most verbs are *validate-then-write* per
// docs/pocv3/design/design-decisions.md: the verb computes the
// projected new tree in memory, runs projectionFindings (which wraps
// check.Run) against the projection, and returns either findings (no
// disk writes occurred) or a Plan (file ops + commit metadata). The
// orchestrator in cmd/aiwf applies the plan only when findings are
// clean. There is no rollback path because nothing is written until
// the projection is known good.
//
// A documented minority of verbs never call projectionFindings, for
// one of four concrete reasons: the field they mutate is validated
// only by a CLI-composed, git-history-dependent rule (area
// membership) that needs data — a touchedByEntity map built by
// scanning commit history — no in-memory projection has, so the rule
// can never fire there regardless of which verb calls it; the commit
// they produce has an empty diff (a sovereign or audit-only act) with
// no entity-content mutation to project; the operation is a purely
// structural multi-entity sweep (archive, rewidth) where mid-sweep
// check noise would be spurious and validation is deferred entirely
// to the pre-push hook's full aiwf check; or the verb belongs to the
// contract subsystem, which writes aiwf.yaml rather than an entity
// file and runs its own narrower validation gate by design. The
// reviewed, exhaustive allowlist — one entry per exempt verb with its
// specific reason — lives in
// internal/policies/projection_findings_presence.go, mechanically
// enforced: any verb newly falling outside both the presence check
// and the allowlist fails CI.
package verb

import (
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/gitops"
)

// Result is what every verb returns. Exactly one of Findings, Plan,
// or the NoOp signal is populated:
//
//   - Findings non-empty   → validation failed; no disk changes pending.
//     Caller renders findings and exits 1.
//   - Plan non-nil         → projection is clean; caller should apply
//     Operations, stage them, and commit with
//     the plan's subject + trailers.
//   - NoOp == true         → validation passed, but the requested change
//     is already in place. Caller prints
//     NoOpMessage on stdout and exits 0. Used by
//     idempotent verbs (bind on exact match, etc.).
type Result struct {
	Findings    []check.Finding
	Plan        *Plan
	NoOp        bool
	NoOpMessage string

	// Metadata carries per-verb-appropriate facts about the mutation
	// (M-0239/AC-2) — e.g. entity_id/from/to for a status transition,
	// swept_count for an archive sweep. Surfaced under the JSON
	// envelope's metadata key, alongside AC-1's correlation_id and
	// (on a successful apply) commit_sha. nil for verbs that report
	// nothing beyond those two.
	Metadata map[string]any
}

// Plan describes the work the orchestrator must do after validation
// passes: a set of file operations to apply on disk, plus the commit
// subject, optional body, and trailers to record once they're staged.
//
// Body is free-form prose: typically the human-supplied --reason for a
// status transition. Empty when the verb has no narrative to record.
// Stored in the commit body (between subject and trailers), surfaced
// by `aiwf history` for events that carry one.
//
// AllowEmpty signals that the plan's commit has no file-level diff and
// must be created via `git commit --allow-empty`. Used by `aiwf
// authorize` (which records a scope event in trailers without touching
// any entity file) and the `--audit-only` recovery mode added in plan
// step 5b. The default (false) is the normal verb behaviour: a commit
// without staged changes errors.
type Plan struct {
	Subject    string
	Body       string
	Trailers   []gitops.Trailer
	Ops        []FileOp
	AllowEmpty bool
}

// OpType discriminates between file operations.
//
// The set is deliberately closed to OpWrite and OpMove — there is no
// OpDelete, by design, not omission. aiwf never deletes an entity
// file: "removal" is a status flip to a terminal value (cancelled /
// wontfix / rejected / retired) followed by an archive sweep that
// OpMoves the file into its per-kind archive/ subdirectory (ADR-0004).
// A verb that needs to "remove" something promotes it to a terminal
// status and lets `aiwf archive` relocate it; nothing in the kernel
// unlinks a tracked entity.
type OpType int

const (
	// OpWrite creates or overwrites a regular file at Path with
	// Content. Parent directories are created as needed.
	OpWrite OpType = iota
	// OpMove relocates Path to NewPath via `git mv`. Used by rename
	// and reallocate. The source must already be tracked by git.
	OpMove
)

// FileOp is a single planned filesystem mutation.
type FileOp struct {
	Type    OpType
	Path    string // source path (relative to repo root)
	NewPath string // destination path (only for OpMove)
	Content []byte // file contents (only for OpWrite)
}

// findings is a tiny constructor used by verbs that fail validation.
func findings(fs []check.Finding) *Result {
	return &Result{Findings: fs}
}

// plan is a tiny constructor used by verbs that pass validation.
func plan(p *Plan) *Result {
	return &Result{Plan: p}
}
