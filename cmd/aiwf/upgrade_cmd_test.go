package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/version"
)

// TestRunUpgrade_CheckOnly_NoNetworkRequired covers the
// `aiwf upgrade --check` path: with GOPROXY=off, the verb falls back
// to "proxy disabled" messaging, prints the current version, and
// exits 0 without touching disk.
func TestRunUpgrade_CheckOnly_NoNetworkRequired(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	rc, stdout, _ := captureRun(t, func() int {
		return run([]string{"upgrade", "--check"})
	})
	if rc != exitOK {
		t.Fatalf("rc = %d, want %d (stdout: %s)", rc, exitOK, stdout)
	}
	if !strings.Contains(stdout, "current:") {
		t.Errorf("missing 'current:' line in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "proxy disabled") {
		t.Errorf("missing 'proxy disabled' indication in output:\n%s", stdout)
	}
}

// TestRunUpgrade_CheckOnly_FakeProxy verifies the comparison
// rendering when a (fake) proxy returns a real-looking semver tag.
// The test binary's current version is a working-tree build, so
// skew is always Unknown — the assertion is on the labels, not the
// classification.
func TestRunUpgrade_CheckOnly_FakeProxy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, `{"Version":"v9.9.9","Time":"2026-05-03T12:00:00Z"}`)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GOPROXY", srv.URL)

	rc, stdout, _ := captureRun(t, func() int {
		return run([]string{"upgrade", "--check"})
	})
	if rc != exitOK {
		t.Fatalf("rc = %d, want %d (stdout: %s)", rc, exitOK, stdout)
	}
	if !strings.Contains(stdout, "v9.9.9 (tagged)") {
		t.Errorf("expected target line to show v9.9.9 (tagged); stdout:\n%s", stdout)
	}
}

// TestRunUpgrade_NoGoBinary covers the friendly "where is go?"
// branch. When AIWF_GO_BIN is empty and `go` is not on PATH, the
// install step fails before exec.
func TestRunUpgrade_NoGoBinary(t *testing.T) {
	t.Setenv("PATH", "")            // nothing on PATH
	t.Setenv("AIWF_GO_BIN", "")     // no override
	t.Setenv("GOPROXY", "off")      // skip proxy lookup
	t.Setenv("AIWF_NO_REEXEC", "1") // belt-and-braces

	rc, _, stderr := captureRun(t, func() int {
		return run([]string{"upgrade", "--version", "v0.1.0"})
	})
	if rc == exitOK {
		t.Fatalf("expected non-zero exit when go binary is missing")
	}
	if !strings.Contains(stderr, "locating `go`") {
		t.Errorf("expected 'locating `go`' message; stderr:\n%s", stderr)
	}
}

// TestRunUpgrade_FullFlow_NoReexec exercises the install path with
// a fake go binary shim. AIWF_NO_REEXEC stops short of the
// syscall.Exec so the test process survives. The shim records its
// invocation; we assert we hit `install <pkg>@v0.1.0` and the
// per-variable `env GOBIN` query.
func TestRunUpgrade_FullFlow_NoReexec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell shim assumes a POSIX-y env")
	}
	tmp := t.TempDir()

	logPath := filepath.Join(tmp, "go.log")
	gobinDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(gobinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := writeUpgradeShim(t, tmp, logPath)

	t.Setenv("AIWF_GO_BIN", shim)
	t.Setenv("GOPROXY", "off")
	t.Setenv("AIWF_NO_REEXEC", "1")
	t.Setenv("AIWF_TEST_GOBIN", gobinDir)
	t.Setenv("AIWF_TEST_GOPATH", tmp)
	t.Setenv("AIWF_TEST_INSTALL_DIR", gobinDir)

	rc, stdout, stderr := captureRun(t, func() int {
		return run([]string{"upgrade", "--version", "v0.1.0", "--root", tmp})
	})
	if rc != exitOK {
		t.Fatalf("rc = %d, want %d (stdout=%s, stderr=%s)", rc, exitOK, stdout, stderr)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading shim log: %v", err)
	}
	log := string(logBytes)
	if !strings.Contains(log, "install ") || !strings.Contains(log, "@v0.1.0") {
		t.Errorf("shim log missing install line:\n%s", log)
	}
	if !strings.Contains(log, "env GOBIN") {
		t.Errorf("shim log missing env GOBIN query:\n%s", log)
	}
	if strings.Contains(log, "env GOPATH") {
		t.Errorf("env GOPATH should not be queried when GOBIN is set:\n%s", log)
	}
	if !strings.Contains(stdout, "AIWF_NO_REEXEC set") {
		t.Errorf("expected NO_REEXEC notice; stdout:\n%s", stdout)
	}
}

// TestRunUpgrade_FullFlow_GOBINUnset is the seam test for G39. The
// most common Go install setup leaves GOBIN unset, so resolution
// must fall through to GOPATH/bin without choking on `go env`'s
// empty-line-for-unset-variable shape. Mirrors the GOBIN-set test
// but with AIWF_TEST_GOBIN cleared and the install copying into
// $GOPATH/bin.
func TestRunUpgrade_FullFlow_GOBINUnset(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell shim assumes a POSIX-y env")
	}
	tmp := t.TempDir()

	logPath := filepath.Join(tmp, "go.log")
	gopathBin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(gopathBin, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := writeUpgradeShim(t, tmp, logPath)

	t.Setenv("AIWF_GO_BIN", shim)
	t.Setenv("GOPROXY", "off")
	t.Setenv("AIWF_NO_REEXEC", "1")
	t.Setenv("AIWF_TEST_GOBIN", "")
	t.Setenv("AIWF_TEST_GOPATH", tmp)
	t.Setenv("AIWF_TEST_INSTALL_DIR", gopathBin)

	rc, stdout, stderr := captureRun(t, func() int {
		return run([]string{"upgrade", "--version", "v0.1.0", "--root", tmp})
	})
	if rc != exitOK {
		t.Fatalf("rc = %d, want %d (stdout=%s, stderr=%s)", rc, exitOK, stdout, stderr)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading shim log: %v", err)
	}
	log := string(logBytes)
	if !strings.Contains(log, "env GOBIN") {
		t.Errorf("shim log missing env GOBIN query:\n%s", log)
	}
	if !strings.Contains(log, "env GOPATH") {
		t.Errorf("expected fallback to env GOPATH when GOBIN unset:\n%s", log)
	}
	if !strings.Contains(stdout, "AIWF_NO_REEXEC set") {
		t.Errorf("expected NO_REEXEC notice; stdout:\n%s", stdout)
	}
}

// TestGoBinDir_Matrix exercises the four GOBIN/GOPATH shape
// combinations `go env` can produce, driven through the same shim
// used by the integration tests. The "gobin empty, gopath set" row
// is the case G39 was filed for.
func TestGoBinDir_Matrix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell shim assumes a POSIX-y env")
	}
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "go.log")
	shim := writeUpgradeShim(t, tmp, logPath)
	t.Setenv("AIWF_GO_BIN", shim)

	cases := []struct {
		name      string
		gobin     string
		gopath    string
		want      string
		wantErr   bool
		errSubstr string
	}{
		{name: "gobin set", gobin: "/custom/bin", gopath: "/home/u/go", want: "/custom/bin"},
		{name: "gobin empty, gopath set", gobin: "", gopath: "/home/u/go", want: "/home/u/go/bin"},
		{name: "both set, gobin wins", gobin: "/from/gobin", gopath: "/home/u/go", want: "/from/gobin"},
		{name: "both empty", gobin: "", gopath: "", wantErr: true, errSubstr: "GOPATH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AIWF_TEST_GOBIN", tc.gobin)
			t.Setenv("AIWF_TEST_GOPATH", tc.gopath)

			got, err := goBinDir(context.Background())
			switch {
			case tc.wantErr && err == nil:
				t.Fatalf("got = %q, want error containing %q", got, tc.errSubstr)
			case tc.wantErr && !strings.Contains(err.Error(), tc.errSubstr):
				t.Errorf("err = %v, want substring %q", err, tc.errSubstr)
			case !tc.wantErr && err != nil:
				t.Fatalf("unexpected err: %v", err)
			case !tc.wantErr && got != tc.want:
				t.Errorf("got = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestInstallLocationHint covers the env-var precedence the hint
// helper uses to suggest a recovery path when goBinDir resolution
// fails. The home-fallback branch is exercised through GOPATH=""
// with HOME set.
func TestInstallLocationHint(t *testing.T) {
	const pkg = "github.com/23min/aiwf/cmd/aiwf"

	cases := []struct {
		name   string
		gobin  string
		gopath string
		home   string
		want   string
	}{
		{"gobin", "/custom/bin", "/home/u/go", "/home/u", "/custom/bin/aiwf"},
		{"gopath only", "", "/home/u/go", "/home/u", "/home/u/go/bin/aiwf"},
		{"home fallback", "", "", "/home/u", "/home/u/go/bin/aiwf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOBIN", tc.gobin)
			t.Setenv("GOPATH", tc.gopath)
			t.Setenv("HOME", tc.home)
			got := installLocationHint(pkg)
			if got != tc.want {
				t.Errorf("installLocationHint = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPathChangedFromStderr (G46): the regex correctly identifies
// the Go toolchain's "module found but does not contain package"
// failure and captures the missing subpath. Spec source: this is
// the exact error string emitted by `cmd/go/internal/modload/import.go`
// when a tagged release no longer contains the requested package
// at the requested subpath.
func TestPathChangedFromStderr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		stderr string
		want   string
		ok     bool
	}{
		{
			name:   "v0.4.0 reorg case (real reproducer)",
			stderr: "go: github.com/23min/aiwf/tools/cmd/aiwf@latest: module github.com/23min/aiwf@latest found (v0.4.0), but does not contain package github.com/23min/aiwf/tools/cmd/aiwf",
			want:   "github.com/23min/aiwf/tools/cmd/aiwf",
			ok:     true,
		},
		{
			name:   "explicit version (not @latest)",
			stderr: "go: x/y/cmd/foo@v1.2.3: module x/y@v1.2.3 found (v1.2.3), but does not contain package x/y/cmd/foo",
			want:   "x/y/cmd/foo",
			ok:     true,
		},
		{
			name:   "ambient noise around the marker line",
			stderr: "go: downloading thing\nsome other line\ngo: pkg@latest: module mod found (v0.1.0), but does not contain package pkg/missing/sub\ntrailing\n",
			want:   "pkg/missing/sub",
			ok:     true,
		},
		{
			name:   "unrelated install failure (network error)",
			stderr: "go: github.com/owner/repo@latest: reading https://proxy.golang.org/...: 502 Bad Gateway",
			ok:     false,
		},
		{
			name:   "tag does not exist (different shape)",
			stderr: "go: github.com/owner/repo@v9.9.9: invalid version: unknown revision v9.9.9",
			ok:     false,
		},
		{
			name:   "empty stderr",
			stderr: "",
			ok:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := pathChangedFromStderr(tc.stderr)
			if ok != tc.ok {
				t.Errorf("ok = %v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("missing pkg = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRunUpgrade_PathChangedHint (G46): when `go install` fails with
// the path-change signature, runUpgrade prints the structured
// remediation pointing at CHANGELOG and the manual re-install
// command. Drives the verb against a shim that emits the real Go
// toolchain error.
func TestRunUpgrade_PathChangedHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell shim assumes a POSIX-y env")
	}
	tmp := t.TempDir()

	shim := filepath.Join(tmp, "go")
	body := `#!/bin/sh
case "$1" in
  install)
    echo "go: github.com/23min/aiwf/tools/cmd/aiwf@latest: module github.com/23min/aiwf@latest found (v0.4.0), but does not contain package github.com/23min/aiwf/tools/cmd/aiwf" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(shim, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AIWF_GO_BIN", shim)
	t.Setenv("GOPROXY", "off")

	rc, _, stderr := captureRun(t, func() int {
		return run([]string{"upgrade", "--version", "v0.4.0", "--root", tmp})
	})
	if rc != exitInternal {
		t.Errorf("rc = %d, want %d (install fails)", rc, exitInternal)
	}
	for _, want := range []string{
		"hint — the install path may have changed",
		"CHANGELOG.md",
		"go install <new-path>@v0.4.0",
		"aiwf update",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q\nfull stderr:\n%s", want, stderr)
		}
	}
}

// TestRunUpgrade_UnrelatedInstallError_NoHint (G46): a `go install`
// failure that doesn't match the path-change signature must NOT
// surface the G46 hint — false positives would confuse users about
// unrelated network/permission/version errors.
func TestRunUpgrade_UnrelatedInstallError_NoHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell shim assumes a POSIX-y env")
	}
	tmp := t.TempDir()

	shim := filepath.Join(tmp, "go")
	body := `#!/bin/sh
case "$1" in
  install)
    echo "go: github.com/owner/repo@v9.9.9: invalid version: unknown revision v9.9.9" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(shim, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AIWF_GO_BIN", shim)
	t.Setenv("GOPROXY", "off")

	rc, _, stderr := captureRun(t, func() int {
		return run([]string{"upgrade", "--version", "v9.9.9", "--root", tmp})
	})
	if rc != exitInternal {
		t.Errorf("rc = %d, want %d", rc, exitInternal)
	}
	if strings.Contains(stderr, "hint — the install path may have changed") {
		t.Errorf("G46 hint fired on unrelated error (false positive); stderr:\n%s", stderr)
	}
}

// writeUpgradeShim writes a shell shim that fakes `go install` and
// `go env` for upgrade tests. Behavior is parameterized at runtime
// via env vars set by the test:
//
//   - AIWF_TEST_GOBIN: value the shim returns for `go env GOBIN`.
//     Set to the empty string to simulate an unset GOBIN (the
//     shim's output is then a single newline, matching real Go).
//   - AIWF_TEST_GOPATH: value the shim returns for `go env GOPATH`.
//   - AIWF_TEST_INSTALL_DIR: directory `go install` copies the test
//     binary into. Tests must align this with whichever directory
//     goBinDir would resolve to under the chosen GOBIN/GOPATH.
//
// Each invocation is appended to logPath so callers can assert on
// the exact subcommands the upgrade flow issued.
func writeUpgradeShim(t *testing.T, dir, logPath string) string {
	t.Helper()
	shim := filepath.Join(dir, "go")
	body := `#!/bin/sh
echo "$@" >> "` + logPath + `"
case "$1" in
  env)
    case "$2" in
      GOBIN)  printf '%s\n' "$AIWF_TEST_GOBIN"  ;;
      GOPATH) printf '%s\n' "$AIWF_TEST_GOPATH" ;;
    esac
    ;;
  install)
    name=$(echo "$2" | sed 's|.*/||; s|@.*||')
    cp "` + os.Args[0] + `" "$AIWF_TEST_INSTALL_DIR/$name"
    ;;
esac
`
	if err := os.WriteFile(shim, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return shim
}

// TestRunUpgrade_BadFlag covers the usage-error path.
func TestRunUpgrade_BadFlag(t *testing.T) {
	rc, _, _ := captureRun(t, func() int {
		return run([]string{"upgrade", "--nope"})
	})
	if rc != exitUsage {
		t.Errorf("rc = %d, want %d", rc, exitUsage)
	}
}

// TestRenderVersionLabel covers the label-format edge cases.
func TestRenderVersionLabel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ver  string
		want string
	}{
		{"tagged", "v0.1.0", "v0.1.0 (tagged)"},
		{"devel", "(devel)", "(devel) (working-tree build)"},
		{"dirty tagged", "v0.1.0+dirty", "v0.1.0+dirty (working-tree build)"},
		{"dirty pseudo", "v0.0.0-20060102150405-abcdef123456+dirty", "v0.0.0-20060102150405-abcdef123456+dirty (working-tree build)"},
		{"plain pseudo", "v0.0.0-20060102150405-abcdef123456", "v0.0.0-20060102150405-abcdef123456 (pseudo-version)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderVersionLabel(version.Parse(tc.ver))
			if got != tc.want {
				t.Errorf("renderVersionLabel(%q) = %q, want %q", tc.ver, got, tc.want)
			}
		})
	}
}

// captureRun redirects os.Stdout and os.Stderr around fn, returning
// the exit code and captured streams.
func captureRun(t *testing.T, fn func() int) (rc int, stdout, stderr string) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	or, ow, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	er, ew, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = ow, ew
	defer func() {
		os.Stdout, os.Stderr = origOut, origErr
	}()

	rc = fn()

	_ = ow.Close()
	_ = ew.Close()
	o, _ := io.ReadAll(or)
	e, _ := io.ReadAll(er)
	return rc, string(o), string(e)
}
