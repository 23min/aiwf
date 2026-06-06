package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_TrailerKeysViaConstants_PositiveControl is the
// regression guard for G-0231 item 1. It synthesizes a minimal
// fixture tree containing a known violation of
// trailer-keys-via-constants and asserts the policy reports it.
//
// Pre-G-0231 the policy used a text-mode regex
// (`"([^"\\]*)"`) over file bytes. RE2's leftmost-first FindAll
// semantics paired the closing `"` of one Go string literal with
// the opening `"` of the next, so a file containing
// `"aiwf-verb"` immediately followed by other quoted strings on
// the same logical block produced zero hits — the regex marched
// past the real literals into the surrounding code. CI stayed
// green with a known violation in tree (render.go:277-278).
//
// The fix is the AST walk in trailer_keys.go; this test pins
// the contract so a future "small refactor" can't silently
// regress to the old shape without CI noticing.
//
// Note: the live-tree test `TestPolicy_TrailerKeysViaConstants`
// is the negative control (no violations in the actual repo).
// Negative + positive together pin "policy detects the bug it
// is meant to detect, and the repo is clean of it."
func TestPolicy_TrailerKeysViaConstants_PositiveControl(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Synthetic gitops/trailers.go so loadGitopsTrailerNames
	// resolves a non-empty set. Only the constant block matters;
	// the rest of the file is omitted.
	mustWrite(t, filepath.Join(root, "internal", "gitops", "trailers.go"), `package gitops

const (
	TrailerVerb  = "aiwf-verb"
	TrailerActor = "aiwf-actor"
)
`)

	// "Production" file in a non-gitops package containing the
	// literal we want flagged. Shape matches the bug observed in
	// internal/cli/render/render.go:277-278 pre-fix.
	mustWrite(t, filepath.Join(root, "internal", "cli", "render", "render.go"), `package render

import "fmt"

type Trailer struct {
	Key   string
	Value string
}

func emit() {
	trailers := []Trailer{
		{Key: "aiwf-verb", Value: "render-roadmap"},
		{Key: "aiwf-actor", Value: "human/test"},
	}
	fmt.Println(trailers)
}
`)

	vs, err := PolicyTrailerKeysViaConstants(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}

	// Expect at least two violations: one for "aiwf-verb" and one
	// for "aiwf-actor". Both pinned by file path and substring of
	// the detail so a future schema change to Violation doesn't
	// silently weaken the assertion.
	wantFile := "internal/cli/render/render.go"
	wantLiterals := []string{"aiwf-verb", "aiwf-actor"}
	hit := map[string]Violation{}
	for _, v := range vs {
		for _, lit := range wantLiterals {
			if v.File == wantFile && strings.Contains(v.Detail, `"`+lit+`"`) {
				hit[lit] = v
			}
		}
	}
	for _, lit := range wantLiterals {
		v, ok := hit[lit]
		if !ok {
			t.Errorf("policy missed literal %q in %s; got %d total violations: %+v",
				lit, wantFile, len(vs), vs)
			continue
		}
		if v.Line <= 0 {
			t.Errorf("violation for %q reports non-positive line %d", lit, v.Line)
		}
		if v.Policy != "trailer-keys-via-constants" {
			t.Errorf("violation for %q has unexpected Policy=%q", lit, v.Policy)
		}
	}
}

// mustWrite creates path (and parents) and writes contents.
func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
