package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicyMintIDsViaAllocate_Synthetic pins the policy's branches
// against synthetic trees: a hand-rolled zero-pad id-mint Sprintf
// call under internal/verb/ fires; a plain entity.AllocateID call, an
// unrelated zero-pad Sprintf outside internal/verb/, a non-zero-pad
// Sprintf (e.g. AC-sub-id minting, `%d` not `%0*d`), and an
// unparsable file do not fire; an allowlisted file (rewidth.go's
// legitimate re-display of an already-existing id) is exempt.
func TestPolicyMintIDsViaAllocate_Synthetic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		relPath  string
		body     string
		wantFire bool
		wantLine int
	}{
		{
			name:    "hand-rolled zero-pad mint fires",
			relPath: "internal/verb/rogue.go",
			body: `package verb

import "fmt"

func formatID(prefix string, n int) string {
	return fmt.Sprintf("%s%0*d", prefix, 4, n)
}
`,
			wantFire: true,
			wantLine: 6,
		},
		{
			name:    "entity.AllocateID call does not fire",
			relPath: "internal/verb/add.go",
			body: `package verb

import "github.com/23min/aiwf/internal/entity"

func mint(k entity.Kind, entities []*entity.Entity, trunkIDs []string) string {
	return entity.AllocateID(k, entities, trunkIDs)
}
`,
			wantFire: false,
		},
		{
			name:    "zero-pad Sprintf outside internal/verb/ does not fire",
			relPath: "internal/entity/canonicalize.go",
			body: `package entity

import "fmt"

func Canonicalize(id string) string {
	return fmt.Sprintf("%s%0*d", "E-", 4, 1)
}
`,
			wantFire: false,
		},
		{
			name:    "non-zero-pad Sprintf (AC sub-id shape) does not fire",
			relPath: "internal/verb/ac.go",
			body: `package verb

import "fmt"

func nextACID(base, i int) string {
	return fmt.Sprintf("AC-%d", base+i+1)
}
`,
			wantFire: false,
		},
		{
			name:     "unparsable file is skipped without error",
			relPath:  "internal/verb/broken.go",
			body:     "package verb\n\nfunc {{{ not go\n",
			wantFire: false,
		},
		{
			name:    "zero-arg Sprintf does not fire",
			relPath: "internal/verb/degenerate.go",
			body: `package verb

import "fmt"

func degenerate() string {
	return fmt.Sprintf()
}
`,
			wantFire: false,
		},
		{
			name:    "non-literal format arg does not fire",
			relPath: "internal/verb/dynamic.go",
			body: `package verb

import "fmt"

func dynamic(format string, n int) string {
	return fmt.Sprintf(format, n)
}
`,
			wantFire: false,
		},
		{
			name:    "non-Sprintf fmt call with a coincidental id-shaped format string does not fire",
			relPath: "internal/verb/errmsg.go",
			body: `package verb

import "fmt"

func badID(n int) error {
	return fmt.Errorf("id %s%0*d is invalid", "E-", 4, n)
}
`,
			wantFire: false,
		},
		{
			name:    "allowlisted rewidth.go is exempt",
			relPath: "internal/verb/rewidth.go",
			body: `package verb

import "fmt"

func padToCanonical(prefix, digits string) string {
	n := 0
	return fmt.Sprintf("%s-%0*d", prefix, 4, n)
}
`,
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
			violations, err := PolicyMintIDsViaAllocate(root)
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
