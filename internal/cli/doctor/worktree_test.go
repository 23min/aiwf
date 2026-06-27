package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
)

// worktreeDirLine returns the `worktree-dir:` line from doctor output, or
// fails. Scoping the assertion to that one line keeps it structural rather
// than a flat substring grep over the whole report.
func worktreeDirLine(t *testing.T, lines []string) string {
	t.Helper()
	for _, l := range lines {
		if strings.HasPrefix(l, "worktree-dir:") {
			return l
		}
	}
	t.Fatalf("no `worktree-dir:` line in doctor output:\n%s", strings.Join(lines, "\n"))
	return ""
}

// TestDoctorReport_WorktreeDirDefault — M-0189 AC-3. With no worktree.dir
// configured, doctor surfaces the kernel default, annotated `(default)`, as
// a greppable line the M-0190 ritual reads.
func TestDoctorReport_WorktreeDirDefault(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, _ := DoctorReport(root, DoctorOptions{})
	got := worktreeDirLine(t, lines)
	if !strings.Contains(got, config.DefaultWorktreeDir) || !strings.Contains(got, "(default)") {
		t.Errorf("worktree-dir line = %q, want it to contain %q and \"(default)\"", got, config.DefaultWorktreeDir)
	}
}

// TestDoctorReport_WorktreeDirConfigured — M-0189 AC-3. A configured
// worktree.dir is surfaced verbatim, annotated `(configured)`, so the
// M-0190 ritual reads the consumer's override rather than the default.
func TestDoctorReport_WorktreeDirConfigured(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("worktree:\n  dir: .wt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, _ := DoctorReport(root, DoctorOptions{})
	got := worktreeDirLine(t, lines)
	if !strings.Contains(got, ".wt") || !strings.Contains(got, "(configured)") {
		t.Errorf("worktree-dir line = %q, want it to contain %q and \"(configured)\"", got, ".wt")
	}
}
