package cliutil

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// provenance_test.go pins DecorateAndFinish's gateAndDecorate-denial
// branch (M-0252/AC-2, provenance.go:295): when the I2.5 allow-rule
// refuses the act, DecorateAndFinish abandons the plan and reports
// ExitFindings without ever calling FinishVerb/verb.Apply.

// TestDecorateAndFinish_GateDenialReturnsExitFindings drives the
// cheapest deterministic gateAndDecorate failure: a non-human actor
// with no --principal. verb.Allow denies before any scope lookup is
// even consulted (internal/verb/allow.go's "principal required for
// non-human actor" pre-scope usage denial), so no real git history is
// needed — root only needs to exist (loadActiveScopesForActor's
// HasCommits probe degrades to "no events" on a repo-less temp dir).
func TestDecorateAndFinish_GateDenialReturnsExitFindings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	result := &verb.Result{
		Plan: &verb.Plan{Subject: "unused", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}},
	}
	pctx := ProvenanceContext{
		Actor:    "ai/claude",
		VerbKind: verb.VerbAct,
		TargetID: "E-0001",
		// Principal deliberately empty: a non-human actor without a
		// principal is refused by Allow's pre-scope usage check.
	}
	code, sha := DecorateAndFinish(context.Background(), root, "aiwf test", &tree.Tree{}, result, nil, pctx, OutputFormat{Format: "json"})
	if code != ExitFindings {
		t.Errorf("code = %d, want ExitFindings (%d)", code, ExitFindings)
	}
	if sha != "" {
		t.Errorf("sha = %q, want empty", sha)
	}
}
