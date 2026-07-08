package stresstest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestBuildBinary_UsesFreshAbsolutePathNotPATH builds the aiwf binary
// under test via BuildBinary, then invokes the returned path with a
// decoy "aiwf" script placed earlier on PATH. If a caller of
// BuildBinary's result ever fell back to a bare "aiwf" PATH lookup
// instead of the returned absolute path, the decoy's output would
// appear instead of the real binary's — this pins AC-1's "never
// trusting PATH" claim concretely, rather than only checking that the
// returned path looks absolute.
func TestBuildBinary_UsesFreshAbsolutePathNotPATH(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("aiwf is unix-only")
	}

	root := repoRootRelative
	outDir := t.TempDir()

	bin, err := BuildBinary(context.Background(), root, outDir)
	if err != nil {
		t.Fatalf("BuildBinary: %v", err)
	}
	if !filepath.IsAbs(bin) {
		t.Fatalf("BuildBinary returned non-absolute path %q", bin)
	}
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("stat built binary: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("built binary %q is not executable: mode %v", bin, info.Mode())
	}

	decoyDir := t.TempDir()
	decoy := filepath.Join(decoyDir, "aiwf")
	if writeErr := os.WriteFile(decoy, []byte("#!/bin/sh\necho DECOY\n"), 0o755); writeErr != nil {
		t.Fatalf("write decoy: %v", writeErr)
	}

	cmd := exec.Command(bin, "--version")
	cmd.Env = append(os.Environ(), "PATH="+decoyDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running built binary: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "DECOY") {
		t.Fatalf("built binary invocation resolved the decoy on PATH instead of the built binary, got: %s", out)
	}
}

// TestBuildBinary_ErrorsOnBuildFailure confirms a `go build` failure
// surfaces as an error carrying the build output, not a silently empty
// path — the moduleRoot here has no ./cmd/aiwf package to build.
func TestBuildBinary_ErrorsOnBuildFailure(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("aiwf is unix-only")
	}

	bogusRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(bogusRoot, "go.mod"), []byte("module bogus\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if _, err := BuildBinary(context.Background(), bogusRoot, t.TempDir()); err == nil {
		t.Fatal("expected BuildBinary to fail for a module with no ./cmd/aiwf package")
	}
}

// TestBuildBinary_RejectsRelativeOutDir confirms a relative outDir is
// rejected up front rather than silently producing a path that's only
// valid relative to moduleRoot's directory instead of the caller's.
func TestBuildBinary_RejectsRelativeOutDir(t *testing.T) {
	t.Parallel()

	root := repoRootRelative
	_, err := BuildBinary(context.Background(), root, "relative/out/dir")
	if err == nil {
		t.Fatal("expected BuildBinary to reject a relative outDir")
	}
}

// repoRootRelative is the module root relative to this test binary's
// working directory. This file always lives at internal/stresstest/,
// a fixed two levels below the repo root — unlike
// internal/cli/cliutil/testutil.repoRootForTest's general upward
// search (built for helpers called from many depths), a plain
// relative path is enough here.
const repoRootRelative = "../.."
