package cellcoverage

import (
	"os"
	"testing"
)

// TestMain seeds the four GIT identity vars once at startup so test
// functions can call t.Parallel safely (os.Setenv not t.Setenv, which
// panics under parallel). See CLAUDE.md §"Test discipline".
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
