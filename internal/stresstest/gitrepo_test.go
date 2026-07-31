package stresstest

import (
	"os"
	"path/filepath"
	"testing"
)

// gitrepo_test.go — decision coverage for the shared fixture helpers
// in gitrepo.go. Nothing here starts a subprocess or waits on
// anything, so it belongs in the lane that runs on every push, beside
// the helpers it covers rather than inside whichever scenario file
// first needed them.

// TestReadGapFile_ErrorsWhenNoneOrMultipleMatch pins readGapFile's
// count-mismatch branch (zero matches; more than one match).
func TestReadGapFile_ErrorsWhenNoneOrMultipleMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files []string
	}{
		{name: "zero matches", files: nil},
		{name: "two matches", files: []string{"G-0001-a.md", "G-0001-b.md"}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			gapsDir := filepath.Join(root, "work", "gaps")
			if err := os.MkdirAll(gapsDir, 0o755); err != nil {
				t.Fatalf("mkdir gapsDir: %v", err)
			}
			for _, f := range tc.files {
				if err := os.WriteFile(filepath.Join(gapsDir, f), []byte("x"), 0o644); err != nil {
					t.Fatalf("seeding %s: %v", f, err)
				}
			}
			if _, err := readGapFile(root, "G-0001"); err == nil {
				t.Fatalf("expected readGapFile to error for %s", tc.name)
			}
		})
	}
}
