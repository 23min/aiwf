package main

import (
	"os"
	"testing"
)

// TestMain exists so test functions can call t.Parallel safely (see
// CLAUDE.md §"Test discipline"). This package's tests never shell out
// to git, so there's no identity env or HardenGitTestEnv to seed.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
