package testsupport

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// GuidanceSource creates a local corpus with one pack and optional filename patterns.
func GuidanceSource(tb testing.TB, detect ...string) string {
	tb.Helper()
	if detect == nil {
		detect = []string{}
	}
	patterns, err := json.Marshal(detect)
	if err != nil { //coverage:ignore a slice of strings is always JSON-encodable
		tb.Fatal(err)
	}
	root := tb.TempDir()
	guidanceGit(tb, root, "init", "-q", "-b", "main")
	files := map[string]string{
		"catalogue.json":             fmt.Sprintf(`{"packs":[{"id":"sample/base","description":"Sample project guidance","files":["packs/sample/base/guide.md"],"detect":%s}]}`, patterns),
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

// IsolateGuidanceEnvironment gives serial tests an empty personal-guidance
// environment while retaining the real Git and shell process boundaries.
func IsolateGuidanceEnvironment(tb testing.TB) string {
	tb.Helper()
	home := tb.TempDir()
	bin := filepath.Join(home, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil { //coverage:ignore fresh private TempDir is writable; failure requires environmental filesystem failure
		tb.Fatal(err)
	}
	for _, name := range []string{"git", "sh", "claude"} {
		executable, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if err := os.Symlink(executable, filepath.Join(bin, name)); err != nil { //coverage:ignore fresh private directory has no conflicting entries; failure requires environmental filesystem failure
			tb.Fatal(err)
		}
	}
	tb.Setenv("HOME", home)
	tb.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	tb.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	tb.Setenv("PATH", bin)
	return bin
}
