package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

func TestRunWhoami_FromFlag(t *testing.T) {
	out := string(captureStdout(t, func() {
		if rc := run([]string{"whoami", "--actor", "human/peter"}); rc != cliutil.ExitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	want := "human/peter (source: --actor flag)\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

// TestRunWhoami_LegacyConfigActorIgnored: pre-I2.5 repos still carry
// `actor:` in aiwf.yaml. The field is no longer a resolution source —
// runtime identity comes from --actor or git config user.email only.
// whoami must NOT report `aiwf.yaml` as the source even when the
// legacy field is present and matches; the git-config fallback wins.
func TestRunWhoami_LegacyConfigActorIgnored(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("aiwf_version: 0.1.0\nactor: human/from-config\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Configure git in an isolated HOME so the test doesn't depend on
	// the host's identity.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"),
		[]byte("[user]\n\temail = git-user@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := string(captureStdout(t, func() {
		if rc := run([]string{"whoami", "--root", root}); rc != cliutil.ExitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	// Must resolve from git, not from the legacy aiwf.yaml field.
	if !strings.Contains(out, "human/git-user") {
		t.Errorf("stdout = %q, want git-derived actor (legacy aiwf.yaml.actor must be ignored)", out)
	}
	if strings.Contains(out, "from-config") || strings.Contains(out, "aiwf.yaml") {
		t.Errorf("stdout = %q, must not surface the legacy aiwf.yaml.actor or its source", out)
	}
}

func TestRunWhoami_FromGitConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"),
		[]byte("[user]\n\temail = peter@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	out := string(captureStdout(t, func() {
		if rc := run([]string{"whoami"}); rc != cliutil.ExitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	want := "human/peter (source: git config user.email)\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestRunWhoami_NoActorAvailable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	if rc := run([]string{"whoami"}); rc != cliutil.ExitFindings {
		t.Errorf("rc = %d, want %d", rc, cliutil.ExitFindings)
	}
}

func TestRunWhoami_InvalidActorFlag(t *testing.T) {
	t.Parallel()
	if rc := run([]string{"whoami", "--actor", "no-slash"}); rc != cliutil.ExitFindings {
		t.Errorf("rc = %d, want %d", rc, cliutil.ExitFindings)
	}
}
