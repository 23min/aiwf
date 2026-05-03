package gitops

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseTestMetrics(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   TestMetrics
		wantOK bool
	}{
		{
			name:   "all four keys",
			input:  "pass=12 fail=1 skip=2 total=15",
			want:   TestMetrics{Pass: 12, Fail: 1, Skip: 2, Total: 15},
			wantOK: true,
		},
		{
			name:   "subset (pass/fail/skip; total derived)",
			input:  "pass=12 fail=0 skip=0",
			want:   TestMetrics{Pass: 12},
			wantOK: true,
		},
		{
			name:   "single recognized key",
			input:  "fail=3",
			want:   TestMetrics{Fail: 3},
			wantOK: true,
		},
		{
			name:   "unknown keys ignored, recognized kept",
			input:  "pass=5 duration=120ms category=unit fail=2",
			want:   TestMetrics{Pass: 5, Fail: 2},
			wantOK: true,
		},
		{
			name:   "malformed value skipped",
			input:  "pass=ok fail=4 skip=NaN",
			want:   TestMetrics{Fail: 4},
			wantOK: true,
		},
		{
			name:   "negative value skipped",
			input:  "pass=-1 fail=2",
			want:   TestMetrics{Fail: 2},
			wantOK: true,
		},
		{
			name:   "no equals sign tokens skipped",
			input:  "passing fail=2 nope skip=1",
			want:   TestMetrics{Fail: 2, Skip: 1},
			wantOK: true,
		},
		{
			name:   "empty value after equals skipped",
			input:  "pass= fail=3",
			want:   TestMetrics{Fail: 3},
			wantOK: true,
		},
		{
			name:   "empty input",
			input:  "",
			want:   TestMetrics{},
			wantOK: false,
		},
		{
			name:   "whitespace only",
			input:  "   \t  ",
			want:   TestMetrics{},
			wantOK: false,
		},
		{
			name:   "no recognized keys",
			input:  "duration=120ms category=unit",
			want:   TestMetrics{},
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseTestMetrics(tc.input)
			if ok != tc.wantOK {
				t.Errorf("ParseTestMetrics(%q) ok = %v, want %v", tc.input, ok, tc.wantOK)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseTestMetrics(%q) mismatch (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}

func TestTotalOrDerive(t *testing.T) {
	cases := []struct {
		name string
		m    TestMetrics
		want int
	}{
		{"total set", TestMetrics{Pass: 5, Fail: 1, Skip: 1, Total: 7}, 7},
		{"total absent — derive", TestMetrics{Pass: 5, Fail: 1, Skip: 1}, 7},
		{"all zero", TestMetrics{}, 0},
		{"only fail", TestMetrics{Fail: 3}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.m.TotalOrDerive(); got != tc.want {
				t.Errorf("TotalOrDerive() = %d, want %d", got, tc.want)
			}
		})
	}
}
