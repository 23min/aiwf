package logger

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// retentionDays is the number of days a daily log file is kept before
// OpenDestination sweeps it, per ADR-0017 Decision #4.
const retentionDays = 30

// logFileLayout is the date format embedded in a daily log file's
// name (aiwf-YYYY-MM-DD.log), matching Go's reference-time layout.
const logFileLayout = "2006-01-02"

// OpenDestination resolves cfg.Destination into an io.WriteCloser:
// "stderr" opens os.Stderr, an explicit absolute path opens that file
// for append, and "" (the default) opens the daily
// $XDG_STATE_HOME/aiwf/logs/aiwf-YYYY-MM-DD.log file (falling back to
// ~/.local/state/aiwf/logs when XDG_STATE_HOME is unset), creating its
// directory and sweeping entries older than 30 days only on this call.
//
// Returns (nil, nil) when cfg.Enabled is false: OpenDestination never
// touches disk for a disabled config, matching ADR-0017's default-off
// constraint regardless of what a caller passes.
//
// now and getenv are injected so callers (and tests) control the
// clock and environment — internal/logger sits below the layering
// tier that may read the ambient wall clock directly (no-time-now-in-core).
func OpenDestination(cfg Config, now time.Time, getenv func(string) string) (io.WriteCloser, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	switch cfg.Destination {
	case "stderr":
		return os.Stderr, nil
	case "":
		return openDefault(now, getenv)
	default:
		return appendFile(cfg.Destination)
	}
}

// appendFile opens path for append, creating it if absent. This is
// the one legitimate exception to the repo's temp+rename atomic-write
// discipline (ADR-0017 Decision #5): the diagnostic log is a shared,
// append-only, multi-writer stream, so O_APPEND + one Write() call
// per record is the correct pattern here, not atomic replace.
func appendFile(path string) (io.WriteCloser, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s: %w", path, err)
	}
	return f, nil
}

// openDefault opens the default XDG-state-home daily log file,
// creating its directory and sweeping expired entries first — both
// only ever reached from a cfg.Enabled call, per OpenDestination's
// default-off guarantee.
func openDefault(now time.Time, getenv func(string) string) (io.WriteCloser, error) {
	dir, err := defaultLogDir(getenv)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory %s: %w", dir, err)
	}
	if err := sweepExpired(dir, now); err != nil {
		return nil, fmt.Errorf("sweeping expired logs in %s: %w", dir, err)
	}
	return appendFile(filepath.Join(dir, logFileName(now)))
}

// defaultLogDir returns the directory the default destination's daily
// files live in: $XDG_STATE_HOME/aiwf/logs, or
// ~/.local/state/aiwf/logs when XDG_STATE_HOME is unset (ADR-0017
// Decision #4). Returns an error when neither resolves — silently
// falling back to a bare relative path would write into whatever
// directory the process happens to be running in.
func defaultLogDir(getenv func(string) string) (string, error) {
	if xdg := getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "aiwf", "logs"), nil
	}
	home := getenv("HOME")
	if home == "" {
		return "", errors.New("logging: neither XDG_STATE_HOME nor HOME is set; cannot resolve the default log directory")
	}
	return filepath.Join(home, ".local", "state", "aiwf", "logs"), nil
}

// logFileName renders the daily log file's name for the UTC date of
// now.
func logFileName(now time.Time) string {
	return fmt.Sprintf("aiwf-%s.log", now.UTC().Format(logFileLayout))
}

// sweepExpired removes daily log files in dir whose embedded date is
// more than retentionDays before now. Files not matching the
// aiwf-YYYY-MM-DD.log naming pattern are left alone. A missing dir is
// not an error — nothing to sweep.
func sweepExpired(dir string, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	today := now.UTC()
	cutoff := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -retentionDays)
	for _, e := range entries {
		date, ok := parseLogFileDate(e.Name())
		if !ok {
			continue
		}
		if date.Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return fmt.Errorf("removing expired log %s: %w", e.Name(), err)
			}
		}
	}
	return nil
}

// parseLogFileDate extracts the UTC date embedded in a daily log
// file's name (aiwf-YYYY-MM-DD.log), reporting false for any name
// that doesn't match the pattern.
func parseLogFileDate(name string) (time.Time, bool) {
	const prefix, suffix = "aiwf-", ".log"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return time.Time{}, false
	}
	datePart := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	t, err := time.Parse(logFileLayout, datePart)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
