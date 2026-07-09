package stresstest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// disk_fault.go — M-0242/AC-4: DiskFaultScenario proves a
// permission-denied write surfaces as a clean, wrapped error — not a
// corrupted file, not a panic. It seeds one gap entity, then revokes
// write permission on its parent directory (a fixture directory, per
// the AC's own text; simulating a real disk-full condition needs
// privileged filesystem-quota setup this harness doesn't have, so
// permission-denied is the portable fault this scenario exercises —
// documented in the milestone spec's Reviewer notes, not a silent
// scope narrowing).
//
// permissionDeniedDirMode matches the precedent already established
// in internal/verb/apply_test.go: read+execute (list, traverse) but
// no write (create/unlink) — enough for aiwf's tree-load read step to
// still succeed while the later write fails.
const permissionDeniedDirMode = 0o500

// DiskFaultScenario implements Scenario.
type DiskFaultScenario struct {
	aiwfBin    string
	violations []Violation
}

// NewDiskFaultScenario builds a scenario driving aiwfBin (the real
// compiled aiwf binary) against a permission-denied fixture directory.
func NewDiskFaultScenario(aiwfBin string) *DiskFaultScenario {
	return &DiskFaultScenario{aiwfBin: aiwfBin}
}

// Setup git-inits dir and seeds one gap entity via a real `aiwf add`.
func (s *DiskFaultScenario) Setup(dir string) error {
	if err := gitInitAndConfig(dir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}
	addEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "gap", "--title", "diskfault", "--body", "seed for the disk-fault scenario")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("seeding gap: %w", err)
	}
	if addEnv.Status != "ok" {
		return fmt.Errorf("seeding gap: aiwf did not report ok (status=%s, error=%+v)", addEnv.Status, addEnv.Error)
	}
	return nil
}

// Run revokes write permission on the seeded gap's parent directory,
// attempts a promote against it, restores permissions so cleanup can
// proceed regardless of outcome, and classifies the result.
func (s *DiskFaultScenario) Run(dir string) error {
	const id = "G-0001"
	gapsDir := filepath.Join(dir, "work", "gaps")

	beforeBytes, err := readGapFile(dir, id)
	if err != nil {
		return fmt.Errorf("reading pre-attempt bytes: %w", err)
	}
	beforeCommits, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: git rev-list on a repo this scenario itself just created and is still driving has no realistic failure mode
		return fmt.Errorf("counting commits before the attempt: %w", err)
	}

	if chmodErr := os.Chmod(gapsDir, permissionDeniedDirMode); chmodErr != nil { //coverage:ignore defensive: chmod on a directory this scenario itself just created has no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("revoking write permission on %s: %w", gapsDir, chmodErr)
	}
	// Safety net for any early return below — restoring here explicitly
	// (not relying on this defer alone) is what lets the subsequent
	// afterBytes read below see a readable directory before Run returns.
	defer func() { _ = os.Chmod(gapsDir, 0o755) }()

	env, err := runAiwfJSON(s.aiwfBin, dir, "promote", id, "wontfix")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("running the fault-injected promote: %w", err)
	}

	afterCommits, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: see the "before" call above
		return fmt.Errorf("counting commits after the attempt: %w", err)
	}
	strayTempFiles, err := globTempFiles(gapsDir)
	if err != nil { //coverage:ignore defensive: filepath.Glob's own literal pattern is well-formed by construction (see readGapFile's identical rationale)
		return fmt.Errorf("checking for stray temp files: %w", err)
	}

	if chmodErr := os.Chmod(gapsDir, 0o755); chmodErr != nil { //coverage:ignore defensive: chmod restoring a directory this scenario itself owns has no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("restoring write permission on %s: %w", gapsDir, chmodErr)
	}
	afterBytes, err := readGapFile(dir, id)
	if err != nil { //coverage:ignore defensive: readGapFile's own mismatch/glob branches are pinned directly at their source (TestReadGapFile_ErrorsWhenNoneOrMultipleMatch); reaching this specific call site requires the gap file to vanish or duplicate strictly between the promote attempt above and this read, a window no external black-box test can arrange without instrumenting Run itself
		return fmt.Errorf("reading post-attempt bytes: %w", err)
	}

	s.violations = classifyDiskFaultOutcome(diskFaultOutcome{
		env:            env,
		beforeBytes:    beforeBytes,
		afterBytes:     afterBytes,
		beforeCommits:  beforeCommits,
		afterCommits:   afterCommits,
		strayTempFiles: strayTempFiles,
	})
	return nil
}

// Verify returns the violations Run collected.
func (s *DiskFaultScenario) Verify(_ string) []Violation {
	return s.violations
}

// globTempFiles returns any pathutil.AtomicWriteFile sibling temp
// file (the ".aiwf-tmp-" naming convention its own doc comment
// documents — no code change, per M-0242/AC-3) left behind in dir.
func globTempFiles(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.aiwf-tmp-*"))
	if err != nil { //coverage:ignore defensive: the only error filepath.Glob returns is ErrBadPattern, and this package's own literal pattern is well-formed by construction
		return nil, fmt.Errorf("globbing %s for temp files: %w", dir, err)
	}
	return matches, nil
}

// diskFaultOutcome is the raw evidence classifyDiskFaultOutcome
// judges.
type diskFaultOutcome struct {
	env                         verbEnvelope
	beforeBytes, afterBytes     []byte
	beforeCommits, afterCommits int
	strayTempFiles              []string
}

// classifyDiskFaultOutcome judges one fault-injected write attempt.
// Every check runs independently — a broken run can fail more than
// one at once, and each is reported rather than short-circuited.
func classifyDiskFaultOutcome(o diskFaultOutcome) []Violation {
	var violations []Violation
	if o.env.Status != "ok" {
		if o.env.Error == nil {
			violations = append(violations, Violation{Message: "aiwf reported a non-ok status with no error payload — a malformed envelope"})
		} else if strings.Contains(o.env.Error.Message, "panic:") || strings.Contains(o.env.Error.Message, "goroutine ") {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"the reported error looks like a Go panic/stack trace, not a clean wrapped error: %q", o.env.Error.Message)})
		}
	} else {
		violations = append(violations, Violation{Message: "expected the fault-injected write to be refused, but aiwf reported ok"})
	}
	if !bytes.Equal(o.afterBytes, o.beforeBytes) {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"entity file changed despite the write being refused (before %d bytes, after %d bytes) — corruption", len(o.beforeBytes), len(o.afterBytes))})
	}
	if len(o.strayTempFiles) > 0 {
		violations = append(violations, Violation{Message: fmt.Sprintf("stray temp file(s) left behind: %v", o.strayTempFiles)})
	}
	if o.afterCommits != o.beforeCommits {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"refused write still landed a commit (%d -> %d)", o.beforeCommits, o.afterCommits)})
	}
	return violations
}
