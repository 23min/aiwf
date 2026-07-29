package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
			name: "one identifier reused across two verbs credits neither",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	res, _ = verb.Bar(ctx)
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
}`,
			// Fixture setup that happens to reuse the assertion's identifier.
			// This walk has no statement order, so it cannot tell which call
			// the assertion is about — and crediting both would let a verb
			// with no NoOp coverage of its own ride along on a neighbour's
			// test. Under-crediting is the safe answer: the policy fires and
			// a human writes the missing test.
			want: nil,
		},
		{
			name: "sibling closures are independent scopes and each credits its own verb",
			src: `package verb_test
func TestX(t *testing.T) {
	t.Run("rename", func(t *testing.T) {
		res, _ := verb.Foo(ctx)
		if !res.NoOp {
			t.Errorf("want a NoOp")
		}
	})
	t.Run("retitle", func(t *testing.T) {
		res, _ := verb.Bar(ctx)
		if !res.NoOp {
			t.Errorf("want a NoOp")
		}
	})
}`,
			// Two subtests each declaring their own `res` is idiomatic and
			// unambiguous — nothing is rebound. Reading the whole test function
			// as one namespace would make this indistinguishable from the case
			// above and refuse credit for both, a false negative on real
			// coverage; per-literal scoping is what keeps them apart.
			want: []string{"Bar", "Foo"},
		},
		{
			name: "a nested scope shadowing a name does not cost the outer scope its credit",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
	t.Run("sub", func(t *testing.T) {
		res, _ := verb.Bar(ctx)
		if !res.NoOp {
			t.Errorf("want a NoOp")
		}
	})
}`,
			// Each scope binds its own `res`, and the credit decision for a
			// scope is made before the walk descends into it, so a child's
			// binding can neither reach its parent nor cost it credit.
			want: []string{"Bar", "Foo"},
		},
		{
			name: "a sibling shadowing a name does not cost its sibling the inherited credit",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	t.Run("a", func(t *testing.T) {
		res, _ := verb.Bar(ctx)
		_ = res
	})
	t.Run("b", func(t *testing.T) {
		if !res.NoOp {
			t.Errorf("want a NoOp")
		}
	})
}`,
			// Sibling "a" never inspects its own Result, so Bar earns nothing;
			// sibling "b" inspects the Result bound in the enclosing scope, so
			// Foo does. A binding that escaped "a" would cost Foo its credit.
			want: []string{"Foo"},
		},
		{
			name: "a second := in the same scope rebinds, so neither verb is credited",
			src: `package verb_test
func TestX(t *testing.T) {
	res, err := verb.Foo(ctx)
	_ = err
	if !res.NoOp {
		t.Errorf("want a NoOp")
	}
	res, err2 := verb.Bar(ctx)
	_ = err2
	_ = res
}`,
			// `res, err2 := ...` redeclares res in the SAME scope: one variable
			// rebound, not a new one shadowing an outer name. Only the first
			// `:=` for a name gets to replace what it holds; this one
			// accumulates, so the two-verb rule refuses both. Crediting Bar
			// instead would pass a verb this function asserts nothing about.
			want: nil,
		},
		{
			name: "a closure's = stays inside that closure, so sibling order cannot change the verdict",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	t.Run("a", func(t *testing.T) {
		res, _ = verb.Bar(ctx)
	})
	t.Run("b", func(t *testing.T) {
		if !res.NoOp {
			t.Errorf("want a NoOp")
		}
	})
}`,
			// Siblings are walked in source order. If they shared one verb set,
			// "a"'s `=` would push the name "b" inherits to two verbs and cost
			// Foo its credit — an answer that flips when the two t.Run blocks
			// swap places. Copying the set per scope makes source order
			// irrelevant, which is the only defensible reading for a walk that
			// has no statement order to consult.
			want: []string{"Foo"},
		},
		{
			name: "a closure inspects a Result bound in the enclosing scope",
			src: `package verb_test
func TestX(t *testing.T) {
	res, _ := verb.Foo(ctx)
	t.Run("sub", func(t *testing.T) {
		if !res.NoOp {
			t.Errorf("want a NoOp")
		}
	})
}`,
			want: []string{"Foo"},
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

// TestVerbResultNoOpInvariant_SurfacesAWalkFailure pins the error path: a root
// the walk cannot read is reported as an error, not swallowed into the zero
// violations an empty scan would otherwise produce.
func TestVerbResultNoOpInvariant_SurfacesAWalkFailure(t *testing.T) {
	t.Parallel()
	if _, err := PolicyVerbResultNoOpInvariant(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("walking an absent root returned no error, want one — a policy that cannot read the tree must not report success")
	}
}

// TestVerbResultNoOpInvariant_SkipsAFileThatDoesNotParse pins that a file under
// internal/verb/ which fails to parse is skipped rather than aborting the scan.
// The tree here holds a broken file alongside a covered entry point, so the
// policy must still see Foo's coverage and report nothing.
func TestVerbResultNoOpInvariant_SkipsAFileThatDoesNotParse(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "verb")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("creating the fixture verb dir: %v", err)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	write("broken.go", "package verb\n\nfunc Oops( { }\n")
	write("v.go", "package verb\n\nfunc Foo() (*Result, error) { return nil, nil }\n")
	write("v_test.go", "package verb\n\nimport \"testing\"\n\n"+
		"func TestFoo(t *testing.T) {\n\tres, _ := Foo()\n\tif !res.NoOp {\n\t\tt.Errorf(\"want a NoOp\")\n\t}\n}\n")

	violations, err := PolicyVerbResultNoOpInvariant(root)
	if err != nil {
		t.Fatalf("a tree holding one unparseable file returned an error, want it skipped: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("got %d violations, want 0 — Foo is covered by the parseable half, and broken.go must be skipped rather than fatal: %+v", len(violations), violations)
	}
}

// TestVerbResultNoOpInvariant_FailsClosedOnEmptyEntrySet pins the fail-closed
// arm: pointed at a tree with no verb entry points, the policy must report a
// violation rather than the zero-violation "green" an empty scan would
// otherwise produce. Without this the chokepoint silently stops guarding the
// moment internal/verb/ is renamed.
func TestVerbResultNoOpInvariant_FailsClosedOnEmptyEntrySet(t *testing.T) {
	t.Parallel()
	violations, err := PolicyVerbResultNoOpInvariant(t.TempDir())
	if err != nil {
		t.Fatalf("running the policy over an empty tree: %v", err)
	}
	if !hasPolicyViolation(violations, "verb-result-noop-invariant") {
		t.Fatalf("an empty tree yielded %d violations, want a self-policy violation — "+
			"the policy must not report green while scanning nothing", len(violations))
	}
	if !strings.Contains(violations[0].Detail, "scanning nothing") {
		t.Errorf("Detail = %q, want it to say the policy is scanning nothing", violations[0].Detail)
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
