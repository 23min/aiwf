package cliutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// apply_test.go pins FinishVerb (M-0252/AC-2) — the shared post-verb
// exit-code chokepoint every mutating verb's Run calls, previously
// untested. Each subtest drives one of the five outcome branches with
// a synthetic (*verb.Result, error) pair, asserting the returned
// (code, sha) pair directly; the emitted stdout/stderr envelope is
// exercised separately by TestOutputFormat_EmitHelpers
// (outputformat_test.go), so these tests don't re-assert it.

// TestFinishVerb_ErrBranches covers the err != nil path (lines 33-40):
// a Coded error resolves to ExitFindings, any other error to
// ExitUsage.
func TestFinishVerb_ErrBranches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	out := OutputFormat{Format: "json"}

	t.Run("coded error -> ExitFindings", func(t *testing.T) {
		t.Parallel()
		codedErr := &entity.FSMTransitionError{Kind: entity.KindGap, From: entity.StatusOpen, To: "bogus"}
		code, sha := FinishVerb(context.Background(), root, "aiwf test", nil, codedErr, out)
		if code != ExitFindings {
			t.Errorf("code = %d, want ExitFindings (%d)", code, ExitFindings)
		}
		if sha != "" {
			t.Errorf("sha = %q, want empty", sha)
		}
	})

	t.Run("plain error -> ExitUsage", func(t *testing.T) {
		t.Parallel()
		code, sha := FinishVerb(context.Background(), root, "aiwf test", nil, errors.New("boom"), out)
		if code != ExitUsage {
			t.Errorf("code = %d, want ExitUsage (%d)", code, ExitUsage)
		}
		if sha != "" {
			t.Errorf("sha = %q, want empty", sha)
		}
	})
}

// TestFinishVerb_NilResult covers the result == nil branch (line 41):
// a nil error paired with a nil result is treated as an internal
// error — every real verb path fills in a Result, so this guards
// against a verb bug rather than a user-facing condition.
func TestFinishVerb_NilResult(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	code, sha := FinishVerb(context.Background(), root, "aiwf test", nil, nil, OutputFormat{Format: "json"})
	if code != ExitInternal {
		t.Errorf("code = %d, want ExitInternal (%d)", code, ExitInternal)
	}
	if sha != "" {
		t.Errorf("sha = %q, want empty", sha)
	}
}

// TestFinishVerb_ErrorFindings covers the check.HasErrors branch (line
// 45): an error-severity finding on an otherwise successful result
// short-circuits to ExitFindings before the plan is ever applied.
func TestFinishVerb_ErrorFindings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	result := &verb.Result{
		Findings: []check.Finding{
			{Code: check.CodeStatusValid, Severity: check.SeverityError, Message: "bad status", EntityID: "E-0001"},
		},
		Plan: &verb.Plan{Subject: "unused", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}},
	}
	code, sha := FinishVerb(context.Background(), root, "aiwf test", result, nil, OutputFormat{Format: "json"})
	if code != ExitFindings {
		t.Errorf("code = %d, want ExitFindings (%d)", code, ExitFindings)
	}
	if sha != "" {
		t.Errorf("sha = %q, want empty", sha)
	}
}

// TestFinishVerb_NoOp covers the result.NoOp branch (line 49): a
// no-op result reports success without ever reaching the plan/apply
// steps below it.
func TestFinishVerb_NoOp(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	result := &verb.Result{NoOp: true, NoOpMessage: "already at target state"}
	code, sha := FinishVerb(context.Background(), root, "aiwf test", result, nil, OutputFormat{Format: "json"})
	if code != ExitOK {
		t.Errorf("code = %d, want ExitOK (%d)", code, ExitOK)
	}
	if sha != "" {
		t.Errorf("sha = %q, want empty", sha)
	}
}

// TestFinishVerb_NilPlan covers the result.Plan == nil branch (line
// 53): validation passed (no error, no findings, not a NoOp) but the
// verb produced no plan — a verb-implementation bug FinishVerb refuses
// to paper over.
func TestFinishVerb_NilPlan(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	code, sha := FinishVerb(context.Background(), root, "aiwf test", &verb.Result{}, nil, OutputFormat{Format: "json"})
	if code != ExitInternal {
		t.Errorf("code = %d, want ExitInternal (%d)", code, ExitInternal)
	}
	if sha != "" {
		t.Errorf("sha = %q, want empty", sha)
	}
}

// TestFinishVerb_ApplyFails covers the applyErr != nil branch (line
// 65): a syntactically valid plan that verb.Apply itself refuses. A
// plan with zero Ops and AllowEmpty unset is verb.Apply's own
// "nothing to commit" guard (see internal/verb/apply_test.go's
// TestApply_RefusesNothingToCommit) — the cheapest deterministic
// trigger, requiring no lock contention or filesystem tricks.
func TestFinishVerb_ApplyFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")
	if err := os.WriteFile(filepath.Join(root, "seed.txt"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, root, "add", "seed.txt")
	runGit(t, root, "commit", "-q", "-m", "seed")

	result := &verb.Result{
		Plan: &verb.Plan{Subject: "empty plan", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}},
	}
	code, sha := FinishVerb(context.Background(), root, "aiwf test", result, nil, OutputFormat{Format: "json"})
	if code != ExitInternal {
		t.Errorf("code = %d, want ExitInternal (%d)", code, ExitInternal)
	}
	if sha != "" {
		t.Errorf("sha = %q, want empty", sha)
	}
}

// TestFinishVerb_Success covers the terminal happy path (lines 69-75):
// a real plan applies cleanly, reporting ExitOK and the resulting
// commit SHA.
func TestFinishVerb_Success(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")
	if err := os.WriteFile(filepath.Join(root, "seed.txt"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, root, "add", "seed.txt")
	runGit(t, root, "commit", "-q", "-m", "seed")

	result := &verb.Result{
		Plan: &verb.Plan{
			Subject:  "add a file",
			Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
			Ops:      []verb.FileOp{{Type: verb.OpWrite, Path: "new.md", Content: []byte("hi\n")}},
		},
	}
	code, sha := FinishVerb(context.Background(), root, "aiwf test", result, nil, OutputFormat{Format: "json"})
	if code != ExitOK {
		t.Errorf("code = %d, want ExitOK (%d)", code, ExitOK)
	}
	if sha == "" {
		t.Error("sha is empty, want the resulting commit SHA")
	}
}
