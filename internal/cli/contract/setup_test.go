package contract_test

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity once at startup so tests can run with
// t.Parallel() without t.Setenv panics. The identity values are
// immutable for the test binary's lifetime; once-setup is correct.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
