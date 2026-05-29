package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAppendMarketplaceOverlapReport_MalformedSettings surfaces the
// loadEnabledPlugins error branch of the de-dupe guard: a malformed
// `.claude/settings.json` produces a `plugins:` error line rather than
// a silent skip. Same-package (internal) test because the helper is
// unexported and the error path is awkward to reach end-to-end.
func TestAppendMarketplaceOverlapReport_MalformedSettings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := appendMarketplaceOverlapReport(nil, root)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "plugins:") {
		t.Errorf("malformed settings.json should surface a plugins: error line; got:\n%s", joined)
	}
}

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
