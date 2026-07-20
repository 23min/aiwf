package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestShow_FailsLoudOnUnreadableHistory pins M-0269/AC-2 (G-0427):
// `aiwf show` propagates a history-read failure instead of silently
// leaving the History field empty and exiting 0. Mirrors
// TestRenderSinglePass_FailsLoudInProcess's fixture (same fault: HEAD
// still resolves via `git rev-parse --verify`, but the loose object
// backing it is gone, so `git log` — the primitive both render and
// show's history read share — fails mid-walk).
//
// Table-driven over a plain id and a composite `M-NNN/AC-N` id: show.go
// carries this error-propagation branch twice (BuildShowView and
// BuildCompositeShowView are separate function bodies, the latter
// reached only via entity.IsCompositeID's delegation), so both need
// their own exercise per the branch-coverage hard rule. Serial —
// CaptureRun swaps the process stdout/stderr fds.
func TestShow_FailsLoudOnUnreadableHistory(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   string
	}{
		{name: "plain id", id: "E-0001"},
		{name: "composite AC id", id: "M-0001/AC-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			if err := osExec(t, repo, "git", "init", "-q", "-b", "main"); err != nil {
				t.Fatalf("git init: %v", err)
			}
			for _, kv := range [][]string{{"user.email", "test@example.com"}, {"user.name", "test"}, {"commit.gpgsign", "false"}, {"gc.auto", "0"}} {
				if err := osExec(t, repo, "git", "config", kv[0], kv[1]); err != nil {
					t.Fatalf("git config %s: %v", kv[0], err)
				}
			}
			if rc := cli.Execute([]string{"init", "--root", repo, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
				t.Fatalf("init: %d", rc)
			}
			if rc := cli.Execute([]string{"add", "epic", "--root", repo, "--actor", "human/test", "--title", "Foundations"}); rc != cliutil.ExitOK {
				t.Fatalf("add epic: %d", rc)
			}
			if rc := cli.Execute([]string{"add", "milestone", "--root", repo, "--actor", "human/test", "--epic", "E-0001", "--tdd", "none", "--title", "Bootstrap"}); rc != cliutil.ExitOK {
				t.Fatalf("add milestone: %d", rc)
			}
			if rc := cli.Execute([]string{"add", "ac", "M-0001", "--root", repo, "--actor", "human/test", "--title", "first behavior"}); rc != cliutil.ExitOK {
				t.Fatalf("add ac: %d", rc)
			}

			// HEAD's own commit object gone: `git rev-parse --verify HEAD`
			// (cliutil.HasCommits' gate) still resolves the ref to a SHA
			// without touching the object store, but `git log` — needed to
			// actually walk history — fails with "bad object HEAD".
			rootOut, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").CombinedOutput()
			if err != nil {
				t.Fatalf("git rev-parse HEAD: %v\n%s", err, rootOut)
			}
			sha := strings.TrimSpace(string(rootOut))
			if rmErr := os.Remove(filepath.Join(repo, ".git", "objects", sha[:2], sha[2:])); rmErr != nil {
				t.Fatalf("removing HEAD object: %v", rmErr)
			}

			rc, _, stderr := testutil.CaptureRun(t, func() int {
				return cli.Execute([]string{"show", "--root", repo, tc.id})
			})
			if rc != cliutil.ExitInternal {
				t.Fatalf("show %s on corrupt history: rc = %d, want ExitInternal (%d)\nstderr:\n%s", tc.id, rc, cliutil.ExitInternal, stderr)
			}
			if !strings.Contains(stderr, "reading history") {
				t.Errorf("show %s on corrupt history: stderr missing 'reading history':\n%s", tc.id, stderr)
			}
		})
	}
}
