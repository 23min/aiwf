//go:build windows

package gitops

import "os/exec"

// The CLI refuses unsupported Windows execution; retain cross-compilation.
func cancelCloneGroup(_ *exec.Cmd) {}
