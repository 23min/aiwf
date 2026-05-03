package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/version"
)

// TestRunUpgrade_CheckOnly_NoNetworkRequired covers the
// `aiwf upgrade --check` path: with GOPROXY=off, the verb falls back
// to "proxy disabled" messaging, prints the current version, and
// exits 0 without touching disk.
func TestRunUpgrade_CheckOnly_NoNetworkRequired(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	rc, stdout, _ := captureRun(t, func() int {
		return runUpgrade([]string{"--check"})
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
		return runUpgrade([]string{"--check"})
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
		return runUpgrade([]string{"--version", "v0.1.0"})
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
// invocation; we assert we hit `install <pkg>@v0.1.0` and
// `env GOBIN GOPATH`.
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
	shim := filepath.Join(tmp, "go")
	shimBody := `#!/bin/sh
echo "$@" >> "` + logPath + `"
case "$1" in
  env)
    echo "` + gobinDir + `"
    echo "` + tmp + `"
    ;;
  install)
    name=$(echo "$2" | sed 's|.*/||; s|@.*||')
    cp "` + os.Args[0] + `" "` + gobinDir + `/$name"
    ;;
esac
`
	if err := os.WriteFile(shim, []byte(shimBody), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AIWF_GO_BIN", shim)
	t.Setenv("GOPROXY", "off")
	t.Setenv("AIWF_NO_REEXEC", "1")

	rc, stdout, stderr := captureRun(t, func() int {
		return runUpgrade([]string{"--version", "v0.1.0", "--root", tmp})
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
	if !strings.Contains(log, "env GOBIN GOPATH") {
		t.Errorf("shim log missing env query:\n%s", log)
	}
	if !strings.Contains(stdout, "AIWF_NO_REEXEC set") {
		t.Errorf("expected NO_REEXEC notice; stdout:\n%s", stdout)
	}
}

// TestRunUpgrade_BadFlag covers the usage-error path.
func TestRunUpgrade_BadFlag(t *testing.T) {
	rc, _, _ := captureRun(t, func() int {
		return runUpgrade([]string{"--nope"})
	})
	if rc != exitUsage {
		t.Errorf("rc = %d, want %d", rc, exitUsage)
	}
}

// TestRenderVersionLabel covers the label-format edge cases.
func TestRenderVersionLabel(t *testing.T) {
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
