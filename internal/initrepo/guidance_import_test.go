package initrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

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

// TestInit_WiresGuidanceImport: `aiwf init` wires the marker-wrapped
// import automatically (default-on, no flag) and announces it (M-0164/AC-1, AC-4).
func TestInit_WiresGuidanceImport(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !strings.Contains(readClaudeMd(t, root), "@.claude/aiwf-guidance.md") {
		t.Errorf("AC-1: CLAUDE.md missing guidance import line")
	}
	// AC-4: the wiring is announced as a ledger step (which init/update print).
	step := findStep(t, res.Steps, "CLAUDE.md (aiwf guidance import)")
	if step.Action == "" || step.Detail == "" {
		t.Errorf("AC-4: import wiring not announced as a ledger step; got %+v", step)
	}
}

// TestInit_GuidanceOptOutViaConfig: a consumer who sets
// guidance.wire_claudemd: false in aiwf.yaml is not wired (M-0164/AC-1, opt-out).
func TestInit_GuidanceOptOutViaConfig(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("guidance:\n  wire_claudemd: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if strings.Contains(readClaudeMd(t, root), "@.claude/aiwf-guidance.md") {
		t.Errorf("opt-out: CLAUDE.md should not be wired when guidance.wire_claudemd=false")
	}
}

// --- direct ensureGuidanceImport branch coverage (M-0164) ---

// Opt-out at the function level: no write, skipped.
func TestEnsureGuidanceImport_OptOut(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: false})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionSkipped || !strings.Contains(st.Detail, "guidance.wire_claudemd") {
		t.Errorf("expected skipped via config knob, got %+v", st)
	}
	if _, statErr := os.Stat(filepath.Join(root, "CLAUDE.md")); !os.IsNotExist(statErr) {
		t.Error("opt-out must not create CLAUDE.md")
	}
}

// AC-2 (created-if-absent) + AC-5 (import resolves to the materialized file).
func TestEnsureGuidanceImport_CreatesWhenAbsent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionCreated {
		t.Errorf("AC-2: expected created, got %q", st.Action)
	}
	if want := "@" + skills.GuidanceFile; !strings.Contains(readClaudeMd(t, root), want) {
		t.Errorf("AC-5: import line %q not present", want)
	}
}

// AC-2: existing user content outside the markers is preserved.
func TestEnsureGuidanceImport_PreservesOutsideContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeClaudeMd(t, root, "# My project\n\nSome notes.\n")
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true}); err != nil {
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
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true}); err != nil {
		t.Fatal(err)
	}
	first := readClaudeMd(t, root)
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true})
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

// AC-3 (self-heal): a removed block is RE-ADDED on the next run — the
// automagical replacement for the old nudge-not-re-add behavior.
func TestEnsureGuidanceImport_SelfHealsRemovedBlock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true}); err != nil {
		t.Fatal(err)
	}
	// Operator removes the block entirely.
	writeClaudeMd(t, root, "# just my notes\n")
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionUpdated {
		t.Errorf("AC-3 self-heal: expected updated (re-added), got %q", st.Action)
	}
	got := readClaudeMd(t, root)
	if !strings.Contains(got, "@"+skills.GuidanceFile) {
		t.Error("AC-3 self-heal: removed block was not re-added")
	}
	if !strings.Contains(got, "# just my notes") {
		t.Error("AC-3 self-heal: user content lost during re-add")
	}
}

// AC-3 (refresh): a stale block in place is refreshed without touching outside content.
func TestEnsureGuidanceImport_RefreshesStaleBlock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	stale := "# u\n\n" + guidanceImportStartMarker + "\n@.claude/OLD.md\n" + guidanceImportEndMarker + "\n"
	writeClaudeMd(t, root, stale)
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true})
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

// Review-hardening: marker strings inside a PROSE line are inert
// (line-anchored), so user text is never clobbered (ADR-0018 "clobbers nothing").
func TestEnsureGuidanceImport_MarkersInProseAreInert(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	prose := "We wrap regions with " + guidanceImportStartMarker + " then text then " + guidanceImportEndMarker + " inline.\n"
	writeClaudeMd(t, root, prose)
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true}); err != nil {
		t.Fatal(err)
	}
	got := readClaudeMd(t, root)
	if !strings.Contains(got, "then text then") {
		t.Errorf("prose between marker mentions was clobbered:\n%s", got)
	}
	if !strings.Contains(got, "@"+skills.GuidanceFile) {
		t.Errorf("block should have been appended below the prose:\n%s", got)
	}
}

// Review-hardening (F2): a pre-existing bare import line (no markers) is
// wrapped in markers, not duplicated.
func TestEnsureGuidanceImport_WrapsBareImportLine(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeClaudeMd(t, root, "# u\n@.claude/aiwf-guidance.md\n")
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionUpdated {
		t.Errorf("expected updated (wrapped), got %q", st.Action)
	}
	got := readClaudeMd(t, root)
	if n := strings.Count(got, "@.claude/aiwf-guidance.md"); n != 1 {
		t.Errorf("expected exactly one import line after wrapping, got %d:\n%s", n, got)
	}
	if !strings.Contains(got, guidanceImportStartMarker) {
		t.Errorf("bare line not wrapped in markers:\n%s", got)
	}
}

// AC-6: a one-sided (damaged) marker pair is refused and the file is untouched.
func TestEnsureGuidanceImport_DamagedMarkerRefused(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	damaged := "# u\n\n" + guidanceImportStartMarker + "\n@.claude/aiwf-guidance.md\n" // START, no END
	writeClaudeMd(t, root, damaged)
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true})
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

// AC-6: reversed markers (END before START) are also refused, untouched.
func TestEnsureGuidanceImport_ReversedMarkersRefused(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	reversed := guidanceImportEndMarker + "\nstuff\n" + guidanceImportStartMarker + "\n"
	writeClaudeMd(t, root, reversed)
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Action != ActionSkipped || !strings.Contains(st.Detail, "damaged") {
		t.Errorf("reversed markers should be refused; got %+v", st)
	}
	if readClaudeMd(t, root) != reversed {
		t.Errorf("reversed markers must be left untouched; got:\n%s", readClaudeMd(t, root))
	}
}

// Branch coverage: content without a trailing newline takes the
// double-newline separator arm.
func TestEnsureGuidanceImport_NoTrailingNewline(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeClaudeMd(t, root, "# no newline at end")
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true}); err != nil {
		t.Fatal(err)
	}
	got := readClaudeMd(t, root)
	if !strings.Contains(got, "# no newline at end") || !strings.Contains(got, "@"+skills.GuidanceFile) {
		t.Errorf("content+block not both present:\n%s", got)
	}
}

// Branch coverage: a read error that is not fs.ErrNotExist (CLAUDE.md is a dir).
func TestEnsureGuidanceImport_ReadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "CLAUDE.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true}); err == nil {
		t.Error("expected a read error when CLAUDE.md is a directory")
	}
}

// Branch coverage: dry-run reports the action but writes nothing (add path).
func TestEnsureGuidanceImport_DryRunAddDoesNotWrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true, DryRun: true})
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
	st, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true, DryRun: true})
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

// Branch coverage: AtomicWriteFile error in the add path (read-only root).
func TestEnsureGuidanceImport_AddWriteError(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true}); err == nil {
		t.Error("expected a write error into a read-only root")
	}
}

// Branch coverage: loadWireClaudeMd defaults to true when aiwf.yaml is absent.
func TestLoadWireClaudeMd_DefaultsWhenNoConfig(t *testing.T) {
	t.Parallel()
	got, err := loadWireClaudeMd(t.TempDir())
	if err != nil {
		t.Fatalf("loadWireClaudeMd: %v", err)
	}
	if !got {
		t.Error("no aiwf.yaml should default WireClaudeMd to true")
	}
}

// Branch coverage: loadWireClaudeMd propagates a non-ErrNotFound parse error.
func TestLoadWireClaudeMd_PropagatesParseError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("guidance: [not a map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadWireClaudeMd(root); err == nil {
		t.Error("malformed aiwf.yaml should propagate an error")
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
	if _, err := ensureGuidanceImport(root, RefreshOptions{WireClaudeMd: true}); err == nil {
		t.Error("expected a write error refreshing into a read-only root")
	}
}
