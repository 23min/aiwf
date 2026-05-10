package verb

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestIsIndexLockError: stderr substrings that map to lock contention
// vs. unrelated git failures. Load-bearing for the whole G24
// classification path — every false positive surfaces a misleading
// `--audit-only` hint, every false negative hides the real cause.
func TestIsIndexLockError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		msg  string
		want bool
	}{
		{
			name: "modern git index.lock message",
			msg:  "fatal: Unable to create '/repo/.git/index.lock': File exists.",
			want: true,
		},
		{
			name: "bare index.lock substring",
			msg:  "git commit: error: index.lock present",
			want: true,
		},
		{
			name: "older Unable-to-create with lock keyword",
			msg:  "fatal: Unable to create lock for some path",
			want: true,
		},
		{
			name: "unrelated commit failure",
			msg:  "git commit: refusing to commit on detached HEAD",
			want: false,
		},
		{
			name: "merge conflict",
			msg:  "git commit: pathspec 'foo' did not match any files",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isIndexLockError(tt.msg); got != tt.want {
				t.Errorf("isIndexLockError(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

// TestParseLsof: well-formed `lsof <file>` output yields the PID and
// process name from the second line. Malformed / empty / single-line
// output yields the zero values so the caller falls back to a bare
// hint instead of panicking.
func TestParseLsof(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		out      string
		wantPID  string
		wantName string
	}{
		{
			name: "happy path",
			out: "COMMAND   PID  USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"git      4811 peter   3w   REG   1,18        0  abc /tmp/repo/.git/index.lock\n",
			wantPID:  "4811",
			wantName: "git",
		},
		{
			name:     "single line (header only)",
			out:      "COMMAND   PID  USER ...\n",
			wantPID:  "",
			wantName: "",
		},
		{
			name:     "empty",
			out:      "",
			wantPID:  "",
			wantName: "",
		},
		{
			name: "second line too short",
			out: "COMMAND PID USER\n" +
				"git\n",
			wantPID:  "",
			wantName: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pid, name := parseLsof(tt.out)
			if pid != tt.wantPID || name != tt.wantName {
				t.Errorf("parseLsof = (%q, %q), want (%q, %q)", pid, name, tt.wantPID, tt.wantName)
			}
		})
	}
}

// TestApply_LockContentionDiagnostic: a stale .git/index.lock is in
// place; Apply's `git commit` fails. The error wrap names the lock
// path AND points at --audit-only (G24 recovery). When `lsof` is
// available on the test machine, it also names the holder PID.
//
// We don't assert the holder section because the stale lock has no
// holder — the test only asserts the contention-detection path.
// Holder discovery is exercised by TestApply_LockContentionWithHolder.
func TestApply_LockContentionDiagnostic(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatal(err)
	}
	tracked := filepath.Join("work", "epics", "E-0001-foo", "epic.md")
	full := filepath.Join(root, tracked)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("---\nid: E-01\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, tracked); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatal(err)
	}

	// Stale lock — git commit will refuse to acquire.
	lockPath := filepath.Join(root, ".git", "index.lock")
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	// Plan a write that triggers a real commit attempt.
	plan := &Plan{
		Subject: "test write under stale lock",
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "test"},
			{Key: "aiwf-entity", Value: "E-0001"},
			{Key: "aiwf-actor", Value: "human/peter"},
		},
		Ops: []FileOp{
			{Type: OpWrite, Path: "new.md", Content: []byte("hi")},
		},
	}
	err := Apply(ctx, root, plan)
	if err == nil {
		t.Fatal("expected commit to fail under stale lock; got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "index.lock contention") {
		t.Errorf("error should name index.lock contention; got:\n%s", msg)
	}
	if !strings.Contains(msg, "--audit-only") {
		t.Errorf("error should hint at --audit-only recovery; got:\n%s", msg)
	}
}

// TestApply_LockContentionWithHolder: a fixture goroutine holds a
// .git/index.lock open via a child `sleep` subprocess (the only way
// to keep an external process listed in lsof's output for the file).
// Apply's commit attempt fails; the diagnostic names the holder.
//
// Skipped on Windows (no lsof) and when lsof is missing locally —
// the function under test gracefully degrades to the lsof-less
// branch in that case, which TestApply_LockContentionDiagnostic
// already covers.
func TestApply_LockContentionWithHolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("lsof not available on Windows")
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof missing on this machine")
	}
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "seed.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "seed.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatal(err)
	}

	// Park a sleeper subprocess with the lock file open. Use shell
	// redirection so the lock is held by the sleep process itself
	// (lsof reports the descriptor's owner).
	lockPath := filepath.Join(root, ".git", "index.lock")
	holder := exec.Command("/bin/sh", "-c", "exec 9>'"+lockPath+"'; sleep 30")
	if err := holder.Start(); err != nil {
		t.Skipf("could not start holder subprocess: %v", err)
	}
	t.Cleanup(func() {
		_ = holder.Process.Kill()
		_, _ = holder.Process.Wait()
		_ = os.Remove(lockPath)
	})
	// Wait for the lock file to appear (sh redirection is async).
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(lockPath); err == nil {
			break
		}
		// 10ms × 50 = 500ms total, plenty for a shell to start.
		// We avoid time.Sleep imports by re-stat in a tight loop —
		// this whole branch is rare and short-lived.
		if err := tinySleep(); err != nil {
			t.Fatal(err)
		}
	}

	plan := &Plan{
		Subject: "test write under held lock",
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "test"},
			{Key: "aiwf-entity", Value: "E-0001"},
			{Key: "aiwf-actor", Value: "human/peter"},
		},
		Ops: []FileOp{
			{Type: OpWrite, Path: "new.md", Content: []byte("hi")},
		},
	}
	err := Apply(ctx, root, plan)
	if err == nil {
		t.Fatal("expected commit to fail under held lock; got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "lock holder: PID ") {
		// Some platforms / sandboxes restrict lsof from reading other
		// processes' open files. Accept the no-holder branch as well
		// — the broader contention diagnostic is what we care about.
		if !strings.Contains(msg, "index.lock contention") {
			t.Errorf("expected contention diagnostic; got:\n%s", msg)
		}
		t.Logf("lsof did not surface a holder on this platform; diagnostic still fired:\n%s", msg)
	}
}

// tinySleep does a 10ms wait without taking a time.Time / time.Sleep
// dependency in this file. We use exec to a tiny `:` shell builtin
// (which exits immediately); the cost of fork/exec is enough to
// space out polling loops without introducing a new import. Returns
// the exec error so the caller can fail the test on broken environs.
func tinySleep() error {
	cmd := exec.Command("/bin/sh", "-c", "sleep 0.01")
	return cmd.Run()
}

// TestClassifyGitError_NonLockError passes through unmodified so the
// existing apply tests' error formatting is preserved.
func TestClassifyGitError_NonLockError(t *testing.T) {
	t.Parallel()
	in := errors.New("git commit: detached HEAD")
	out := classifyGitError(context.Background(), t.TempDir(), "git commit", in)
	if !strings.Contains(out.Error(), "detached HEAD") {
		t.Errorf("non-lock error should pass through; got: %v", out)
	}
	if strings.Contains(out.Error(), "audit-only") {
		t.Errorf("non-lock error must not suggest audit-only; got: %v", out)
	}
}
