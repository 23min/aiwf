package cliutil

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// readOneDiagRecord reads the single JSON diagnostic record the finish
// closure wrote to an explicit AIWF_LOG_FILE destination.
func readOneDiagRecord(t *testing.T, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var rec map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &rec); err != nil {
		t.Fatalf("record %q did not parse as JSON: %v", content, err)
	}
	return rec
}

// enabledDiagEnv is the fake environment that turns the diagnostic
// logger on and points it at a JSON file the test can read back.
func enabledDiagEnv(path string) func(string) string {
	return fakeGetenv(map[string]string{
		"AIWF_LOG":        "info",
		"AIWF_LOG_FILE":   path,
		"AIWF_LOG_FORMAT": "json",
	})
}

// staticActor is the actor-provider a mutating verb passes: the actor
// was already resolved in the prelude, so the provider just returns it.
func staticActor(actor string) func() string {
	return func() string { return actor }
}

func TestBeginVerbDiagCore_EnabledEmitsCompletedWithFinalCodeAndSHA(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	finish := beginVerbDiagCore(t.TempDir(), enabledDiagEnv(path),
		"promote", "M-0001", staticActor("human/peter"), "corr-123")

	// Pointer capture: the verb assigns code/sha mid-body, after the
	// defer was registered. finish dereferences the pointers at call
	// time, so it reports the final assigned values.
	code := ExitOK
	sha := "deadbeef"
	finish(&code, &sha)

	rec := readOneDiagRecord(t, path)
	for field, want := range map[string]string{
		"msg":    "verb.completed",
		"sha":    "deadbeef",
		"verb":   "promote",
		"entity": "M-0001",
		"actor":  "human/peter",
		"run_id": "corr-123",
	} {
		if rec[field] != want {
			t.Errorf("record[%q] = %v, want %q", field, rec[field], want)
		}
	}
}

func TestBeginVerbDiagCore_EmptyCorrelationID_GeneratesRunID(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	finish := beginVerbDiagCore(t.TempDir(), enabledDiagEnv(path),
		"promote", "M-0001", staticActor("human/peter"), "")

	code := ExitOK
	sha := ""
	finish(&code, &sha)

	rec := readOneDiagRecord(t, path)
	runID, ok := rec["run_id"].(string)
	if !ok || len(runID) != 16 {
		// logger.NewRunID() is 16 hex chars from 8 random bytes.
		t.Errorf("run_id = %v (len %d), want a 16-char generated id", rec["run_id"], len(runID))
	}
}

func TestBeginVerbDiagCore_NonOK_EmitsFailedWithErrorClass(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	finish := beginVerbDiagCore(t.TempDir(), enabledDiagEnv(path),
		"promote", "M-0001", staticActor("human/peter"), "corr-123")

	code := ExitInternal
	sha := ""
	finish(&code, &sha)

	rec := readOneDiagRecord(t, path)
	if rec["msg"] != "verb.failed" {
		t.Errorf("msg = %v, want %q", rec["msg"], "verb.failed")
	}
	if rec["error_class"] != "internal" {
		t.Errorf("error_class = %v, want %q", rec["error_class"], "internal")
	}
	if got, ok := rec["exit_code"].(float64); !ok || int(got) != ExitInternal {
		t.Errorf("exit_code = %v, want %d", rec["exit_code"], ExitInternal)
	}
}

func TestBeginVerbDiagCore_Enabled_ResolvesActorExactlyOnce(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	calls := 0
	resolveActor := func() string { calls++; return "human/lazy" }
	finish := beginVerbDiagCore(t.TempDir(), enabledDiagEnv(path),
		"show", "M-0001", resolveActor, "corr-123")

	code := ExitOK
	sha := ""
	finish(&code, &sha)

	if calls != 1 {
		t.Errorf("resolveActor called %d times, want exactly 1", calls)
	}
	if rec := readOneDiagRecord(t, path); rec["actor"] != "human/lazy" {
		t.Errorf("actor = %v, want %q", rec["actor"], "human/lazy")
	}
}

func TestBeginVerbDiagCore_Disabled_DoesNotResolveActorNoOutputNoPanic(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	called := false
	// fakeGetenv(nil): AIWF_LOG unset → the logger resolves to disabled,
	// exercising the false arm of the Enabled guard. The lazy actor
	// resolver must NOT run — that is the exec-avoidance the read-verb
	// variant preserves.
	resolveActor := func() string { called = true; return "human/never" }
	finish := beginVerbDiagCore(t.TempDir(), fakeGetenv(nil),
		"show", "M-0001", resolveActor, "corr-123")

	code := ExitOK
	sha := "deadbeef"
	finish(&code, &sha) // must not panic

	if called {
		t.Error("resolveActor ran while logging disabled; the lazy-actor exec-avoidance is broken")
	}
	if content, err := os.ReadFile(path); err == nil && len(content) > 0 {
		t.Errorf("disabled logger wrote %q, want no output", content)
	}
}

// TestBestEffortActor_DerivesFromGitConfig and _MissingReturnsEmpty are
// serial: they t.Setenv the git-locator env vars (HOME/XDG_CONFIG_HOME/
// GIT_CONFIG_NOSYSTEM), so they must not run under t.Parallel.

func TestBestEffortActor_DerivesFromGitConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"),
		[]byte("[user]\n\temail = reader@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := bestEffortActor(t.TempDir()); got != "human/reader" {
		t.Errorf("bestEffortActor = %q, want human/reader", got)
	}
}

func TestBestEffortActor_MissingIdentityReturnsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// No .gitconfig written: git config user.email finds nothing, so
	// ResolveActor errors and bestEffortActor collapses it to "".
	if got := bestEffortActor(t.TempDir()); got != "" {
		t.Errorf("bestEffortActor = %q, want %q on missing identity", got, "")
	}
}
