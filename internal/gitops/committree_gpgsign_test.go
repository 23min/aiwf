package gitops

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// gpgSignFixture is built once per test binary: a throwaway GNUPGHOME
// holding a single ephemeral, passphrase-less signing key, plus a
// wrapper script that points a repo's gpg.program at that GNUPGHOME.
// The wrapper avoids mutating the process-wide GNUPGHOME env var
// (t.Setenv panics under t.Parallel, and a real GNUPGHOME must not
// leak across unrelated parallel tests). Read-only after creation —
// every gpgsign test points its own repo's user.signingkey/gpg.program
// at this fixture and shares it, never regenerating the key.
var (
	gpgSignFixtureOnce sync.Once
	gpgSignHome        string
	gpgSignProgram     string
	gpgSignFingerprint string
	gpgSignFixtureErr  error
)

func gpgSignFixture(t *testing.T) (program, fingerprint string) { // do not mutate
	t.Helper()
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg not installed")
	}
	gpgSignFixtureOnce.Do(func() {
		gpgSignFixtureErr = buildGPGSignFixture()
	})
	if gpgSignFixtureErr != nil {
		t.Fatalf("building gpg test fixture: %v", gpgSignFixtureErr)
	}
	return gpgSignProgram, gpgSignFingerprint
}

func buildGPGSignFixture() error {
	home, err := os.MkdirTemp("", "aiwf-gpgsign-home-*")
	if err != nil {
		return fmt.Errorf("creating GNUPGHOME: %w", err)
	}
	// Recorded before the directory is populated so cleanupGPGSignFixture
	// reaches it even when a later step fails partway through: generating
	// the key starts the agent, so a failure after that point still leaves
	// a daemon to shut down.
	gpgSignHome = home
	if chmodErr := os.Chmod(home, 0o700); chmodErr != nil {
		return fmt.Errorf("chmod GNUPGHOME: %w", chmodErr)
	}

	if genErr := generateGPGKey(home); genErr != nil {
		return genErr
	}

	listCmd := exec.Command("gpg", "--list-secret-keys", "--with-colons")
	listCmd.Env = append(os.Environ(), "GNUPGHOME="+home)
	out, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("listing test signing key: %w", err)
	}
	fpr := parseGPGFingerprint(string(out))
	if fpr == "" {
		return fmt.Errorf("no fingerprint found in gpg --list-secret-keys output:\n%s", out)
	}

	// gpg.program points git at this wrapper instead of the real `gpg`
	// binary, so GNUPGHOME travels with the repo config rather than
	// the process environment.
	wrapperPath := filepath.Join(home, "gpg-wrapper.sh")
	wrapper := "#!/bin/sh\nexport GNUPGHOME=" + home + "\nexec gpg \"$@\"\n"
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o700); err != nil {
		return fmt.Errorf("writing gpg wrapper script: %w", err)
	}

	gpgSignProgram = wrapperPath
	gpgSignFingerprint = fpr
	return nil
}

// generateGPGKey creates a single ephemeral, passphrase-less signing key in
// home. It also starts a gpg-agent daemon rooted there, which outlives the
// process that triggered it — see killGPGAgent.
func generateGPGKey(home string) error {
	cmd := exec.Command("gpg", "--batch", "--pinentry-mode", "loopback",
		"--passphrase", "", "--quick-generate-key",
		"aiwf-test <aiwf-test@example.com>", "default", "default", "never")
	cmd.Env = append(os.Environ(), "GNUPGHOME="+home)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generating test signing key: %w\n%s", err, out)
	}
	return nil
}

// errEmptyGPGHome refuses the one input gpgconf reads as meaningful rather
// than absent.
var errEmptyGPGHome = errors.New("empty GNUPGHOME")

// killGPGAgent shuts down the gpg-agent daemon rooted at home. The daemon
// keeps running after the process that started it exits, so something has to
// end it explicitly.
//
// Removing home ends it too, within a few milliseconds where GnuPG watches
// the socket with inotify and within a timer tick elsewhere. That is a GnuPG
// implementation detail rather than a contract, though, and leaning on it
// would make the teardown "delete the directory and hope", so the daemon is
// shut down explicitly. Call this before removing home: gpgconf reaches the
// agent through a socket that, in the common configuration, lives there.
//
// gpgconf is synchronous with respect to the sockets, not the process — they
// are unlinked before it returns, while the daemon itself is usually still
// exiting for another millisecond or two.
func killGPGAgent(home string) error {
	// gpgconf resolves an empty --homedir to the invoking user's real
	// GnuPG home, so passing one through would shut down the developer's
	// own agent — inside a devcontainer, often a socket forwarded from the
	// host. Nothing upstream needs that call to happen, so refuse it here,
	// at the seam that runs the destructive command.
	if home == "" {
		return errEmptyGPGHome
	}
	out, err := exec.Command("gpgconf", "--homedir", home, "--kill", "all").CombinedOutput()
	if err != nil {
		return fmt.Errorf("killing gpg-agent in %s: %w\n%s", home, err, out)
	}
	return nil
}

// teardownGPGHome shuts down the agent rooted at home and removes the
// directory.
//
// Best-effort: a failure leaks exactly what this exists to prevent, but
// failing the run over it would replace a leaked daemon with a red suite
// reporting the wrong thing. The zero value needs no special case — an empty
// home means the fixture was never built, killGPGAgent refuses it, and
// os.RemoveAll("") is a documented no-op.
func teardownGPGHome(home string) {
	_ = killGPGAgent(home)
	_ = os.RemoveAll(home)
}

// cleanupGPGSignFixture tears down the shared fixture. TestMain calls it once
// m.Run returns: os.Exit runs no deferred calls, and the fixture outlives
// every individual test, so a t.Cleanup would fire while later tests still
// needed it.
//
// Reading gpgSignHome unsynchronized is safe here: m.Run has returned, so
// every test goroutine that could have written it through the sync.Once has
// finished.
func cleanupGPGSignFixture() {
	teardownGPGHome(gpgSignHome)
}

// gpgAgentSocket returns the path the agent rooted at home listens on. The
// socket is not necessarily inside home: where a per-user runtime directory
// exists, GnuPG puts its sockets under that instead, so the location is asked
// for rather than assumed.
//
// The two forms coincide on a host without such a directory, so tests that
// depend on this distinction only exercise it where one exists — a container
// without logind will not notice the difference either way.
func gpgAgentSocket(t *testing.T, home string) string {
	t.Helper()
	out, err := exec.Command("gpgconf", "--homedir", home, "--list-dirs", "agent-socket").Output()
	if err != nil {
		t.Fatalf("locating the agent socket for %s: %v", home, err)
	}
	return strings.TrimSpace(string(out))
}

// gpgAgentAnswering reports whether an agent is listening on socket. A dial is
// the liveness signal rather than the socket file's presence, because an agent
// that cannot unlink its socket on the way out leaves the file behind.
func gpgAgentAnswering(socket string) bool {
	// Bounded rather than a plain Dial: connecting to a unix socket whose
	// backlog is full blocks instead of being refused.
	conn, err := net.DialTimeout("unix", socket, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// emptyGPGWrapper writes a gpg.program wrapper pointing at a fresh,
// key-less GNUPGHOME under t's temp dir — deterministically reproducing
// "signing requested, no usable key" regardless of what the ambient
// environment's real keyring happens to hold.
func emptyGPGWrapper(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg not installed")
	}
	home := t.TempDir()
	if err := os.Chmod(home, 0o700); err != nil {
		t.Fatalf("chmod GNUPGHOME: %v", err)
	}
	wrapperPath := filepath.Join(home, "gpg-wrapper.sh")
	wrapper := "#!/bin/sh\nexport GNUPGHOME=" + home + "\nexec gpg \"$@\"\n"
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o700); err != nil {
		t.Fatalf("writing gpg wrapper script: %v", err)
	}
	return wrapperPath
}

func parseGPGFingerprint(colonOutput string) string {
	for line := range strings.SplitSeq(colonOutput, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) > 9 && fields[0] == "fpr" {
			return fields[9]
		}
	}
	return ""
}

// TestCommitTree_SignsCommitWhenGPGSignEnabled pins M-0186/AC-4: `git
// commit-tree` does not consult commit.gpgsign the way `git commit`
// does, so CommitTree must replicate that behavior explicitly. With
// commit.gpgsign=true and a signing key configured, the resulting
// commit must carry a signature that `git verify-commit` accepts.
func TestCommitTree_SignsCommitWhenGPGSignEnabled(t *testing.T) {
	t.Parallel()
	program, fingerprint := gpgSignFixture(t)
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, kv := range [][2]string{
		{"user.signingkey", fingerprint},
		{"gpg.program", program},
		{"commit.gpgsign", "true"},
	} {
		if err := run(ctx, root, "config", kv[0], kv[1]); err != nil {
			t.Fatalf("config %s: %v", kv[0], err)
		}
	}

	sha, err := CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("hi\n")}}, "signed commit", "", nil)
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}

	if err := run(ctx, root, "verify-commit", sha); err != nil {
		t.Fatalf("expected commit %s to carry a valid signature, verify-commit failed: %v", sha, err)
	}
}

// TestCommitTree_NoSignatureWhenGPGSignNotEnabled pins the other half
// of AC-4: without commit.gpgsign=true, CommitTree must not sign —
// covers both the config key being entirely unset and explicitly set
// to false.
func TestCommitTree_NoSignatureWhenGPGSignNotEnabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		gpgsign string // empty means leave commit.gpgsign unset
	}{
		{name: "unset", gpgsign: ""},
		{name: "explicitly false", gpgsign: "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			root := t.TempDir()

			if err := Init(ctx, root); err != nil {
				t.Fatalf("init: %v", err)
			}
			if tt.gpgsign != "" {
				if err := run(ctx, root, "config", "commit.gpgsign", tt.gpgsign); err != nil {
					t.Fatalf("config commit.gpgsign: %v", err)
				}
			}

			sha, err := CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("hi\n")}}, "unsigned commit", "", nil)
			if err != nil {
				t.Fatalf("CommitTree: %v", err)
			}

			if err := run(ctx, root, "verify-commit", sha); err == nil {
				t.Fatalf("expected commit %s to carry no signature, but verify-commit succeeded", sha)
			}
		})
	}
}

// TestCommitTree_ErrorsWhenSigningKeyUnavailable pins a real, ordinary
// misconfiguration: commit.gpgsign=true with no usable signing key
// (key never configured, revoked, or the agent unreachable). `git
// commit-tree -S` fails in exactly this shape — this is not database
// corruption, it is a reachable input-driven branch, so CommitTree
// must surface the failure rather than silently committing unsigned or
// leaving a partial commit (HEAD must not move).
func TestCommitTree_ErrorsWhenSigningKeyUnavailable(t *testing.T) {
	t.Parallel()
	program := emptyGPGWrapper(t)
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "base.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "base.md"); err != nil {
		t.Fatalf("add base.md: %v", err)
	}
	if err := Commit(ctx, root, "initial commit", "", nil); err != nil {
		t.Fatalf("initial commit: %v", err)
	}
	for _, kv := range [][2]string{
		{"gpg.program", program},
		{"commit.gpgsign", "true"},
	} {
		if err := run(ctx, root, "config", kv[0], kv[1]); err != nil {
			t.Fatalf("config %s: %v", kv[0], err)
		}
	}
	headBefore, err := output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}

	_, err = CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("hi\n")}}, "should not land", "", nil)
	if err == nil {
		t.Fatal("expected CommitTree to fail when no signing key is available, got nil error")
	}

	headAfter, err := output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	if headAfter != headBefore {
		t.Fatalf("HEAD must not advance on signing failure: before %q, after %q", headBefore, headAfter)
	}
}

// TestCommitTree_MalformedGPGSignConfigIsAnError pins the other
// gpgSignEnabled failure shape: a config value git itself cannot parse
// as a boolean (e.g. commit.gpgsign = banana) is a plain user
// misconfiguration, not corruption — `git commit` itself hard-errors
// on it (`fatal: bad boolean config value`), so CommitTree must too,
// rather than silently defaulting to unsigned.
func TestCommitTree_MalformedGPGSignConfigIsAnError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := run(ctx, root, "config", "commit.gpgsign", "banana"); err != nil {
		t.Fatalf("config commit.gpgsign: %v", err)
	}

	_, err := CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("hi\n")}}, "should not land", "", nil)
	if err == nil {
		t.Fatal("expected CommitTree to fail on a malformed commit.gpgsign value, got nil error")
	}
}

// shortTempDir returns a scratch directory whose path is deliberately short.
// gpg-agent's Unix sockets live inside its GNUPGHOME, and sun_path caps a
// socket path at roughly 108 bytes; a t.TempDir() path embeds the test's own
// name and can exhaust that budget, at which point the agent fails to start
// and every gpg operation reports "No agent running".
func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gpgt-*")
	if err != nil {
		t.Fatalf("creating scratch dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// gpgHomeReportPrefix marks the line the child test prints its GNUPGHOME on.
const gpgHomeReportPrefix = "GNUPGHOME="

// TestGPGSignFixture_ReportsItsGNUPGHOME builds the fixture and prints where
// it put its GNUPGHOME. It exists to be the test selected in the child binary
// that TestGPGSignFixture_RemovesGNUPGHOMEAtBinaryExit runs: the parent cannot
// observe the teardown from inside a test, and unless the child names the
// directory it created, "nothing survived" is also what a parent looking in
// the wrong place would see.
func TestGPGSignFixture_ReportsItsGNUPGHOME(t *testing.T) {
	t.Parallel()
	gpgSignFixture(t)
	t.Logf("%s%s", gpgHomeReportPrefix, gpgSignHome)
}

// TestGPGSignFixture_RemovesGNUPGHOMEAtBinaryExit pins the fixture teardown:
// the throwaway GNUPGHOME must not outlive the test binary that created it.
// Seeing that requires a second binary, because the teardown runs after m.Run
// returns and so is invisible from inside any test — this re-executes the
// current binary with TMPDIR redirected into a scratch dir, then asserts
// against the exact path the child reports building.
func TestGPGSignFixture_RemovesGNUPGHOMEAtBinaryExit(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg not installed")
	}

	tmp := shortTempDir(t)
	var reported string
	// Registered after the scratch dir's own cleanup, so it runs first: on
	// failure this reaps the agent the child leaked, rather than letting
	// the directory removal strand it.
	t.Cleanup(func() {
		if reported != "" {
			_ = killGPGAgent(reported)
			return
		}
		// A child that died before naming its GNUPGHOME leaves nothing
		// to reap by name. Matching the fixture's own pattern covers
		// that path; it is only ever a reaper, never an assertion,
		// which is why the test below asserts on the reported path.
		homes, _ := filepath.Glob(filepath.Join(tmp, "aiwf-gpgsign-home-*"))
		for _, home := range homes {
			_ = killGPGAgent(home)
		}
	})

	// Without a deadline the child inherits none — go test's -test.timeout
	// applies to this process only — so a wedged gpg would be killed along
	// with the parent and orphan the very daemon this test is about.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	const childTest = "TestGPGSignFixture_ReportsItsGNUPGHOME"
	child := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run", "^"+childTest+"$")
	child.Env = append(os.Environ(), "TMPDIR="+tmp)
	out, err := child.CombinedOutput()
	if err != nil {
		t.Fatalf("child test binary failed: %v\n%s", err, out)
	}
	// A -test.run that selects nothing also exits 0, which would leave
	// every assertion below passing over a child that built no fixture.
	// Requiring the selected test to report PASS keeps this test honest if
	// childTest is ever renamed out from under it.
	if want := "--- PASS: " + childTest; !strings.Contains(string(out), want) {
		t.Fatalf("child did not run %s (looked for %q):\n%s", childTest, want, out)
	}

	reported = reportedGPGHome(t, string(out))
	// The child must have built inside the directory this test watches.
	// Without this, a TMPDIR redirection that silently stopped taking
	// effect would leave the assertion below passing over an empty dir.
	if got := filepath.Dir(reported); got != tmp {
		t.Fatalf("child built its GNUPGHOME at %s, outside the watched dir %s", reported, tmp)
	}
	if _, err := os.Stat(reported); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("GNUPGHOME %s survived the test binary, stat returned: %v", reported, err)
	}
}

// reportedGPGHome extracts the GNUPGHOME the child test logged.
func reportedGPGHome(t *testing.T, out string) string {
	t.Helper()
	for line := range strings.SplitSeq(out, "\n") {
		if _, path, found := strings.Cut(line, gpgHomeReportPrefix); found {
			if path = strings.TrimSpace(path); path != "" {
				return path
			}
		}
	}
	t.Fatalf("child never reported its GNUPGHOME (no %q line):\n%s", gpgHomeReportPrefix, out)
	return ""
}

// TestTeardownGPGHome_EndsTheAgentWithoutRelyingOnDirectoryRemoval pins the
// explicit shutdown as its own guarantee. Removing the GNUPGHOME would end the
// agent by itself within milliseconds, so the two paths are indistinguishable
// under normal conditions — this makes the directory unwritable first, so
// os.RemoveAll can unlink nothing and the agent has no reason to quit on its
// own. What is left is exactly the property the explicit kill contributes.
func TestTeardownGPGHome_EndsTheAgentWithoutRelyingOnDirectoryRemoval(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg not installed")
	}

	home := shortTempDir(t)
	// Runs before the scratch dir's removal, and restores the write bit the
	// assertion below strips, so neither a failure nor the fault injection
	// can strand a daemon or an undeletable directory.
	t.Cleanup(func() {
		_ = killGPGAgent(home)
		_ = os.Chmod(home, 0o700)
	})
	if err := os.Chmod(home, 0o700); err != nil {
		t.Fatalf("chmod GNUPGHOME: %v", err)
	}
	if err := generateGPGKey(home); err != nil {
		t.Fatalf("generating key: %v", err)
	}

	socket := gpgAgentSocket(t, home)
	if !gpgAgentAnswering(socket) {
		t.Fatalf("expected key generation to leave an agent answering on %s", socket)
	}

	if err := os.Chmod(home, 0o500); err != nil {
		t.Fatalf("making GNUPGHOME unwritable: %v", err)
	}
	teardownGPGHome(home)

	// gpgconf unlinks the sockets before returning but the daemon is
	// usually still exiting for a few milliseconds more, so this waits
	// rather than sampling once. A teardown leaning on the directory
	// removal never converges at all — its agent keeps home and socket
	// alike and runs until the deadline.
	deadline := time.Now().Add(10 * time.Second)
	for gpgAgentAnswering(socket) {
		if time.Now().After(deadline) {
			t.Fatalf("agent rooted at %s still answering on %s after teardown", home, socket)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestKillGPGAgent_RefusesAnEmptyHome pins the guard standing between this
// teardown and the developer's own GnuPG home: gpgconf reads an empty
// --homedir as "use the default", so an unguarded call would shut down the
// real agent — inside a devcontainer, frequently a socket forwarded from the
// host.
func TestKillGPGAgent_RefusesAnEmptyHome(t *testing.T) {
	t.Parallel()
	if err := killGPGAgent(""); !errors.Is(err, errEmptyGPGHome) {
		t.Fatalf("killGPGAgent(\"\") must refuse rather than target the default home, got: %v", err)
	}
}
