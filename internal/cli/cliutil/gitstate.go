package cliutil

import (
	"context"
	"os/exec"
)

// HasCommits reports whether root's HEAD points at a real commit.
// `git log` on an empty repo errors with "your current branch X does
// not have any commits yet"; this guard converts that into "no events".
func HasCommits(ctx context.Context, root string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = root
	return cmd.Run() == nil
}
