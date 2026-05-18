// Command aiwf is the ai-workflow framework's single binary.
//
// The binary's entry point is intentionally minimal — main() defers
// every concern (OS preflight, dispatch, error rendering) to
// internal/cli.Execute. The Cobra command tree, version helpers, and
// help text all live in internal/cli/; per-verb implementations live
// in internal/cli/<verb>/.
//
// The two thin wrappers below — run and newRootCmd — exist solely so
// the M-0118-era cmd/aiwf/*_test.go files keep compiling. M-0118/AC-6
// relocates those tests to internal/cli/integration/ where they call
// cli.Execute / cli.NewRootCmd directly; M-0118/AC-5 then deletes
// these wrappers and main.go is just `func main`.
package main

import (
	"os"

	"github.com/23min/aiwf/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}

// run is a test-compat shim around cli.Execute. Removed at AC-5 once
// AC-6 has relocated every cmd/aiwf-side test off it.
func run(args []string) int {
	return cli.Execute(args)
}

// newRootCmd is a test-compat shim around cli.NewRootCmd. Removed at
// AC-5 once AC-6 has relocated every cmd/aiwf-side test off it.
func newRootCmd() *cobra.Command {
	return cli.NewRootCmd()
}
