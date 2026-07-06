package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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
	if chmodErr := os.Chmod(home, 0o700); chmodErr != nil {
		return fmt.Errorf("chmod GNUPGHOME: %w", chmodErr)
	}

	genCmd := exec.Command("gpg", "--batch", "--pinentry-mode", "loopback",
		"--passphrase", "", "--quick-generate-key",
		"aiwf-test <aiwf-test@example.com>", "default", "default", "never")
	genCmd.Env = append(os.Environ(), "GNUPGHOME="+home)
	if out, genErr := genCmd.CombinedOutput(); genErr != nil {
		return fmt.Errorf("generating test signing key: %w\n%s", genErr, out)
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
	wrapperPath := filepath.Join(t.TempDir(), "gpg-wrapper.sh")
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
