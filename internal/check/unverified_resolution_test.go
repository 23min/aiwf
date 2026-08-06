package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestMarkUnverifiedResolution_DowngradesOnlyWhenTheScanWasSkipped is
// the core of G-0558: `unresolved` is a claim about every tier, so a
// surface that built one tier reports the weaker subcode instead.
func TestMarkUnverifiedResolution_DowngradesOnlyWhenTheScanWasSkipped(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		scanned      bool
		wantSubcode  string
		wantSeverity Severity
	}{
		{"scanned: every tier consulted, so unresolved is earned", true, "unresolved", SeverityError},
		{"not scanned: the claim outruns the evidence", false, "unresolved-unverified", SeverityWarning},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := []Finding{{
				Code:     CodeRefsResolve,
				Severity: SeverityError,
				Subcode:  "unresolved",
				Message:  `milestone field "parent" references unknown id "E-9999"`,
			}}
			got := MarkUnverifiedResolution(in, &tree.Tree{CrossBranchScanned: tc.scanned})
			if got[0].Subcode != tc.wantSubcode {
				t.Errorf("Subcode = %q, want %q", got[0].Subcode, tc.wantSubcode)
			}
			if got[0].Severity != tc.wantSeverity {
				t.Errorf("Severity = %q, want %q", got[0].Severity, tc.wantSeverity)
			}
		})
	}
}

// TestMarkUnverifiedResolution_LeavesEverythingElseAlone guards the
// blast radius. The pass runs over a whole findings slice, so it must
// touch exactly the subcodes whose verdict rests on the tier stack, and
// leave the rest.
//
// `unresolved-ac` is the interesting exclusion: it fires only after the
// parent entity was found in the working tree, and asserts something
// about the AC list in that file, which the caller is holding. A
// sibling branch cannot change it. `unresolved-milestone` is the
// opposite case and IS downgraded — it asserts the parent is allocated
// nowhere — so it is covered in the contract test rather than here.
func TestMarkUnverifiedResolution_LeavesEverythingElseAlone(t *testing.T) {
	t.Parallel()
	in := []Finding{
		{Code: CodeRefsResolve, Severity: SeverityWarning, Subcode: "cross-branch-pending"},
		{Code: CodeRefsResolve, Severity: SeverityError, Subcode: "wrong-kind"},
		{Code: CodeBodyProseID, Severity: SeverityError, Subcode: "malformed-shape"},
		{Code: CodeBodyProseID, Severity: SeverityError, Subcode: "unresolved-ac"},
		{Code: CodeRefsResolve, Severity: SeverityError, Subcode: "unresolved-ac"},
		{Code: CodeIDsUnique, Severity: SeverityError, Subcode: "unresolved"},
		{Code: CodeIDsUnique, Severity: SeverityError, Subcode: "unresolved-milestone"},
	}
	want := make([]Finding, len(in))
	copy(want, in)

	got := MarkUnverifiedResolution(in, &tree.Tree{CrossBranchScanned: false})
	for i := range got {
		if got[i].Subcode != want[i].Subcode || got[i].Severity != want[i].Severity {
			t.Errorf("finding %d (%s/%s) was rewritten to %s/%s; only refs-resolve and body-prose-id `unresolved` may change",
				i, want[i].Code, want[i].Subcode, got[i].Severity, got[i].Subcode)
		}
	}
}

// TestMarkUnverifiedResolution_MessageStatesWhatWasEstablished pins
// that the downgraded finding stops asserting the id is unknown and
// says what the surface actually checked. The message is the whole
// reason this subcode exists rather than silent suppression: without
// it, two surfaces disagree and nothing explains why.
func TestMarkUnverifiedResolution_MessageStatesWhatWasEstablished(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   Finding
		want string
	}{
		{
			name: "structured field",
			in: Finding{
				Code: CodeRefsResolve, Severity: SeverityError, Subcode: "unresolved",
				Message: `milestone field "parent" references unknown id "E-9999"`,
			},
			want: `milestone field "parent" references "E-9999", which resolves to no entity in this working tree; the cross-branch view was not built, so it may exist on an unmerged branch`,
		},
		{
			name: "prose token",
			in: Finding{
				Code: CodeBodyProseID, Severity: SeverityError, Subcode: "unresolved",
				Message: `G-0001 body prose contains unknown id "M-9999" (no entity allocated at this id)`,
			},
			want: `G-0001 body prose contains "M-9999", which resolves to no entity in this working tree; the cross-branch view was not built, so it may exist on an unmerged branch`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := MarkUnverifiedResolution([]Finding{tc.in}, &tree.Tree{CrossBranchScanned: false})
			if got[0].Message != tc.want {
				t.Errorf("Message =\n  %q\nwant\n  %q", got[0].Message, tc.want)
			}
			if strings.Contains(got[0].Message, "unknown id") {
				t.Errorf("message still asserts the id is unknown, which this surface did not establish: %q", got[0].Message)
			}
			if got[0].Hint == "" {
				t.Error("downgraded finding carries no hint; the operator needs the surface that can settle it")
			}
		})
	}
}

// TestMarkUnverifiedResolution_NilTreeIsInert covers the defensive arm:
// a nil tree carries no evidence either way, and rewriting findings on
// that basis would be a claim of its own.
func TestMarkUnverifiedResolution_NilTreeIsInert(t *testing.T) {
	t.Parallel()
	in := []Finding{{Code: CodeRefsResolve, Severity: SeverityError, Subcode: "unresolved"}}
	got := MarkUnverifiedResolution(in, nil)
	if got[0].Subcode != "unresolved" || got[0].Severity != SeverityError {
		t.Errorf("nil tree rewrote the finding to %s/%s, want error/unresolved", got[0].Severity, got[0].Subcode)
	}
}

// TestMarkUnverifiedResolution_ContractWithTheRules derives its input
// from the rules themselves rather than a hand-written literal.
//
// The rewrite recognizes the two phrasings refsResolve and
// classifyBodyToken use for `unresolved`. That is a coupling to their
// wording, and a literal fixture would keep passing after a reword
// while the shipped message silently degraded to the fallback. Driving
// the real rules is what makes the reword break a test instead.
func TestMarkUnverifiedResolution_ContractWithTheRules(t *testing.T) {
	t.Parallel()

	t.Run("refsResolve", func(t *testing.T) {
		t.Parallel()
		tr := &tree.Tree{Entities: []*entity.Entity{
			{ID: "M-0001", Kind: entity.KindMilestone, Parent: "E-9999", Path: "synthetic.md"},
		}}
		got := MarkUnverifiedResolution(refsResolve(tr), tr)
		if len(got) != 1 || got[0].Subcode != SubcodeUnresolvedUnverified {
			t.Fatalf("want one downgraded finding, got %+v", got)
		}
		assertStatesOnlyWhatWasEstablished(t, got[0].Message)
	})

	t.Run("bodyProseID", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		ents := writeBodyProseFixture(t, root, "See M-9999 for the proposed rule.")
		tr := &tree.Tree{Root: root, Entities: ents}
		got := MarkUnverifiedResolution(bodyProseID(tr), tr)
		if len(got) != 1 || got[0].Subcode != SubcodeUnresolvedUnverified {
			t.Fatalf("want one downgraded finding, got %+v", got)
		}
		assertStatesOnlyWhatWasEstablished(t, got[0].Message)
	})

	t.Run("refsResolve composite parent is NOT downgraded", func(t *testing.T) {
		t.Parallel()
		// resolveCompositeRef resolves the parent against the
		// working-tree index alone, so its verdict is the same under
		// either loader. Downgrading it here would make --fast disagree
		// with the full check about a finding they actually agree on —
		// manufacturing the divergence this pass removes elsewhere.
		tr := &tree.Tree{Entities: []*entity.Entity{
			{ID: "G-0001", Kind: entity.KindGap, AddressedBy: []string{"M-9999/AC-1"}, Path: "synthetic.md"},
		}}
		got := MarkUnverifiedResolution(refsResolve(tr), tr)
		if len(got) != 1 {
			t.Fatalf("want one finding, got %+v", got)
		}
		if got[0].Subcode != "unresolved-milestone" || got[0].Severity != SeverityError {
			t.Errorf("got %s/%s, want error/unresolved-milestone — this verdict does not depend on the tier stack",
				got[0].Severity, got[0].Subcode)
		}
	})

	t.Run("bodyProseID composite parent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		ents := writeBodyProseFixture(t, root, "See M-9999/AC-1 for the proposed rule.")
		tr := &tree.Tree{Root: root, Entities: ents}
		got := MarkUnverifiedResolution(bodyProseID(tr), tr)
		if len(got) != 1 || got[0].Subcode != SubcodeUnresolvedUnverified {
			t.Fatalf("want one downgraded finding, got %+v", got)
		}
		assertStatesOnlyWhatWasEstablished(t, got[0].Message)
	})
}

// assertStatesOnlyWhatWasEstablished fails on a message that still
// asserts the id is unallocated — the claim a ref-less load cannot
// make — or that omits why the surface is not deciding.
//
// The neutral-clause check is what makes this sensitive to a reworded
// rule. Greping for today's overclaim phrasings alone would not be: a
// reword removes the phrase being searched for, so the message sails
// through the fallback arm carrying its new overclaim, and the test
// stays green for the one change it exists to catch.
func assertStatesOnlyWhatWasEstablished(t *testing.T, msg string) {
	t.Helper()
	for _, overclaim := range []string{"unknown id", "no entity allocated at this id", "is not allocated", "does not exist"} {
		if strings.Contains(msg, overclaim) {
			t.Errorf("message still claims %q, which this surface did not establish: %q", overclaim, msg)
		}
	}
	if !strings.Contains(msg, unverifiedNeutralClause) {
		t.Errorf("message lacks %q, so it hit unverifiedMessage's fallback — a rule's wording drifted past the cases it handles: %q", unverifiedNeutralClause, msg)
	}
	if !strings.Contains(msg, "the cross-branch view was not built") {
		t.Errorf("message does not say why the surface is not deciding: %q", msg)
	}
}

// TestUnverifiedMessage_UnrecognizedShapeStillQualifies covers the
// fallback: a message matching neither known phrasing is qualified by
// appending rather than dropped or mangled. It is the arm a reworded
// rule would land in, and the contract test above is what would catch
// the reword itself.
func TestUnverifiedMessage_UnrecognizedShapeStillQualifies(t *testing.T) {
	t.Parallel()
	got := unverifiedMessage("some future phrasing with no recognizable id clause")
	if !strings.HasPrefix(got, "some future phrasing with no recognizable id clause") {
		t.Errorf("fallback dropped the original message: %q", got)
	}
	if !strings.Contains(got, "the cross-branch view was not built") {
		t.Errorf("fallback did not qualify the claim: %q", got)
	}
	// The fallback deliberately omits the neutral clause, which is the
	// link that makes a reworded rule fail the contract test above
	// rather than degrade quietly.
	if strings.Contains(got, unverifiedNeutralClause) {
		t.Errorf("fallback emitted %q, so a reworded rule would look rewritten and the contract test would stay green: %q", unverifiedNeutralClause, got)
	}
}
