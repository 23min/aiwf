package doctor

import (
	"testing"
)

// TestAppendRecommendedPluginsReport_NilCfg_NoOp: helper called with
// nil cfg returns input unchanged. Reaches the `cfg == nil`
// early-return guard the public DoctorReport relies on when
// config.Load failed for a non-NotFound reason (cfg comes back nil).
// Same-package (internal) test because the helper is unexported.
func TestAppendRecommendedPluginsReport_NilCfg_NoOp(t *testing.T) {
	t.Parallel()
	in := []string{"line a", "line b"}
	out := appendRecommendedPluginsReport(in, nil, t.TempDir())
	if len(out) != len(in) {
		t.Fatalf("len = %d, want %d (helper must not mutate input on nil cfg)", len(out), len(in))
	}
	for i, want := range in {
		if out[i] != want {
			t.Errorf("[%d] = %q, want %q", i, out[i], want)
		}
	}
}
