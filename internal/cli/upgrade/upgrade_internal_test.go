package upgrade

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/version"
)

// TestRun_ReexecFails_StillEmittedInstallCompleted pins M-0238/AC-1's
// upgrade-specific classification decision (documented in Run's own
// comment above the diagnostic emission): "install.completed" fires
// the moment install succeeds, not on the final process exit code.
// This is deliberate, not an oversight — a successful reexec replaces
// the process via syscall.Exec, so there is no Go code path after a
// successful reexec that could ever emit a diagnostic event; gating
// on the final exit code would mean the event almost never fires in a
// real production upgrade. Swaps the unexported reexecUpdate var
// (package-internal access), so this test is serial: t.Parallel would
// race any other test that also swaps it.
func TestRun_ReexecFails_StillEmittedInstallCompleted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell shim assumes a POSIX-y env")
	}
	tmp := t.TempDir()
	gobinDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(gobinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(tmp, "go")
	shimBody := `#!/bin/sh
case "$1" in
  env)
    case "$2" in
      GOBIN)  printf '%s\n' "` + gobinDir + `" ;;
      GOPATH) printf '\n' ;;
    esac
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

	origReexec := reexecUpdate
	reexecUpdate = func(string, string) error { return errTestReexecFailure }
	t.Cleanup(func() { reexecUpdate = origReexec })

	t.Setenv("AIWF_GO_BIN", shim)
	t.Setenv("GOPROXY", "off")

	diagLogPath := filepath.Join(tmp, "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", diagLogPath)

	rc := Run(tmp, "v0.1.0", false)
	if rc != cliutil.ExitInternal {
		t.Fatalf("Run() = %d, want ExitInternal (%d) when reexec fails", rc, cliutil.ExitInternal)
	}

	raw, err := os.ReadFile(diagLogPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		Msg  string `json:"msg"`
		Verb string `json:"verb"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.Msg != "install.completed" || rec.Verb != "upgrade" {
		t.Errorf("diagnostic record = %+v, want install.completed/upgrade despite the reexec failure", rec)
	}
}

var errTestReexecFailure = errors.New("simulated reexec failure")

// TestProxyStaleHint covers the helper that emits the
// "proxy may be stale" hint when a pseudo-version's base is newer
// than the resolved latest. Closes G-0149.
//
// The positive case is unreachable through Run() under `go test`
// (runtime/debug.ReadBuildInfo() always returns "(devel)" for a test
// binary, so version.Current() never carries a real pseudo-version),
// hence this white-box test exercises the helper directly with
// synthetic Info values.
func TestProxyStaleHint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		current        string
		resolved       string
		wantHint       bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "pseudo-base ahead of target (the G-0149 case)",
			current:      "v0.8.1-0.20260516161658-02c349f629d7",
			resolved:     "v0.8.0",
			wantHint:     true,
			wantContains: []string{"hint:", "pseudo-base v0.8.1", "target v0.8.0", "GOPROXY=direct"},
		},
		{
			name:           "pseudo-base equal to target — already-aligned, no hint",
			current:        "v0.8.1-0.20260516161658-02c349f629d7",
			resolved:       "v0.8.1",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "pseudo-base behind target — real upgrade available, no hint",
			current:        "v0.8.1-0.20260516161658-02c349f629d7",
			resolved:       "v0.9.0",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "current is clean tag — no hint regardless of skew",
			current:        "v0.8.0",
			resolved:       "v0.8.1",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "current is devel — no hint",
			current:        "(devel)",
			resolved:       "v0.8.1",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "current is +dirty pseudo — no hint",
			current:        "v0.8.1-0.20260516161658-02c349f629d7+dirty",
			resolved:       "v0.8.0",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "form-1 pseudo with v0.0.0 base, target v0.1.0 — base behind, no hint",
			current:        "v0.0.0-20260503120000-abcdef123456",
			resolved:       "v0.1.0",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := proxyStaleHint(version.Parse(tc.current), version.Parse(tc.resolved))
			if tc.wantHint && got == "" {
				t.Fatalf("proxyStaleHint returned empty, want non-empty hint")
			}
			if !tc.wantHint && got != "" {
				t.Fatalf("proxyStaleHint returned %q, want empty", got)
			}
			for _, sub := range tc.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("hint missing substring %q; got:\n%s", sub, got)
				}
			}
			for _, sub := range tc.wantNotContain {
				if strings.Contains(got, sub) {
					t.Errorf("hint unexpectedly contained %q; got:\n%s", sub, got)
				}
			}
		})
	}
}

// TestProxyLookupFailedHint covers the advisory hint shown when the
// pre-flight latest-version proxy lookup fails (the v0.10.0 post-release
// `context deadline exceeded` friction). White-box: the helper is pure;
// the Run() default branch that calls it requires a proxy timeout that
// is awkward to provoke under `go test`. Asserts the three remediations
// are named (retry, pin a version, GOPROXY=direct).
func TestProxyLookupFailedHint(t *testing.T) {
	t.Parallel()
	pkg := "github.com/23min/aiwf/cmd/aiwf"
	got := proxyLookupFailedHint(pkg)
	for _, want := range []string{
		"hint:",
		pkg + "@vX.Y.Z", // pin-a-version remediation names the package
		"GOPROXY=direct",
		"retry",
		"`,direct` fallback", // explains go install may still succeed
	} {
		if !strings.Contains(got, want) {
			t.Errorf("proxyLookupFailedHint missing %q; got:\n%s", want, got)
		}
	}
}
