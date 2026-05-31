package skills

import "testing"

// TestRitualTrailerVerbs_DerivedFromEmbedded is the G-0190 drift guard:
// the auto-derived ritual-verb set must contain exactly the verbs
// literally stamped by `--trailer "aiwf-verb: <verb>"` patterns in the
// embedded snapshot under embedded-rituals. If a new ritual stamp
// lands in upstream rituals (refreshed via `make sync-rituals`), this
// test surfaces the addition — the allowlist used by
// `check.RunTrailerVerbUnknown` updates by construction, but the test
// fixes the expected baseline so a sweep that *accidentally* adds or
// drops a stamp is caught.
//
// Update the expected set when an intentional ritual-stamp change
// lands. Pre-G-0190 the kernel held a hand-maintained map containing
// both `wrap-epic` and `wrap-milestone`; the latter was a defensive
// future-proofing entry not actually stamped by any embedded skill.
// G-0193 retired the upstream authoring channel and landed the
// wrap-milestone trailer stamp on the embedded snapshot in the same
// patch, so `wrap-milestone` joins `wrap-epic` in the expected set.
func TestRitualTrailerVerbs_DerivedFromEmbedded(t *testing.T) {
	t.Parallel()
	got, err := RitualTrailerVerbs()
	if err != nil {
		t.Fatalf("RitualTrailerVerbs(): %v", err)
	}
	want := map[string]struct{}{
		"wrap-epic":      {},
		"wrap-milestone": {},
	}
	if len(got) != len(want) {
		t.Errorf("RitualTrailerVerbs set size = %d, want %d (got=%v want=%v)", len(got), len(want), sortedKeys(got), sortedKeys(want))
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing expected ritual verb %q (got=%v)", k, sortedKeys(got))
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("unexpected ritual verb %q in derived set — if intentional, update want{} in this test", k)
		}
	}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
