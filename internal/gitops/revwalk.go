package gitops

import (
	"context"
)

// CommitRecord is one commit observed by [BulkRevwalk]: the commit's
// SHA, its parent SHAs in git's declared order (first-parent first),
// the paths it touched (with rename info when -M detected one), and
// the aiwf-* trailers parsed from the commit message.
//
// For multi-parent (merge) commits, Paths is the union of paths that
// differ from at least one parent — the consumer does per-parent
// comparison via blob reads (see [cat-file --batch] pump in
// internal/gitops/catfile.go once AC-2 lands).
//
// Trailers is keyed by the bare trailer name (no "aiwf-" prefix
// stripping). Multi-value trailers collapse to the last value, matching
// internal/cli/history's existing single-value-per-key shape; consumers
// needing multi-value semantics use the [Trailers] slice form via
// [HeadTrailers] / [ParseTrailers] instead.
type CommitRecord struct {
	Commit   string
	Parents  []string
	Paths    []PathTouch
	Trailers map[string]string
}

// PathTouch is one path touched by a commit. Status is the git
// --name-status code: "A" added, "M" modified, "D" deleted, "R"
// renamed (SrcPath set to the pre-rename path), "C" copied (SrcPath
// set to the source path). The "T" (type change) code is rare in
// the aiwf planning tree (no symlinks, no submodules) and collapses
// to "M" via the parser's prefix match.
type PathTouch struct {
	Status  string
	Path    string
	SrcPath string
}

// BulkRevwalk streams [CommitRecord] values from a single
// `git log --all --name-status -M --pretty=...` subprocess, calling fn
// for each commit in walk order. The single-subprocess shape replaces
// the per-entity `git log --follow` fan-out used by callers that walk
// every entity (fsm-history-consistent, status worktree views, show
// scope views) — collapsing ~3,000 fork/execs on the kernel tree into
// one long-lived process.
//
// If fn returns a non-nil error, BulkRevwalk halts the walk and
// returns that error verbatim. Use this to short-circuit when the
// consumer has found what it needs.
//
// Returns nil (no error, no callbacks) when root is empty, is not a
// git repo, or is a repo with no commits — the same "nothing to walk"
// semantic as [internal/cli/history.readHistory] uses.
//
// The walk includes all reachable refs (--all) so feature-branch
// history is observed; the -M flag enables rename detection so a
// rename commit emits PathTouch{Status: "R", SrcPath: <old>, Path:
// <new>} rather than separate D + A entries. -m (per-parent diff
// fan-out) is set so merge commits' name-status output is non-empty;
// see CommitRecord.Paths above for how merges are encoded.
func BulkRevwalk(ctx context.Context, root string, fn func(CommitRecord) error) error {
	// Stub for AC-1 red phase. Implementation lands in green.
	_ = ctx
	_ = root
	_ = fn
	return nil
}
