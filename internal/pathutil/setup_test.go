package pathutil

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the
// test binary, so once-setup is correct.
//
// pathutil itself doesn't shell out to git, but the template is
// uniform across internal/* per M-0091 — packages that subprocess
// out via gitops downstream get a sane environment without each
// test caring. Harmless where it isn't needed.
//
// Serial tests: none. Every Test* function in this package is
// pure path/symlink reasoning with no shared state or env mutation.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
