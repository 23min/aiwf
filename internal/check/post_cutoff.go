package check

import (
	"context"
	"os/exec"
	"strings"
)

// post_cutoff.go — G-0218 Patch 2 gather-layer helper.
//
// WalkPostCutoffSHAs returns the set of commit SHAs reachable from
// HEAD that descend from HookInstallSHA — the SHA at which the
// commit-msg hook materialization landed (Patch 1). The set is the
// input the trailer-verb-unknown rule reads to decide which findings
// emit at SeverityError (post-cutoff = the commit-msg hook would
// have caught the fabrication at composition time, so landing it
// required `--no-verify` or git plumbing) vs. SeverityWarning
// (pre-cutoff = historical trunk; the G-0150 baseline preserved so
// addressed_by_commit refs aren't invalidated by retroactive
// breakage).
//
// Algorithm: a single `git rev-list <cutoff>..HEAD` call. The range
// form excludes the cutoff itself (the hook-install commit was
// authored before its own hook existed, so it cannot be policed by
// the hook it installs).
//
// Fallback contract — returns nil for every failure mode:
//   - non-git directory (no .git/) → silent
//   - cutoff SHA not reachable from HEAD (shallow clone, fork
//     diverged before the hook landed, repo unrelated to aiwf
//     trunk) → silent
//   - rev-list errored for any other reason → silent
//
// The rule's "nil postCutoffSHAs → all warnings" fallback (see
// trailer_verb_unknown.go:RunTrailerVerbUnknown) consumes nil
// identically to "no commits descend from the cutoff." Either
// way, pre-cutoff baseline is preserved.

// WalkPostCutoffSHAs walks `git rev-list HookInstallSHA..HEAD` once
// and returns the resulting SHA set. See file-level docstring above
// for failure-mode contracts. The caller is the CLI gather layer at
// internal/cli/check/provenance.go; the result threads through
// RunTrailerVerbUnknown's postCutoffSHAs parameter.
//
// The production wrapper delegates to walkPostCutoffSHAsFromCutoff
// so tests can exercise the walker against fixture cutoff SHAs;
// pinning the live constant in production keeps the gather layer's
// call site terse and the cutoff knowledge concentrated in
// trailer_verb_unknown.go.
func WalkPostCutoffSHAs(ctx context.Context, root string) map[string]bool {
	return walkPostCutoffSHAsFromCutoff(ctx, root, HookInstallSHA)
}

// walkPostCutoffSHAsFromCutoff is the parameterized inner helper.
// Unexported because no caller outside this package needs it; the
// production surface is WalkPostCutoffSHAs which pins the cutoff to
// HookInstallSHA.
//
// Tests use this form to walk against a fresh-fixture cutoff SHA;
// no production code path passes a non-HookInstallSHA value.
func walkPostCutoffSHAsFromCutoff(ctx context.Context, root, cutoffSHA string) map[string]bool {
	if root == "" || cutoffSHA == "" || !hasGitCommits(ctx, root) {
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "rev-list", cutoffSHA+"..HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// Most common cause: cutoffSHA not in the object database
		// (shallow clone, fork that diverged before the hook
		// landed, fresh repo unrelated to aiwf trunk). The rule's
		// nil-fallback handles this.
		return nil
	}
	post := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		sha := strings.TrimSpace(line)
		if sha == "" {
			continue
		}
		post[sha] = true
	}
	if len(post) == 0 {
		return nil
	}
	return post
}
