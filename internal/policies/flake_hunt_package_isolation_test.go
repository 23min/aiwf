package policies

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPolicy_FlakeHuntPackageIsolation(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyFlakeHuntPackageIsolation)
}

func TestPolicy_FlakeHuntPackageIsolation_MissingWorkflowErrors(t *testing.T) {
	t.Parallel()
	if _, err := PolicyFlakeHuntPackageIsolation(t.TempDir()); err == nil {
		t.Fatal("expected an error when flake-hunt.yml is absent, got nil")
	}
}

func TestLogicalLines(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		content string
		want    []sourceLine
	}{
		{
			name:    "plain lines number from one",
			content: "a\nb\n",
			want:    []sourceLine{{Num: 1, Text: "a"}, {Num: 2, Text: "b"}, {Num: 3, Text: ""}},
		},
		{
			name:    "a continuation joins and reports the line it starts on",
			content: "x\ngo test -race \\\n  -parallel 8\ny\n",
			want: []sourceLine{
				{Num: 1, Text: "x"},
				{Num: 2, Text: "go test -race   -parallel 8"},
				{Num: 4, Text: "y"},
				{Num: 5, Text: ""},
			},
		},
		{
			name:    "trailing whitespace after the backslash still continues",
			content: "a \\  \nb\n",
			want:    []sourceLine{{Num: 1, Text: "a b"}, {Num: 3, Text: ""}},
		},
		{
			name:    "an unterminated continuation flushes at EOF",
			content: "a \\",
			want:    []sourceLine{{Num: 1, Text: "a "}},
		},
		{
			name:    "a comment does not continue across its backslash",
			content: "# prose \\\nrun: go test\n",
			want:    []sourceLine{{Num: 1, Text: "# prose \\"}, {Num: 2, Text: "run: go test"}, {Num: 3, Text: ""}},
		},
		{
			name:    "a comment interrupting a join loses neither side",
			content: "run: cmd \\\n# aside\n  --flag\n",
			want: []sourceLine{
				{Num: 1, Text: "run: cmd "},
				{Num: 2, Text: "# aside"},
				{Num: 3, Text: "  --flag"},
				{Num: 4, Text: ""},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if diff := cmp.Diff(tc.want, logicalLines(tc.content)); diff != "" {
				t.Errorf("logicalLines mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// A wildcard split from its `go test` by a continuation is the same
// sweep as one written inline, so joining is what the ban rests on.
func TestPolicyFlakeHuntPackageIsolation_ContinuationIsNotAnEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWrite(t, root+"/.github/workflows/flake-hunt.yml",
		"name: y\nrun: go test -race -parallel 8 -count=10 \\\n  ./...\n")

	vs, err := PolicyFlakeHuntPackageIsolation(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("want 1 violation, got %d: %+v", len(vs), vs)
	}
	if vs[0].Line != 2 {
		t.Errorf("violation should report the line the invocation starts on, got %d", vs[0].Line)
	}
}

// The enumerator is the one command that must name every package, so
// the ban has to let it through while still catching the invocation.
func TestPolicyFlakeHuntPackageIsolation_EnumeratorIsAllowed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWrite(t, root+"/.github/workflows/flake-hunt.yml",
		"name: y\nrun: go list -f '{{.ImportPath}}' ./...\nrun: go test -race -parallel 8 \"${{ matrix.package }}\"\n")

	vs, err := PolicyFlakeHuntPackageIsolation(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("want no violations, got %d: %+v", len(vs), vs)
	}
}

// The shipped enumerator is a continued block, so appending a sweep to
// it is the cheapest route back to a whole-module run — the exemption
// has to end where `go test` begins.
func TestPolicyFlakeHuntPackageIsolation_EnumeratorExemptionEndsAtGoTest(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"sweep appended to the enumerator":  "name: y\nrun: go list ./... > pkgs.txt && \\\n  go test -race -count=10 ./...\n",
		"sweep sharing the enumerator line": "name: y\nrun: go list ./... && go test -race ./...\n",
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			mustWrite(t, root+"/.github/workflows/flake-hunt.yml", content)

			vs, err := PolicyFlakeHuntPackageIsolation(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if len(vs) != 1 {
				t.Fatalf("want 1 violation, got %d: %+v", len(vs), vs)
			}
		})
	}
}

// A commented backslash must not swallow the invocation beneath it,
// which would let a sweep ride in under a comment.
func TestPolicyFlakeHuntPackageIsolation_CommentDoesNotSwallowTheNextLine(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWrite(t, root+"/.github/workflows/flake-hunt.yml",
		"name: y\n# prose ending in a backslash \\\nrun: go test -race -count=10 ./...\n")

	vs, err := PolicyFlakeHuntPackageIsolation(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("want 1 violation, got %d: %+v", len(vs), vs)
	}
	if vs[0].Line != 3 {
		t.Errorf("violation should report the invocation's own line, got %d", vs[0].Line)
	}
}
