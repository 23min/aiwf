package cliutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
)

// TestParseKind covers the closed-set lookup behind every verb that
// takes a <kind> positional argument. Each entity.Kind constant must
// round-trip through its lowercase string; anything outside the set
// returns ("", false) so callers fall through to a usage error.
func TestParseKind(t *testing.T) {
	t.Parallel()
	for _, k := range entity.AllKinds() {
		got, ok := cliutil.ParseKind(string(k))
		if !ok {
			t.Errorf("ParseKind(%q): ok = false; want true", string(k))
		}
		if got != k {
			t.Errorf("ParseKind(%q) = %q; want %q", string(k), got, k)
		}
	}
	for _, bad := range []string{"", "Epic", "MILESTONE", "unknown", "ac"} {
		got, ok := cliutil.ParseKind(bad)
		if ok {
			t.Errorf("ParseKind(%q) accepted; want ok=false, got=%q", bad, got)
		}
	}
}

// TestParseTestsFlag covers the three behavior arms: empty input
// (flag unset, returns nil/nil), valid input (returns parsed metrics),
// malformed input (returns parse error, writes one-line to stderr).
func TestParseTestsFlag(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		got, err := cliutil.ParseTestsFlag("", "aiwf test")
		if err != nil {
			t.Errorf("empty: err = %v; want nil", err)
		}
		if got != nil {
			t.Errorf("empty: metrics = %+v; want nil", got)
		}
	})
	t.Run("whitespace_only", func(t *testing.T) {
		t.Parallel()
		got, err := cliutil.ParseTestsFlag("   ", "aiwf test")
		if err != nil {
			t.Errorf("whitespace: err = %v; want nil", err)
		}
		if got != nil {
			t.Errorf("whitespace: metrics = %+v; want nil", got)
		}
	})
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		got, err := cliutil.ParseTestsFlag("pass=12 fail=0 skip=0", "aiwf test")
		if err != nil {
			t.Fatalf("valid: err = %v", err)
		}
		if got == nil {
			t.Fatal("valid: metrics = nil; want non-nil")
		}
		if got.Pass != 12 || got.Fail != 0 || got.Skip != 0 {
			t.Errorf("valid: metrics = %+v; want pass=12 fail=0 skip=0", *got)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		t.Parallel()
		// Redirect stderr for the duration of the call so the test
		// output stays clean.
		oldStderr := os.Stderr
		os.Stderr, _ = os.Open(os.DevNull)
		defer func() { os.Stderr = oldStderr }()
		_, err := cliutil.ParseTestsFlag("not-a-metric=garbage", "aiwf test")
		if err == nil {
			t.Errorf("malformed: err = nil; want non-nil")
		}
	})
}

// TestReadBodyFile covers reading a file from disk; the stdin ("-")
// branch is exercised in the integration tests of consumers (those
// drive the dispatcher with cmd.SetIn).
func TestReadBodyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "body.md")
	want := []byte("body content\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := cliutil.ReadBodyFile(path)
	if err != nil {
		t.Fatalf("ReadBodyFile: %v", err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("body mismatch (-want +got):\n%s", diff)
	}
}

// TestSplitCommaList covers the comma-trim-drop behavior shared by
// every multi-value CLI flag (--relates-to, --linked-adr, --depends-on,
// etc.). Empty input must return nil, not a zero-length slice — callers
// branch on len(s) > 0 to decide whether the flag was set.
func TestSplitCommaList(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single", "a", []string{"a"}},
		{"two", "a,b", []string{"a", "b"}},
		{"trim_whitespace", "  a , b  ,c", []string{"a", "b", "c"}},
		{"drop_empty", "a,,b,,,c", []string{"a", "b", "c"}},
		{"only_commas", ",,,", nil},
		{"only_whitespace", "   ", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := cliutil.SplitCommaList(tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SplitCommaList(%q) mismatch (-want +got):\n%s", tc.in, diff)
			}
		})
	}
}
