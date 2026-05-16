package cliutil

import (
	"strings"
	"testing"
)

// TestAssertSupportedOS_Windows is the load-bearing test for G13: a
// Windows host must produce a clear up-front error rather than a
// confusing failure deeper in the call stack.
func TestAssertSupportedOS_Windows(t *testing.T) {
	t.Parallel()
	err := AssertSupportedOS("windows")
	if err == nil {
		t.Fatal("expected error on windows")
	}
	if !strings.Contains(err.Error(), "Windows") {
		t.Errorf("error should name Windows; got %q", err)
	}
	if !strings.Contains(err.Error(), "POSIX") {
		t.Errorf("error should explain why (POSIX assumptions); got %q", err)
	}
}

// TestAssertSupportedOS_SupportedHosts: linux and darwin are the
// two supported PoC platforms and must produce no error.
func TestAssertSupportedOS_SupportedHosts(t *testing.T) {
	t.Parallel()
	for _, goos := range []string{"linux", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			if err := AssertSupportedOS(goos); err != nil {
				t.Errorf("%s should be supported; got %v", goos, err)
			}
		})
	}
}
