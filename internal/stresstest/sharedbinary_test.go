package stresstest

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// prebuiltBinaryEnvVar/-LockHolderEnvVar name env vars mutate-hunt.yml
// sets to binaries it built once, up front. sync.Once below only
// dedupes a build within one process, but gremlins spawns a fresh `go
// test` process per mutant, so every mutant would otherwise re-pay
// the full build cost on top of its own test run. Reusing one
// externally-built binary across every mutant is safe regardless of
// which package gremlins is currently mutating: the real-subprocess
// scenarios below always treat the aiwf/lockholder binaries as fixed,
// already-built tools (testing real OS-process behavior — kill
// signals, fd cleanup), never as the code under mutation. A mutation
// to cmd/aiwf or lockholder/main.go's run() is caught by that
// package's own tests compiling the mutated source fresh, wholly
// independent of whichever binary these helpers hand back.
const (
	prebuiltBinaryEnvVar           = "AIWF_STRESSTEST_PREBUILT_BINARY"
	prebuiltLockHolderBinaryEnvVar = "AIWF_STRESSTEST_PREBUILT_LOCKHOLDER_BINARY"
)

// resolvePrebuiltBinary reports whether envVar is set and, if so,
// whether it names a usable (stat-able) file. A set-but-unusable path
// is reported as an error rather than treated like unset, so a
// misconfigured mutate-hunt.yml fails loudly instead of silently
// falling back to a build that defeats the whole reuse optimization.
func resolvePrebuiltBinary(envVar string) (path string, set bool, err error) {
	path = os.Getenv(envVar)
	if path == "" {
		return "", false, nil
	}
	if _, statErr := os.Stat(path); statErr != nil {
		return "", true, statErr
	}
	return path, true, nil
}

// sharedBinaryOnce/-Path/-Err coordinate a single BuildBinary call
// shared across every test in this package's test binary process.
// The AC-1..AC-5 real-subprocess scenarios all need a built aiwf
// binary; rebuilding it per test would multiply an already-slow `go
// build` across dozens of subtests for no benefit. Mirrors this
// repo's own sync.Once-shared-fixture convention (CLAUDE.md "Test
// discipline"). Do not mutate the file the returned path names.
var (
	sharedBinaryOnce sync.Once
	sharedBinaryPath string
	sharedBinaryErr  error
)

// sharedTestBinary returns the absolute path to a real aiwf binary,
// built once per test process — or, when prebuiltBinaryEnvVar names
// an existing file, that file directly, skipping the build.
func sharedTestBinary(t *testing.T) string {
	t.Helper()
	if path, set, err := resolvePrebuiltBinary(prebuiltBinaryEnvVar); set {
		if err != nil { //coverage:ignore triggering this fails the test process by design; the error-producing condition itself is covered by TestResolvePrebuiltBinary
			t.Fatalf("%s set but not usable: %v", prebuiltBinaryEnvVar, err)
		}
		return path
	}
	sharedBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "stresstest-shared-bin-")
		if err != nil {
			sharedBinaryErr = err
			return
		}
		sharedBinaryPath, sharedBinaryErr = BuildBinary(context.Background(), repoRootRelative, dir)
	})
	if sharedBinaryErr != nil {
		t.Fatalf("building shared test binary: %v", sharedBinaryErr)
	}
	return sharedBinaryPath
}

// sharedLockHolderOnce/-Path/-Err mirror sharedBinaryOnce/-Path/-Err
// for M-0242/AC-1's lockholder helper binary — do not mutate the file
// the returned path names.
var (
	sharedLockHolderOnce sync.Once
	sharedLockHolderPath string
	sharedLockHolderErr  error
)

// sharedLockHolderBinary returns the absolute path to the lockholder
// helper binary (internal/stresstest/lockholder), built once per test
// process — or, when prebuiltLockHolderBinaryEnvVar names an existing
// file, that file directly, skipping the build.
func sharedLockHolderBinary(t *testing.T) string {
	t.Helper()
	if path, set, err := resolvePrebuiltBinary(prebuiltLockHolderBinaryEnvVar); set {
		if err != nil { //coverage:ignore triggering this fails the test process by design; the error-producing condition itself is covered by TestResolvePrebuiltBinary
			t.Fatalf("%s set but not usable: %v", prebuiltLockHolderBinaryEnvVar, err)
		}
		return path
	}
	sharedLockHolderOnce.Do(func() {
		dir, err := os.MkdirTemp("", "stresstest-shared-lockholder-")
		if err != nil {
			sharedLockHolderErr = err
			return
		}
		sharedLockHolderPath, sharedLockHolderErr = BuildLockHolder(context.Background(), repoRootRelative, dir)
	})
	if sharedLockHolderErr != nil {
		t.Fatalf("building shared lockholder binary: %v", sharedLockHolderErr)
	}
	return sharedLockHolderPath
}

// TestResolvePrebuiltBinary exhaustively covers resolvePrebuiltBinary's
// three outcomes — unset, set to an existing file, set to a missing
// file — including the error case that sharedTestBinary/
// sharedLockHolderBinary can only turn into a t.Fatalf (untestable
// in-process without failing the test that triggers it). Serial: uses
// t.Setenv.
func TestResolvePrebuiltBinary(t *testing.T) {
	const envVar = "AIWF_STRESSTEST_TEST_RESOLVE_PREBUILT_BINARY"

	tests := []struct {
		name    string
		setEnv  bool
		exists  bool
		wantSet bool
		wantErr bool
	}{
		{name: "unset", setEnv: false, wantSet: false, wantErr: false},
		{name: "set to existing file", setEnv: true, exists: true, wantSet: true, wantErr: false},
		{name: "set to missing file", setEnv: true, exists: false, wantSet: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wantPath string
			if tt.setEnv {
				if tt.exists {
					wantPath = filepath.Join(t.TempDir(), "fake-binary")
					if err := os.WriteFile(wantPath, []byte("fake"), 0o755); err != nil {
						t.Fatalf("writing fake binary: %v", err)
					}
				} else {
					wantPath = filepath.Join(t.TempDir(), "does-not-exist")
				}
				t.Setenv(envVar, wantPath)
			}

			path, set, err := resolvePrebuiltBinary(envVar)

			if set != tt.wantSet {
				t.Errorf("set = %v, want %v", set, tt.wantSet)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantSet && !tt.wantErr && path != wantPath {
				t.Errorf("path = %q, want %q", path, wantPath)
			}
		})
	}
}

// TestSharedBinaryHelpers_PrebuiltEnvVar confirms sharedTestBinary and
// sharedLockHolderBinary return the prebuilt path directly, without
// building, when their env var names an existing file. Serial: uses
// t.Setenv.
func TestSharedBinaryHelpers_PrebuiltEnvVar(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		fn     func(*testing.T) string
	}{
		{"sharedTestBinary", prebuiltBinaryEnvVar, sharedTestBinary},
		{"sharedLockHolderBinary", prebuiltLockHolderBinaryEnvVar, sharedLockHolderBinary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "fake-binary")
			if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
				t.Fatalf("writing fake binary: %v", err)
			}
			t.Setenv(tt.envVar, bin)
			if got := tt.fn(t); got != bin {
				t.Errorf("got %q, want %q", got, bin)
			}
		})
	}
}

// skipIfUnsupported gates the real-subprocess scenario tests in this
// package: they need `go build` (slow; skip under -short) and are
// unix-only, matching aiwf itself.
func skipIfUnsupported(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping real-subprocess stress scenario (-short); requires go build")
	}
	if runtime.GOOS == "windows" {
		t.Skip("aiwf is unix-only; stress scenarios follow suit")
	}
}
