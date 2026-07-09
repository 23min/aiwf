// Command lockholder acquires internal/repolock's exclusive lock on
// the repo root passed as its only argument, prints "ACQUIRED" to
// signal readiness, then blocks until killed. It exists purely so
// M-0242/AC-1's scenario can hold the lock from a real, independently
// killable OS process — internal/repolock itself is never modified to
// support this.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/23min/aiwf/internal/repolock"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin))
}

func run(args []string, stdout, stderr, stdin *os.File) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: lockholder <repo-root>")
		return 2
	}
	lock, err := repolock.Acquire(args[0], 5*time.Second)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "acquire: %v\n", err)
		return 1
	}
	defer func() { _ = lock.Release() }()

	_, _ = fmt.Fprintln(stdout, "ACQUIRED")
	// Block until the scenario kills this process — SIGKILL bypasses
	// this read entirely, and the deferred Release above, which is
	// exactly the kernel fd-cleanup property AC-1 tests — or the
	// scenario closes stdin.
	buf := make([]byte, 1)
	_, _ = stdin.Read(buf)
	return 0
}
