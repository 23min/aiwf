package gitops

import (
	"context"
	"reflect"
	"testing"
)

// TestBulkRevwalk_EmptyRoot pins the early-return for an empty root
// path: BulkRevwalk doesn't shell out and emits no callbacks.
func TestBulkRevwalk_EmptyRoot(t *testing.T) {
	t.Parallel()
	calls := 0
	err := BulkRevwalk(context.Background(), "", func(CommitRecord) error {
		calls++
		return nil
	})
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if calls != 0 {
		t.Errorf("callback invoked %d times, want 0", calls)
	}
}

// TestSplitOnMarker walks the splitOnMarker contract: empty input,
// no marker, only marker, leading/trailing marker, multiple markers.
func TestSplitOnMarker(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  string
		marker string
		want   []string
	}{
		{name: "empty input", input: "", marker: "===M==="},
		{name: "no marker, single line", input: "hello\n", marker: "===M===", want: []string{"hello\n"}},
		{name: "only marker", input: "===M===\n", marker: "===M==="},
		{name: "marker + content", input: "===M===\nbody\n", marker: "===M===", want: []string{"body\n"}},
		{
			name:   "two records",
			input:  "===M===\nbody1\n===M===\nbody2\n",
			marker: "===M===",
			want:   []string{"body1\n", "body2\n"},
		},
		{
			name:   "content-before-first-marker is captured",
			input:  "prefix\n===M===\nbody\n",
			marker: "===M===",
			want:   []string{"prefix\n", "body\n"},
		},
		{
			name:   "marker as substring within a line is NOT a boundary",
			input:  "===M===\nbody contains ===M=== inline\n===M===\ntail\n",
			marker: "===M===",
			want:   []string{"body contains ===M=== inline\n", "tail\n"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := splitOnMarker(tc.input, tc.marker)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitOnMarker(%q, %q) = %#v, want %#v",
					tc.input, tc.marker, got, tc.want)
			}
		})
	}
}

// TestParseBulkChunk_Malformed exercises the defensive returns:
// missing paths marker, too-few fields, empty SHA.
func TestParseBulkChunk_Malformed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		chunk string
	}{
		{name: "missing paths marker", chunk: "sha1\x1fparent\x1f\n"},
		{name: "too few fields", chunk: "sha1\n===AIWF-PATHS===\n"},
		{
			name: "empty SHA",
			chunk: "\x1fparent" +
				makeTrailerStub() +
				"\n===AIWF-PATHS===\n",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec, ok := parseBulkChunk(tc.chunk)
			if ok {
				t.Errorf("parseBulkChunk(%q) ok=true (rec=%+v), want false", tc.chunk, rec)
			}
		})
	}
}

// TestParseBulkChunk_NoTrailingNewline pins the end-of-output
// fallback: the paths marker appears without a trailing newline (last
// chunk in `git log` output).
func TestParseBulkChunk_NoTrailingNewline(t *testing.T) {
	t.Parallel()
	chunk := "abc123\x1fparent1" + makeTrailerStub() + "\n===AIWF-PATHS==="
	rec, ok := parseBulkChunk(chunk)
	if !ok {
		t.Fatalf("parseBulkChunk(%q) ok=false, want true", chunk)
	}
	if rec.Commit != "abc123" {
		t.Errorf("Commit = %q, want abc123", rec.Commit)
	}
	if len(rec.Paths) != 0 {
		t.Errorf("Paths = %v, want empty", rec.Paths)
	}
}

// TestParseBulkTrailers_EmptyFields covers the all-empty case (no
// aiwf-* trailers on the commit at all): the helper returns nil to
// distinguish from a zero-length-but-non-nil map.
func TestParseBulkTrailers_EmptyFields(t *testing.T) {
	t.Parallel()
	fields := make([]string, len(bulkTrailerKeys))
	got := parseBulkTrailers(fields)
	if got != nil {
		t.Errorf("parseBulkTrailers(empty) = %#v, want nil", got)
	}
}

// TestParseBulkTrailers_EmptySlice covers the defensive entry-guard:
// a zero-length slice (no trailer fields at all) returns nil rather
// than panicking.
func TestParseBulkTrailers_EmptySlice(t *testing.T) {
	t.Parallel()
	got := parseBulkTrailers(nil)
	if got != nil {
		t.Errorf("parseBulkTrailers(nil) = %#v, want nil", got)
	}
}

// TestParseBulkTrailers_FewerFieldsThanKeys covers the break path
// inside parseBulkTrailers: the input slice is shorter than
// bulkTrailerKeys (a future git that drops a field). The available
// keys are populated; the loop breaks at the slice's end without
// out-of-bounds.
func TestParseBulkTrailers_FewerFieldsThanKeys(t *testing.T) {
	t.Parallel()
	fields := []string{"add", "M-0137"}
	got := parseBulkTrailers(fields)
	want := map[string]string{
		"aiwf-verb":   "add",
		"aiwf-entity": "M-0137",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseBulkTrailers(short) = %#v, want %#v", got, want)
	}
}

// TestParsePathsBlock covers the parser branches: empty input,
// well-formed A/M/D/T, R/C with srcpath, malformed lines that are
// silently dropped.
func TestParsePathsBlock(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		block string
		want  []PathTouch
	}{
		{name: "empty block", block: ""},
		{
			name:  "single A line",
			block: "A\talpha.md\n",
			want:  []PathTouch{{Status: "A", Path: "alpha.md"}},
		},
		{
			name:  "M + D",
			block: "M\talpha.md\nD\tbeta.md\n",
			want: []PathTouch{
				{Status: "M", Path: "alpha.md"},
				{Status: "D", Path: "beta.md"},
			},
		},
		{
			name:  "rename with similarity",
			block: "R100\told.md\tnew.md\n",
			want:  []PathTouch{{Status: "R", SrcPath: "old.md", Path: "new.md"}},
		},
		{
			name:  "copy with similarity",
			block: "C087\tsrc.md\tdst.md\n",
			want:  []PathTouch{{Status: "C", SrcPath: "src.md", Path: "dst.md"}},
		},
		{
			name:  "type change passes through",
			block: "T\tx.md\n",
			want:  []PathTouch{{Status: "T", Path: "x.md"}},
		},
		{
			name:  "skip too-few parts",
			block: "A\nM\talpha.md\n",
			want:  []PathTouch{{Status: "M", Path: "alpha.md"}},
		},
		{
			name:  "skip rename with missing dst path",
			block: "R100\told.md\nM\talpha.md\n",
			want:  []PathTouch{{Status: "M", Path: "alpha.md"}},
		},
		{
			name:  "skip empty status code",
			block: "\tnopath.md\nM\talpha.md\n",
			want:  []PathTouch{{Status: "M", Path: "alpha.md"}},
		},
		{
			name:  "skip empty lines between entries",
			block: "A\talpha.md\n\nM\tbeta.md\n",
			want: []PathTouch{
				{Status: "A", Path: "alpha.md"},
				{Status: "M", Path: "beta.md"},
			},
		},
		// `git log --raw` lines (production shape): the ':<srcmode>
		// <dstmode> <presha> <postsha> <status>' prefix carries the blob
		// object ids that the FSM walker reads by id (M-0216 AC-2).
		{
			name:  "raw M line carries pre/post blob ids",
			block: ":100644 100644 1111111111111111111111111111111111111111 2222222222222222222222222222222222222222 M\talpha.md\n",
			want: []PathTouch{{
				Status:  "M",
				Path:    "alpha.md",
				PreSHA:  "1111111111111111111111111111111111111111",
				PostSHA: "2222222222222222222222222222222222222222",
			}},
		},
		{
			name:  "raw A line has all-zero pre-image",
			block: ":000000 100644 0000000000000000000000000000000000000000 3333333333333333333333333333333333333333 A\talpha.md\n",
			want: []PathTouch{{
				Status:  "A",
				Path:    "alpha.md",
				PreSHA:  "0000000000000000000000000000000000000000",
				PostSHA: "3333333333333333333333333333333333333333",
			}},
		},
		{
			name:  "raw D line has all-zero post-image",
			block: ":100644 000000 4444444444444444444444444444444444444444 0000000000000000000000000000000000000000 D\tbeta.md\n",
			want: []PathTouch{{
				Status:  "D",
				Path:    "beta.md",
				PreSHA:  "4444444444444444444444444444444444444444",
				PostSHA: "0000000000000000000000000000000000000000",
			}},
		},
		{
			name:  "raw R line carries both paths and blob ids",
			block: ":100644 100644 5555555555555555555555555555555555555555 6666666666666666666666666666666666666666 R100\told.md\tnew.md\n",
			want: []PathTouch{{
				Status:  "R",
				SrcPath: "old.md",
				Path:    "new.md",
				PreSHA:  "5555555555555555555555555555555555555555",
				PostSHA: "6666666666666666666666666666666666666666",
			}},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parsePathsBlock(tc.block)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parsePathsBlock(%q) = %#v, want %#v", tc.block, got, tc.want)
			}
		})
	}
}

// TestParseRawPathLine exercises parseRawPathLine directly — the
// `git log --raw` line parser — including the malformed-shape guards
// that the line-format the production walk emits never hits but that
// must reject cleanly (returning ok=false so parsePathsBlock falls
// back to the name-status branch).
func TestParseRawPathLine(t *testing.T) {
	t.Parallel()
	const pre = "1111111111111111111111111111111111111111"
	const post = "2222222222222222222222222222222222222222"
	cases := []struct {
		name string
		line string
		want PathTouch
		ok   bool
	}{
		{
			name: "valid M",
			line: ":100644 100644 " + pre + " " + post + " M\talpha.md",
			want: PathTouch{Status: "M", Path: "alpha.md", PreSHA: pre, PostSHA: post},
			ok:   true,
		},
		{
			name: "valid R with two operands",
			line: ":100644 100644 " + pre + " " + post + " R100\told.md\tnew.md",
			want: PathTouch{Status: "R", SrcPath: "old.md", Path: "new.md", PreSHA: pre, PostSHA: post},
			ok:   true,
		},
		{name: "no leading colon", line: "M\talpha.md"},
		{name: "no tab", line: ":100644 100644 " + pre + " " + post + " M"},
		{name: "wrong meta field count", line: ":100644 100644 " + pre + " M\talpha.md"},
		{name: "rename missing second operand", line: ":100644 100644 " + pre + " " + post + " R100\tonly.md"},
		{name: "empty single operand", line: ":100644 100644 " + pre + " " + post + " M\t"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseRawPathLine(tc.line)
			if ok != tc.ok {
				t.Fatalf("parseRawPathLine(%q) ok = %v, want %v", tc.line, ok, tc.ok)
			}
			if ok && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseRawPathLine(%q) = %#v, want %#v", tc.line, got, tc.want)
			}
		})
	}
}

// makeTrailerStub returns a sequence of `\x1f<empty>` separators
// covering every bulkTrailerKeys slot, so chunk fixtures have the
// expected field count without spelling each trailer out.
func makeTrailerStub() string {
	stub := ""
	for range bulkTrailerKeys {
		stub += "\x1f"
	}
	return stub
}
