package gitops

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

func TestProjectFiles_ReportsIgnoredListFailure(t *testing.T) {
	// Serial: changes PATH at the subprocess boundary.
	bin := t.TempDir()
	if err := testsupport.WriteExecutable(filepath.Join(bin, "git"), []byte(`#!/bin/sh
case " $* " in
 *" --ignored "*) echo "cannot read ignored entries" >&2; exit 42 ;;
 *) printf 'file.xyz\000' ;;
esac
`)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := ProjectFiles(t.Context(), t.TempDir())
	var exitErr *exec.ExitError
	if got != nil || !errors.As(err, &exitErr) || exitErr.ExitCode() != 42 {
		t.Fatalf("partial result or lost error: %v, %v", got, err)
	}
}
