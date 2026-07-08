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

// TestUpgradeDiag_EmitsVerbCompletedEvent pins M-0238/AC-1+AC-2+AC-5
// for `aiwf upgrade`: a successful install (stopped short of the
// process-replacing re-exec via AIWF_NO_REEXEC) fires an
// "install.completed" event with the installed version bound as
// entity and a run_id (never a sha — upgrade produces no entity
// commit). upgrade has no --actor flag, so actor is bound empty.
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
		RunID  string `json:"run_id"`
		SHA    string `json:"sha"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.Msg != "install.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "install.completed")
	}
	if rec.Verb != "upgrade" {
		t.Errorf("verb = %q, want %q", rec.Verb, "upgrade")
	}
	if rec.Entity != "v0.1.0" {
		t.Errorf("entity = %q, want %q", rec.Entity, "v0.1.0")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
	if rec.SHA != "" {
		t.Errorf("sha = %q, want empty (upgrade produces no entity commit)", rec.SHA)
	}
}

// TestUpgradeDiag_FailedInstallEmitsFailedEvent pins M-0238/AC-6: a
// failed `go install` emits "install.failed" (not "install.completed"
// — that event never fires, since install never actually succeeded)
// carrying the exit code's error class.
func TestUpgradeDiag_FailedInstallEmitsFailedEvent(t *testing.T) {
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
		return cli.Execute([]string{"upgrade", "--version", "v0.1.0", "--root", tmp})
	})
	if rc != cliutil.ExitInternal {
		t.Fatalf("rc = %d, want ExitInternal (%d) when the go binary is missing", rc, cliutil.ExitInternal)
	}

	raw, err := os.ReadFile(diagLogPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		Msg        string `json:"msg"`
		Verb       string `json:"verb"`
		ExitCode   int    `json:"exit_code"`
		ErrorClass string `json:"error_class"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.Msg != "install.failed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "install.failed")
	}
	if rec.Verb != "upgrade" {
		t.Errorf("verb = %q, want %q", rec.Verb, "upgrade")
	}
	if rec.ExitCode != cliutil.ExitInternal {
		t.Errorf("exit_code = %d, want %d", rec.ExitCode, cliutil.ExitInternal)
	}
	if rec.ErrorClass != "internal" {
		t.Errorf("error_class = %q, want %q", rec.ErrorClass, "internal")
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
