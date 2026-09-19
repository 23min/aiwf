package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCellGenerator_UsesScriptLocation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	script, readErr := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "m0162-build-ac3-cells.py"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, dir := range []string{"scripts", "internal/cli/integration", "internal/workflows/spec/branch"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	scriptPath := filepath.Join(root, "scripts", "m0162-build-ac3-cells.py")
	if err := os.WriteFile(scriptPath, script, 0o644); err != nil {
		t.Fatal(err)
	}
	fixture := "CellID: \"branch-cell-root-location\",\nName: \"Uses this checkout\",\n"
	if err := os.WriteFile(filepath.Join(root, "internal", "cli", "integration", "location_test.go"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", scriptPath)
	cmd.Dir = t.TempDir()
	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("generator: %v\n%s", runErr, output)
	}
	output, readErr := os.ReadFile(filepath.Join(root, "internal", "workflows", "spec", "branch", "rules_m0162_ac3.go"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(output), `ID:      "branch-cell-root-location"`) {
		t.Fatalf("generated catalog lacks this checkout's fixture:\n%s", output)
	}
}
