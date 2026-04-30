// Package verb implements aiwf's mutating verbs: add, promote, cancel,
// rename, reallocate.
//
// Every verb is *validate-then-write* per docs/poc-design-decisions.md:
// the verb computes the projected new tree in memory, runs the
// check.Run validators against the projection, and returns either
// findings (no disk writes occurred) or a Plan (file ops + commit
// metadata). The orchestrator in cmd/aiwf applies the plan only when
// findings are clean. There is no rollback path because nothing is
// written until the projection is known good.
package verb

import (
	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
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
}

// Plan describes the work the orchestrator must do after validation
// passes: a set of file operations to apply on disk, plus the commit
// subject, optional body, and trailers to record once they're staged.
//
// Body is free-form prose: typically the human-supplied --reason for a
// status transition. Empty when the verb has no narrative to record.
// Stored in the commit body (between subject and trailers), surfaced
// by `aiwf history` for events that carry one.
type Plan struct {
	Subject  string
	Body     string
	Trailers []gitops.Trailer
	Ops      []FileOp
}

// OpType discriminates between file operations.
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
