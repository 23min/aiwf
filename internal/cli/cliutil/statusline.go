package cliutil

import (
	"fmt"
	"os"

	"github.com/23min/aiwf/internal/skills"
)

// RunStatuslineScaffold invokes the shared scaffold-if-absent helper
// in skills/ and prints an operator-readable summary of what happened.
// Shared between `aiwf init --statusline` and `aiwf update --statusline`
// (M-0155) so the two entry points behave identically — same
// destination resolution, same scaffold-if-absent semantics, same
// activation-snippet shape.
//
// Returns one of the Exit* codes. The function never writes to a
// settings file — settings wiring is M-0156's responsibility, gated by
// explicit per-invocation consent per ADR-0015.
func RunStatuslineScaffold(rootDir, scope string) int {
	res, err := skills.ScaffoldStatusline(rootDir, skills.StatuslineScope(scope))
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf --statusline: %v\n", err)
		return ExitUsage
	}
	if res.Wrote {
		fmt.Printf("\naiwf --statusline: wrote %s\n", res.Path)
	} else {
		fmt.Printf("\naiwf --statusline: %s already exists, left untouched\n", res.Path)
	}
	if res.GitignoreAppended {
		fmt.Println("aiwf --statusline: appended `.claude/statusline.sh` to .gitignore")
	}
	fmt.Println("\nTo activate, add this to your Claude Code settings file")
	fmt.Println("(see ADR-0015 for the consent flow; M-0156 will wire this automatically):")
	fmt.Println()
	fmt.Println(res.Snippet)
	return ExitOK
}
