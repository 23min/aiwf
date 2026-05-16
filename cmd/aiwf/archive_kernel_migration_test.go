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
	t.Parallel()
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

	// Synthesize a terminal-active gap inside the tempdir so the
	// test's premise (the tree has something to sweep) is
	// guaranteed regardless of the live kernel tree's housekeeping
	// state. See G-0106 for the design discussion — without this,
	// the test silently became a no-op every time `aiwf archive
	// --apply` was run against the kernel tree, and noisily failed
	// when it sat clean. The synthetic gap uses a high-numbered
	// sentinel id (G-9999) that's far above current allocation and
	// only lives inside this tempdir, so there's no collision risk
	// with the live kernel tree.
	synthGapRel := writeSyntheticTerminalGap(t, repo)
	mustExec(t, repo, "git", "add", "-A")
	mustExec(t, repo, "git", "commit", "-q", "-m", "AC-7 test: seed synthetic terminal-active gap (G-9999)")

	// Structural pre-sweep assertion: the synthesized file exists
	// at the active-tree path. Stronger than substring-matching
	// `aiwf check` output (which rolls up findings per code, so a
	// specific entity's id may not appear in the rendered detail
	// when multiple entities fire the same rule) and survives
	// future renderer-format changes. The verb's effect is verified
	// structurally post-sweep below.
	syntheticActivePath := filepath.Join(repo, synthGapRel)
	if _, err := os.Stat(syntheticActivePath); err != nil {
		t.Fatalf("pre-sweep: synthetic G-9999 not found at %s — synthesis failed: %v", synthGapRel, err)
	}

	// Capture commit count immediately before the verb so the
	// "exactly one commit per invocation" assertion is robust to
	// any preparatory commits the test makes upstream (the seed
	// commit + the synthesis commit today; possibly more in the
	// future).
	commitCountBefore := commitCountInRepo(t, repo)

	// Run the archive verb.
	archOut, archErr := runBinary(bin, "archive", "--apply", "--root", repo, "--actor", "human/test")
	if archErr != nil {
		t.Fatalf("aiwf archive --apply failed: %v\noutput:\n%s", archErr, archOut)
	}

	// Single commit produced. ADR-0004 §"`aiwf archive` verb" + kernel
	// principle #7: one verb invocation = one commit.
	commitCountAfter := commitCountInRepo(t, repo)
	if delta := commitCountAfter - commitCountBefore; delta != 1 {
		t.Errorf("archive --apply produced %d commit(s), want exactly 1\narchive output:\n%s", delta, archOut)
	}

	// Post-sweep: the synthesized G-9999 should have moved into
	// work/gaps/archive/. Assert structurally rather than via the
	// check output, so the assertion doesn't drift when the
	// renderer's format changes.
	if _, err := os.Stat(syntheticActivePath); !os.IsNotExist(err) {
		t.Errorf("post-sweep: synthetic G-9999 still present at active-tree path %s — archive verb did not sweep it", synthGapRel)
	}
	archivedSyntheticPath := filepath.Join(repo, "work", "gaps", "archive", filepath.Base(synthGapRel))
	if _, err := os.Stat(archivedSyntheticPath); err != nil {
		t.Errorf("post-sweep: synthetic G-9999 not found at archive path %s — archive verb's move target diverged from work/gaps/archive/: %v", archivedSyntheticPath, err)
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

// writeSyntheticTerminalGap plants a freshly-authored gap entity
// with terminal status inside repo's work/gaps/ directory. Used by
// the AC-7 test to pin the "pre-sweep tree contains an unswept
// terminal" premise independently of the live kernel tree's
// housekeeping state (per G-0106).
//
// The synthetic id (G-9999) is a sentinel high enough above current
// allocation that it will not collide with the live kernel tree
// even on the unlikely contingency that someone copies a test
// artifact into the kernel's planning tree. The slug names the
// intent explicitly so a reader of the post-sweep archive directory
// recognizes the entry as test-synthesized, not historical.
//
// Returns the repo-relative path the helper wrote, for use in
// post-sweep assertions.
func writeSyntheticTerminalGap(t *testing.T, repo string) string {
	t.Helper()
	const id = "G-9999"
	const slug = id + "-synthetic-terminal-anchor-for-ac-7-test"
	relPath := filepath.Join("work", "gaps", slug+".md")
	absDir := filepath.Join(repo, "work", "gaps")
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", absDir, err)
	}
	body := "---\n" +
		"id: " + id + "\n" +
		"title: synthetic-terminal-anchor-for-ac-7-test\n" +
		"status: wontfix\n" +
		"---\n# Problem\n\n" +
		"Synthesized by TestBinary_ArchiveKernelMigration_LeavesCheckClean\n" +
		"to guarantee the kernel-tree copy carries at least one unswept\n" +
		"terminal entity, regardless of the live kernel tree's housekeeping\n" +
		"state. The test pre-sweep premise (and the verb's substantive\n" +
		"path) depend on having something to sweep. See G-0106 for the\n" +
		"original design discussion.\n"
	absPath := filepath.Join(repo, relPath)
	if err := os.WriteFile(absPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write synthetic gap: %v", err)
	}
	return relPath
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
