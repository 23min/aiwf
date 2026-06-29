package cliutil

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
)

// FetchTrunkBestEffort refreshes the configured trunk ref's
// remote-tracking branch via a single `git fetch <remote> <branch>`,
// so the allocator's `max` is computed against the freshest published
// trunk (M-0213's opt-in `aiwf add --fetch`).
//
// It returns a descriptive error when the fetch could not run or
// failed; the CALLER owns the best-effort policy (warn and continue,
// never block the add). Keeping the swallow-or-propagate decision at
// the caller — not buried here — lets the dispatcher emit a single,
// uniform operator warning while this function stays honest about what
// happened.
//
// The configured trunk ref must be a remote-tracking ref
// (refs/remotes/<remote>/<branch>); a local ref (refs/heads/...) or a
// malformed value has nothing to fetch and returns an error naming the
// reason.
func FetchTrunkBestEffort(ctx context.Context, rootDir string) error {
	cfg, err := config.Load(rootDir)
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return fmt.Errorf("loading aiwf.yaml: %w", err)
	}
	ref, _ := cfg.AllocateTrunkRef() // cfg may be nil → default ref
	remote, branch, ok := parseRemoteTrackingRef(ref)
	if !ok {
		return fmt.Errorf("trunk ref %q is not a remote-tracking ref (refs/remotes/<remote>/<branch>); nothing to fetch", ref)
	}
	if err := gitops.FetchBranch(ctx, rootDir, remote, branch); err != nil {
		return fmt.Errorf("fetching %s/%s: %w", remote, branch, err)
	}
	return nil
}

// parseRemoteTrackingRef splits a remote-tracking ref
// "refs/remotes/<remote>/<branch...>" into its remote and branch parts.
// The branch may itself contain slashes (refs/remotes/origin/feature/x
// → "origin", "feature/x"). Returns ok=false for any other shape: a
// local ref, a bare remote with no branch segment, a trailing slash, or
// a non-ref string.
func parseRemoteTrackingRef(ref string) (remote, branch string, ok bool) {
	const prefix = "refs/remotes/"
	if !strings.HasPrefix(ref, prefix) {
		return "", "", false
	}
	rest := ref[len(prefix):]
	i := strings.IndexByte(rest, '/')
	if i <= 0 || i == len(rest)-1 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}
