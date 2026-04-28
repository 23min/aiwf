package main

import (
	"flag"
	"fmt"
	"os"
)

// runWhoami handles `aiwf whoami`: prints the resolved actor for the
// current context, plus the source label that produced it. Useful to
// confirm what `aiwf-actor:` trailer the next mutating verb would write.
func runWhoami(args []string) int {
	fs := flag.NewFlagSet("whoami", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root (default: discover via aiwf.yaml)")
	actor := fs.String("actor", "", "actor override; echoes back if valid")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf whoami: %v\n", err)
		return exitUsage
	}

	resolved, source, err := resolveActorWithSource(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf whoami: %v\n", err)
		return exitFindings
	}
	fmt.Printf("%s (source: %s)\n", resolved, source)
	return exitOK
}
