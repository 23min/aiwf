package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/repolock"
)

// fixtures.go collects the reusable failure triggers for the handful
// of CLI-verb error-handling guards that recur across many `Run`
// functions (M-0252/AC-1): actor-resolution failure, repo-lock
// contention, and a malformed/corrupt tree entity. Callers building
// coverage for a specific verb's error branches (M-0253 through
// M-0256) reach for these instead of re-deriving the trigger.
//
// Deliberately NOT covered here: cliutil.ResolveRoot's error branch.
// Every existing call site marks it `//coverage:ignore` (see
// internal/cli/archive/archive.go's `//coverage:ignore
// cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent
// --root path`) because it essentially never fails inside a normal
// test harness. There is no fixture for it, and M-0253-M-0256 should
// keep using the same `//coverage:ignore` precedent rather than
// looking for one.
//
// Also not covered: an explicitly malformed `--actor` string (e.g.
// "notanactor", missing the required `<role>/<identifier>` slash).
// That failure needs no fixture — pass the literal string directly to
// cliutil.ResolveActor or a verb's Run function.

// BrokenGitIdentity isolates the current test's process environment so
// `git config user.email` cannot yield a usable identity, guaranteeing
// cliutil.ResolveActor("", root) (and any verb's Run that resolves an
// actor with an empty --actor) fails via the "no actor" error path.
//
// It points HOME and XDG_CONFIG_HOME at a fresh, empty-of-gitconfig
// temp directory and sets GIT_CONFIG_NOSYSTEM=1 to keep the host's
// real git identity from leaking in, then writes a `.gitconfig` whose
// user.email has no "@" separator — the same degenerate-but-legal git
// state pinned by actor_test.go's TestResolveActor_MalformedGitEmail.
// All three env vars are set via t.Setenv, so isolation reverts
// automatically at test cleanup; no manual teardown is needed.
func BrokenGitIdentity(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	gitconfig := filepath.Join(home, ".gitconfig")
	if err := os.WriteFile(gitconfig, []byte("[user]\n\temail = no-at-sign\n"), 0o644); err != nil {
		t.Fatalf("BrokenGitIdentity: write %s: %v", gitconfig, err)
	}
}

// HoldRepoLock takes root's repo lock (via repolock.Acquire, the same
// mechanism cliutil.AcquireRepoLock wraps) and returns a release func
// the caller must invoke to free it. Call this before invoking a verb
// that itself calls cliutil.AcquireRepoLock against the same root, to
// force the busy/contention branch — a second Acquire against a
// held lock always fails with repolock.ErrBusy.
//
// The "other error" branch of cliutil.AcquireRepoLock (a root whose
// lockfile can't even be opened) doesn't need a fixture: pass a
// non-existent temp directory as root, e.g.
// filepath.Join(t.TempDir(), "does-not-exist") — the one-line pattern
// already used at internal/cli/renamearea/renamearea_test.go and
// internal/cli/setarea/setarea_test.go.
func HoldRepoLock(t *testing.T, root string) (release func()) {
	t.Helper()
	lock, err := repolock.Acquire(root, 0)
	if err != nil {
		t.Fatalf("HoldRepoLock: %v", err)
	}
	return func() { _ = lock.Release() }
}

// WriteMalformedEntity writes a file at root/relPath containing
// frontmatter with an unclosed YAML string literal — the same
// malformed-YAML shape internal/tree/tree_test.go's
// TestLoad_ParseErrorBecomesLoadError pins — guaranteeing
// tree.Load(ctx, root) reports exactly one LoadError for relPath
// while leaving the rest of the tree loaded normally (tree.Load
// treats a per-file parse error as non-fatal).
//
// relPath must land in one of entity.PathKind's recognized shapes
// (e.g. "work/epics/E-0099-broken/epic.md" or
// "work/gaps/G-0099-broken.md") so the loader's directory walk
// classifies it as an entity file in the first place; anywhere else
// it is treated as a stray and silently skipped, producing no
// LoadError.
func WriteMalformedEntity(t *testing.T, root, relPath string) {
	t.Helper()
	full := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("WriteMalformedEntity: mkdir %s: %v", filepath.Dir(full), err)
	}
	const malformed = `---
id: E-02
title: "Unclosed quote
status: active
---
`
	if err := os.WriteFile(full, []byte(malformed), 0o644); err != nil {
		t.Fatalf("WriteMalformedEntity: write %s: %v", full, err)
	}
}

// InvalidFormat is a --format value no verb recognizes. It exercises
// only ONE of the two `--format` verb shapes in this codebase:
//
//   - Read-style verbs (history, list, schema, show, template, check,
//     contract verify, status, render) validate a hardcoded
//     text/json string themselves, first thing in Run(...), before
//     touching root or tree — passing InvalidFormat there returns
//     cliutil.ExitUsage. See internal/cli/check/check.go's format
//     guard, pinned by check_test.go's TestRun_BadFormat.
//   - Mutating verbs going through cliutil.OutputFormat do NOT
//     validate --format at all: anything other than "json" silently
//     degrades to text via OutputFormat.JSON(). InvalidFormat does
//     NOT trigger a failure there — there is no bad-format branch to
//     cover on that verb shape.
const InvalidFormat = "xml"
