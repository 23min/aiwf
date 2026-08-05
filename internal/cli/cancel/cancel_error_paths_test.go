package cancel_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cancel"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestRun_AuditOnlyBranch_EntityNotFound covers M-0253/AC-1's sole
// cancel.go flagged branch: the --audit-only dispatch arm (`if
// auditOnly { ... }`), which no existing test reached — the one
// integration test exercising `aiwf cancel --audit-only`
// (internal/cli/integration/auditonly_cmd_test.go) drives a separate
// compiled binary as a subprocess, invisible to this package's
// coverage instrumentation.
//
// A nonexistent id is enough to prove the branch's two statements
// (the verb.CancelAuditOnly call and the DecorateAndFinish call)
// execute; verb.CancelAuditOnly's own not-found handling is out of
// this milestone's scope (internal/verb has its own coverage).
func TestRun_AuditOnlyBranch_EntityNotFound(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rc := cancel.Run(cancel.Options{
		ID:        "G-0001",
		Actor:     "human/test",
		Root:      root,
		Reason:    "manual flip from earlier",
		AuditOnly: true,
	})
	if rc == cliutil.ExitOK {
		t.Errorf("audit-only cancel of a nonexistent entity: rc = ExitOK, want a non-OK exit code")
	}
}

// TestRun_NonHumanForceIsRefusedAtTheDispatcher covers the
// sovereign-force pre-check Run makes right after the prelude
// (M-0293/AC-3). The guard's own arms are covered in
// internal/cli/cliutil; what this pins is that the dispatcher calls it,
// and that a human actor is not caught by it.
//
// The human case is the discriminator: both invocations name an entity
// no tree here contains, so both fail, and only the non-human one may
// fail with the legality exit the coherence refusal carries.
func TestRun_NonHumanForceIsRefusedAtTheDispatcher(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	opts := cancel.Options{
		ID:     "G-0001",
		Root:   root,
		Reason: "an agent reaching for a sovereign act",
		Force:  true,
	}

	nonHuman := opts
	nonHuman.Actor = "ai/claude"
	if rc := cancel.Run(nonHuman); rc != cliutil.ExitFindings {
		t.Errorf("non-human --force: rc = %d, want ExitFindings (%d) — the dispatcher never "+
			"consulted the sovereign-force guard", rc, cliutil.ExitFindings)
	}

	human := opts
	human.Actor = "human/test"
	if rc := cancel.Run(human); rc == cliutil.ExitFindings {
		t.Error("human --force reached the same exit as the non-human one, so the assertion " +
			"above proves nothing about the guard — both may simply be failing on the missing entity")
	}
}
