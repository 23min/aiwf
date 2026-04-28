package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWhoami_FromFlag(t *testing.T) {
	out := string(captureStdout(t, func() {
		if rc := runWhoami([]string{"--actor", "human/peter"}); rc != exitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	want := "human/peter (source: --actor flag)\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestRunWhoami_FromConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("aiwf_version: 0.1.0\nactor: human/from-config\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := string(captureStdout(t, func() {
		if rc := runWhoami([]string{"--root", root}); rc != exitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	if !strings.Contains(out, "human/from-config") || !strings.Contains(out, "aiwf.yaml") {
		t.Errorf("stdout = %q, want actor + aiwf.yaml source", out)
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
		if rc := runWhoami(nil); rc != exitOK {
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

	if rc := runWhoami(nil); rc != exitFindings {
		t.Errorf("rc = %d, want %d", rc, exitFindings)
	}
}

func TestRunWhoami_InvalidActorFlag(t *testing.T) {
	if rc := runWhoami([]string{"--actor", "no-slash"}); rc != exitFindings {
		t.Errorf("rc = %d, want %d", rc, exitFindings)
	}
}
