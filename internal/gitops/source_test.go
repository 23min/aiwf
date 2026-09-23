package gitops

import (
	"errors"
	"os/exec"
	"testing"
)

func TestReadRegularFromHEAD_ModesAndLiteralPaths(t *testing.T) {
	t.Parallel()
	root := initTestRepo(t)
	commitFile(t, t.Context(), root, "guide.md", "ordinary")
	commitFile(t, t.Context(), root, "*.md", "literal")
	mustRun(t, t.Context(), root, "update-index", "--chmod=+x", "guide.md")
	mustRun(t, t.Context(), root, "commit", "-qm", "executable markdown")
	for _, tc := range []struct{ name, want string }{{"guide.md", "ordinary"}, {"*.md", "literal"}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ReadRegularFromHEAD(t.Context(), root, tc.name)
			if err != nil || string(got) != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestReadRegularFromHEAD_CommandFailure(t *testing.T) {
	t.Parallel()
	_, err := ReadRegularFromHEAD(t.Context(), t.TempDir(), "guide.md")
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("lost git failure: %v", err)
	}
}
