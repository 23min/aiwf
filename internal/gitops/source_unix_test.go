//go:build !windows

package gitops

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestCancelCloneGroup_CompletedProcess(t *testing.T) {
	t.Parallel()
	cmd := exec.CommandContext(t.Context(), "git", "--version")
	cancelCloneGroup(cmd)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("got %v", err)
	}
}
