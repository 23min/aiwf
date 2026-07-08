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

// TestResolveTraceLogger_EnabledWithoutAIWFLOG pins the whole point of
// --trace: no AIWF_LOG set anywhere, yet the resolved logger is
// enabled at debug — and, critically, still honors AIWF_LOG_FILE/
// AIWF_LOG_FORMAT from the real environment. This is the exact bug a
// naive "patch Enabled+Level after ResolveConfig returns" approach
// missed: ResolveConfig short-circuits to a bare zero Config (format/
// destination never resolved) when no level is set anywhere.
func TestResolveTraceLogger_EnabledWithoutAIWFLOG(t *testing.T) {
	t.Parallel()
	logPath := filepath.Join(t.TempDir(), "trace.log")
	l, closeFn := ResolveTraceLogger(t.TempDir(), fakeGetenv(map[string]string{
		"AIWF_LOG_FORMAT": "json",
		"AIWF_LOG_FILE":   logPath,
	}))
	defer func() { _ = closeFn() }()
	if !l.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("ResolveTraceLogger must enable debug level even with no AIWF_LOG set")
	}
	l.Debug("phase.apply", "elapsed_ms", 5)
	_ = closeFn()
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading %s: %v (AIWF_LOG_FILE was not honored)", logPath, err)
	}
	if len(raw) == 0 {
		t.Fatal("log file is empty")
	}
}

// TestResolveTraceLogger_ClampsAMoreRestrictiveAIWFLOG pins the
// override half: an operator with AIWF_LOG=warn set (deliberately
// more restrictive than debug) still gets phase-apply timing when
// they also pass --trace — the flag always wins for this invocation.
func TestResolveTraceLogger_ClampsAMoreRestrictiveAIWFLOG(t *testing.T) {
	t.Parallel()
	l, closeFn := ResolveTraceLogger(t.TempDir(), fakeGetenv(map[string]string{
		"AIWF_LOG":      "warn",
		"AIWF_LOG_FILE": "stderr",
	}))
	defer func() { _ = closeFn() }()
	if !l.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("--trace must clamp an operator's own AIWF_LOG=warn down to debug for this invocation")
	}
}

// TestResolveTraceLogger_MissingAiwfYAML_TreatedAsNoLoggingBlock
// mirrors ResolveLogger's identical pre-init tolerance.
func TestResolveTraceLogger_MissingAiwfYAML_TreatedAsNoLoggingBlock(t *testing.T) {
	t.Parallel()
	l, closeFn := ResolveTraceLogger(t.TempDir(), fakeGetenv(map[string]string{
		"AIWF_LOG_FILE": "stderr",
	}))
	defer func() { _ = closeFn() }()
	if !l.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("ResolveTraceLogger must still enable debug level when aiwf.yaml is entirely absent")
	}
}

// TestResolveTraceLogger_InvalidFormat_FallsBackButStaysEnabled pins
// logger.ResolveConfig's own error path (an invalid AIWF_LOG_FORMAT
// value) — --trace's whole reason for existing is "give me timing
// regardless," so even here the logger must still come back enabled
// at debug (falling back to text format), never a hard failure.
func TestResolveTraceLogger_InvalidFormat_FallsBackButStaysEnabled(t *testing.T) {
	t.Parallel()
	l, closeFn := ResolveTraceLogger(t.TempDir(), fakeGetenv(map[string]string{
		"AIWF_LOG_FORMAT": "not-a-format",
		"AIWF_LOG_FILE":   "stderr",
	}))
	defer func() { _ = closeFn() }()
	if !l.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("ResolveTraceLogger must stay enabled at debug even when logger.ResolveConfig errors on an invalid format")
	}
}

// TestResolveTraceLogger_InvalidFormat_FallsBackToYAMLDestination
// covers the other half of the fallback's destination resolution: no
// AIWF_LOG_FILE in the environment at all, so the fallback must still
// honor aiwf.yaml's logging.destination rather than dropping it too.
func TestResolveTraceLogger_InvalidFormat_FallsBackToYAMLDestination(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, "logging:\n  destination: stderr\n")
	l, closeFn := ResolveTraceLogger(root, fakeGetenv(map[string]string{
		"AIWF_LOG_FORMAT": "not-a-format",
	}))
	defer func() { _ = closeFn() }()
	if !l.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("ResolveTraceLogger must stay enabled at debug and honor aiwf.yaml's logging.destination on a format error")
	}
}

// TestResolveTraceLogger_DestinationOpenFails_FallsBackToDisabled
// mirrors ResolveLogger's identical destination-open-failure test —
// the one case where --trace legitimately can't proceed (no writable
// destination), and diagnostic logging must never escalate that into
// a verb failure.
func TestResolveTraceLogger_DestinationOpenFails_FallsBackToDisabled(t *testing.T) {
	t.Parallel()
	l, closeFn := ResolveTraceLogger(t.TempDir(), fakeGetenv(map[string]string{
		"AIWF_LOG_FILE": t.TempDir(),
	}))
	if l.Enabled(context.Background(), slog.LevelError) {
		t.Error("logger must fall back to disabled when the destination fails to open, even under --trace")
	}
	if err := closeFn(); err != nil {
		t.Errorf("closeFn() = %v, want nil", err)
	}
}
