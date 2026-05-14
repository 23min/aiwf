package render

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the
// test binary, so once-setup is correct.
//
// render itself doesn't invoke git, but the template is uniform
// across internal/* per M-0091.
//
// Serial tests: none. Every Test* function is pure rendering over
// in-memory findings into a bytes.Buffer.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
