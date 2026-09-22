package testsupport

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestHardenGitTestEnv checks locator isolation and Git configuration.
// Serial: mutates process environment; t.Setenv restores it before parallel tests.
func TestHardenGitTestEnv(t *testing.T) {
	for _, v := range gitLocatorEnvVars {
		t.Setenv(v, "/tmp/should-be-scrubbed")
	}
	t.Setenv("GIT_CONFIG_COUNT", os.Getenv("GIT_CONFIG_COUNT"))
	for i := range gitTestConfig {
		for _, prefix := range []string{"GIT_CONFIG_KEY_", "GIT_CONFIG_VALUE_"} {
			name := prefix + strconv.Itoa(i)
			t.Setenv(name, os.Getenv(name))
		}
	}

	// A helper stand-in keeps the bypass probe offline even if hardening fails.
	t.Setenv("GIT_ALLOW_PROTOCOL", "https")
	helpers := t.TempDir()
	if err := WriteExecutable(filepath.Join(helpers, "git-remote-https"), []byte("#!/bin/sh\necho REMOTE_HELPER_REACHED >&2\nexit 1\n")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_EXEC_PATH", helpers)
	HardenGitTestEnv()
	output, err := exec.CommandContext(t.Context(), "git", "ls-remote", "https://example.invalid/repo").CombinedOutput()
	if err == nil || !strings.Contains(string(output), "transport 'https' not allowed") {
		t.Errorf("inherited transport override bypassed restriction: %s (%v)", output, err)
	}

	for _, v := range gitLocatorEnvVars {
		if got, ok := os.LookupEnv(v); ok {
			t.Errorf("%s still set after HardenGitTestEnv: %q", v, got)
		}
	}

	if got, want := os.Getenv("GIT_CONFIG_COUNT"), strconv.Itoa(len(gitTestConfig)); got != want {
		t.Errorf("GIT_CONFIG_COUNT = %q, want %q", got, want)
	}
	for i, kv := range gitTestConfig {
		gotKey := os.Getenv("GIT_CONFIG_KEY_" + strconv.Itoa(i))
		gotVal := os.Getenv("GIT_CONFIG_VALUE_" + strconv.Itoa(i))
		if gotKey != kv[0] || gotVal != kv[1] {
			t.Errorf("GIT_CONFIG_[%d] = (%q, %q), want (%q, %q)", i, gotKey, gotVal, kv[0], kv[1])
		}
	}
}

func TestGitTestEnvironment_OnlyLocalTransport(t *testing.T) {
	t.Parallel()
	source := GuidanceSource(t)
	for _, tc := range []struct {
		source string
		denied bool
	}{
		{source, false}, {"https://example.invalid/guidance.git", true},
	} {
		output, err := exec.CommandContext(t.Context(), "git", "ls-remote", tc.source).CombinedOutput()
		if tc.denied {
			if err == nil || !strings.Contains(string(output), "transport 'https' not allowed") {
				t.Fatalf("network transport not refused: %s (%v)", output, err)
			}
		} else if err != nil || !strings.Contains(string(output), "refs/heads/main") {
			t.Fatalf("local transport failed: %s (%v)", output, err)
		}
	}
}
