// Package integration holds the cobra-driven integration tests for
// the aiwf binary. They exercise the dispatcher via cli.Execute
// in-process — the verbs build, the trailers land, the FSM
// transitions, all without spinning a subprocess.
//
// Binary-subprocess tests (the ones that build the binary and exec
// it to catch buildinfo-only bug classes per G-0027) continue to
// live at cmd/aiwf/ next to the binary entry point.
//
// Why this package exists at all (M-0118/AC-6): pre-M-0118 every
// dispatcher test lived in cmd/aiwf/ as `package main`, calling the
// in-package `run` and `newRootCmd`. After AC-1 moved those into
// internal/cli/, the cobra-driven tests followed so cmd/aiwf could
// shrink to its entry-only G-0107 shape.
package integration

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the test
// binary, so once-setup is correct.
//
// Per M-0091/M-0092 the test discipline is "t.Parallel by default,
// document serial cases in this comment block." Serial tests in this
// package (mutate process-level state or saturate shared resources):
//
//   - Any test calling testutil.CaptureStdout / CaptureStderr /
//     CaptureRun — mutates os.Stdout / os.Stderr, which are
//     process-level fds shared by every goroutine. Lots of these in
//     this package (every verb-level test that asserts on printed
//     output goes through capture).
//   - Any test calling t.Setenv (panics under t.Parallel). The
//     doctor/actor/whoami tests use this to isolate $HOME and
//     $XDG_CONFIG_HOME.
//   - Any test calling os.Chdir (process-wide cwd mutation). The
//     completion-helpers and whoami tests do this.
//
// The cmd/aiwf-side setup_test.go remains the home for binary-
// subprocess test discipline (the integration_g37 fan-out, etc).
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
