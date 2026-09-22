package gitops

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrNotRegular identifies a missing or non-regular Git tree entry.
var ErrNotRegular = errors.New("not a regular Git file")

// CloneDefault clones the source's default branch without checking out files.
// Credentials and transport settings come from the existing Git environment.
func CloneDefault(ctx context.Context, source, destination string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--quiet", "--no-checkout", "--no-local", "--depth=1", "--single-branch", "--", source, destination)
	cmd.Env = gitEnv()
	cancelCloneGroup(cmd)
	// Bound pipe draining even if a user-configured transport detaches itself.
	cmd.WaitDelay = time.Second
	return runCommand(cmd)
}

// ReadRegularFromHEAD reads a regular blob without following symbolic links or
// applying checkout filters. Literal pathspecs prevent filenames acting as globs.
func ReadRegularFromHEAD(ctx context.Context, workdir, name string) ([]byte, error) {
	mode, err := output(ctx, workdir, "--literal-pathspecs", "ls-tree", "--format=%(objectmode)", "HEAD", "--", name)
	if err != nil {
		return nil, err
	}
	switch strings.TrimSpace(mode) {
	case "100644", "100755":
		return ReadFromHEAD(ctx, workdir, name)
	default:
		return nil, fmt.Errorf("%w: %s", ErrNotRegular, name)
	}
}
