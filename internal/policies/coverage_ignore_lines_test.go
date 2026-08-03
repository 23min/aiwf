package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestCoverageIgnoreLines pins which comments open the coverage escape. The
// matcher's own contract is TestHasDirectiveComment; what is pinned here is
// that the escape is read off parsed comments, so the shapes that are not
// comments — or not directives — do not suppress a finding.
func TestCoverageIgnoreLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// body is placed inside a func at a known offset; want names the
		// lines of body (1-based within body) that carry a directive.
		body string
		want []int
	}{
		{
			name: "trailing directive with a reason",
			body: "\t_ = 1 //coverage:ignore defensive guard, unreachable in fixtures\n",
			want: []int{1},
		},
		{
			name: "own-line directive with a reason",
			body: "\t//coverage:ignore defensive guard, unreachable in fixtures\n\t_ = 1\n",
			want: []int{1},
		},
		{
			name: "bare marker is not a directive",
			body: "\t_ = 1 //coverage:ignore\n",
			want: nil,
		},
		{
			name: "prose naming the escape is not a directive",
			body: "\t// see the //coverage:ignore escape described in CLAUDE.md\n\t_ = 1\n",
			want: nil,
		},
		{
			name: "a longer word opening with the marker is not a directive",
			body: "\t_ = 1 //coverage:ignoreable\n",
			want: nil,
		},
		{
			name: "a string literal wearing the marker is not a directive",
			body: "\ts := \"//coverage:ignore\"\n\t_ = s\n",
			want: nil,
		},
		{
			name: "a block comment is not a directive",
			body: "\t/* //coverage:ignore unreachable */\n\t_ = 1\n",
			want: nil,
		},
		{
			name: "a directive on a continuation line of a group is found",
			body: "\t// a note that opens the group\n\t//coverage:ignore defensive guard, unreachable\n\t_ = 1\n",
			want: []int{2},
		},
		{
			// Files carry several annotations — reporting only the first
			// would silently un-exempt every later one.
			name: "every directive in a file is reported, not just the first",
			body: "\t_ = 1 //coverage:ignore first guard, unreachable\n" +
				"\t_ = 2\n" +
				"\t_ = 3 //coverage:ignore second guard, unreachable\n",
			want: []int{1, 3},
		},
		{
			// The sibling escapes annotate different properties. Neither
			// may open this one.
			name: "a sibling escape does not open this one",
			body: "\t_ = 1 //history:ok legacy on-disk format\n\t_ = 2 //exec:ok the mode is the subject\n",
			want: nil,
		},
		{
			name: "the marker is case-sensitive",
			body: "\t_ = 1 //COVERAGE:IGNORE unreachable\n",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			const header = "package p\n\nfunc F() {\n"
			const headerLines = 3 // body line 1 lands on file line 4

			path := filepath.Join(t.TempDir(), "seam.go")
			if err := os.WriteFile(path, []byte(header+tt.body+"}\n"), 0o600); err != nil {
				t.Fatalf("writing fixture: %v", err)
			}

			ignored, err := coverageIgnoreLines(path)
			if err != nil {
				t.Fatalf("coverageIgnoreLines: %v", err)
			}

			var want []int
			for _, ln := range tt.want {
				want = append(want, ln+headerLines)
			}
			var got []int
			for ln := 1; ln <= headerLines+strings.Count(tt.body, "\n")+1; ln++ {
				if ignored[ln] {
					got = append(got, ln)
				}
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("directive lines mismatch (-want +got):\n%s\nsource:\n%s", diff, header+tt.body+"}\n")
			}
		})
	}
}

// TestCoverageIgnoreLines_LineDirective pins the coordinate system: under a
// //line directive the escape is recorded at the remapped line, because the
// coverage profile numbers its blocks the same way and a block span is the
// only thing the set is ever compared against.
//
// Verified against real `go test -coverprofile` output: a file carrying
// `//line x.go:100` reports its blocks at 100/101/103, not at 3/4/6.
func TestCoverageIgnoreLines_LineDirective(t *testing.T) {
	t.Parallel()

	//  1 package p
	//  2
	//  3 //line seam.go:100   → the next line is 100, so the directive is 101
	//  4 func F() {
	//  5 <tab>_ = 1 //coverage:ignore unreachable in fixtures
	//  6 }
	src := "package p\n\n//line seam.go:100\nfunc F() {\n\t_ = 1 //coverage:ignore unreachable in fixtures\n}\n"
	path := filepath.Join(t.TempDir(), "seam.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	ignored, err := coverageIgnoreLines(path)
	if err != nil {
		t.Fatalf("coverageIgnoreLines: %v", err)
	}
	if diff := cmp.Diff(map[int]bool{101: true}, ignored); diff != "" {
		t.Errorf("directive lines mismatch (-want +got):\n%s", diff)
	}
}

// TestCoverageIgnoreLines_Unreadable pins that a file the audit cannot read
// surfaces as an error rather than as "nothing is annotated", which would
// silently report every uncovered block in it.
func TestCoverageIgnoreLines_Unreadable(t *testing.T) {
	t.Parallel()

	if _, err := coverageIgnoreLines(filepath.Join(t.TempDir(), "absent.go")); err == nil {
		t.Fatal("want an error for a missing file, got nil")
	}
}

// TestCoverageIgnoreLines_Unparseable pins the same for source that does not
// parse. Every file reaching the audit has a coverage profile and so compiled;
// reaching this means the tree moved out from under the profile, which is not
// a condition to absorb into an empty annotation set.
func TestCoverageIgnoreLines_Unparseable(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "broken.go")
	if err := os.WriteFile(path, []byte("package p\n\nfunc F( {\n"), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if _, err := coverageIgnoreLines(path); err == nil {
		t.Fatal("want a parse error for malformed source, got nil")
	}
}
