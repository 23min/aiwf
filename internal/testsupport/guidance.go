package testsupport

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// GuidanceSource creates a local corpus with one explicitly selectable pack.
func GuidanceSource(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	guidanceGit(tb, root, "init", "-q", "-b", "main")
	files := map[string]string{
		"catalogue.json":             `{"packs":[{"id":"sample/base","description":"Sample project guidance","files":["packs/sample/base/guide.md"],"detect":[]}]}`,
		"packs/sample/base/guide.md": "# Sample guidance\nInitial upstream content.\n",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	CommitGuidanceSource(tb, root)
	return root
}

// CommitGuidanceSource records fixture changes and returns their exact revision.
func CommitGuidanceSource(tb testing.TB, root string) string {
	tb.Helper()
	guidanceGit(tb, root, "add", ".")
	guidanceGit(tb, root, "commit", "-qm", "fixture guidance")
	return guidanceGit(tb, root, "rev-parse", "HEAD")
}

func guidanceGit(tb testing.TB, root string, args ...string) string {
	tb.Helper()
	cmd := exec.CommandContext(tb.Context(), "git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("fixture git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}
