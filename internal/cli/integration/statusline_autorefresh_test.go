package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/skills"
)

// TestBinary_PlainUpdate_StatuslineUpgradeOnly is the end-to-end proof of
// G-0344's upgrade-only auto-refresh: a plain `aiwf update` (no
// `--statusline`), run by a binary stamped at a concrete release version,
// refreshes an *older* installed statusline up to that version but refuses
// to downgrade a *newer* one — the fleet-safety property for a `~/.claude`
// shared across containers running different aiwf versions.
//
// A stamped subprocess is required because the decision reads
// version.Current(), whose tagged value only exists in an
// ldflags-stamped binary; a `go test` binary reports `(devel)`, which is
// deliberately unorderable and would skip every copy.
func TestBinary_PlainUpdate_StatuslineUpgradeOnly(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	tmp := t.TempDir()
	// A clean semver tag (no pre-release suffix) so version.Compare can
	// order it — the aiwf upgrade flow only compares concrete releases.
	const binaryVersion = "v1.5.0"
	bin := testutil.BuildBinary(t, tmp, "-ldflags=-X github.com/23min/aiwf/internal/version.Stamp="+binaryVersion)

	repo := t.TempDir()
	home := t.TempDir()
	testutil.MustExec(t, repo, "git", "init", "-q")
	testutil.MustExec(t, repo, "git", "config", "user.email", "test@example.com")
	testutil.MustExec(t, repo, "git", "config", "user.name", "aiwf-test")

	// Quickstart scaffold (hermetic HOME so nothing touches the real
	// ~/.claude). --skip-hook: the test drives no commits that need one.
	runAt(t, repo, home, bin, "init", "--actor", "human/test", "--skip-hook")

	// An OLDER project-scope copy and a NEWER user-scope copy, both
	// aiwf-marked, placed after init (init does not scaffold a statusline
	// without --statusline).
	projPath := filepath.Join(repo, ".claude", "statusline.sh")
	userPath := filepath.Join(home, ".claude", "statusline.sh")
	writeFile(t, projPath, skills.RenderStatusline("v1.0.0"))
	userNewer := skills.RenderStatusline("v2.0.0")
	writeFile(t, userPath, userNewer)

	// Plain update — no --statusline.
	runAt(t, repo, home, bin, "update", "--root", repo)

	// Project copy (older) upgraded to the binary's version.
	if got := readFile(t, projPath); !bytes.Equal(got, skills.RenderStatusline(binaryVersion)) {
		t.Errorf("project-scope older statusline must be upgraded to %s by a plain update", binaryVersion)
	}
	// User copy (newer) left exactly as-is — never downgraded.
	if got := readFile(t, userPath); !bytes.Equal(got, userNewer) {
		t.Errorf("user-scope newer statusline must NOT be downgraded by a plain update")
	}
}

// runAt runs bin with args in workdir, with HOME overridden to home so
// user-scope resolution is hermetic. Fails the test on a non-zero exit.
func runAt(t *testing.T, workdir, home, bin string, args ...string) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "HOME="+home)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", filepath.Base(bin), args, err, out)
	}
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
