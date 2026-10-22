package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestBinary_StaleBinaryWarnsBeforeEveryVerb pins M-0363/AC-2 through the
// real CLI: in a repository whose module is aiwf's, a binary built from
// another commit than HEAD prints the warning on stderr while the verb's
// stdout and exit status stay as they are; in a consumer repository it
// prints nothing.
func TestBinary_StaleBinaryWarnsBeforeEveryVerb(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	const stamp = "main@v0.8.0-3-gabcdef123"
	bin := testutil.BuildBinary(t, t.TempDir(), "-ldflags=-X github.com/23min/aiwf/internal/version.Stamp="+stamp)

	for _, tc := range []struct {
		name     string
		module   string
		wantWarn bool
	}{
		{"aiwf repository", "github.com/23min/aiwf", true},
		{"consumer repository", "example.com/consumer", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := t.TempDir()
			testutil.MustExec(t, repo, "git", "init", "-q", "-b", "main")
			testutil.MustExec(t, repo, "git", "-c", "user.email=t@example.com", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init")
			if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module "+tc.module+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(bin, "version")
			cmd.Dir = repo
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("aiwf version: %v\n%s", err, stderr.String())
			}
			if got := stdout.String(); got != stamp+"\n" {
				t.Errorf("stdout = %q, want the version alone", got)
			}
			if warned := bytes.Contains(stderr.Bytes(), []byte("aiwf: warning: ")); warned != tc.wantWarn {
				t.Errorf("stderr = %q, want warn=%v", stderr.String(), tc.wantWarn)
			}
		})
	}
}
