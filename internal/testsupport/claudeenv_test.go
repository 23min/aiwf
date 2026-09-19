package testsupport

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Serial: the TestMain helper temporarily changes process PATH.
func TestRunWithClaudeOnPATH_RestoresEnvironmentAndRemovesCommand(t *testing.T) {
	original := t.TempDir()
	t.Setenv("PATH", original)
	var command string
	code := RunWithClaudeOnPATH(func() int {
		var err error
		command, err = exec.LookPath("claude")
		if err != nil {
			t.Fatal(err)
		}
		return 73
	})
	if code != 73 {
		t.Fatalf("callback exit code = %d", code)
	}
	if got := os.Getenv("PATH"); got != original {
		t.Fatalf("PATH = %q, want %q", got, original)
	}
	if _, err := os.Stat(command); !os.IsNotExist(err) {
		t.Fatalf("temporary command remains: %v", err)
	}
}

func TestRunWithClaudeOnPATH_FailedSetupDoesNotRunTests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", path)
	called := false
	code := RunWithClaudeOnPATH(func() int { called = true; return 0 })
	if code != 1 || called {
		t.Fatalf("failed setup: code=%d, ran=%v", code, called)
	}
}

func TestRunWithClaudeOnPATH_RestoresAbsentPATH(t *testing.T) {
	t.Setenv("PATH", "")
	if err := os.Unsetenv("PATH"); err != nil {
		t.Fatal(err)
	}
	if code := RunWithClaudeOnPATH(func() int { return 0 }); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if value, exists := os.LookupEnv("PATH"); exists {
		t.Fatalf("PATH became set to %q", value)
	}
}
