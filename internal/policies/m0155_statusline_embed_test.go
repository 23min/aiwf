package policies

import (
	"bytes"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// TestM0155_AC1_StatuslineEmbedded asserts M-0155/AC-1: the aiwf-aware
// Claude Code statusline script is embedded in the aiwf binary and
// exposed via `skills.StatuslineBytes()`.
//
// The embedded snapshot at
// `internal/skills/embedded-statusline/statusline.sh` is the single
// source of truth — the same posture as the embedded rituals (ADR-0014).
// M-0155's scaffold-if-absent write path (AC-3..AC-5) materializes
// `skills.StatuslineBytes()` into a consumer's `.claude/statusline.sh`;
// the repo no longer tracks a separate canonical copy for the embed to
// drift against, so the former byte-equality-vs-canonical assertion is
// obsolete. Scaffold-equality (the write path emits exactly these bytes)
// is pinned in m0155_statusline_scaffold_test.go.
//
// Two assertions:
//
//   - Presence: `skills.StatuslineBytes()` returns a non-empty slice
//     (the embed directive is wired and the source file is non-empty).
//   - Shape: the embed begins with a shebang, so the materialized script
//     is directly executable.
func TestM0155_AC1_StatuslineEmbedded(t *testing.T) {
	t.Parallel()
	embedded := skills.StatuslineBytes()
	if len(embedded) == 0 {
		t.Fatal("AC-1: skills.StatuslineBytes() returned empty — the go:embed directive is not wired or the source file is empty")
	}
	if !bytes.HasPrefix(embedded, []byte("#!")) {
		t.Error("AC-1: embedded statusline must begin with a shebang (#!) line so the materialized script is executable")
	}
}
