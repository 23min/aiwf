package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// rewriteBody replaces the body of the entity file at path, keeping its
// frontmatter byte-for-byte — the shape a working-copy edit in $EDITOR
// produces, and the only one bless mode accepts.
func rewriteBody(t *testing.T, path, body string) {
	t.Helper()
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	fm, _, ok := entity.Split(original)
	if !ok {
		t.Fatalf("%s lacks a frontmatter delimiter", path)
	}
	edited := append(append([]byte("---\n"), fm...), append([]byte("---\n\n"), body...)...)
	if err := os.WriteFile(path, edited, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestEditBody_ExplicitAndBlessAnswerAlike pins that both of
// edit-body's modes judge the same edit the same way (M-0329/AC-3).
//
// The two modes take different inputs — bytes handed to the verb, and
// a working copy the verb reads — but the write they produce is the
// same, so an answer that differs by mode means the same edit to the
// same entity is refused or committed depending on which flag the
// operator reached for. Measured before the shared rule, on a gap whose
// committed body already omitted `## Why it matters` and an edit that
// still omitted it: --body-file refused, bless committed.
//
// The rows are the two halves of one rule. Dropping a section HEAD
// carries is refused; keeping an omission HEAD already had is not,
// which is what leaves an already-incomplete entity editable through
// either mode.
func TestEditBody_ExplicitAndBlessAnswerAlike(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		// headBody is what the entity is created with, and next is the
		// body the edit produces by either route.
		headBody    []byte
		headForced  bool
		next        string
		wantRefusal bool
	}{
		{
			name:        "dropping a section the committed body carries",
			headBody:    bornCompleteFixtureBody(entity.KindGap),
			next:        "## What's missing\n\nRewritten, and the second section is gone.\n",
			wantRefusal: true,
		},
		{
			name:        "keeping an omission the committed body already had",
			headBody:    []byte("## What's missing\n\nThe committed body never carried the second section.\n"),
			headForced:  true,
			next:        "## What's missing\n\nRewritten; the second section is still absent.\n",
			wantRefusal: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			explicit := editOutcome(t, tc.headBody, tc.headForced, func(r *runner, id, path, next string) error {
				_, err := verb.EditBody(r.ctx, r.tree(), id, []byte(next), testActor, "")
				return err
			}, tc.next)
			bless := editOutcome(t, tc.headBody, tc.headForced, func(r *runner, id, path, next string) error {
				rewriteBody(t, filepath.Join(r.root, path), next)
				_, err := verb.EditBody(r.ctx, r.tree(), id, nil, testActor, "")
				return err
			}, tc.next)

			if (explicit != nil) != (bless != nil) {
				t.Fatalf("the two modes disagree about the same edit — --body-file: %v; bless: %v", explicit, bless)
			}
			if (explicit != nil) != tc.wantRefusal {
				t.Fatalf("refused = %v, want %v (error: %v)", explicit != nil, tc.wantRefusal, explicit)
			}
			if !tc.wantRefusal {
				return
			}
			// Naming the section is the assertion. Both modes refuse
			// several other ways on this path — an unchanged file,
			// touched frontmatter, a malformed id-shaped token — so an
			// error-is-non-nil check passes with the rule absent and the
			// section quietly gone.
			for mode, err := range map[string]error{"--body-file": explicit, "bless": bless} {
				if !strings.Contains(err.Error(), "Why it matters") {
					t.Errorf("%s refusal does not name the dropped section: %v", mode, err)
				}
			}
		})
	}
}

// editOutcome seeds a gap whose committed body is headBody, applies one
// edit through the route edit does, and returns whatever that route
// refused with.
func editOutcome(t *testing.T, headBody []byte, forced bool, edit func(r *runner, id, path, next string) error, next string) error {
	t.Helper()
	r := newRunner(t)
	opts := verb.AddOptions{BodyOverride: headBody}
	if forced {
		opts.Force = true
		opts.Reason = "seeding the already-omitting state this rule must leave editable"
	}
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Subject gap", testActor, opts))
	return edit(r, "G-0001", r.tree().ByID("G-0001").Path, next)
}
