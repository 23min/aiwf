package initrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// TestInit_WiresGuidanceImport: `aiwf init` wires the marker-wrapped
// `@.claude/aiwf-guidance.md` import line into CLAUDE.md by default
// (M-0164/AC-1).
func TestInit_WiresGuidanceImport(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(data), "@.claude/aiwf-guidance.md") {
		t.Errorf("AC-1: CLAUDE.md missing guidance import line; got:\n%s", data)
	}
	// AC-4: the wiring is announced as a ledger step (which init/update
	// print), and the notice names the opt-out.
	step := findStep(t, res.Steps, "CLAUDE.md (aiwf guidance import)")
	if !strings.Contains(step.Detail, "--no-wire-claudemd") {
		t.Errorf("AC-4: import notice should name the opt-out; got Detail=%q", step.Detail)
	}
}

// TestInit_NoWireClaudeMd_SkipsImport: --no-wire-claudemd opts out of
// wiring the import (M-0164/AC-1).
func TestInit_NoWireClaudeMd_SkipsImport(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{NoWireClaudeMd: true}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if strings.Contains(string(data), "@.claude/aiwf-guidance.md") {
		t.Errorf("AC-1: --no-wire-claudemd should skip wiring; got:\n%s", data)
	}
}

// --- direct ensureGuidanceImport branch coverage (M-0164/AC-2..AC-6) ---

func writeClaudeMd(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing CLAUDE.md: %v", err)
	}
}

func readClaudeMd(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	return string(b)
}

// AC-1 opt-out at the function level: no write, skipped+declined.
func TestEnsureGuidanceImport_NoWire(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	st, err := ensureGuidanceImport(root, RefreshOptions{NoWireClaudeMd: true, WireClaudeMdIfAbsent: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionSkipped || !strings.Contains(st.Detail, "--no-wire-claudemd") {
		t.Errorf("expected skipped+declined, got %+v", st)
	}
	if _, statErr := os.Stat(filepath.Join(root, "CLAUDE.md")); !os.IsNotExist(statErr) {
		t.Error("--no-wire-claudemd must not create CLAUDE.md")
	}
}

// AC-2 (created-if-absent) + AC-5 (import resolves to the materialized file)
// + AC-4 (notice names the opt-out).
func TestEnsureGuidanceImport_CreatesWhenAbsent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionCreated {
		t.Errorf("AC-2: expected created, got %q", st.Action)
	}
	got := readClaudeMd(t, root)
	if want := "@" + skills.GuidanceFile; !strings.Contains(got, want) {
		t.Errorf("AC-5: import line %q not present in:\n%s", want, got)
	}
	if !strings.Contains(st.Detail, "--no-wire-claudemd") {
		t.Errorf("AC-4: notice must name the opt-out; got Detail=%q", st.Detail)
	}
}

// AC-2: existing user content outside the markers is preserved.
func TestEnsureGuidanceImport_PreservesOutsideContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeClaudeMd(t, root, "# My project\n\nSome notes.\n")
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true}); err != nil {
		t.Fatal(err)
	}
	got := readClaudeMd(t, root)
	if !strings.Contains(got, "# My project") || !strings.Contains(got, "Some notes.") {
		t.Errorf("AC-2: user content not preserved:\n%s", got)
	}
	if !strings.Contains(got, "@"+skills.GuidanceFile) {
		t.Errorf("AC-2: block not appended:\n%s", got)
	}
}

// AC-3: re-running with the block present is idempotent (no diff).
func TestEnsureGuidanceImport_Idempotent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeClaudeMd(t, root, "# u\n")
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true}); err != nil {
		t.Fatal(err)
	}
	first := readClaudeMd(t, root)
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionPreserved {
		t.Errorf("AC-3: second run should be preserved, got %q", st.Action)
	}
	if readClaudeMd(t, root) != first {
		t.Error("AC-3: not idempotent across re-run")
	}
}

// AC-3 (refresh) + AC-2 (preserve on refresh): a stale block in place is
// refreshed to the canonical import without touching outside content.
func TestEnsureGuidanceImport_RefreshesStaleBlock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	stale := "# u\n\n" + guidanceImportStartMarker + "\n@.claude/OLD.md\n" + guidanceImportEndMarker + "\n"
	writeClaudeMd(t, root, stale)
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: false})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionUpdated {
		t.Errorf("AC-3: stale block should refresh to updated, got %q", st.Action)
	}
	got := readClaudeMd(t, root)
	if strings.Contains(got, "@.claude/OLD.md") || !strings.Contains(got, "@"+skills.GuidanceFile) {
		t.Errorf("AC-3: stale import not refreshed:\n%s", got)
	}
	if !strings.Contains(got, "# u") {
		t.Errorf("AC-2: outside content lost on refresh:\n%s", got)
	}
}

// AC-3 (nudge): on update, an absent block is reported, not re-added.
func TestEnsureGuidanceImport_UpdateNudgesWhenAbsent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeClaudeMd(t, root, "# u, block was removed\n")
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: false})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionSkipped || !strings.Contains(st.Detail, "not re-adding") {
		t.Errorf("AC-3: update should nudge, not re-add; got %+v", st)
	}
	if strings.Contains(readClaudeMd(t, root), "@"+skills.GuidanceFile) {
		t.Error("AC-3: update must not re-add a removed block")
	}
}

// AC-6: a damaged marker pair (only one of START/END) is refused and the
// file is left untouched.
func TestEnsureGuidanceImport_DamagedMarkerRefused(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	damaged := "# u\n\n" + guidanceImportStartMarker + "\n@.claude/aiwf-guidance.md\n" // START, no END
	writeClaudeMd(t, root, damaged)
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionSkipped || !strings.Contains(st.Detail, "damaged") {
		t.Errorf("AC-6: damaged marker should be refused; got %+v", st)
	}
	if readClaudeMd(t, root) != damaged {
		t.Error("AC-6: damaged-marker file must be left untouched")
	}
}

// Branch coverage: content without a trailing newline takes the
// double-newline separator arm when appending the block.
func TestEnsureGuidanceImport_NoTrailingNewline(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeClaudeMd(t, root, "# no newline at end")
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true}); err != nil {
		t.Fatal(err)
	}
	got := readClaudeMd(t, root)
	if !strings.Contains(got, "# no newline at end") || !strings.Contains(got, "@"+skills.GuidanceFile) {
		t.Errorf("content+block not both present:\n%s", got)
	}
}

// Branch coverage: a read error that is not fs.ErrNotExist (here CLAUDE.md
// is a directory) propagates.
func TestEnsureGuidanceImport_ReadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "CLAUDE.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true}); err == nil {
		t.Error("expected a read error when CLAUDE.md is a directory")
	}
}

// Branch coverage: dry-run reports the action but writes nothing (add path).
func TestEnsureGuidanceImport_DryRunAddDoesNotWrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionCreated {
		t.Errorf("dry-run add should still report Created, got %q", st.Action)
	}
	if _, statErr := os.Stat(filepath.Join(root, "CLAUDE.md")); !os.IsNotExist(statErr) {
		t.Error("dry-run must not write CLAUDE.md")
	}
}

// Branch coverage: dry-run reports the refresh but writes nothing.
func TestEnsureGuidanceImport_DryRunRefreshDoesNotWrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	stale := guidanceImportStartMarker + "\n@.claude/OLD.md\n" + guidanceImportEndMarker + "\n"
	writeClaudeMd(t, root, stale)
	st, err := ensureGuidanceImport(root, RefreshOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionUpdated {
		t.Errorf("dry-run refresh should report Updated, got %q", st.Action)
	}
	if readClaudeMd(t, root) != stale {
		t.Error("dry-run must not rewrite CLAUDE.md")
	}
}

// Branch coverage: reversed markers (END before START) can't be safely
// parsed by replaceGuidanceBlock — left untouched, not corrupted.
func TestEnsureGuidanceImport_ReversedMarkers(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	reversed := guidanceImportEndMarker + "\nstuff\n" + guidanceImportStartMarker + "\n"
	writeClaudeMd(t, root, reversed)
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true}); err != nil {
		t.Fatal(err)
	}
	if readClaudeMd(t, root) != reversed {
		t.Errorf("reversed markers must be left untouched; got:\n%s", readClaudeMd(t, root))
	}
}

// Branch coverage: AtomicWriteFile error in the add path (read-only root).
func TestEnsureGuidanceImport_AddWriteError(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMdIfAbsent: true}); err == nil {
		t.Error("expected a write error into a read-only root")
	}
}

// Branch coverage: AtomicWriteFile error in the refresh path (read-only root).
func TestEnsureGuidanceImport_RefreshWriteError(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := guidanceImportStartMarker + "\n@.claude/OLD.md\n" + guidanceImportEndMarker + "\n"
	writeClaudeMd(t, root, stale)
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	if _, err := ensureGuidanceImport(root, RefreshOptions{}); err == nil {
		t.Error("expected a write error refreshing into a read-only root")
	}
}
