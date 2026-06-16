package testsupport

import (
	"os"
	"strconv"
	"testing"
)

// TestHardenGitTestEnv asserts both effects: every git locator var is
// unset, and the gc-disabling GIT_CONFIG_* vars are exported. Serial
// (it mutates process env; no t.Parallel, and t.Setenv would panic
// under it anyway). The GIT_CONFIG_* vars are not registered with
// t.Setenv, so a cleanup restores the process env for any later test
// in this package.
func TestHardenGitTestEnv(t *testing.T) {
	for _, v := range gitLocatorEnvVars {
		t.Setenv(v, "/tmp/should-be-scrubbed")
	}
	t.Cleanup(func() {
		os.Unsetenv("GIT_CONFIG_COUNT")
		for i := range gitTestConfig {
			os.Unsetenv("GIT_CONFIG_KEY_" + strconv.Itoa(i))
			os.Unsetenv("GIT_CONFIG_VALUE_" + strconv.Itoa(i))
		}
	})

	HardenGitTestEnv()

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
