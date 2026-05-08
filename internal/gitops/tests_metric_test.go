package gitops

import (
	"strings"
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

func TestParseStrictTestMetrics(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    TestMetrics
		wantErr string
	}{
		{name: "all keys", input: "pass=12 fail=1 skip=2 total=15", want: TestMetrics{Pass: 12, Fail: 1, Skip: 2, Total: 15}},
		{name: "subset", input: "pass=5 fail=0 skip=0", want: TestMetrics{Pass: 5}},
		{name: "empty input ok", input: "", want: TestMetrics{}},
		{name: "whitespace only ok", input: "  \t ", want: TestMetrics{}},
		{name: "unknown key rejected", input: "pass=1 duration=120ms", wantErr: `unknown key "duration"`},
		{name: "non-integer rejected", input: "pass=ok", wantErr: `pass="ok"`},
		{name: "negative rejected", input: "pass=-1", wantErr: "must be non-negative"},
		{name: "missing equals rejected", input: "pass 12", wantErr: `token "pass"`},
		{name: "trailing equals rejected", input: "pass=", wantErr: `token "pass="`},
		{name: "leading equals rejected", input: "=12", wantErr: `token "=12"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseStrictTestMetrics(tc.input)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseStrictTestMetrics(%q) err = nil, want containing %q", tc.input, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("ParseStrictTestMetrics(%q) err = %v, want containing %q", tc.input, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseStrictTestMetrics(%q) err = %v, want nil", tc.input, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseStrictTestMetrics(%q) mismatch (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}

func TestFormatTestMetrics(t *testing.T) {
	cases := []struct {
		name string
		m    TestMetrics
		want string
	}{
		{"zero value emits empty", TestMetrics{}, ""},
		{"pass only", TestMetrics{Pass: 12}, "pass=12 fail=0 skip=0"},
		{"all four", TestMetrics{Pass: 12, Fail: 1, Skip: 2, Total: 15}, "pass=12 fail=1 skip=2 total=15"},
		{"total omitted when zero", TestMetrics{Pass: 5, Fail: 1}, "pass=5 fail=1 skip=0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatTestMetrics(tc.m); got != tc.want {
				t.Errorf("FormatTestMetrics(%+v) = %q, want %q", tc.m, got, tc.want)
			}
		})
	}
}

// TestTestMetrics_StrictParseRoundTrip pins the strict-parse / format
// round-trip: anything FormatTestMetrics emits must round-trip
// through ParseStrictTestMetrics losslessly. Load-bearing for the
// kernel write path — the verb composes the trailer from a TestMetrics,
// and aiwf history reads it back through the tolerant parser; we want
// the strict→format→strict path to be a fixed-point.
func TestTestMetrics_StrictParseRoundTrip(t *testing.T) {
	cases := []TestMetrics{
		{Pass: 12, Fail: 1, Skip: 2, Total: 15},
		{Pass: 5},
		{Pass: 0, Fail: 1, Skip: 0},
	}
	for _, m := range cases {
		t.Run(FormatTestMetrics(m), func(t *testing.T) {
			parsed, err := ParseStrictTestMetrics(FormatTestMetrics(m))
			if err != nil {
				t.Fatalf("round-trip parse: %v", err)
			}
			if diff := cmp.Diff(m, parsed); diff != "" {
				t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
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
