package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicyEnvelopeVersionSource_FiresOnPackageGlobal proves the
// policy catches the regression class M-0118/AC-8 is designed to
// prevent: a production .go file building a render.Envelope with
// `Version:` sourced from a package-global identifier instead of
// `version.Current().Version`.
//
// Per CLAUDE.md §"Test untested code paths": a positive-only test
// (the live repo is clean) proves nothing about whether the policy
// actually fires when drift appears. This test drives the policy
// against a synthetic fixture tree.
func TestPolicyEnvelopeVersionSource_FiresOnPackageGlobal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "cli", "drift")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package drift

import "github.com/23min/aiwf/internal/render"

var Version = "dev"

func bad() render.Envelope {
	return render.Envelope{
		Tool:    "aiwf",
		Version: Version,
		Status:  "ok",
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "drift.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyEnvelopeVersionSource(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected at least one violation on the synthetic drift fixture; got none")
	}
	foundDrift := false
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" && v.Line == 10 {
			foundDrift = true
			break
		}
	}
	if !foundDrift {
		t.Errorf("expected violation on drift.go:10; got: %+v", violations)
	}
}

// TestPolicyEnvelopeVersionSource_AcceptsCanonical proves the policy
// accepts version.Current().Version as the canonical source.
func TestPolicyEnvelopeVersionSource_AcceptsCanonical(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "cli", "good")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package good

import (
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/version"
)

func ok() render.Envelope {
	return render.Envelope{
		Tool:    "aiwf",
		Version: version.Current().Version,
		Status:  "ok",
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "good.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyEnvelopeVersionSource(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/cli/good/good.go" {
			t.Errorf("policy fired on canonical fixture: %+v", v)
		}
	}
}
