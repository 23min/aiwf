package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/scope"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// runProvenanceCheck walks every commit reachable from HEAD that
// carries any `aiwf-*` trailer and runs the I2.5 standing rules
// against the result. It also runs the step-7b untrailered-entity-
// commit warning over the unpushed range (`@{u}..HEAD` when an
// upstream is set, else all of HEAD). Returns a single concatenated
// finding slice; transport errors propagate.
//
// Why grep on `^aiwf-` for the standing rules: every rule is keyed
// on at least one aiwf trailer (actor, principal, scope-ends, etc.).
// Untrailered commits are handled by the separate step-7b audit pass,
// which uses a different filter (range scoped to the unpushed
// commits, no trailer grep).
func runProvenanceCheck(ctx context.Context, root string, t *tree.Tree) ([]check.Finding, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	commits, err := readProvenanceCommits(ctx, root)
	if err != nil {
		return nil, err
	}
	findings := check.RunProvenance(commits, t)

	untrailed, uErr := readUntrailedCommits(ctx, root)
	if uErr != nil {
		return nil, uErr
	}
	findings = append(findings, check.RunUntrailedAudit(untrailed)...)
	return findings, nil
}

// readUntrailedCommits returns the commits in the unpushed range
// (`@{u}..HEAD`, or all of HEAD when no upstream exists) along with
// their trailer set and the relative paths each commit touched.
//
// The range is what step 7b cares about: already-pushed commits are
// either pre-aiwf (correctly silent) or are someone else's
// responsibility to repair. Once an untrailered commit has been
// pushed, surfacing it locally is noise.
//
// Implementation: a single `git log <range> --name-only --pretty=...`
// invocation with custom record/field separators. The empty unpushed
// range (HEAD == @{u}) returns no commits, no findings.
func readUntrailedCommits(ctx context.Context, root string) ([]check.UntrailedCommit, error) {
	rangeArg, err := unpushedRange(ctx, root)
	if err != nil {
		return nil, err
	}
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	args := []string{
		"log",
		"--reverse",
		rangeArg,
		"--name-only",
		"--pretty=tformat:" + recSep + "%H" + fieldSep + "%(trailers:only=true,unfold=true)" + fieldSep,
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

// unpushedRange picks the rev-range step 7b walks. Falls back to all
// of HEAD (`HEAD`) when no upstream is configured, so a brand-new
// branch surfaces every untrailered entity commit until the first
// push.
func unpushedRange(ctx context.Context, root string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		return "@{u}..HEAD", nil
	}
	return "HEAD", nil
}

// parseUntrailedCommits unpacks the multi-record stream produced by
// readUntrailedCommits. The format is:
//
//	<RS>{SHA}<US>{trailers}<US>
//	{file1}
//	{file2}
//	...
//	<RS>{SHA}<US>...
//
// Trailers and file lists are both newline-delimited. Empty input
// (no unpushed commits) returns nil.
func parseUntrailedCommits(s string) []check.UntrailedCommit {
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	var out []check.UntrailedCommit
	for _, rec := range strings.Split(s, recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 3)
		if len(parts) < 3 {
			continue
		}
		var paths []string
		for _, line := range strings.Split(parts[2], "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			paths = append(paths, line)
		}
		out = append(out, check.UntrailedCommit{
			SHA:      strings.TrimSpace(parts[0]),
			Trailers: parseTrailerLines(parts[1]),
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
