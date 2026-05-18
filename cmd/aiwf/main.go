// Command aiwf is the ai-workflow framework's single binary.
//
// The binary's entry point is intentionally minimal — main() defers
// every concern (OS preflight, dispatch, error rendering) to
// internal/cli.Execute. The Cobra command tree, version helpers, and
// help text all live in internal/cli/; per-verb implementations live
// in internal/cli/<verb>/. Integration tests for the dispatcher live
// at internal/cli/integration/.
//
// G-0107 fully closed at M-0118/AC-5: cmd/aiwf/ contains main.go only.
package main

import (
	"os"

	"github.com/23min/aiwf/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
