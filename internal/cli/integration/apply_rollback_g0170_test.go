package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// apply_rollback_g0170_test.go — M-0160/AC-3: binary-level real-git
// E2E regression pin for G-0170.
//
// G-0170 (closed by commit `ed0b5014`) hardened
// `internal/verb/apply.go::applyTx.rollback` to restore touched
// paths to their captured pre-Apply worktree bytes — not to HEAD.
// Pre-G-0170, a failed commit (flaky pre-commit hook, lock
// contention, empty git identity) ran `git restore --staged
// --worktree` against the touched path, which reverts the worktree
// file to HEAD. When the operator had an uncommitted hand-edit at
// that path, the rollback silently discarded it.
//
// The acute case is `aiwf edit-body` bless mode, whose INPUT is the
// operator's working-copy edit (`internal/verb/editbody.go` reads
// `workingBytes` via `os.ReadFile`). Bless-mode + commit failure +
// pre-G-0170 rollback = the operator's hand-authored prose is gone,
// with only a confusing "no changes to commit" on retry as the
// signal. The bug is general (any mutating verb that touches a path
// the operator was already editing loses that edit on commit
// failure), but bless mode is the certain-loss case.
//
// Unit coverage already comprehensive at
// `internal/verb/apply_test.go`:
//
//	- TestApply_RollbackPreservesPreExistingDirtyContent (empty-
//	  identity commit-failure path)
//	- TestApply_RollbackIsFullyClean_G0170Regression (lock-
//	  contention path — rollback's own git restore fails, yet the
//	  captured-bytes write-back is pure filesystem and survives)
//	- TestRollback_RemoveErrorIsCapturedWhenRestoreSucceeds
//	  (captured-absent removal failure path)
//
// What this AC adds: the **binary-level seam** through `aiwf
// edit-body` → `verb.Apply` → `applyTx.rollback`. The unit tests
// drive `verb.Apply` directly with hand-crafted Plans; this test
// drives the full subprocess pipeline against the worktree-built
// binary, exercising the verb-side glue that wires bless-mode
// reading to Apply's plan to the rollback machinery. Without this
// seam, a regression in `editbody.go`'s working-copy capture or
// in the dispatcher's Apply invocation would surface only at
// operator-encounter time.
//
// The test is intentionally free-form (no RunScenarios / no
// Expectation envelope assertion). The framework's Expect is
// designed for `aiwf check` envelope assertions; AC-3's
// load-bearing assertions are filesystem state (worktree bytes)
// and git state (HEAD SHA), not check-rule output.

// TestApplyRollback_AC3_G0170_BlessModePreservesPreApplyDirtyBytes
// reconstructs the canonical G-0170 failure shape end-to-end
// through the worktree-built aiwf binary:
//
//  1. Operator creates an entity (`aiwf add gap`).
//  2. Operator hand-edits the entity file in the worktree without
//     committing (the pre-Apply dirty bytes).
//  3. Operator runs `aiwf edit-body G-NNNN` (bless mode) while a
//     commit-failure trigger is in flight (empty git identity env
//     vars). The verb reads the operator's hand-edit as input,
//     attempts to commit, and fails at the commit step.
//  4. The rollback machinery activates.
//
// Load-bearing assertions:
//
//   - HEAD did NOT advance (commit failure preserved the ref state).
//   - Worktree bytes match the pre-Apply dirty bytes (the operator's
//     hand-edit survived — this is the G-0170 contract).
//   - Exit code is non-zero AND the error envelope does NOT mislead
//     the operator into thinking "no changes to commit" (the pre-fix
//     failure mode where bless-mode retry after rollback saw a clean
//     worktree and reported the misleading message at
//     internal/verb/editbody.go:118).
//
// Sabotage-verified at AC-3 RED: skipping step 2 of
// `applyTx.rollback` at `internal/verb/apply.go:437-452` (the
// captured-bytes write-back loop) fires the worktree-bytes
// assertion — the operator's hand-edit is reverted to HEAD by step
// 1's `git restore`, the load-bearing G-0170 fix is bypassed, and
// the test discriminates with informative output. Discrimination
// confirmed end-to-end through the binary.
func TestApplyRollback_AC3_G0170_BlessModePreservesPreApplyDirtyBytes(t *testing.T) {
	t.Parallel()
	env := newScenarioEnv(t)

	// Step 1: Create a gap entity. `aiwf add gap` produces a stub
	// body the kernel recognizes; the entity gives us a real
	// kernel-managed path to test bless-mode rollback against.
	env.MustRunBin("add", "gap", "--title", "G-0170 fixture entity")

	entityPath := findEntityFile(t, env, "G-0001")
	if entityPath == "" {
		t.Fatal("G-0001 not found after `aiwf add gap`")
	}
	fullPath := filepath.Join(env.Root, entityPath)

	headBefore := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))

	// Step 2: Hand-edit the entity body in the worktree. No
	// `git add`. These are the pre-Apply dirty bytes the rollback
	// must preserve — modeling the bless-mode shape where the
	// operator authored prose directly in the working copy and
	// will hit `aiwf edit-body` to commit it.
	headBytes, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("read HEAD-version of %s: %v", entityPath, err)
	}
	dirty := string(headBytes) + handEditFixtureAC3
	if werr := os.WriteFile(fullPath, []byte(dirty), 0o644); werr != nil {
		t.Fatalf("write hand-edit to %s: %v", entityPath, werr)
	}

	// Step 3: Invoke `aiwf edit-body` with empty git identity env
	// vars. Empty identity makes `git commit` fail deterministically
	// (the unit test at internal/verb/apply_test.go uses the same
	// trigger). The verb's working-copy bytes pass succeeds; the
	// commit step fails; rollback runs.
	//
	// Env vars are appended LAST in testutil.RunBin's env composition,
	// so they override the default GIT_* identity that RunBin sets up
	// for happy-path tests (the AC-6/M-0159 last-wins discovery).
	commitFailureEnv := []string{
		"GIT_AUTHOR_NAME=",
		"GIT_AUTHOR_EMAIL=",
		"GIT_COMMITTER_NAME=",
		"GIT_COMMITTER_EMAIL=",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	}
	out, runErr := testutil.RunBin(t, env.Root, env.BinDir, commitFailureEnv,
		"edit-body", "G-0001",
		"--reason", "M-0160/AC-3 fixture: induce commit failure to exercise G-0170 rollback")
	if runErr == nil {
		t.Fatalf("expected `aiwf edit-body` to fail under empty git identity; got success\n%s", out)
	}

	// Load-bearing assertion 1: HEAD did not advance.
	headAfter := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
	if headBefore != headAfter {
		t.Errorf("HEAD advanced from %s to %s; commit failure should leave HEAD unchanged", headBefore, headAfter)
	}

	// Load-bearing assertion 2: worktree bytes match the pre-Apply
	// dirty state. Pre-G-0170 the rollback's `git restore` reverted
	// the worktree to HEAD, silently discarding the hand-edit.
	// Post-G-0170 the captured-bytes write-back restores the
	// operator's pre-Apply state. This is the G-0170 contract.
	postBytes, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("re-read %s after failed verb: %v", entityPath, err)
	}
	if string(postBytes) != dirty {
		t.Errorf("worktree bytes did not survive failed-verb rollback (G-0170 regressed)\n--- pre-Apply dirty bytes (want): ---\n%s\n--- post-rollback bytes (got): ---\n%s",
			dirty, string(postBytes))
	}

	// Load-bearing assertion 3: the error envelope is informative,
	// not the misleading "no changes to commit" that the pre-G-0170
	// retry path produced when the rollback reverted to HEAD and a
	// subsequent bless attempt saw clean state. G-0170's design note
	// at the fix commit says: "a blind retry wrapper around bless
	// mode is actively harmful: the first failure already destroyed
	// the input, so the retry runs against a clean worktree and
	// reports 'no changes to commit', masking the real cause."
	// Post-G-0170 the rollback preserves the hand-edit, so the
	// retry message can be honest about the real failure.
	if strings.Contains(out, "no changes to commit") {
		t.Errorf("error envelope contains the misleading 'no changes to commit' message — the pre-G-0170 failure mode where rollback destroyed the input and a retry saw clean state. Post-G-0170 the operator's bytes survive, so this message must not appear on the FIRST failure path either.\nenvelope:\n%s", out)
	}
}

// handEditFixtureAC3 is the synthetic operator-authored prose
// appended to the entity body in the worktree before the failed
// `aiwf edit-body` invocation. Extracted to a constant for
// readability (reviewer nit M-0160/AC-3 N-1) so the test body
// stays focused on the load-bearing flow.
const handEditFixtureAC3 = `
## Operator Hand-Edit

The operator authored this prose directly in the worktree without committing first. If the rollback discards these bytes, the operator's work is silently lost — exactly the G-0170 failure shape this AC pins against.
`
