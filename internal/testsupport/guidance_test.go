package testsupport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsolateGuidanceEnvironment(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("CODEX_HOME", "/unrelated-codex")
	t.Setenv("CLAUDE_CONFIG_DIR", "/unrelated-claude")
	bin := IsolateGuidanceEnvironment(t)
	home := os.Getenv("HOME")
	if os.Getenv("PATH") != bin || os.Getenv("CODEX_HOME") != filepath.Join(home, ".codex") || os.Getenv("CLAUDE_CONFIG_DIR") != filepath.Join(home, ".claude") {
		t.Fatal("personal profile environment was not isolated")
	}
	entries, err := os.ReadDir(bin)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unavailable optional commands not skipped: %v, %v", entries, err)
	}
}
