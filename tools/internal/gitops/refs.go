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
