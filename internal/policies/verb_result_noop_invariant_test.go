package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// entryNamesFixture is the pretend entry-point set the credit-relation cases
// resolve against — two verbs, so a case can show credit landing on one and
// not the other.
var entryNamesFixture = map[string]bool{"Foo": true, "Bar": true}

// parseSoleFunc parses a one-function source snippet and returns its
// declaration, so a case can state the test shape it means to describe as
// ordinary Go rather than as a hand-built AST.
func parseSoleFunc(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "x_test.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parsing the fixture snippet: %v", err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
			return fn
		}
	}
	t.Fatalf("fixture snippet declares no function with a body:\n%s", src)
	return nil
}

// TestNoOpInspectedVerbs_CreditRequiresDataflow pins the credit relation behind
// the AC-6 policy: a verb earns NoOp coverage only when a test binds that
// verb's *Result to an identifier and references that same identifier's NoOp
// field. The cases are written as the shapes real tests take, and half of them
// are the false-green shapes a body-text scan credits and this relation must
// not: a mention in a comment, a mention in a format string, a verb called
// only as fixture setup, and a discarded Result.
func TestNoOpInspectedVerbs_CreditRequiresDataflow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "external-test spelling, positive assertion",
			src: `package verb_test
func TestX(t *testing.T) {
	res, err := verb.Foo(ctx)
	_ = err
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: []string{"Foo"},
		},
		{
			name: "in-package spelling, no qualifier",
			src: `package verb
func TestX(t *testing.T) {
	res, err := Foo(ctx)
	_ = err
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: []string{"Foo"},
		},
		{
			name: "NoOpMessage counts as inspecting the outcome",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	if !strings.Contains(res.NoOpMessage, "already") {
		t.Errorf("want the message to name the current state")
	}
}`,
			want: []string{"Foo"},
		},
		{
			name: "assertion inside a t.Run closure is reached",
			src: `package verb_test
func TestX(t *testing.T) {
	t.Run("sub", func(t *testing.T) {
		res, _ := verb.Foo(ctx)
		if !res.NoOp {
			t.Errorf("want a NoOp")
		}
	})
}`,
			want: []string{"Foo"},
		},
		{
			name: "one identifier reused across two verbs credits both",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	res, _ = verb.Bar(ctx)
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: []string{"Bar", "Foo"},
		},
		{
			name: "polarity is not judged: a not-a-NoOp assertion still credits",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	if res.NoOp {
		t.Errorf("want a real mutation, not a NoOp")
	}
}`,
			want: []string{"Foo"},
		},

		// The false-green shapes. Each mentions ".NoOp" and calls the verb,
		// so a body-text scan credits every one of them.
		{
			name: "mention in a comment does not credit",
			src: `package verb_test
func TestX(t *testing.T) {
	// A same-state call would return res.NoOp here once the guard lands.
	res, err := verb.Foo(ctx)
	_, _ = res, err
}`,
			want: nil,
		},
		{
			name: "mention in a format string does not credit",
			src: `package verb_test
func TestX(t *testing.T) {
	res, err := verb.Foo(ctx)
	_ = res
	if err != nil {
		t.Fatalf("res.NoOp assertion unreachable: %v", err)
	}
}`,
			want: nil,
		},
		{
			name: "fixture setup does not borrow the asserted verb's credit",
			src: `package verb_test
func TestX(t *testing.T) {
	setup, _ := verb.Foo(ctx)
	_ = setup
	res, _ := verb.Bar(ctx)
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: []string{"Bar"},
		},
		{
			name: "fixture setup passed to a helper binds nothing",
			src: `package verb_test
func TestX(t *testing.T) {
	r.must(verb.Foo(ctx))
	res, _ := verb.Bar(ctx)
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: []string{"Bar"},
		},
		{
			name: "discarded Result cannot be inspected",
			src: `package verb_test
func TestX(t *testing.T) {
	_, err := verb.Foo(ctx)
	_ = err
	other, _ := verb.Bar(ctx)
	if !other.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: []string{"Bar"},
		},
		{
			name: "a Result laundered through a helper earns no credit",
			src: `package verb_test
func TestX(t *testing.T) {
	res := mustNoOp(t, verb.Foo(ctx))
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: nil,
		},
		{
			name: "a same-named call qualified by another package is not the entry point",
			src: `package verb_test
func TestX(t *testing.T) {
	res, err := harness.Foo(ctx)
	_ = err
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: nil,
		},
		{
			name: "a wrapper whose name merely contains the verb is not the entry point",
			src: `package verb
func TestX(t *testing.T) {
	res, err := mustFoo(ctx)
	_ = err
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: nil,
		},
		{
			name: "a non-NoOp field reference does not credit on its own",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	if res.Plan != nil {
		t.Errorf("want no plan")
	}
}`,
			want: nil,
		},
		{
			name: "a NoOp reference through a chained selector credits nothing",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	_ = res
	if !outer.inner.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: nil,
		},
		{
			name: "a non-identifier assignment target binds nothing",
			src: `package verb_test
func TestX(t *testing.T) {
	var got map[string]*verb.Result
	var err error
	got["k"], err = verb.Foo(ctx)
	_ = err
	if !got["k"].NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: nil,
		},
		{
			name: "a multi-valued assignment is not a verb call",
			src: `package verb_test
func TestX(t *testing.T) {
	res, want := newResult(), true
	if res.NoOp != want {
		t.Errorf("want a NoOp")
	}
}`,
			want: nil,
		},
		{
			name: "a right-hand side that is not a call binds nothing",
			src: `package verb_test
func TestX(t *testing.T) {
	res := cached
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := noopInspectedVerbs(parseSoleFunc(t, tc.src), entryNamesFixture)
			var names []string
			for name := range got {
				names = append(names, name)
			}
			sort.Strings(names)
			if diff := cmp.Diff(tc.want, names); diff != "" {
				t.Errorf("credited verbs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestVerbResultNoOpInvariant_LiveTreeCreditsEveryNonExemptVerb pins the
// property the policy exists to protect, against the real tree rather than a
// fixture: every exported internal/verb entry point is either on the reviewed
// allowlist or carries a dataflow-connected NoOp assertion. It duplicates
// runPolicy's zero-violation expectation deliberately — this one reports which
// verb lost its coverage, which is the whole diagnostic value when it fires.
func TestVerbResultNoOpInvariant_LiveTreeCreditsEveryNonExemptVerb(t *testing.T) {
	t.Parallel()
	violations, err := PolicyVerbResultNoOpInvariant("../..")
	if err != nil {
		t.Fatalf("running the policy over the live tree: %v", err)
	}
	if len(violations) == 0 {
		return
	}
	var detail []string
	for _, v := range violations {
		detail = append(detail, v.File+": "+v.Detail)
	}
	t.Errorf("%d verb(s) lack dataflow-connected NoOp coverage:\n%s",
		len(violations), strings.Join(detail, "\n"))
}
