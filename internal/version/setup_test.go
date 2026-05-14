package version

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the
// test binary, so once-setup is correct.
//
// Serial tests: every test that calls t.Setenv("GOPROXY", ...) stays
// serial — t.Setenv panics under t.Parallel. These are the proxy-
// resolution tests that need to point Latest at a fake httptest
// server or assert GOPROXY=off behavior:
//   - TestProxyBase (mutates GOPROXY across table cases)
//   - TestLatest_Happy
//   - TestLatest_FallsBackToAtLatest
//   - TestLatest_ProxyError
//   - TestLatest_GoproxyOff
//   - TestLatest_ContextTimeout
//   - TestLatest_Wrapper
//   - TestLatest_RealProxy_ContractTest
//   - TestLatest_PrereleaseExcludedFromHighestSelection
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
