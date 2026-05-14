package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/initrepo"
)

// newUpdateCmd builds `aiwf update`: refreshes every marker-managed
// framework artifact the consumer is opted into. The pipeline is the
// same one `aiwf init` runs after first-time scaffolding —
// `initrepo.RefreshArtifacts` — so init and update converge to the
// same state for a given binary version + aiwf.yaml.
//
// Concretely the verb refreshes:
//   - the embedded skills under .claude/skills/aiwf-*
//   - the .gitignore patterns covering them
//   - the marker-managed pre-push hook
//   - the marker-managed pre-commit hook (gated by
//     aiwf.yaml's status_md.auto_update; default-on)
//
// Hook conflicts (a non-marker hook already in place) are reported
// in the per-step ledger and surface a remediation block, mirroring
// `aiwf init`'s conflict path.
func newUpdateCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Refresh marker-managed framework artifacts (skills, hooks)",
		Example: `  # Refresh skills + hooks against the current binary version
  aiwf update`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runUpdateCmd(root))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	return cmd
}

func runUpdateCmd(root string) int {
	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf update")
	if release == nil {
		return rc
	}
	defer release()

	cfg, err := config.Load(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return exitInternal
	}

	steps, conflict, err := initrepo.RefreshArtifacts(context.Background(), rootDir, initrepo.RefreshOptions{
		StatusMdAutoUpdate: cfg.StatusMdAutoUpdate(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return exitInternal
	}

	for _, s := range steps {
		if s.Detail != "" {
			fmt.Printf("  %-9s  %s  (%s)\n", s.Action, s.What, s.Detail)
		} else {
			fmt.Printf("  %-9s  %s\n", s.Action, s.What)
		}
	}

	if conflict {
		fmt.Println()
		fmt.Println("aiwf update: hook chain collision (G45).")
		fmt.Println("A non-aiwf hook would auto-migrate to its `.local` sibling, but a `.local`")
		fmt.Println("file already exists at .git/hooks/pre-push.local or .git/hooks/pre-commit.local.")
		fmt.Println("Resolve manually: merge the existing hook's content into the `.local` file,")
		fmt.Println("delete the original (non-`.local`) hook, and re-run `aiwf update`.")
		return exitFindings
	}

	fmt.Println("\naiwf update: done.")
	return exitOK
}
