package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// check_commit_msg_seam_test.go drives `aiwf check --commit-msg` through the
// Cobra dispatcher, which is how the installed hook reaches it. The in-package
// tests call runCommitMsg directly and so never prove the flag is wired, nor
// that --root reaches the guard that reads the index.
func TestRun_CheckCommitMsgReachesTheGuards(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	write := func(name, body string) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return p
	}

	for _, tc := range []struct {
		name string
		msg  string
		want int
	}{
		{"clean message passes", "chore(x): a subject\n\naiwf-verb: promote\naiwf-entity: M-0001\n", cliutil.ExitOK},
		{"unknown verb is refused", "chore(x): a subject\n\naiwf-verb: frobnicate\naiwf-entity: M-0001\n", cliutil.ExitFindings},
		{"an AC-scoped subject without its trailer is refused", "feat(x): a thing (M-0001/AC-1)\n\naiwf-entity: M-0001\n", cliutil.ExitFindings},
		{"a hidden trailer block is refused", "chore(x): a subject\n\naiwf-verb: promote\naiwf-entity: M-0001\n\nCo-Authored-By: A <a@example.com>\n", cliutil.ExitFindings},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := write(tc.name+".msg", tc.msg)
			if rc := cli.Execute([]string{"check", "--commit-msg", path, "--root", dir}); rc != tc.want {
				t.Errorf("aiwf check --commit-msg: rc = %d, want %d", rc, tc.want)
			}
		})
	}
}
