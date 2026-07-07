package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestUpgradeDiag_EmitsVerbCompletedEvent pins M-0238/AC-1+AC-2 for
// `aiwf upgrade`: a successful install (stopped short of the
// process-replacing re-exec via AIWF_NO_REEXEC) fires a
// "verb.completed" event with the installed version bound as entity.
// upgrade has no --actor flag, so actor is bound empty.
func TestUpgradeDiag_EmitsVerbCompletedEvent(t *testing.T) {
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

	diagLogPath := filepath.Join(tmp, "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", diagLogPath)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"upgrade", "--version", "v0.1.0", "--root", tmp})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("rc = %d, want %d (stdout=%s, stderr=%s)", rc, cliutil.ExitOK, stdout, stderr)
	}

	raw, err := os.ReadFile(diagLogPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		Msg    string `json:"msg"`
		Verb   string `json:"verb"`
		Entity string `json:"entity"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "upgrade" {
		t.Errorf("verb = %q, want %q", rec.Verb, "upgrade")
	}
	if rec.Entity != "v0.1.0" {
		t.Errorf("entity = %q, want %q", rec.Entity, "v0.1.0")
	}
}

// TestUpgradeDiag_FailedInstallEmitsNoEvent: a failed `go install`
// must not emit "verb.completed" even with AIWF_LOG=info set.
func TestUpgradeDiag_FailedInstallEmitsNoEvent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PATH", "")
	t.Setenv("AIWF_GO_BIN", "")
	t.Setenv("GOPROXY", "off")
	t.Setenv("AIWF_NO_REEXEC", "1")

	diagLogPath := filepath.Join(tmp, "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", diagLogPath)

	rc, _, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"upgrade", "--version", "v0.1.0"})
	})
	if rc == cliutil.ExitOK {
		t.Fatalf("expected non-zero exit when go binary is missing")
	}

	if raw, err := os.ReadFile(diagLogPath); err == nil {
		t.Errorf("diagnostic log %q written despite a failed install", raw)
	} else if !os.IsNotExist(err) {
		t.Errorf("reading diagnostic log: %v", err)
	}
}

// TestUpgradeDiag_DisabledByDefault_NoLogFileCreated pins the
// default-off half: without AIWF_LOG set, `aiwf upgrade` creates no
// diagnostic log.
func TestUpgradeDiag_DisabledByDefault_NoLogFileCreated(t *testing.T) {
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

	diagLogPath := filepath.Join(tmp, "diag.log")
	t.Setenv("AIWF_LOG", "")
	t.Setenv("AIWF_LOG_FILE", diagLogPath)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"upgrade", "--version", "v0.1.0", "--root", tmp})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("rc = %d, want %d (stdout=%s, stderr=%s)", rc, cliutil.ExitOK, stdout, stderr)
	}
	if _, err := os.Stat(diagLogPath); !os.IsNotExist(err) {
		t.Errorf("diagnostic log file exists at %s despite AIWF_LOG unset (err=%v)", diagLogPath, err)
	}
}
