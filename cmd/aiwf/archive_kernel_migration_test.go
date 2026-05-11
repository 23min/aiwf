package main

// archive_kernel_migration_test.go — M-0085 AC-7 binary-level
// integration test: the first `aiwf archive --apply` against a copy
// of the kernel's actual `work/` + `docs/adr/` tree (the historical
// migration per ADR-0004 §"Migration") leaves `aiwf check` with 0
// error-severity findings.
//
// Per CLAUDE.md "Test the seam, not just the layer": this test runs
// the actual built binary as a subprocess against a real fixture
// (the kernel's own planning tree, copied into a temp dir so the
// production tree is never mutated under test). Unit tests of
// `verb.Archive` exercise the helper; this test exercises the
// end-to-end seam from `os.Args` through the verb's commit through
// post-sweep validation.
//
// Cost: ~3-5s for the binary build + a multi-hundred-entity walk.
// Gated on `-short` per the rest of the binary-integration suite.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBinary_ArchiveKernelMigration_LeavesCheckClean is AC-7's
// binary-level proof: the first `aiwf archive --apply` against a
// faithful copy of the kernel's planning tree produces a single
// commit, sweeps every terminal-status entity into archive/, and
// leaves `aiwf check` with no error-severity findings.
func TestBinary_ArchiveKernelMigration_LeavesCheckClean(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp /* no ldflags */)

	// Locate the kernel's tree by walking up from this test file's
	// runtime directory until a `go.mod` is found. (The test runs
	// from the package dir; the kernel root is the parent of
	// cmd/aiwf/.)
	kernelRoot := findKernelRoot(t)

	// Stage a temp-dir copy of the kernel's `work/`, `docs/adr/`, and
	// `aiwf.yaml` so the verb has a real tree to sweep. Anything
	// outside those paths (Go source, .git, hooks) is irrelevant —
	// the verb only touches entity files.
	repo := t.TempDir()
	mustCopyDir(t, filepath.Join(kernelRoot, "work"), filepath.Join(repo, "work"))
	mustCopyDir(t, filepath.Join(kernelRoot, "docs", "adr"), filepath.Join(repo, "docs", "adr"))
	mustCopyFile(t, filepath.Join(kernelRoot, "aiwf.yaml"), filepath.Join(repo, "aiwf.yaml"))

	mustExec(t, repo, "git", "init", "-q")
	mustExec(t, repo, "git", "config", "user.email", "test@example.com")
	mustExec(t, repo, "git", "config", "user.name", "aiwf-archive-test")
	mustExec(t, repo, "git", "add", "-A")
	mustExec(t, repo, "git", "commit", "-q", "-m", "seed kernel-tree copy for AC-7 migration test")

	// Pre-sweep: confirm aiwf check finds the expected drift —
	// terminal-entity-not-archived warnings + the aggregate. This
	// is the load-bearing input invariant of AC-7: the test asserts
	// the migration's effect, not just that the verb runs. `aiwf
	// check` exits 0 when only warnings fire (per cmd/aiwf/main.go:
	// HasErrors gates the exit code), so we don't gate on the exit
	// status — the output content is the assertion.
	preOut, _ := runBinary(bin, "check", "--root", repo)
	if !strings.Contains(preOut, "archive-sweep-pending") &&
		!strings.Contains(preOut, "terminal-entity-not-archived") {
		t.Fatalf("pre-sweep aiwf check did not surface terminal-entity-not-archived or archive-sweep-pending — fixture not in expected pre-sweep state\noutput:\n%s", preOut)
	}

	// Run the archive verb.
	archOut, archErr := runBinary(bin, "archive", "--apply", "--root", repo, "--actor", "human/test")
	if archErr != nil {
		t.Fatalf("aiwf archive --apply failed: %v\noutput:\n%s", archErr, archOut)
	}

	// Single commit produced. ADR-0004 §"`aiwf archive` verb" + kernel
	// principle #7: one verb invocation = one commit.
	commitCountBefore := 1 // the seed commit we made above
	commitCountAfter := commitCountInRepo(t, repo)
	if delta := commitCountAfter - commitCountBefore; delta != 1 {
		t.Errorf("archive --apply produced %d commit(s), want exactly 1\narchive output:\n%s", delta, archOut)
	}

	// Post-sweep: aiwf check has 0 error-severity findings. Warnings
	// may remain (e.g., the kernel's gap-resolved-has-resolver
	// finding on G-0093); this AC scopes to errors only.
	postOut, postErr := runBinary(bin, "check", "--root", repo)
	// Exit 0 means clean; exit 1 means findings (warnings or errors).
	// We need to distinguish: parse the output for "X errors" and
	// require it to be 0. The verbose output line ends with
	// `N findings (E errors, W warnings)` or `ok — no findings`.
	if postErr == nil {
		// All clean — including warnings. That's the strongest
		// possible outcome; AC-7 is satisfied.
		return
	}
	if !exitedWithCode(postErr, 1) {
		t.Fatalf("aiwf check exited with unexpected code: %v\noutput:\n%s", postErr, postOut)
	}
	// Exit 1: parse the summary line to confirm 0 errors.
	if !checkOutputHasZeroErrors(postOut) {
		t.Fatalf("post-sweep aiwf check has error-severity findings (AC-7 requires 0):\n%s", postOut)
	}
}

// findKernelRoot walks upward from the test working directory until a
// directory containing both `go.mod` and `aiwf.yaml` is found. The
// caller uses this as the source of truth for the kernel's planning
// tree.
func findKernelRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		_, modErr := os.Stat(filepath.Join(dir, "go.mod"))
		_, yamlErr := os.Stat(filepath.Join(dir, "aiwf.yaml"))
		if modErr == nil && yamlErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find kernel root (go.mod + aiwf.yaml) walking up from %s", dir)
	return "" //coverage:ignore unreachable: t.Fatalf above terminates
}

// mustCopyDir recursively copies srcDir to dstDir. Existing files in
// dstDir are overwritten. Symlinks are followed (the kernel tree has
// none worth preserving). Used by the AC-7 test to stage a faithful
// copy of the kernel's planning tree without touching the original.
func mustCopyDir(t *testing.T, srcDir, dstDir string) {
	t.Helper()
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(srcDir, path)
		if relErr != nil {
			return relErr
		}
		dst := filepath.Join(dstDir, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		return copyFileBytes(path, dst)
	})
	if err != nil {
		t.Fatalf("copy %s -> %s: %v", srcDir, dstDir, err)
	}
}

// mustCopyFile is the single-file analog of mustCopyDir.
func mustCopyFile(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	if err := copyFileBytes(src, dst); err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}

// copyFileBytes is a tiny helper: read all of src, write to dst.
// Adequate for the small markdown files the kernel tree carries; not
// optimized for large binaries.
func copyFileBytes(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// commitCountInRepo asks raw git for HEAD's commit count in repo.
// Independent of the binary under test, so the assertion doesn't
// circle back through aiwf's own surface.
func commitCountInRepo(t *testing.T, repo string) int {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "rev-list", "--count", "HEAD").Output()
	if err != nil {
		t.Fatalf("git rev-list --count HEAD: %v", err)
	}
	n := 0
	for _, r := range strings.TrimSpace(string(out)) {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}

// checkOutputHasZeroErrors parses `aiwf check` output and reports
// whether the summary line ends with "(0 errors, ...)". Two surfaces:
// the `ok — no findings` line (zero everything) and the
// `N findings (E errors, W warnings)` line. We inspect the `errors,`
// portion via substring search since the format is stable.
func checkOutputHasZeroErrors(out string) bool {
	if strings.Contains(out, "no findings") {
		return true
	}
	return strings.Contains(out, "(0 errors")
}
