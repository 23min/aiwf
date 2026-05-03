package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Binary integration tests close G27's bug class: they build the
// actual cmd binary and run verbs as subprocesses so the production
// path — `runtime/debug.ReadBuildInfo` plus the ldflags-stamped
// Version global — is exercised the way a user's installed binary
// would be.
//
// `go test` on its own cannot catch the v0.1.0 bug class
// (`aiwf version` returning "dev" while doctor read buildinfo
// correctly); the test binary's buildinfo always reads as "(devel)"
// and there is no runtime way to spoof a different value. Subprocess-
// ing a freshly-built binary is the only path.
//
// Tests are gated under `-short` because each shells out to
// `go build` (~3-5s on a warm cache); CI's default `go test` opts
// in, faster local iterations skip via `-short`.

// TestBinary_VersionVerb_RespectsLdflags pins the ldflags-stamped
// path: a binary built with `-ldflags="-X main.Version=v0.99.0-…"`
// must report that exact value from `aiwf version`. This is the
// `make install` path the kernel-dev repo uses today.
func TestBinary_VersionVerb_RespectsLdflags(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	const stampedVersion = "v0.99.0-binary-integration-test"
	bin := buildBinary(t, tmp, "-ldflags=-X main.Version="+stampedVersion)

	out, err := runBinary(bin, "version")
	if err != nil {
		t.Fatalf("aiwf version: %v\n%s", err, out)
	}
	got := strings.TrimSpace(out)
	if got != stampedVersion {
		t.Errorf("aiwf version = %q, want %q\n(G27: ldflags-stamped value did not reach the verb)", got, stampedVersion)
	}
}

// TestBinary_VersionVerb_FallsBackToBuildInfo pins the seam between
// `aiwf version` and `aiwf doctor`'s `binary:` row when no ldflags
// stamp is present. Both surfaces must report the *same* underlying
// value — a regression that updates one source of truth without the
// other (the v0.1.0 shape) fails this test even when both surfaces
// individually look "right" in isolation.
func TestBinary_VersionVerb_FallsBackToBuildInfo(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp /* no ldflags */)

	verOut, err := runBinary(bin, "version")
	if err != nil {
		t.Fatalf("aiwf version: %v\n%s", err, verOut)
	}
	verVer := strings.TrimSpace(verOut)
	if verVer == "" {
		t.Fatal("aiwf version printed empty output")
	}
	if verVer == "dev" {
		t.Errorf("aiwf version returned literal sentinel %q — G27 regression: the no-ldflags path should defer to runtime/debug.ReadBuildInfo", verVer)
	}

	// doctor's binary: row carries `<version> (<state-label>)`. Pull
	// the version token and assert it matches `aiwf version`'s output.
	doctorOut, err := runBinary(bin, "doctor", "--root", tmp)
	if err != nil && !exitedWithCode(err, 1) {
		// doctor exits 1 ("findings") when aiwf.yaml is missing in
		// --root; that's expected here. Anything else is a real fail.
		t.Fatalf("aiwf doctor: %v\n%s", err, doctorOut)
	}
	row := extractRow(doctorOut, "binary:")
	if row == "" {
		t.Fatalf("aiwf doctor missing 'binary:' row\n%s", doctorOut)
	}
	docVer := versionTokenFromBinaryRow(row)
	if docVer == "" {
		t.Fatalf("could not extract version token from doctor row %q", row)
	}
	if docVer != verVer {
		t.Errorf("seam mismatch (G27): aiwf version = %q, doctor binary: row version = %q\nrow: %s", verVer, docVer, strings.TrimSpace(row))
	}
}

// skipIfShortOrUnsupported gates the binary integration tests:
// requires `go` on PATH, skipped under `-short`, skipped on Windows
// (aiwf is unix-only).
func skipIfShortOrUnsupported(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping binary integration test (-short); requires go build")
	}
	if runtime.GOOS == "windows" {
		t.Skip("aiwf is unix-only; binary integration test follows suit")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go not on PATH: %v", err)
	}
}

// buildBinary compiles ./tools/cmd/aiwf into tmp/aiwf with the given
// extra `go build` args (typically `-ldflags=…`) and returns the path.
// Builds happen from the repo root so the relative package path
// resolves regardless of which package the test runs in.
func buildBinary(t *testing.T, tmp string, extraArgs ...string) string {
	t.Helper()
	out := filepath.Join(tmp, "aiwf")
	args := append([]string{"build"}, extraArgs...)
	args = append(args, "-o", out, "./tools/cmd/aiwf")
	cmd := exec.Command("go", args...)
	cmd.Dir = repoRootForTest(t)
	if buildOut, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, buildOut)
	}
	return out
}

// repoRootForTest walks up from the test's cwd looking for go.mod
// and returns the absolute directory containing it. The test binary
// runs in the package directory (tools/cmd/aiwf); the repo root is
// two levels up.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	for i := 0; i < 6; i++ {
		if _, err := exec.Command("test", "-f", filepath.Join(dir, "go.mod")).Output(); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate repo root (no go.mod in 6 parents)")
	return ""
}

// runBinary invokes bin with args and returns combined stdout+stderr.
// Combined output is what a user sees, so the assertions read the
// same bytes the user would.
func runBinary(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// exitedWithCode reports whether err is an *exec.ExitError with the
// given exit code. Used to tolerate doctor's `exitFindings` (1) when
// no aiwf.yaml is present in --root.
func exitedWithCode(err error, code int) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == code
	}
	return false
}

// extractRow returns the first line of haystack whose prefix (after
// trimming leading whitespace) matches prefix. Empty if not found.
func extractRow(haystack, prefix string) string {
	for _, line := range strings.Split(haystack, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), prefix) {
			return line
		}
	}
	return ""
}

// versionTokenFromBinaryRow extracts the version string from a
// doctor `binary:` row of the shape "binary:    <version> (<state>)".
// Returns the value between the colon-space and the trailing
// state-label parenthetical. Empty when the row doesn't match.
func versionTokenFromBinaryRow(row string) string {
	row = strings.TrimSpace(row)
	const prefix = "binary:"
	if !strings.HasPrefix(row, prefix) {
		return ""
	}
	rest := strings.TrimSpace(row[len(prefix):])
	if i := strings.LastIndex(rest, " ("); i > 0 {
		return rest[:i]
	}
	return rest
}
