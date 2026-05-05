package gitops

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrRefNotFound reports that the requested ref does not resolve in
// workdir's git repository. Wrapped by HasRef and LsTreePaths so
// callers can distinguish "ref absent" (potentially a sandbox repo)
// from "git failed for some other reason."
var ErrRefNotFound = errors.New("ref not found")

// HasRemotes reports whether workdir has any configured git remote.
// A repo with no remotes has no possible cross-branch coordination
// surface, so the trunk-aware allocator skips its check there.
func HasRemotes(ctx context.Context, workdir string) (bool, error) {
	out, err := output(ctx, workdir, "remote")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// HasAnyRemoteTrackingRefs reports whether workdir has any
// refs/remotes/* ref recorded locally. Used by the trunk-awareness
// policy to distinguish "remote configured but never populated"
// (e.g., a clone of an empty bare repo, before the first push) from
// "remote configured and the trunk ref just doesn't match what's
// fetched" (a real misconfiguration).
//
// Returns (false, nil) when no tracking refs exist; (true, nil) when
// at least one does. Other git failures propagate as wrapped errors.
func HasAnyRemoteTrackingRefs(ctx context.Context, workdir string) (bool, error) {
	out, err := output(ctx, workdir, "for-each-ref", "--count=1", "--format=%(refname)", "refs/remotes/")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// AddCommitSHA returns the SHA of the commit that introduced
// relPath into the repo. Returns ("", nil) when the file has no add
// commit visible from HEAD (newly staged but never committed).
// Wraps git failures.
//
// `git log --diff-filter=A --pretty=%H -- <path>` is git's "when
// did this exact path first appear" query. We deliberately do NOT
// pass `--follow`: it traces *content* across renames as a
// heuristic, which produces wrong answers in the duplicate-id case
// the reallocate tiebreaker cares about — two entity files of the
// same kind have nearly-identical frontmatter/body shapes, and
// `--follow` will frequently mis-attribute one's add commit to the
// other's. The exact-path query is what we actually want: the
// commit that first put bytes at this exact path.
func AddCommitSHA(ctx context.Context, workdir, relPath string) (string, error) {
	out, err := output(ctx, workdir, "log", "--diff-filter=A", "--pretty=%H", "--", relPath)
	if err != nil {
		return "", fmt.Errorf("finding add commit for %s: %w", relPath, err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// `git log` lists newest first; with --diff-filter=A the *last*
	// line is the original add. That's what callers want when
	// ranking two entities by birth order.
	for i := len(lines) - 1; i >= 0; i-- {
		s := strings.TrimSpace(lines[i])
		if s != "" {
			return s, nil
		}
	}
	return "", nil
}

// IsAncestor reports whether commit is an ancestor of ref (i.e.
// `git merge-base --is-ancestor <commit> <ref>` succeeds). Returns
// (false, nil) when commit is not an ancestor; (true, nil) when it
// is; an error only on real git failures (bad refs, missing repo).
//
// The reallocate tiebreaker uses this to ask "which side already
// exists on trunk?" — the side that does keeps the id; the side
// that doesn't gets renumbered.
func IsAncestor(ctx context.Context, workdir, commit, ref string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", commit, ref)
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Exit 1 = not an ancestor. Exit 128 = bad ref / repo issue.
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
			return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %w", commit, ref, err)
		}
		return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %w", commit, ref, err)
	}
	return true, nil
}

// HasRef reports whether ref resolves to an object in workdir's repo.
// Returns (false, nil) when the ref is absent — distinguishing it
// from any other git failure, which propagates as a wrapped error.
func HasRef(ctx context.Context, workdir, ref string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("git rev-parse --verify %s: %w", ref, err)
	}
	return true, nil
}

// LsTreePaths returns the file paths under ref's tree, optionally
// filtered to those whose slash-normalized path begins with any of the
// supplied prefixes. Pass no prefixes to list every path. Paths are
// repo-relative and slash-separated; ordering is git's (sorted).
//
// Returns ErrRefNotFound (wrapped) when ref does not resolve. Other
// git failures propagate as wrapped errors. An existing but empty
// ref tree returns ([]string{}, nil).
func LsTreePaths(ctx context.Context, workdir, ref string, prefixes ...string) ([]string, error) {
	exists, err := HasRef(ctx, workdir, ref)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrRefNotFound, ref)
	}
	out, err := output(ctx, workdir, "ls-tree", "--full-tree", "-r", "--name-only", "-z", ref)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	parts := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	paths := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		if len(prefixes) == 0 {
			paths = append(paths, p)
			continue
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(p, prefix) {
				paths = append(paths, p)
				break
			}
		}
	}
	return paths, nil
}
