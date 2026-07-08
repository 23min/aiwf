// Command stresstest is the on-demand correctness stress harness for
// aiwf (E-0062). It is dev-only tooling — never installed alongside
// cmd/aiwf — built and invoked by hand (see `make stress`).
package main

import "os"

func main() {
	os.Exit(run(os.Args[1:]))
}

// run builds the stresstest command tree and executes it against
// args, returning a process exit code. Kept separate from main() so
// it's directly testable without spawning a subprocess.
func run(args []string) int {
	cmd := newRootCmd()
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		return 1
	}
	return 0
}
