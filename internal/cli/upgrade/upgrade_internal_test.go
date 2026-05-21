package upgrade

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/version"
)

// TestProxyStaleHint covers the helper that emits the
// "proxy may be stale" hint when a pseudo-version's base is newer
// than the resolved latest. Closes G-0149.
//
// The positive case is unreachable through Run() under `go test`
// (runtime/debug.ReadBuildInfo() always returns "(devel)" for a test
// binary, so version.Current() never carries a real pseudo-version),
// hence this white-box test exercises the helper directly with
// synthetic Info values.
func TestProxyStaleHint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		current        string
		resolved       string
		wantHint       bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "pseudo-base ahead of target (the G-0149 case)",
			current:      "v0.8.1-0.20260516161658-02c349f629d7",
			resolved:     "v0.8.0",
			wantHint:     true,
			wantContains: []string{"hint:", "pseudo-base v0.8.1", "target v0.8.0", "GOPROXY=direct"},
		},
		{
			name:           "pseudo-base equal to target — already-aligned, no hint",
			current:        "v0.8.1-0.20260516161658-02c349f629d7",
			resolved:       "v0.8.1",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "pseudo-base behind target — real upgrade available, no hint",
			current:        "v0.8.1-0.20260516161658-02c349f629d7",
			resolved:       "v0.9.0",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "current is clean tag — no hint regardless of skew",
			current:        "v0.8.0",
			resolved:       "v0.8.1",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "current is devel — no hint",
			current:        "(devel)",
			resolved:       "v0.8.1",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "current is +dirty pseudo — no hint",
			current:        "v0.8.1-0.20260516161658-02c349f629d7+dirty",
			resolved:       "v0.8.0",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
		{
			name:           "form-1 pseudo with v0.0.0 base, target v0.1.0 — base behind, no hint",
			current:        "v0.0.0-20260503120000-abcdef123456",
			resolved:       "v0.1.0",
			wantHint:       false,
			wantNotContain: []string{"hint:"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := proxyStaleHint(version.Parse(tc.current), version.Parse(tc.resolved))
			if tc.wantHint && got == "" {
				t.Fatalf("proxyStaleHint returned empty, want non-empty hint")
			}
			if !tc.wantHint && got != "" {
				t.Fatalf("proxyStaleHint returned %q, want empty", got)
			}
			for _, sub := range tc.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("hint missing substring %q; got:\n%s", sub, got)
				}
			}
			for _, sub := range tc.wantNotContain {
				if strings.Contains(got, sub) {
					t.Errorf("hint unexpectedly contained %q; got:\n%s", sub, got)
				}
			}
		})
	}
}
