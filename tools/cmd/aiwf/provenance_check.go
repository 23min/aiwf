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
// against the result. Returns the slice of findings (possibly empty)
// and a transport error only when the git subprocess itself fails;
// missing-history (a fresh repo with no commits) returns no findings.
//
// Why grep on `^aiwf-`: the standing rules only ever care about
// trailered commits — every rule is keyed on at least one aiwf trailer
// (actor, principal, scope-ends, etc.). Untrailered commits would be
// silently skipped by the in-package logic anyway, but pre-filtering
// keeps git log cheap on big histories.
func runProvenanceCheck(ctx context.Context, root string, t *tree.Tree) ([]check.Finding, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	commits, err := readProvenanceCommits(ctx, root)
	if err != nil {
		return nil, err
	}
	return check.RunProvenance(commits, t), nil
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
