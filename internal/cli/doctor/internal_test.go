package doctor

import (
	"strings"
	"testing"
)

// TestAppendMaterializedRitualsReport_EmptyRoot exercises the missing
// branch directly: an empty root reports the artifacts as not
// materialized and points at `aiwf update`.
func TestAppendMaterializedRitualsReport_EmptyRoot(t *testing.T) {
	t.Parallel()
	out := appendMaterializedRitualsReport(nil, t.TempDir())
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "not materialized") || !strings.Contains(joined, "aiwf update") {
		t.Errorf("empty root should report rituals not materialized; got:\n%s", joined)
	}
}
