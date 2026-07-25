package cliutil

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
)

// isolateGitIdentity points the git config locator at an empty HOME so
// `git config user.email` finds nothing, making ResolveActor("", root)
// deterministically fail. Callers must NOT run t.Parallel — t.Setenv panics
// under it, and these tests also capture the process-global streams.
func isolateGitIdentity(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func TestResolvePrelude_Success(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	wantRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	gotRoot, gotActor, code, ok := ResolvePrelude("aiwf test-verb", root, "human/peter")
	if !ok {
		t.Fatal("ok = false, want true on a resolvable root + explicit actor")
	}
	if gotRoot != wantRoot {
		t.Errorf("rootDir = %q, want %q", gotRoot, wantRoot)
	}
	if gotActor != "human/peter" {
		t.Errorf("actorStr = %q, want %q", gotActor, "human/peter")
	}
	if code != ExitOK {
		t.Errorf("code = %d, want ExitOK (%d)", code, ExitOK)
	}
}

// TestResolvePrelude_ActorError_EmitsUsageError is serial: it isolates the
// git identity (t.Setenv) and captures the process-global os.Stderr.
func TestResolvePrelude_ActorError_EmitsUsageError(t *testing.T) {
	isolateGitIdentity(t)
	root := t.TempDir()
	// Derive the exact error the shared usage arm should print, in the same
	// isolated environment, so the assertion is byte-exact rather than a
	// prefix probe.
	_, wantErr := ResolveActor("", root)
	if wantErr == nil {
		t.Fatal("precondition: ResolveActor must fail with no derivable identity")
	}
	var gotRoot, gotActor string
	var code int
	var ok bool
	_, stderr := captureStdStreams(t, func() {
		gotRoot, gotActor, code, ok = ResolvePrelude("aiwf test-verb", root, "")
	})
	if want := fmt.Sprintf("aiwf test-verb: %v\n", wantErr); stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
	if ok {
		t.Error("ok = true, want false on actor-resolution failure")
	}
	if code != ExitUsage {
		t.Errorf("code = %d, want ExitUsage (%d)", code, ExitUsage)
	}
	if gotRoot != "" || gotActor != "" {
		t.Errorf("root/actor = %q/%q, want both empty on failure", gotRoot, gotActor)
	}
}

func TestResolvePreludeEnvelope_Success(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	wantRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	gotRoot, gotActor, code, ok := ResolvePreludeEnvelope(
		context.Background(), "aiwf test-verb", root, "human/peter", OutputFormat{Format: "text"})
	if !ok {
		t.Fatal("ok = false, want true on a resolvable root + explicit actor")
	}
	if gotRoot != wantRoot {
		t.Errorf("rootDir = %q, want %q", gotRoot, wantRoot)
	}
	if gotActor != "human/peter" {
		t.Errorf("actorStr = %q, want %q", gotActor, "human/peter")
	}
	if code != ExitOK {
		t.Errorf("code = %d, want ExitOK (%d)", code, ExitOK)
	}
}

// TestResolvePreludeEnvelope_ActorError_EmitsEnvelope is serial: it isolates
// the git identity (t.Setenv) and captures the process-global os.Stdout that
// the JSON error envelope is written to.
func TestResolvePreludeEnvelope_ActorError_EmitsEnvelope(t *testing.T) {
	isolateGitIdentity(t)
	root := t.TempDir()
	if _, wantErr := ResolveActor("", root); wantErr == nil {
		t.Fatal("precondition: ResolveActor must fail with no derivable identity")
	}
	var gotRoot, gotActor string
	var code int
	var ok bool
	stdout, _ := captureStdStreams(t, func() {
		gotRoot, gotActor, code, ok = ResolvePreludeEnvelope(
			context.Background(), "aiwf test-verb", root, "", OutputFormat{Format: "json"})
	})
	// The envelope arm emits a structured error envelope (status "error"),
	// not a plain stderr line — that is what distinguishes it from the text
	// helper and what --format=json consumers rely on.
	var env map[string]any
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout %q did not parse as a JSON envelope: %v", stdout, err)
	}
	if env["status"] != "error" {
		t.Errorf("envelope status = %v, want %q", env["status"], "error")
	}
	if ok {
		t.Error("ok = true, want false on actor-resolution failure")
	}
	if code != ExitUsage {
		t.Errorf("code = %d, want ExitUsage (%d)", code, ExitUsage)
	}
	if gotRoot != "" || gotActor != "" {
		t.Errorf("root/actor = %q/%q, want both empty on failure", gotRoot, gotActor)
	}
}
