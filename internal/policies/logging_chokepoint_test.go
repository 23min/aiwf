package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicyLoggingChokepoint_Synthetic pins the policy's branches
// against synthetic trees: bare fmt.Println/Print/Printf and
// fmt.Fprintln/Fprintf(os.Stdout|os.Stderr, …) fire; the same calls
// against an arbitrary writer, and an unparsable file, do not; an
// allowlisted file is exempt.
func TestPolicyLoggingChokepoint_Synthetic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		relPath  string
		body     string
		wantFire bool
		wantLine int
	}{
		{
			name:    "bare fmt.Println fires",
			relPath: "internal/cli/drift/drift.go",
			body: `package drift

import "fmt"

func bad() {
	fmt.Println("hello")
}
`,
			wantFire: true,
			wantLine: 6,
		},
		{
			name:    "bare fmt.Print fires",
			relPath: "internal/cli/drift/drift.go",
			body: `package drift

import "fmt"

func bad() {
	fmt.Print("hello")
}
`,
			wantFire: true,
			wantLine: 6,
		},
		{
			name:    "bare fmt.Printf fires",
			relPath: "internal/cli/drift/drift.go",
			body: `package drift

import "fmt"

func bad() {
	fmt.Printf("hello %s", "world")
}
`,
			wantFire: true,
			wantLine: 6,
		},
		{
			name:    "fmt.Fprintln(os.Stderr, ...) fires",
			relPath: "internal/cli/drift/drift.go",
			body: `package drift

import (
	"fmt"
	"os"
)

func bad() {
	fmt.Fprintln(os.Stderr, "hello")
}
`,
			wantFire: true,
			wantLine: 9,
		},
		{
			name:    "fmt.Fprintf(os.Stdout, ...) fires",
			relPath: "internal/cli/drift/drift.go",
			body: `package drift

import (
	"fmt"
	"os"
)

func bad() {
	fmt.Fprintf(os.Stdout, "hello %s", "world")
}
`,
			wantFire: true,
			wantLine: 9,
		},
		{
			// fmt.Fprintln() with zero args doesn't type-check (Fprintln
			// requires a writer), so this never occurs in real code — the
			// len(call.Args) < 1 guard exists purely to keep the AST walk
			// (which never runs go/types) from indexing call.Args[0] on a
			// syntactically-valid-but-type-invalid call.
			name:    "zero-arg fmt.Fprintln does not panic or fire",
			relPath: "internal/cli/clean/clean.go",
			body: `package clean

import "fmt"

func good() {
	fmt.Fprintln()
}
`,
			wantFire: false,
		},
		{
			name:    "fmt.Fprintln to an arbitrary writer does not fire",
			relPath: "internal/cli/clean/clean.go",
			body: `package clean

import (
	"fmt"
	"io"
)

func good(w io.Writer) {
	fmt.Fprintln(w, "hello")
}
`,
			wantFire: false,
		},
		{
			// The writer argument is a selector under a package OTHER
			// than os — exercises isOSStdioWriter's pkg.Name != "os"
			// arm specifically, distinct from the "not a selector at
			// all" and "os.* but not Stdout/Stderr" cases above/below.
			name:    "fmt.Fprintln to a non-os package selector does not fire",
			relPath: "internal/cli/clean/clean.go",
			body: `package clean

import (
	"bytes"
	"fmt"
)

func good() {
	fmt.Fprintln(bytes.MinRead, "hello")
}
`,
			wantFire: false,
		},
		{
			// The writer argument IS an os.* selector, but not Stdout or
			// Stderr — exercises isOSStdioWriter's final name comparison,
			// not just its earlier "is this a selector at all" guard.
			name:    "fmt.Fprintln to a non-stdio os.* selector does not fire",
			relPath: "internal/cli/clean/clean.go",
			body: `package clean

import (
	"fmt"
	"os"
)

func good() {
	fmt.Fprintln(os.Args, "hello")
}
`,
			wantFire: false,
		},
		{
			name:     "unparsable file is skipped without error",
			relPath:  "internal/cli/broken/broken.go",
			body:     "package broken\n\nfunc {{{ not go\n",
			wantFire: false,
		},
		{
			name:    "allowlisted outputformat.go is exempt",
			relPath: "internal/cli/cliutil/outputformat.go",
			body: `package cliutil

import (
	"fmt"
	"os"
)

func good() {
	fmt.Fprintln(os.Stderr, "hello")
}
`,
			wantFire: false,
		},
		{
			name:    "allowlisted textio.go is exempt",
			relPath: "internal/cli/cliutil/textio.go",
			body: `package cliutil

import "fmt"

func good() {
	fmt.Println("hello")
}
`,
			wantFire: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			full := filepath.Join(root, filepath.FromSlash(tc.relPath))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			violations, err := PolicyLoggingChokepoint(root)
			if err != nil {
				t.Fatalf("policy: %v", err)
			}
			found := false
			for _, v := range violations {
				if v.File == tc.relPath && (tc.wantLine == 0 || v.Line == tc.wantLine) {
					found = true
				}
			}
			if found != tc.wantFire {
				t.Errorf("fire = %v, want %v; violations: %+v", found, tc.wantFire, violations)
			}
		})
	}
}
