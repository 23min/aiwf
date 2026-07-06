package gitops

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Worktree describes one entry in a repo's worktree set: the working
// directory path, the currently-checked-out branch (empty for detached
// HEAD), and the HEAD commit SHA.
//
// Used by `aiwf status --worktrees` (G-0122) and any future worktree-
// aware verb. Returned in the order `git worktree list --porcelain`
// emits them — main checkout first, linked worktrees in
// administrative-listing order.
type Worktree struct {
	Path    string // absolute path to the worktree's working directory
	Branch  string // branch name without the `refs/heads/` prefix, or "" for detached HEAD
	HeadSHA string // 40-char SHA at HEAD
}

// ListWorktrees enumerates every worktree linked to the repo containing
// workdir. Returns the main checkout plus every linked worktree.
// Parses `git worktree list --porcelain` per the documented format:
// each entry is a sequence of `key value` lines terminated by a blank
// line. Recognized keys: `worktree` (path), `HEAD` (sha), `branch`
// (full ref name).
//
// A worktree with detached HEAD has `detached` in lieu of `branch`;
// Branch comes back empty in that case. Bare repos (no working tree)
// are skipped.
//
// G-0122.
func ListWorktrees(ctx context.Context, workdir string) ([]Worktree, error) {
	out, err := output(ctx, workdir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(out), nil
}

// parseWorktreeList is the pure-text parser for `git worktree list
// --porcelain` output. Split from ListWorktrees so tests can drive
// every output shape (main-only, linked-worktree, detached-HEAD, bare)
// against synthetic fixtures without spinning up a real repo.
func parseWorktreeList(out string) []Worktree {
	var worktrees []Worktree
	var cur Worktree
	flush := func() {
		if cur.Path != "" {
			worktrees = append(worktrees, cur)
		}
		cur = Worktree{}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		key, value, found := strings.Cut(line, " ")
		if !found {
			// Single-token lines (e.g. "bare", "detached", "locked")
			// — recognized for completeness but only "bare" affects
			// flow: a bare repo entry has no working tree to track.
			if line == "bare" {
				cur = Worktree{} // discard; bare repos have no working tree
			}
			continue
		}
		switch key {
		case "worktree":
			cur.Path = value
		case "HEAD":
			cur.HeadSHA = value
		case "branch":
			cur.Branch = strings.TrimPrefix(value, "refs/heads/")
		}
	}
	flush()
	return worktrees
}

// BranchExists reports whether branch exists as a local branch in the
// repo containing workdir. `aiwf worktree add`'s caller uses this to
// pick between reusing an existing branch (WorktreeAdd) and creating
// a fresh one (WorktreeAddNewBranch) — plain `git worktree add <path>
// <branch>` fails on a nonexistent branch, and `-b` fails when the
// branch already exists, so the two forms are not interchangeable.
//
// Mirrors StashTopRef's existence-probe shape: `--verify --quiet`
// exits 1 with empty output when the ref is absent, which maps to
// (false, nil); any other failure is wrapped and returned.
func BranchExists(ctx context.Context, workdir, branch string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("git rev-parse refs/heads/%s: %w", branch, err) //coverage:ignore requires git rev-parse to fail with a non-exit-1 error (git missing or repo corruption mid-call)
	}
	return true, nil
}

// WorktreeAdd creates a linked worktree at path checked out to the
// existing local branch. Any git failure (branch already checked out
// elsewhere, path already exists, etc.) is surfaced verbatim via run's
// combined-output wrap — the caller must not report success on error.
func WorktreeAdd(ctx context.Context, workdir, path, branch string) error {
	return run(ctx, workdir, "worktree", "add", path, branch)
}

// WorktreeAddNewBranch creates a linked worktree at path with a fresh
// local branch, starting from base. An empty base defers to git's own
// default (HEAD).
func WorktreeAddNewBranch(ctx context.Context, workdir, path, branch, base string) error {
	args := []string{"worktree", "add", "-b", branch, path}
	if base != "" {
		args = append(args, base)
	}
	return run(ctx, workdir, args...)
}
