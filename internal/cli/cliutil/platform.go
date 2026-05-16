package cliutil

import "fmt"

// AssertSupportedOS gates aiwf to platforms whose POSIX assumptions
// the engine relies on. The PoC is Unix-only by design (per
// docs/pocv3/design/design-decisions.md): the pre-push hook is a /bin/sh
// script, contract validators shell out via POSIX semantics, and the
// repo lock uses flock(2). On Windows the engine would fail at the
// first such call with an opaque error; a clear up-front refusal is
// kinder to the user than a confusing failure deeper in the stack.
//
// Pass runtime.GOOS in main; tests pass canned values to exercise
// every branch without depending on the test runner's host OS.
func AssertSupportedOS(goos string) error {
	if goos == "windows" {
		return fmt.Errorf("aiwf is not supported on Windows in the PoC (POSIX-only assumptions: /bin/sh hook, exec.LookPath validators, flock(2) repo lock). See docs/pocv3/design/design-decisions.md and the README's Known Limitations section")
	}
	return nil
}
