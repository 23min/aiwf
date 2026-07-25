package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// writeVerbScaffoldTree lays down a synthetic internal/cli tree in a
// fresh temp dir: a cliutil stub declaring the wrapped primitives (so
// the relocation anchor stays satisfied and detection cases assert
// only on the re-inline signal), plus the case's own file at relPath.
// Returns the tree root.
func writeVerbScaffoldTree(t *testing.T, relPath, body string) string {
	t.Helper()
	root := t.TempDir()
	writeVerbScaffoldGo(t, root, "internal/cli/cliutil/prim.go", `package cliutil

func ResolveLogger()          {}
func EmitVerbOutcome()        {}
func ResolveActor()           {}
func ResolveActorWithSource() {}
func BeginVerbDiag()          {}
func ResolvePrelude()         {}
`)
	writeVerbScaffoldGo(t, root, relPath, body)
	return root
}

// writeVerbScaffoldGo writes body to root/relPath (forward-slash
// relative), creating parent dirs.
func writeVerbScaffoldGo(t *testing.T, root, relPath, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// verbScaffoldFires reports whether the policy produced a violation for
// file rel (line-agnostic when wantLine==0).
func verbScaffoldFires(t *testing.T, root, rel string, wantLine int) bool {
	t.Helper()
	vs, err := PolicyVerbScaffoldSingleSeam(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range vs {
		if v.File == rel && (wantLine == 0 || v.Line == wantLine) {
			return true
		}
	}
	return false
}

// TestPolicyVerbScaffold_DiagBlock is the mechanical evidence for
// M-0280/AC-1: the guard fires when a verb reconstructs the diagnostic
// block inline (a direct cliutil.ResolveLogger / EmitVerbOutcome call),
// is exempt for the documented non-member, and stays silent for a verb
// that routes through cliutil.BeginVerbDiag or lives outside the verb
// layer.
func TestPolicyVerbScaffold_DiagBlock(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		relPath  string
		body     string
		wantFire bool
	}{
		{
			name:    "direct ResolveLogger re-inline fires",
			relPath: "internal/cli/frob/frob.go",
			body: `package frob

import "github.com/23min/aiwf/internal/cli/cliutil"

func Run() {
	_ = cliutil.ResolveLogger()
}
`,
			wantFire: true,
		},
		{
			name:    "direct EmitVerbOutcome re-inline fires",
			relPath: "internal/cli/frob/frob.go",
			body: `package frob

import "github.com/23min/aiwf/internal/cli/cliutil"

func Run() {
	cliutil.EmitVerbOutcome()
}
`,
			wantFire: true,
		},
		{
			name:    "allowlisted upgrade non-member is exempt",
			relPath: "internal/cli/upgrade/upgrade.go",
			body: `package upgrade

import "github.com/23min/aiwf/internal/cli/cliutil"

func Run() {
	_ = cliutil.ResolveLogger()
	cliutil.EmitVerbOutcome()
}
`,
			wantFire: false,
		},
		{
			name:    "verb routing through BeginVerbDiag does not fire",
			relPath: "internal/cli/frob/frob.go",
			body: `package frob

import "github.com/23min/aiwf/internal/cli/cliutil"

func Run() {
	_ = cliutil.BeginVerbDiag()
}
`,
			wantFire: false,
		},
		{
			name:    "non-verb package calling a primitive is out of scope",
			relPath: "internal/stresstest/scenario.go",
			body: `package stresstest

import "github.com/23min/aiwf/internal/cli/cliutil"

func run() {
	_ = cliutil.ResolveLogger()
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
			root := writeVerbScaffoldTree(t, tc.relPath, tc.body)
			if got := verbScaffoldFires(t, root, tc.relPath, 0); got != tc.wantFire {
				t.Errorf("fire = %v, want %v", got, tc.wantFire)
			}
		})
	}
}
