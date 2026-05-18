package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// captureExecuteOutput runs Execute with args, swapping stdout to a
// pipe so the test can assert on the printed output. Returns
// (stdoutBytes, exitCode). os.Stdout is mutated, so this test (and
// the captured-stderr variant below) sit on the serial-skip list per
// the package's setup_test.go comment.
func captureExecuteOutput(t *testing.T, args []string) ([]byte, int) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	done := make(chan []byte)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()

	code := Execute(args)
	_ = w.Close()
	out := <-done
	return out, code
}

// TestExecute_Version asserts that `aiwf --version` prints a single
// non-empty version string and exits 0.
func TestExecute_Version(t *testing.T) {
	out, code := captureExecuteOutput(t, []string{"--version"})
	if code != cliutil.ExitOK {
		t.Errorf("exit code: got %d, want %d", code, cliutil.ExitOK)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		t.Errorf("expected non-empty version output, got empty")
	}
}

// TestExecute_VersionVerb asserts that `aiwf version` (the verb) emits
// the same string as `aiwf --version` — both go through ResolvedVersion.
func TestExecute_VersionVerb(t *testing.T) {
	outFlag, codeFlag := captureExecuteOutput(t, []string{"--version"})
	if codeFlag != cliutil.ExitOK {
		t.Fatalf("--version exit code: got %d, want %d", codeFlag, cliutil.ExitOK)
	}
	outVerb, codeVerb := captureExecuteOutput(t, []string{"version"})
	if codeVerb != cliutil.ExitOK {
		t.Errorf("version verb exit code: got %d, want %d", codeVerb, cliutil.ExitOK)
	}
	if !bytes.Equal(bytes.TrimSpace(outFlag), bytes.TrimSpace(outVerb)) {
		t.Errorf("--version and version verb diverged: %q vs %q",
			bytes.TrimSpace(outFlag), bytes.TrimSpace(outVerb))
	}
}

// TestExecute_Help asserts that `aiwf --help` writes the printHelp
// content (a substring known to be in the help banner) and exits 0.
func TestExecute_Help(t *testing.T) {
	out, code := captureExecuteOutput(t, []string{"--help"})
	if code != cliutil.ExitOK {
		t.Errorf("exit code: got %d, want %d", code, cliutil.ExitOK)
	}
	if !strings.Contains(string(out), "ai-workflow framework CLI") {
		t.Errorf("expected help banner, got: %s", string(out))
	}
}

// TestNewRootCmd_HasExpectedVerbs asserts the root command tree has
// every verb the framework ships. Structural pinning catches
// accidental verb removal during the cmd/aiwf → internal/cli
// migration.
func TestNewRootCmd_HasExpectedVerbs(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	expected := []string{
		"check", "add", "promote", "cancel", "rename", "retitle",
		"edit-body", "move", "reallocate", "rewidth", "archive",
		"init", "update", "upgrade", "history", "doctor", "render",
		"import", "whoami", "status", "list", "schema", "show",
		"template", "contract", "milestone", "authorize", "version",
	}
	got := map[string]bool{}
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}
	for _, v := range expected {
		if !got[v] {
			t.Errorf("verb %q missing from root command tree", v)
		}
	}
}

// TestResolvedVersion_FallsBackToBuildInfo: when the package-level
// Version is at its default sentinel "dev", ResolvedVersion returns
// the buildinfo-derived value. Catches the bug class where
// resolvedVersion shorts to the wrong field.
func TestResolvedVersion_FallsBackToBuildInfo(t *testing.T) {
	t.Parallel()
	// Save and restore Version since this test can't run parallel
	// with TestResolvedVersion_PrefersStampedValue (both mutate it).
	// The two are also mutually exclusive with anything that calls
	// Execute (which reads Version via the root RunE).
	orig := Version
	defer func() { Version = orig }()

	Version = "dev"
	got := ResolvedVersion()
	if got == "" || got == "dev" {
		t.Errorf("with Version=dev expected buildinfo fallback (e.g. (devel) or a tag); got %q", got)
	}
}
