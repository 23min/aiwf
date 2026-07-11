package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestReallocateEpic_RefusesStagedEditNestedInMilestone reproduces
// G-0377 through the real `aiwf reallocate` CLI path — not the
// synthetic verb.Apply() Plan the internal/verb unit tests drive.
// Reallocating an epic moves its directory, which contains a nested
// milestone file (the same shape as
// TestReallocateScenarios_AC1_HistoricalCorpus's scenario 6). A staged
// edit to that milestone file, made before the reallocate runs, must
// refuse the reallocate rather than being silently swept into the
// reallocate's commit — checkStagedConflict's guard must see the
// staged path even though it is nested under the epic directory, not
// the directory path itself.
func TestReallocateEpic_RefusesStagedEditNestedInMilestone(t *testing.T) {
	t.Parallel()
	env := newScenarioEnv(t)

	env.MustRunBin("add", "epic", "--title", "Sample epic")
	env.MustRunBin("add", "milestone", "--epic", "E-0001", "--tdd", "advisory", "--title", "Sample milestone")

	milestonePath := findEntityFile(t, env, "M-0001")
	if milestonePath == "" || !strings.Contains(milestonePath, "E-0001-") {
		t.Fatalf("M-0001 should live inside E-0001's directory pre-reallocate; got %q", milestonePath)
	}

	// User stages an edit to the nested milestone file, unaware a
	// reallocate is about to move its containing epic directory.
	fullPath := filepath.Join(env.Root, milestonePath)
	original, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("read milestone file: %v", err)
	}
	userContent := append(append([]byte{}, original...), []byte("\nuser's staged note\n")...)
	if writeErr := os.WriteFile(fullPath, userContent, 0o644); writeErr != nil {
		t.Fatalf("write staged edit: %v", writeErr)
	}
	env.MustRunGit("add", milestonePath)

	out, err := testutil.RunBin(t, env.Root, env.BinDir, nil, "reallocate", "E-0001")
	if err == nil {
		t.Fatalf("expected `aiwf reallocate E-0001` to refuse on a staged edit nested inside the moved directory; got success:\n%s", out)
	}
	if !strings.Contains(out, "pre-staged") {
		t.Errorf("refusal output should explain the pre-staged conflict:\n%s", out)
	}
	if !strings.Contains(out, milestonePath) {
		t.Errorf("refusal output should name the conflicting nested path %q:\n%s", milestonePath, out)
	}

	// Filesystem invariant: the epic must not have moved.
	if !fileExists(t, env, "E-0001") {
		t.Error("E-0001 was moved despite the refusal")
	}
	if fileExists(t, env, "E-0002") {
		t.Error("E-0002 (the reallocate destination) exists despite the refusal")
	}

	// The user's staged content must survive untouched.
	gotStaged, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("re-read milestone file: %v", err)
	}
	if !bytes.Equal(gotStaged, userContent) {
		t.Errorf("reallocate touched the user's staged nested file despite the refusal: got %q, want %q", gotStaged, userContent)
	}
}
