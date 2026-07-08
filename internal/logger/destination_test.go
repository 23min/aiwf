package logger

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixedNow is an arbitrary, deterministic instant used across this
// file's tests. UTC by construction so date-derived paths are
// independent of the test runner's local timezone.
var fixedNow = time.Date(2026, time.July, 7, 12, 0, 0, 0, time.UTC)

func envFrom(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

func TestOpenDestination_Disabled_NoDiskTouch(t *testing.T) {
	t.Parallel()
	xdg := filepath.Join(t.TempDir(), "does-not-exist-yet")
	w, err := OpenDestination(Config{}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg}))
	if err != nil {
		t.Fatalf("OpenDestination() error = %v, want nil", err)
	}
	if w != nil {
		t.Fatalf("OpenDestination() writer = %v, want nil for a disabled config", w)
	}
	if _, statErr := os.Stat(xdg); !os.IsNotExist(statErr) {
		t.Fatalf("XDG_STATE_HOME dir exists after a disabled OpenDestination call, want untouched")
	}
}

func TestOpenDestination_Stderr(t *testing.T) {
	t.Parallel()
	w, err := OpenDestination(Config{Enabled: true, Destination: "stderr"}, fixedNow, envFrom(nil))
	if err != nil {
		t.Fatalf("OpenDestination() error = %v, want nil", err)
	}
	if w != os.Stderr {
		t.Fatalf("OpenDestination() writer = %v, want os.Stderr", w)
	}
}

func TestOpenDestination_ExplicitPath_AppendsAcrossOpens(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "custom.log")
	cfg := Config{Enabled: true, Destination: path}

	w1, openErr := OpenDestination(cfg, fixedNow, envFrom(nil))
	if openErr != nil {
		t.Fatalf("first OpenDestination() error = %v, want nil", openErr)
	}
	if _, writeErr := w1.Write([]byte("first\n")); writeErr != nil {
		t.Fatalf("first Write() error = %v", writeErr)
	}
	if closeErr := w1.Close(); closeErr != nil {
		t.Fatalf("first Close() error = %v", closeErr)
	}

	w2, openErr2 := OpenDestination(cfg, fixedNow, envFrom(nil))
	if openErr2 != nil {
		t.Fatalf("second OpenDestination() error = %v, want nil", openErr2)
	}
	if _, writeErr := w2.Write([]byte("second\n")); writeErr != nil {
		t.Fatalf("second Write() error = %v", writeErr)
	}
	if closeErr := w2.Close(); closeErr != nil {
		t.Fatalf("second Close() error = %v", closeErr)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if want := "first\nsecond\n"; string(got) != want {
		t.Fatalf("file content = %q, want %q (second open must append, not truncate)", got, want)
	}
}

func TestOpenDestination_Default_UsesXDGStateHome(t *testing.T) {
	t.Parallel()
	xdg := t.TempDir()
	w, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg}))
	if err != nil {
		t.Fatalf("OpenDestination() error = %v, want nil", err)
	}
	defer w.Close()

	if _, writeErr := w.Write([]byte("event\n")); writeErr != nil {
		t.Fatalf("Write() error = %v", writeErr)
	}
	want := filepath.Join(xdg, "aiwf", "logs", "aiwf-2026-07-07.log")
	got, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("expected log file %s: %v", want, err)
	}
	if string(got) != "event\n" {
		t.Fatalf("file content = %q, want %q", got, "event\n")
	}
}

func TestOpenDestination_Default_FallsBackToHomeWhenXDGUnset(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	w, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(map[string]string{"HOME": home}))
	if err != nil {
		t.Fatalf("OpenDestination() error = %v, want nil", err)
	}
	defer w.Close()

	want := filepath.Join(home, ".local", "state", "aiwf", "logs", "aiwf-2026-07-07.log")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected log file %s to exist: %v", want, err)
	}
}

func TestOpenDestination_Default_FileAndDirAreOwnerOnly(t *testing.T) {
	t.Parallel()
	xdg := t.TempDir()
	logsDir := filepath.Join(xdg, "aiwf", "logs")

	w, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg}))
	if err != nil {
		t.Fatalf("OpenDestination() error = %v, want nil", err)
	}
	defer w.Close()

	dirInfo, err := os.Stat(logsDir)
	if err != nil {
		t.Fatalf("os.Stat(logsDir) error = %v", err)
	}
	if got, want := dirInfo.Mode().Perm(), fs.FileMode(0o700); got != want {
		t.Errorf("logs dir perm = %o, want %o (per-user, never shared — ADR-0017)", got, want)
	}

	fileInfo, err := os.Stat(filepath.Join(logsDir, logFileName(fixedNow)))
	if err != nil {
		t.Fatalf("os.Stat(log file) error = %v", err)
	}
	if got, want := fileInfo.Mode().Perm(), fs.FileMode(0o600); got != want {
		t.Errorf("log file perm = %o, want %o (per-user, never shared — ADR-0017)", got, want)
	}
}

func TestOpenDestination_Default_BothXDGAndHomeUnsetReturnsError(t *testing.T) {
	t.Parallel()
	// Neither XDG_STATE_HOME nor HOME resolves: falling back to a bare
	// relative path would write into whatever directory the process
	// happens to be running in (e.g. the operator's repo) — refuse
	// instead.
	_, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(nil))
	if err == nil {
		t.Fatalf("OpenDestination() error = nil, want a non-nil error when neither XDG_STATE_HOME nor HOME is set")
	}
}

func TestOpenDestination_Default_DirCreatedOnlyOnOptedInWrite(t *testing.T) {
	t.Parallel()
	xdg := t.TempDir()
	logsDir := filepath.Join(xdg, "aiwf", "logs")

	if _, err := OpenDestination(Config{}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg})); err != nil {
		t.Fatalf("disabled OpenDestination() error = %v, want nil", err)
	}
	if _, statErr := os.Stat(logsDir); !os.IsNotExist(statErr) {
		t.Fatalf("logs dir exists after a disabled call, want absent")
	}

	w, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg}))
	if err != nil {
		t.Fatalf("enabled OpenDestination() error = %v, want nil", err)
	}
	defer w.Close()
	if _, statErr := os.Stat(logsDir); statErr != nil {
		t.Fatalf("logs dir missing after an opted-in write: %v", statErr)
	}
}

func TestOpenDestination_SweepsEntriesOlderThan30Days(t *testing.T) {
	t.Parallel()
	xdg := t.TempDir()
	logsDir := filepath.Join(xdg, "aiwf", "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	old := filepath.Join(logsDir, "aiwf-2026-05-01.log")      // 67 days before fixedNow: expired
	boundary := filepath.Join(logsDir, "aiwf-2026-06-07.log") // exactly 30 days before: kept
	recent := filepath.Join(logsDir, "aiwf-2026-07-06.log")   // 1 day before: kept
	stray := filepath.Join(logsDir, "notes.txt")              // non-matching name: kept
	for _, p := range []string{old, boundary, recent, stray} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("seeding %s: %v", p, err)
		}
	}

	w, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg}))
	if err != nil {
		t.Fatalf("OpenDestination() error = %v, want nil", err)
	}
	w.Close()

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("expired file %s survived the sweep", old)
	}
	for _, p := range []string{boundary, recent, stray} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("non-expired file %s was swept: %v", p, err)
		}
	}
}

func TestSweepExpired_MissingDirIsNotAnError(t *testing.T) {
	t.Parallel()
	err := sweepExpired(filepath.Join(t.TempDir(), "absent"), fixedNow)
	if err != nil {
		t.Fatalf("sweepExpired() on a missing dir error = %v, want nil", err)
	}
}

func TestRemoveExpiredFile_AlreadyRemovedIsNotAnError(t *testing.T) {
	t.Parallel()
	// Reproduces the race two aiwf processes hit when both list the
	// same expired file after a daily rollover and race to delete it:
	// the loser's os.Remove sees a path that's already gone.
	path := filepath.Join(t.TempDir(), "aiwf-2020-01-01.log")
	err := removeExpiredFile(path)
	if err != nil {
		t.Fatalf("removeExpiredFile() on an already-removed path error = %v, want nil", err)
	}
}

func TestRemoveExpiredFile_GenericErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "aiwf-2020-01-01.log")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding file: %v", err)
	}
	// Removing write permission on the containing dir blocks os.Remove
	// (POSIX: unlink needs write on the containing dir, not the file).
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	if err := removeExpiredFile(path); err == nil {
		t.Fatalf("removeExpiredFile() on a permission-denied path error = nil, want a non-nil error")
	}
}

func TestSweepExpired_GenericReadDirErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err := sweepExpired(dir, fixedNow)
	if err == nil {
		t.Fatalf("sweepExpired() on an unreadable dir error = nil, want a non-nil, non-not-exist error")
	}
}

func TestOpenDestination_ExplicitPath_MissingParentDirReturnsError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "missing-subdir", "custom.log")
	cfg := Config{Enabled: true, Destination: path}

	_, err := OpenDestination(cfg, fixedNow, envFrom(nil))
	if err == nil {
		t.Fatalf("OpenDestination() error = nil, want a non-nil error for a missing parent directory")
	}
}

func TestOpenDestination_Default_MkdirAllFailureReturnsError(t *testing.T) {
	t.Parallel()
	xdg := t.TempDir()
	// A regular file where "aiwf" must be a directory blocks MkdirAll
	// from creating the "logs" subdirectory beneath it.
	if err := os.WriteFile(filepath.Join(xdg, "aiwf"), []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("seeding blocking file: %v", err)
	}

	_, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg}))
	if err == nil {
		t.Fatalf("OpenDestination() error = nil, want a non-nil error when the log directory can't be created")
	}
}

func TestOpenDestination_Default_FileOpenFailureReturnsError(t *testing.T) {
	t.Parallel()
	xdg := t.TempDir()
	logsDir := filepath.Join(xdg, "aiwf", "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	// No write permission on the directory: MkdirAll (already exists)
	// and the (empty) sweep both succeed, but creating today's file
	// fails — isolating the final os.OpenFile error path.
	if err := os.Chmod(logsDir, 0o555); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { os.Chmod(logsDir, 0o755) })

	_, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg}))
	if err == nil {
		t.Fatalf("OpenDestination() error = nil, want a non-nil error when the day's log file can't be created")
	}
}

func TestOpenDestination_Default_SweepFailurePropagates(t *testing.T) {
	t.Parallel()
	xdg := t.TempDir()
	logsDir := filepath.Join(xdg, "aiwf", "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	expired := filepath.Join(logsDir, "aiwf-2020-01-01.log")
	if err := os.WriteFile(expired, []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding expired file: %v", err)
	}
	// Removing write permission on the directory blocks os.Remove of
	// the expired file inside it (POSIX: unlink needs write on the
	// containing dir, not the file itself).
	if err := os.Chmod(logsDir, 0o555); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { os.Chmod(logsDir, 0o755) })

	_, err := OpenDestination(Config{Enabled: true}, fixedNow, envFrom(map[string]string{"XDG_STATE_HOME": xdg}))
	if err == nil {
		t.Fatalf("OpenDestination() error = nil, want a non-nil error when the retention sweep fails")
	}
}

func TestParseLogFileDate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		file    string
		wantOK  bool
		wantVal time.Time
	}{
		{"valid", "aiwf-2026-07-07.log", true, time.Date(2026, time.July, 7, 0, 0, 0, 0, time.UTC)},
		{"missing prefix", "2026-07-07.log", false, time.Time{}},
		{"missing suffix", "aiwf-2026-07-07.txt", false, time.Time{}},
		{"malformed date", "aiwf-not-a-date.log", false, time.Time{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseLogFileDate(tc.file)
			if ok != tc.wantOK {
				t.Fatalf("parseLogFileDate(%q) ok = %v, want %v", tc.file, ok, tc.wantOK)
			}
			if ok && !got.Equal(tc.wantVal) {
				t.Fatalf("parseLogFileDate(%q) = %v, want %v", tc.file, got, tc.wantVal)
			}
		})
	}
}
