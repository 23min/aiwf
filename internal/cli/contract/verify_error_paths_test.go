package contract_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/contract"
)

// M-0254/AC-1 backfill: Run's ResolveRoot and tree.Load guards, plus
// its two stdout-write guards, are `//coverage:ignore`d in verify.go
// itself. The remaining flagged branch — the LoadContractsBlock guard
// — gets a real test below, reusing the malformed-contracts-block
// trigger already proven at
// internal/cli/add/add_error_paths_test.go.
func TestRun_LoadContractsBlockFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("contracts:\n  bindings:\n    - not a valid binding\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	rc := contract.Run(root, "text", false, "")
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}
