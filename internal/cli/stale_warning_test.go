package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/version"
)

// TestWarnIfStale pins the root hook's output (M-0363/AC-2): run from
// inside the aiwf repository by a binary whose commit differs from HEAD,
// it prints one warning to stderr; a binary that names no commit prints
// nothing.
//
// Serial (no t.Parallel): changes the working directory and captures
// os.Stderr, both process-globals; see setup_test.go.
func TestWarnIfStale(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo, "-c", "user.email=t@example.com", "-c", "user.name=t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	write("go.mod", "module github.com/23min/aiwf\n")
	git("add", ".")
	git("commit", "-q", "-m", "init")
	old := git("rev-parse", "HEAD")
	write("internal/x.go", "package x\n")
	git("add", ".")
	git("commit", "-q", "-m", "change the source")
	t.Chdir(repo)

	stale := testutil.CaptureStderr(t, func() {
		warnIfStale(t.Context(), version.Parse("main@v0.8.0-3-g"+old[:9]), "github.com/23min/aiwf")
	})
	if !strings.Contains(string(stale), "aiwf: warning: ") {
		t.Errorf("stale binary printed %q, want the stale-binary warning", stale)
	}
	devel := testutil.CaptureStderr(t, func() {
		warnIfStale(t.Context(), version.Parse(version.DevelVersion), "github.com/23min/aiwf")
	})
	if len(devel) != 0 {
		t.Errorf("(devel) binary printed %q, want nothing", devel)
	}
}
