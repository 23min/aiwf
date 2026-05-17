package cliutil_test

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
)

// TestCompletionHelpers_AreExported pins M-0114/AC-1: the six completion
// helpers and ResolveRoot live in cliutil as exported symbols, callable
// from any caller package. Compiles only after the move; the test does
// not exercise behavior in depth — that is covered by the package-local
// tests that travel with the helpers — it just pins the export shape.
func TestCompletionHelpers_AreExported(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "fake"}
	cmd.Flags().String("format", "text", "")
	cliutil.RegisterFormatCompletion(cmd)

	if names := cliutil.AllKindNames(); len(names) == 0 {
		t.Errorf("AllKindNames returned empty list")
	}
	if statuses := cliutil.StatusesForID("E-0001"); len(statuses) == 0 {
		t.Errorf("StatusesForID(E-0001) returned empty list")
	}
	if _, dir := cliutil.CompleteEntityIDs(""); dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("CompleteEntityIDs directive = %v, want NoFileComp", dir)
	}
	if fn := cliutil.CompleteEntityIDFlag(entity.KindEpic); fn == nil {
		t.Errorf("CompleteEntityIDFlag returned nil")
	}
	if fn := cliutil.CompleteEntityIDArg(entity.KindEpic, 0); fn == nil {
		t.Errorf("CompleteEntityIDArg returned nil")
	}
	if _, err := cliutil.ResolveRoot(t.TempDir()); err != nil {
		t.Errorf("ResolveRoot returned error: %v", err)
	}
}
