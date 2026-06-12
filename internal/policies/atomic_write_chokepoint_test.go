package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicyAtomicWriteChokepoint_Synthetic pins the policy's
// branches against synthetic trees: raw os.WriteFile, os.Create, and
// write-mode os.OpenFile fire; read-only os.OpenFile, a
// pathutil.AtomicWriteFile call, and an unparsable file do not; an
// allowlisted file is exempt.
func TestPolicyAtomicWriteChokepoint_Synthetic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		relPath  string
		body     string
		wantFire bool
		wantLine int
	}{
		{
			name:    "raw os.WriteFile fires",
			relPath: "internal/cli/drift/drift.go",
			body: `package drift

import "os"

func bad(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
`,
			wantFire: true,
			wantLine: 6,
		},
		{
			name:    "raw os.Create fires",
			relPath: "internal/cli/drift/drift.go",
			body: `package drift

import "os"

func bad(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}
`,
			wantFire: true,
			wantLine: 6,
		},
		{
			name:    "write-mode os.OpenFile fires",
			relPath: "internal/cli/drift/drift.go",
			body: `package drift

import "os"

func bad(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
`,
			wantFire: true,
			wantLine: 6,
		},
		{
			name:    "read-only os.OpenFile does not fire",
			relPath: "internal/cli/clean/clean.go",
			body: `package clean

import "os"

func good(path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	return f.Close()
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
		{
			name:    "AtomicWriteFile call does not fire",
			relPath: "internal/cli/clean/clean.go",
			body: `package clean

import "github.com/23min/aiwf/internal/pathutil"

func good(path string, data []byte) error {
	return pathutil.AtomicWriteFile(path, data, 0o644)
}
`,
			wantFire: false,
		},
		{
			name:    "allowlisted file is exempt",
			relPath: "internal/cli/doctor/selfcheck.go",
			body: `package doctor

import "os"

func sandboxWrite(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
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
			violations, err := PolicyAtomicWriteChokepoint(root)
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
