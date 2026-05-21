package gitops

import (
	"strings"
	"testing"
)

// TestParseBatchHeader covers the helper's three legal shapes and the
// defensive returns: missing, found-with-size, and the malformed
// variants (wrong field count, non-integer size, negative size).
func TestParseBatchHeader(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		line        string
		wantMissing bool
		wantSize    int
		wantErrSub  string
	}{
		{
			name:        "found, size 42",
			line:        "abc123 blob 42",
			wantMissing: false,
			wantSize:    42,
		},
		{
			name:        "found, size 0",
			line:        "abc123 blob 0",
			wantMissing: false,
			wantSize:    0,
		},
		{
			name:        "missing",
			line:        "abc:nonexistent missing",
			wantMissing: true,
		},
		{
			name:       "wrong field count (1)",
			line:       "abc123",
			wantErrSub: "malformed cat-file --batch header",
		},
		{
			name:       "wrong field count (4)",
			line:       "abc123 blob 42 extra",
			wantErrSub: "malformed cat-file --batch header",
		},
		{
			name:       "non-integer size",
			line:       "abc123 blob notanumber",
			wantErrSub: "size parse",
		},
		{
			name:       "negative size",
			line:       "abc123 blob -1",
			wantErrSub: "negative size",
		},
		{
			name:       "empty line",
			line:       "",
			wantErrSub: "malformed cat-file --batch header",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotMissing, gotSize, err := parseBatchHeader(tc.line)
			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("parseBatchHeader(%q) err=nil, want err containing %q", tc.line, tc.wantErrSub)
				}
				if !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Errorf("parseBatchHeader(%q) err=%q, want substring %q", tc.line, err.Error(), tc.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseBatchHeader(%q) unexpected err=%v", tc.line, err)
			}
			if gotMissing != tc.wantMissing {
				t.Errorf("missing = %v, want %v", gotMissing, tc.wantMissing)
			}
			if gotSize != tc.wantSize {
				t.Errorf("size = %d, want %d", gotSize, tc.wantSize)
			}
		})
	}
}
