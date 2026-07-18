package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicyCrossBranchScanConsolidation_Synthetic pins the policy's
// branches against synthetic trees: a direct trunk.DetectCollisions
// call fires; a trunk.ScanCrossBranch call and an unqualified
// (in-package) DetectCollisions call do not; an unparsable file is
// skipped without error.
func TestPolicyCrossBranchScanConsolidation_Synthetic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		relPath  string
		body     string
		wantFire bool
		wantLine int
	}{
		{
			name:    "direct trunk.DetectCollisions fires",
			relPath: "internal/cli/list/list.go",
			body: `package list

import (
	"context"

	"github.com/23min/aiwf/internal/trunk"
)

func rows(ctx context.Context, root string, hits []trunk.RefHit) map[string]bool {
	return trunk.DetectCollisions(ctx, root, hits)
}
`,
			wantFire: true,
			wantLine: 10,
		},
		{
			name:    "trunk.ScanCrossBranch does not fire",
			relPath: "internal/cli/list/list.go",
			body: `package list

import (
	"context"

	"github.com/23min/aiwf/internal/trunk"
)

func rows(ctx context.Context, root string) trunk.CrossBranchScan {
	return trunk.ScanCrossBranch(ctx, root, nil)
}
`,
			wantFire: false,
		},
		{
			name:    "unqualified in-package DetectCollisions does not fire",
			relPath: "internal/trunk/trunk.go",
			body: `package trunk

import "context"

func ScanCrossBranch(ctx context.Context, root string, hits []RefHit) map[string]bool {
	return DetectCollisions(ctx, root, hits)
}
`,
			wantFire: false,
		},
		{
			name:     "unparsable file is skipped without error",
			relPath:  "internal/cli/broken/broken.go",
			body:     "package broken\n\nfunc {{{ not go\n",
			wantFire: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			full := filepath.Join(root, filepath.FromSlash(tc.relPath))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			violations, err := PolicyCrossBranchScanConsolidation(root)
			if err != nil {
				t.Fatalf("policy: %v", err)
			}
			found := false
			for _, v := range violations {
				if v.File == tc.relPath && (tc.wantLine == 0 || v.Line == tc.wantLine) {
					found = true
				}
			}
			if found != tc.wantFire {
				t.Errorf("fire = %v, want %v; violations: %+v", found, tc.wantFire, violations)
			}
		})
	}
}

// TestPolicyCrossBranchScanConsolidation_WalkError covers the defensive
// error return: an unwalkable root surfaces WalkGoFiles' error rather
// than being swallowed.
func TestPolicyCrossBranchScanConsolidation_WalkError(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := PolicyCrossBranchScanConsolidation(missing); err == nil {
		t.Error("want error walking a non-existent root, got nil")
	}
}
