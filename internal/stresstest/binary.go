// Package stresstest provides the on-demand correctness stress
// harness's core scaffolding: building the binary under test, driving
// scenarios, and streaming a report of what happened. It is dev-only
// tooling invoked via cmd/stresstest — never installed alongside
// cmd/aiwf.
package stresstest

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// BuildBinary compiles moduleRoot's ./cmd/aiwf into outDir/aiwf and
// returns its absolute path. outDir must itself be an absolute path —
// the caller (typically a fresh os.MkdirTemp/t.TempDir) is responsible
// for that, so the returned path never depends on resolving the
// process's current working directory. Matches this repo's own
// worktree-binary discipline (make diag-aiwf's precedent; CLAUDE.md
// "Worktree binary discipline"): a stress run builds a fresh binary
// from source at the start of every run and uses the returned path
// throughout — never whatever an "aiwf" PATH lookup would resolve,
// which could be stale or absent.
func BuildBinary(ctx context.Context, moduleRoot, outDir string) (string, error) {
	if !filepath.IsAbs(outDir) {
		return "", fmt.Errorf("outDir must be an absolute path, got %q", outDir)
	}
	bin := filepath.Join(outDir, "aiwf")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/aiwf")
	cmd.Dir = moduleRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("building aiwf binary from %s: %w\n%s", moduleRoot, err, out)
	}
	return bin, nil
}

// BuildLockHolder compiles moduleRoot's
// ./internal/stresstest/lockholder into outDir/lockholder and returns
// its absolute path. See BuildBinary for the outDir-must-be-absolute
// rationale, which applies identically here. M-0242/AC-1's scenario
// uses the returned binary to hold internal/repolock's lock from a
// real, independently killable OS process.
func BuildLockHolder(ctx context.Context, moduleRoot, outDir string) (string, error) {
	if !filepath.IsAbs(outDir) {
		return "", fmt.Errorf("outDir must be an absolute path, got %q", outDir)
	}
	bin := filepath.Join(outDir, "lockholder")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./internal/stresstest/lockholder")
	cmd.Dir = moduleRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("building lockholder binary from %s: %w\n%s", moduleRoot, err, out)
	}
	return bin, nil
}
