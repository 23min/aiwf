package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/scope"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// runProvenanceCheck walks every commit reachable from HEAD that
// carries any `aiwf-*` trailer and runs the I2.5 standing rules
// against the result. It also runs the step-7b untrailered-entity-
// commit warning, scoped per the rules in resolveUntrailedRange:
//   - --since <ref> on the verb wins.
//   - Otherwise `@{u}..HEAD` when an upstream is configured.
//   - Otherwise the audit is SKIPPED with a single
//     `provenance-untrailered-scope-undefined` advisory; the
//     fallback used to be "all of HEAD," which on long-lived
//     branches floods with warnings against commits already
//     merged in from trunk. See issue #5 sub-item 2.
//
// Returns a single concatenated finding slice; transport errors
// propagate.
//
// Why grep on `^aiwf-` for the standing rules: every rule is keyed
// on at least one aiwf trailer (actor, principal, scope-ends, etc.).
// Untrailered commits are handled by the separate step-7b audit pass,
// which uses a different filter (range scoped per resolveUntrailedRange,
// no trailer grep).
func runProvenanceCheck(ctx context.Context, root string, t *tree.Tree, since string) ([]check.Finding, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	commits, err := readProvenanceCommits(ctx, root)
	if err != nil {
		return nil, err
	}
	findings := check.RunProvenance(commits, t)

	rangeArg, advisory, rErr := resolveUntrailedRange(ctx, root, since)
	if rErr != nil {
		return nil, rErr
	}
	if advisory != nil {
		findings = append(findings, *advisory)
		return findings, nil
	}
	untrailed, uErr := readUntrailedCommits(ctx, root, rangeArg)
	if uErr != nil {
		return nil, uErr
	}
	findings = append(findings, check.RunUntrailedAudit(untrailed)...)
	return findings, nil
}

// resolveUntrailedRange picks the `git log` range for the step-7b
// untrailered-entity audit. Three branches:
//
//  1. since != "" — the operator's explicit choice wins. Validates
//     the ref shape via `git rev-parse --verify`; an unrecognized
//     ref returns a usage-error advisory finding so the audit is
//     still skipped (rather than failing the whole check verb).
//  2. else, an upstream is configured — return `@{u}..HEAD`.
//  3. else — return ("", advisory) so the caller skips the scan
//     and surfaces the undefined-scope warning.
func resolveUntrailedRange(ctx context.Context, root, since string) (string, *check.Finding, error) {
	if since != "" {
		// Verify the ref before trusting it: a typo in `--since`
		// would otherwise cause a `git log` failure that aborts
		// the whole `aiwf check` run.
		verify := exec.CommandContext(ctx, "git", "rev-parse", "--verify", since+"^{commit}")
		verify.Dir = root
		if err := verify.Run(); err != nil {
			advisory := &check.Finding{
				Code:     check.CodeProvenanceUntrailedScopeUndefined,
				Severity: check.SeverityWarning,
				Message: fmt.Sprintf("--since %q does not resolve to a commit; provenance audit skipped",
					since),
			}
			return "", advisory, nil
		}
		return since + "..HEAD", nil, nil
	}
	upstream := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	upstream.Dir = root
	if err := upstream.Run(); err == nil {
		return "@{u}..HEAD", nil, nil
	}
	advisory := &check.Finding{
		Code:     check.CodeProvenanceUntrailedScopeUndefined,
		Severity: check.SeverityWarning,
		Message:  "no upstream configured and no --since <ref>; provenance audit skipped",
	}
	return "", advisory, nil
}

// readUntrailedCommits returns the commits in rangeArg (e.g.
// `@{u}..HEAD`, or `<sha>..HEAD` from --since) along with their
// trailer set and the relative paths each commit touched.
//
// The range is supplied by the caller (resolveUntrailedRange);
// readUntrailedCommits is purely the git-log invocation +
// parsing. An empty range (HEAD == @{u}) returns no commits,
// no findings.
//
// `-m --first-parent` walks the integration-branch view (G32):
// merge commits surface their introduced changes (against their
// first parent) so the audit pass sees entity-file paths brought
// in by `git merge`, while feature-branch commits not on
// first-parent ancestry are correctly excluded (those are the
// feature branch's own warning scope, not the integration
// branch's). Without `-m` the default is "show no diff for merge
// commits," which silently bypassed the audit for merges that
// absorbed entity-file changes from a feature branch.
func readUntrailedCommits(ctx context.Context, root, rangeArg string) ([]check.UntrailedCommit, error) {
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	args := []string{
		"log",
		"--reverse",
		"-m",
		"--first-parent",
		rangeArg,
		"--name-only",
		"--pretty=tformat:" + recSep + "%H" + fieldSep + "%s" + fieldSep + "%(trailers:only=true,unfold=true)" + fieldSep,
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log: %w", err)
	}
	return parseUntrailedCommits(string(out)), nil
}

// parseUntrailedCommits unpacks the multi-record stream produced by
// readUntrailedCommits. The format is:
//
//	<RS>{SHA}<US>{subject}<US>{trailers}<US>
//	{file1}
//	{file2}
//	...
//	<RS>{SHA}<US>...
//
// Trailers and file lists are both newline-delimited. Subject is
// the commit's first line, used for the squash-merge specialization
// (G31). Empty input (no unpushed commits) returns nil.
func parseUntrailedCommits(s string) []check.UntrailedCommit {
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	var out []check.UntrailedCommit
	for _, rec := range strings.Split(s, recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 4)
		if len(parts) < 4 {
			continue
		}
		var paths []string
		for _, line := range strings.Split(parts[3], "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			paths = append(paths, line)
		}
		out = append(out, check.UntrailedCommit{
			SHA:      strings.TrimSpace(parts[0]),
			Subject:  strings.TrimSpace(parts[1]),
			Trailers: parseTrailerLines(parts[2]),
			Paths:    paths,
		})
	}
	return out
}

// readProvenanceCommits returns every commit reachable from HEAD whose
// message carries any aiwf-* trailer, oldest-first. The output shape
// matches scope.Commit (SHA + parsed trailers) so check.RunProvenance
// stays I/O-free.
//
// The grep pattern is a basic regex anchored to the start of a
// trailer line. `git log -E` enables ERE so the anchor is honored.
func readProvenanceCommits(ctx context.Context, root string) ([]scope.Commit, error) {
	const fieldSep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"--reverse",
		"-E",
		"--grep", "^aiwf-[a-z-]+:",
		"--pretty=tformat:%H" + fieldSep + "%(trailers:only=true,unfold=true)\x1e",
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log: %w", err)
	}
	var commits []scope.Commit
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 2)
		if len(parts) < 2 {
			continue
		}
		commits = append(commits, scope.Commit{
			SHA:      strings.TrimSpace(parts[0]),
			Trailers: parseTrailerLines(parts[1]),
		})
	}
	return commits, nil
}
