package testsupport

import (
	"log/slog"
	"os"
	"path/filepath"
)

// RunWithClaudeOnPATH supplies the command-presence prerequisite for legacy
// Claude setup fixtures without requiring an installed assistant. Call from
// TestMain. Host-selection tests must set their own controlled PATH instead.
func RunWithClaudeOnPATH(run func() int) int {
	dir, err := os.MkdirTemp("", "aiwf-test-claude-")
	if err != nil {
		slog.Error("test_host_setup_failed", "operation", "create_directory", "error", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if err := WriteExecutable(filepath.Join(dir, "claude"), []byte("#!/bin/sh\nexit 99\n")); err != nil { //coverage:ignore fresh private temp directory is writable; failure requires environmental disk exhaustion or concurrent external interference
		slog.Error("test_host_setup_failed", "operation", "write_command", "error", err)
		return 1
	}
	previous, hadPath := os.LookupEnv("PATH")
	// The constant key and filesystem-derived value cannot contain NUL bytes.
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+previous)
	defer func() {
		if hadPath {
			_ = os.Setenv("PATH", previous)
		} else {
			_ = os.Unsetenv("PATH")
		}
	}()
	return run()
}
