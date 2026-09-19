package doctor

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
)

// TestAppendHostArtifactsReport_EmptyRoot exercises the missing
// branch directly: an empty root reports the artifacts as not
// materialized and points at `aiwf update`.
func TestAppendHostArtifactsReport_EmptyRoot(t *testing.T) {
	t.Parallel()
	out, _ := appendHostArtifactsReport(nil, nil, t.TempDir(), config.HostClaudeCode, nil)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "not materialized") || !strings.Contains(joined, "aiwf update") {
		t.Errorf("empty root should report rituals not materialized; got:\n%s", joined)
	}
}
