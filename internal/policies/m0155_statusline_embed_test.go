package policies

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// TestM0155_AC1_StatuslineEmbedded asserts M-0155/AC-1: the
// aiwf-aware Claude Code statusline script is embedded in the aiwf
// binary via go:embed, exposed via `skills.StatuslineBytes()`, and its
// content is byte-equal to the canonical `.claude/statusline.sh`.
//
// The drift assertion is the load-bearing one — it makes the rest of
// the milestone safe: M-0155's scaffold-if-absent write path
// (AC-3..AC-5) materializes `skills.StatuslineBytes()` to disk, so if
// the embed silently lags the canonical script, every consumer's
// scaffold gets a stale copy with no warning. The test catches that
// regression at CI time. Operators who edit `.claude/statusline.sh`
// must keep `internal/skills/embedded-statusline/statusline.sh` in
// sync (a follow-up could land a `make sync-statusline` target if the
// manual sync becomes friction).
//
// Two assertions:
//
//   - Presence: `skills.StatuslineBytes()` returns a non-empty slice
//     (the embed directive is wired and the source file is non-empty).
//   - Drift: the embed bytes match the canonical `.claude/statusline.sh`
//     byte-for-byte. Any divergence — whether the canonical was edited
//     without re-sync or the embed was edited without re-syncing the
//     canonical — fails this test with a clear remediation hint.
func TestM0155_AC1_StatuslineEmbedded(t *testing.T) {
	t.Parallel()
	embedded := skills.StatuslineBytes()
	if len(embedded) == 0 {
		t.Fatal("AC-1: skills.StatuslineBytes() returned empty — the go:embed directive is not wired or the source file is empty")
	}

	root := repoRoot(t)
	canonical, err := os.ReadFile(filepath.Join(root, ".claude", "statusline.sh"))
	if err != nil {
		t.Fatalf("AC-1: reading canonical .claude/statusline.sh: %v", err)
	}

	if !bytes.Equal(embedded, canonical) {
		t.Errorf("AC-1: embedded statusline (%d bytes) drifted from canonical .claude/statusline.sh (%d bytes); re-sync by copying `.claude/statusline.sh` to `internal/skills/embedded-statusline/statusline.sh`",
			len(embedded), len(canonical))
	}
}
