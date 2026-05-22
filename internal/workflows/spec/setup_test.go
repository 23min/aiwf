package spec

import (
	"os"
	"testing"
)

// TestMain seeds the four GIT identity vars once at startup so test
// functions can call t.Parallel safely (os.Setenv not t.Setenv, which
// panics under parallel). See CLAUDE.md §"Test discipline".
//
// The spec package's tests are pure data-structure unit tests — no
// git invocations — but the project-wide policy
// (PolicyTestSetupPresence) requires every test-bearing internal/*
// package to ship setup_test.go with TestMain, so the chokepoint
// holds uniformly across the module.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
