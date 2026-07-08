package cliutil

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func fakeGetenv(vals map[string]string) func(string) string {
	return func(key string) string { return vals[key] }
}

func TestResolveLogger_DisabledByDefault(t *testing.T) {
	t.Parallel()
	l, closeFn := ResolveLogger(t.TempDir(), fakeGetenv(nil))
	if l.Enabled(context.Background(), slog.LevelError) {
		t.Error("logger must be disabled (Enabled(Error) = true) when AIWF_LOG is unset")
	}
	if err := closeFn(); err != nil {
		t.Errorf("closeFn() = %v, want nil", err)
	}
}

func TestResolveLogger_InvalidLevel_FallsBackToDisabled(t *testing.T) {
	t.Parallel()
	l, closeFn := ResolveLogger(t.TempDir(), fakeGetenv(map[string]string{"AIWF_LOG": "not-a-level"}))
	if l.Enabled(context.Background(), slog.LevelError) {
		t.Error("logger must fall back to disabled on a resolve error, not panic or leave a half-built logger")
	}
	if err := closeFn(); err != nil {
		t.Errorf("closeFn() = %v, want nil", err)
	}
}

func TestResolveLogger_DestinationOpenFails_FallsBackToDisabled(t *testing.T) {
	t.Parallel()
	// AIWF_LOG_FILE names a directory, not a file — appendFile's
	// os.OpenFile(O_WRONLY) fails on it, exercising OpenDestination's
	// error return.
	l, closeFn := ResolveLogger(t.TempDir(), fakeGetenv(map[string]string{
		"AIWF_LOG":      "info",
		"AIWF_LOG_FILE": t.TempDir(),
	}))
	if l.Enabled(context.Background(), slog.LevelError) {
		t.Error("logger must fall back to disabled when the destination fails to open")
	}
	if err := closeFn(); err != nil {
		t.Errorf("closeFn() = %v, want nil", err)
	}
}

func TestResolveLogger_ExplicitFileDestination_WritesAndCloses(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	l, closeFn := ResolveLogger(t.TempDir(), fakeGetenv(map[string]string{
		"AIWF_LOG":      "info",
		"AIWF_LOG_FILE": path,
	}))
	l.Info("event.fired", "verb", "cancel")
	if err := closeFn(); err != nil {
		t.Fatalf("closeFn() = %v, want nil", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(content) == 0 {
		t.Fatal("log file is empty; want the event.fired record")
	}
}

// TestResolveLogger_YAMLLoggingBlock_EnablesWithoutEnvVar pins M-0238/AC-4:
// aiwf.yaml's logging: block alone (no AIWF_LOG set) enables the logger,
// reading through the real config.Load path.
func TestResolveLogger_YAMLLoggingBlock_EnablesWithoutEnvVar(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "diag.log")
	writeAiwfYAML(t, root, "logging:\n  level: info\n  destination: "+path+"\n")

	l, closeFn := ResolveLogger(root, fakeGetenv(nil))
	l.Info("event.fired", "verb", "cancel")
	if err := closeFn(); err != nil {
		t.Fatalf("closeFn() = %v, want nil", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(content) == 0 {
		t.Fatal("log file is empty; want the event.fired record — logging.level in aiwf.yaml should have enabled it")
	}
}

// TestResolveLogger_EnvBeatsYAMLLoggingBlock pins the precedence half of
// AC-4: AIWF_LOG overrides a conflicting aiwf.yaml logging.level, per
// ADR-0017 Decision #3 (env beats yaml beats default, per field).
func TestResolveLogger_EnvBeatsYAMLLoggingBlock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, "logging:\n  level: error\n")

	l, closeFn := ResolveLogger(root, fakeGetenv(map[string]string{
		"AIWF_LOG":      "debug",
		"AIWF_LOG_FILE": "stderr",
	}))
	defer func() { _ = closeFn() }()
	if !l.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("AIWF_LOG=debug should win over aiwf.yaml's logging.level: error")
	}
}

// TestResolveLogger_MissingAiwfYAML_TreatedAsNoLoggingBlock pins the
// pre-init tolerance: no aiwf.yaml at all (config.Load returns
// ErrNotFound) behaves exactly like an aiwf.yaml with no logging: block —
// never a hard failure.
func TestResolveLogger_MissingAiwfYAML_TreatedAsNoLoggingBlock(t *testing.T) {
	t.Parallel()
	l, closeFn := ResolveLogger(t.TempDir(), fakeGetenv(nil))
	if l.Enabled(context.Background(), slog.LevelError) {
		t.Error("logger must be disabled when neither AIWF_LOG nor aiwf.yaml (absent entirely) set a level")
	}
	if err := closeFn(); err != nil {
		t.Errorf("closeFn() = %v, want nil", err)
	}
}

func writeAiwfYAML(t *testing.T, root, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Serial: uses captureStdStreams (see setup_test.go's serial note).
//
// ResolveLogger must run INSIDE the capture window: OpenDestination's
// "stderr" case reads the current os.Stderr package variable, which
// captureStdStreams has already swapped to a pipe by the time this
// closure runs. Calling ResolveLogger outside the window would bind
// closeFn to the real stderr's Close method while the probe write
// below targets the swapped pipe instead — the mismatch that let a
// close-real-stderr mutant survive undetected during this AC's
// wf-vacuity audit.
func TestResolveLogger_StderrDestination_NeverClosesRealStderr(t *testing.T) {
	var closeErr, writeErr error
	_, errOut := captureStdStreams(t, func() {
		l, closeFn := ResolveLogger(t.TempDir(), fakeGetenv(map[string]string{
			"AIWF_LOG":      "info",
			"AIWF_LOG_FILE": "stderr",
		}))
		l.Info("event.fired", "verb", "cancel")
		closeErr = closeFn()
		// If closeFn had closed the resolved stderr stream, this second
		// write would fail instead of landing in the capture.
		_, writeErr = os.Stderr.WriteString("still open\n")
	})
	if closeErr != nil {
		t.Fatalf("closeFn() = %v, want nil", closeErr)
	}
	if writeErr != nil {
		t.Fatalf("os.Stderr unusable after closeFn(): %v", writeErr)
	}
	if errOut == "" {
		t.Fatal("stderr capture is empty; want the event.fired record and the post-close probe write")
	}
}
