package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestRunCheck_LegacyStringFormNoPathFindings pins M-0180/AC-5 end-to-end: an
// E-0043 legacy string-form areas block (members declared as bare strings, no
// paths) flows through config.Load → the AreaPaths projection → the path-axis
// checks and produces NO area-dead-glob / area-overlap findings. Backward
// compatibility: a label-only config keeps validating as it did before the
// path-axis checks landed. This is the seam the unit inertness test cannot
// reach — it exercises the real config decode and projection, not a
// hand-built []AreaPaths.
func TestRunCheck_LegacyStringFormNoPathFindings(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n    - app-a\n    - app-b\n"
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		_ = cli.Execute([]string{"check", "--root", root})
	})
	out := string(captured)
	for _, code := range []string{"area-dead-glob", "area-overlap"} {
		if strings.Contains(out, code) {
			t.Errorf("legacy string-form config must produce no %s finding; got:\n%s", code, out)
		}
	}
}
