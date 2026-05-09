package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/scope"
)

// TestRunAuthorize_OpenPauseResumeRoundTrip drives `aiwf authorize`
// end-to-end through the built binary: open a scope, then read it back
// via loadEntityScopes; pause it; load again and assert paused; resume
// it; load again and assert active. This is the integration-level
// proof that the cmd dispatcher, the verb function, and the scope
// loader all line up on a real consumer repo.
func TestRunAuthorize_OpenPauseResumeRoundTrip(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote E-01 active: %v\n%s", err, out)
	}

	// Open a scope.
	if out, err := runBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "implement E-01"); err != nil {
		t.Fatalf("aiwf authorize --to: %v\n%s", err, out)
	}
	scopes := mustLoadScopes(t, root, "E-0001")
	if len(scopes) != 1 {
		t.Fatalf("after open: scopes len=%d, want 1", len(scopes))
	}
	if scopes[0].State != scope.StateActive || scopes[0].Agent != "ai/claude" || scopes[0].Principal != "human/peter" {
		t.Errorf("after open: scope = %+v", scopes[0])
	}

	// Pause it.
	if out, err := runBin(t, root, binDir, nil,
		"authorize", "E-0001", "--pause", "blocked by E-09"); err != nil {
		t.Fatalf("aiwf authorize --pause: %v\n%s", err, out)
	}
	scopes = mustLoadScopes(t, root, "E-0001")
	if scopes[0].State != scope.StatePaused {
		t.Errorf("after pause: state = %s, want paused", scopes[0].State)
	}

	// Resume it.
	if out, err := runBin(t, root, binDir, nil,
		"authorize", "E-0001", "--resume", "back to it"); err != nil {
		t.Fatalf("aiwf authorize --resume: %v\n%s", err, out)
	}
	scopes = mustLoadScopes(t, root, "E-0001")
	if scopes[0].State != scope.StateActive {
		t.Errorf("after resume: state = %s, want active", scopes[0].State)
	}

	// HEAD trailer set carries the resume.
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "authorize")
	hasTrailer(t, tr, "aiwf-entity", "E-0001")
	hasTrailer(t, tr, "aiwf-scope", "resumed")
	hasTrailer(t, tr, "aiwf-reason", "back to it")
}

// TestRunAuthorize_RefusesNonHumanActor: --actor ai/claude is rejected
// before any state is touched — only humans authorize.
func TestRunAuthorize_RefusesNonHumanActor(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil,
		"authorize", "E-0001", "--actor", "ai/claude", "--to", "ai/cursor")
	if err == nil {
		t.Fatalf("expected non-zero exit for non-human actor; output:\n%s", out)
	}
	if !strings.Contains(out, "human/") {
		t.Errorf("expected human/ requirement in error; got:\n%s", out)
	}
}

// TestRunAuthorize_PauseRefusedWhenNoActiveScope: --pause with no
// open scope on the entity exits non-zero with a clear message.
func TestRunAuthorize_PauseRefusedWhenNoActiveScope(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil,
		"authorize", "E-0001", "--pause", "trying without a scope")
	if err == nil {
		t.Fatalf("expected non-zero exit; output:\n%s", out)
	}
	if !strings.Contains(out, "no active scope") {
		t.Errorf("expected no-active-scope error; got:\n%s", out)
	}
}

// TestRunAuthorize_RejectsMixedModes: passing both --pause and
// --resume (or --to + --pause) is a usage error.
func TestRunAuthorize_RejectsMixedModes(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	out, err := runBin(t, root, binDir, nil,
		"authorize", "E-0001", "--pause", "x", "--resume", "y")
	if err == nil {
		t.Fatalf("expected mixed-mode usage error; got:\n%s", out)
	}
	if !strings.Contains(out, "exactly one") {
		t.Errorf("expected usage error mentioning exactly-one; got:\n%s", out)
	}
}

func mustLoadScopes(t *testing.T, root, id string) []*scope.Scope {
	t.Helper()
	scopes, err := loadEntityScopes(context.Background(), root, id)
	if err != nil {
		t.Fatalf("loadEntityScopes: %v", err)
	}
	return scopes
}

func hasTrailer(t *testing.T, trailers []gitops.Trailer, key, value string) {
	t.Helper()
	for _, tr := range trailers {
		if tr.Key == key && tr.Value == value {
			return
		}
	}
	t.Errorf("trailer %s=%q not found in %+v", key, value, trailers)
}
