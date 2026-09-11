package verb_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// bodyOmitting renders a body carrying every section kind k requires,
// each under real prose, except omit — which is left out entirely.
//
// The discrimination is the point. A fixture built from the kind's
// scaffold leaves every required heading empty, so deleting one leaves
// a survivor for the emptiness gate to refuse and the refusal proves
// nothing about absence. Here the surviving headings are all filled, so
// the only thing left to report is the one that is not there. The
// section list comes from the kernel table, so a kind that later gains
// a section cannot leave this fixture quietly complete.
func bodyOmitting(t *testing.T, k entity.Kind, omit string) []byte {
	t.Helper()
	sections := entity.RequiredSections(k)
	if !slices.Contains(sections, omit) {
		t.Fatalf("bodyOmitting: %q is not a required section of %s (%v); the fixture would be complete and the test vacuous", omit, k, sections)
	}
	var b strings.Builder
	for _, s := range sections {
		if s == omit {
			continue
		}
		b.WriteString("## " + s + "\n\nReal prose under " + s + ".\n\n")
	}
	return []byte(b.String())
}

// TestAdd_BodyGate_OmittedRequiredSectionRefused pins M-0329/AC-2's
// first two claims: an explicit body that omits a required section is
// refused, naming it, and that holds for every kind carrying a required
// set rather than only the born-complete ones the emptiness gate
// covers.
//
// Kind is not the axis here — the three rows are three rules. The
// born-complete and draft-bearing rows differ in whether the gate
// reaches the kind at all, which is the whole distinction between
// scoping absence the way emptiness is scoped and scoping it to every
// writer of a body. Emptiness stays born-complete-only because a draft
// epic is meant to land with its headings empty and be filled in; no
// kind is meant to land without them, since the scaffold that seeds
// both writes every one.
//
// The headless row pins the gate's domain rather than another input: a
// gate that ran the absence check only once it had found a heading to
// anchor on would satisfy both rows above and still let `--body "b"` —
// the shape an operator actually types — land a body with nothing in
// it.
func TestAdd_BodyGate_OmittedRequiredSectionRefused(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		kind      entity.Kind
		body      []byte
		wantNamed []string
	}{
		{
			name:      "born-complete kind omitting one required section",
			kind:      entity.KindGap,
			body:      bodyOmitting(t, entity.KindGap, "Why it matters"),
			wantNamed: []string{"Why it matters"},
		},
		{
			name:      "draft-bearing kind omitting one required section",
			kind:      entity.KindEpic,
			body:      bodyOmitting(t, entity.KindEpic, "Out of scope"),
			wantNamed: []string{"Out of scope"},
		},
		{
			name:      "prose carrying no headings at all",
			kind:      entity.KindGap,
			body:      []byte("a one-line body, which is what --body \"...\" usually supplies\n"),
			wantNamed: []string{"What's missing", "Why it matters"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			_, err := verb.Add(r.ctx, r.tree(), tc.kind, "Probe entity", testActor, verb.AddOptions{
				BodyOverride: tc.body,
			})
			if err == nil {
				t.Fatalf("%s: expected refusal for a body omitting %v, got nil error", tc.kind, tc.wantNamed)
			}
			for _, want := range tc.wantNamed {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%s: error %q should name the omitted section %q", tc.kind, err, want)
				}
			}
		})
	}
}

// TestAdd_BodyGate_RefusalPromisesOnlyWhatTheCheckDelivers pins
// M-0329/AC-2's third claim, as the relationship it asserts rather than
// as a phrase: the gate tells the operator `aiwf check` will block
// until the section is filled, and that is true of a section present
// and empty and false of one deleted. The asymmetry is the hole this
// milestone closes — an operator who satisfies the emptiness refusal by
// deleting the heading is told the stricter surface will catch it, and
// nothing does.
//
// The expectation is derived by running the check over the body in
// question, not written down beside it, so the day entity-body-empty
// starts reporting an absent section this test fails rather than
// agreeing with a message that has become true by accident.
func TestAdd_BodyGate_RefusalPromisesOnlyWhatTheCheckDelivers(t *testing.T) {
	t.Parallel()
	const promise = "aiwf check will still"
	cases := []struct {
		name string
		body []byte
	}{
		{"required section present and empty", []byte("## What's missing\n\n\n\n## Why it matters\n\nReal prose.\n")},
		{"required section omitted outright", bodyOmitting(t, entity.KindGap, "What's missing")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			_, err := verb.Add(r.ctx, r.tree(), entity.KindGap, "Probe gap", testActor, verb.AddOptions{
				BodyOverride: tc.body,
			})
			if err == nil {
				t.Fatalf("expected a refusal for %s", tc.name)
			}

			// Land the same body over the gate, so the check has a real
			// entity to judge, and ask it what it reports.
			r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Landed gap", testActor, verb.AddOptions{
				BodyOverride: tc.body,
				Force:        true,
				Reason:       "landing the body the gate refused, to observe what the check says about it",
			}))
			checkBlocks := false
			for _, f := range check.Run(r.tree(), nil) {
				if f.Code == check.CodeEntityBodyEmpty {
					checkBlocks = true
				}
			}

			if promised := strings.Contains(err.Error(), promise); promised != checkBlocks {
				t.Errorf("refusal for %s promises `aiwf check` will block: %v; the check actually reports %s: %v\nmessage: %s",
					tc.name, promised, check.CodeEntityBodyEmpty, checkBlocks, err)
			}
		})
	}
}
