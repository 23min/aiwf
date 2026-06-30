package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadM0215Spec resolves the M-0215 spec via the tree loader (not a
// hardcoded path) so it survives an archive sweep, per the repo's
// "policy tests must resolve via the loader" rule.
func loadM0215Spec(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("M-0215")
	if e == nil {
		t.Fatal("entity M-0215 not found in tree")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("loading %s: %v", e.Path, err)
	}
	return string(data)
}

// TestM0215_AC1_BaselineRecordsCheckProfile asserts AC-1: the M-0215
// §Validation section records the `aiwf check` CPU profile (the
// subprocess-wait-bound finding with its utilization figure) and the
// git-subprocess attribution (total spawn count + the merge-base
// fan-out). Structural: scoped to §Validation, not a flat file grep.
func TestM0215_AC1_BaselineRecordsCheckProfile(t *testing.T) {
	t.Parallel()
	section := extractMarkdownSection(loadM0215Spec(t), 2, "Validation")
	if section == "" {
		t.Fatal("AC-1: M-0215 spec must have a `## Validation` section")
	}
	// CPU profile: the subprocess-wait-bound conclusion + utilization.
	for _, m := range []string{"5.31%", "subprocess-wait bound", "CPU"} {
		if !strings.Contains(section, m) {
			t.Errorf("AC-1: §Validation must record CPU-profile marker %q", m)
		}
	}
	// git-subprocess attribution: total spawns + the merge-base fan-out.
	for _, m := range []string{"895", "683", "merge-base"} {
		if !strings.Contains(section, m) {
			t.Errorf("AC-1: §Validation must record subprocess marker %q", m)
		}
	}
}

// TestM0215_AC2_BaselineRecordsPoliciesTiming asserts AC-2: the M-0215
// §Validation section records the internal/policies per-test wall-time
// snapshot ranking the floor-gating tests. Structural: scoped to
// §Validation.
func TestM0215_AC2_BaselineRecordsPoliciesTiming(t *testing.T) {
	t.Parallel()
	section := extractMarkdownSection(loadM0215Spec(t), 2, "Validation")
	if section == "" {
		t.Fatal("AC-2: M-0215 spec must have a `## Validation` section")
	}
	for _, m := range []string{"internal/policies", "TestM0162_AC2_BuildTagExclusion", "4.7s"} {
		if !strings.Contains(section, m) {
			t.Errorf("AC-2: §Validation must record policies-timing marker %q", m)
		}
	}
}
