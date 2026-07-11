package stresstest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// disk_fault_test.go — real-subprocess coverage for DiskFaultScenario
// (M-0242/AC-4). The pure decision logic (classifyDiskFaultOutcome) is
// pinned exhaustively in disk_fault_classify_test.go against
// fabricated outcomes; this is the actual AC-4 scenario, driving a
// real aiwf subprocess against a real permission-denied fixture.

func TestDiskFaultScenario_RealBinary_ConfirmsCleanRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewDiskFaultScenario(bin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("disk-fault scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
	}
}

func TestDiskFaultScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewDiskFaultScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"))
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "seeding gap") {
		t.Fatalf("expected the launch failure to name the seeding step, got: %v", err)
	}
}

// TestDiskFaultScenario_RealBinary_RunErrorsWhenGapFileMissing deletes
// the seeded gap file after a successful Setup, pinning Run's own
// initial readGapFile call (reading the pre-attempt bytes) rather
// than readGapFile's already-unit-tested internals.
func TestDiskFaultScenario_RealBinary_RunErrorsWhenGapFileMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewDiskFaultScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "work", "gaps", "G-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected exactly one seeded gap file, got %v (err=%v)", matches, err)
	}
	if err := os.Remove(matches[0]); err != nil {
		t.Fatalf("removing seeded gap file: %v", err)
	}

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to error when the seeded gap file is missing")
	} else if !strings.Contains(err.Error(), "reading pre-attempt bytes") {
		t.Fatalf("expected the error to name the pre-attempt read step, got: %v", err)
	}
}

// TestGlobTempFiles_RealBinary pins globTempFiles' two direct
// outcomes: no matches in a clean directory, and a match once a
// temp-shaped file is present.
func TestGlobTempFiles_RealBinary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	matches, err := globTempFiles(dir)
	if err != nil {
		t.Fatalf("globTempFiles on a clean dir: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no matches in a clean dir, got %v", matches)
	}

	if writeErr := os.WriteFile(filepath.Join(dir, "entity.md.aiwf-tmp-12345"), []byte("x"), 0o644); writeErr != nil {
		t.Fatalf("seeding a temp file: %v", writeErr)
	}
	matches, err = globTempFiles(dir)
	if err != nil {
		t.Fatalf("globTempFiles with a temp file present: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one match, got %v", matches)
	}
}

// TestDiskFaultScenario_RealBinary_SetupSurfacesASeedingRefusal
// pre-seeds a colliding G-0001 entity file (an id collision the
// ids-unique rule refuses at error severity, mirroring M-0241/AC-5's
// same pre-seed technique) so Setup's `add gap` call reports
// something other than "ok", pinning that Setup wraps and surfaces
// the refusal.
func TestDiskFaultScenario_RealBinary_SetupSurfacesASeedingRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	gapsDir := filepath.Join(dir, "work", "gaps")
	if err := os.MkdirAll(gapsDir, 0o755); err != nil {
		t.Fatalf("mkdir colliding gap dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gapsDir, "G-0001-collision.md"), []byte("not valid frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write colliding gap file: %v", err)
	}

	s := NewDiskFaultScenario(bin)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to surface the id-collision refusal from the `aiwf add` call")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to be reported as a non-ok status, got: %v", err)
	}
}

// TestDiskFaultScenario_RealBinary_RunLeavesTheDirWritable confirms
// Run restores the gaps directory's permissions even though the
// scenario's own oracle is byte/commit-based, not permission-based —
// a passing scenario must be fully cleanable by RunScenario's own
// os.RemoveAll, which needs write access on every constituent dir.
func TestDiskFaultScenario_RealBinary_RunLeavesTheDirWritable(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewDiskFaultScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := s.Run(dir); err != nil {
		t.Fatalf("Run: %v", err)
	}

	gapsDir := filepath.Join(dir, "work", "gaps")
	info, err := os.Stat(gapsDir)
	if err != nil {
		t.Fatalf("stat gapsDir: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o200 == 0 {
		t.Fatalf("expected gapsDir to be writable again after Run, got mode %v", info.Mode().Perm())
	}
}
