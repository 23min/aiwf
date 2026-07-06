package cliutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/version"
)

// TestRunStatuslineScaffoldForVersion_UntaggedRefusesWithoutOverride pins
// G-0367: an untagged (dev/worktree) binary must not write the statusline
// script without confirmation. Under `go test` there is no TTY, so the
// interactive prompt path can't fire either — the call must refuse
// (ExitOK, matching the sibling ADR-0015 consent-declined shape) and leave
// no script on disk.
func TestRunStatuslineScaffoldForVersion_UntaggedRefusesWithoutOverride(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	untagged := version.Parse(version.DevelVersion)

	rc := RunStatuslineScaffoldForVersion(StatuslineOpts{
		RootDir: root,
		Scope:   "project",
	}, untagged)
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}

	dest := filepath.Join(root, ".claude", "statusline.sh")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("untagged binary without --allow-untagged-statusline must not write %s; stat err = %v", dest, err)
	}
}

// TestRunStatuslineScaffoldForVersion_UntaggedProceedsWithAllowUntagged
// pins the explicit override: AllowUntagged bypasses the confirmation
// gate and the write proceeds exactly as it would for a tagged binary.
func TestRunStatuslineScaffoldForVersion_UntaggedProceedsWithAllowUntagged(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	untagged := version.Parse(version.DevelVersion)

	rc := RunStatuslineScaffoldForVersion(StatuslineOpts{
		RootDir:       root,
		Scope:         "project",
		AllowUntagged: true,
	}, untagged)
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}

	dest := filepath.Join(root, ".claude", "statusline.sh")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("AllowUntagged=true must write %s; stat err = %v", dest, err)
	}
}

// TestRunStatuslineScaffoldForVersion_TaggedNeedsNoOverride pins the
// unaffected case: a tagged (release) binary writes unconditionally, same
// as before G-0367 — the gate only applies to untagged binaries.
func TestRunStatuslineScaffoldForVersion_TaggedNeedsNoOverride(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tagged := version.Parse("v1.2.3")

	rc := RunStatuslineScaffoldForVersion(StatuslineOpts{
		RootDir: root,
		Scope:   "project",
	}, tagged)
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}

	dest := filepath.Join(root, ".claude", "statusline.sh")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("tagged binary must write %s unconditionally; stat err = %v", dest, err)
	}
}

// TestRunStatuslineScaffold_UsesCurrentBinaryVersion asserts the public
// entry point threads version.Current() into the testable core — a thin
// smoke test since version.Current() under `go test` is itself untagged
// ((devel) or a pseudo-version), so this call takes the same refuse path
// as the explicit-untagged test above.
func TestRunStatuslineScaffold_UsesCurrentBinaryVersion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	rc := RunStatuslineScaffold(StatuslineOpts{
		RootDir: root,
		Scope:   "project",
	})
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
}
