package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
)

func TestReferencePreservesDeclaredSemantics(t *testing.T) {
	t.Parallel()
	applicability := []Applicability{
		{Kind: entity.KindEpic, Verb: "promote", Applies: true},
		{Kind: KindTDDPhase, Verb: "cancel", Reason: "status only"},
	}
	rules := []Rule{
		{Kind: entity.KindEpic, FromState: "active", Verb: "promote", ToState: "done", Outcome: OutcomeLegal},
		{
			Kind: entity.KindEpic, FromState: "active", Verb: "promote", ToState: "done", Outcome: OutcomeIllegal,
			Preconditions:  []Predicate{{Subject: "self.one", Op: "==", Value: "false"}, {Subject: "self.two", Op: "!=", Value: "yes"}},
			RejectionLayer: RejectionLayerVerbTime, BlockingStrict: true, ExpectedErrorCode: "blocked",
		},
		{Kind: KindTDDPhase, Verb: "promote", Outcome: OutcomeNoOp},
	}
	globals := []Rule{{Verb: "authorize", Preconditions: []Predicate{{Subject: "self.kind", Op: "!=", Value: "epic"}}, Outcome: OutcomeIllegal, RejectionLayer: RejectionLayerCheckTime, ExpectedErrorCode: "scope"}}
	got := string(renderReference(applicability, rules, globals))
	for section, want := range map[string][]string{
		"Applicability": {
			"| epic | promote | true |  |",
			"| tdd-phase | cancel | false | status only |",
		},
		"Declared transitions": {
			"| epic | active | promote | done | Legal |  | none | false |  |",
			"| epic | active | promote | done | Illegal | self.one == false AND self.two != yes | verb-time | true | blocked |",
			"| tdd-phase |  | promote |  | NoOp |  | none | false |  |",
		},
		"Global restrictions": {
			"|  | authorize | self.kind != epic | Illegal | check-time | false | scope |",
		},
	} {
		t.Run(section, func(t *testing.T) {
			t.Parallel()
			_, body, found := strings.Cut(got, "## "+section+"\n")
			if !found {
				t.Fatalf("missing section %s", section)
			}
			body, _, _ = strings.Cut(body, "\n## ")
			var rows []string
			for _, line := range strings.Split(body, "\n") {
				if strings.HasPrefix(line, "| ") {
					rows = append(rows, line)
				}
			}
			if len(rows) == 0 {
				t.Fatal("missing table")
			}
			if diff := cmp.Diff(want, rows[1:]); diff != "" {
				t.Errorf("rows (-want +got):\n%s", diff)
			}
		})
	}
	if diff := cmp.Diff(got, string(renderReference(applicability, rules, globals))); diff != "" {
		t.Errorf("nondeterministic render: %s", diff)
	}
}

func TestReferenceEscapesTableData(t *testing.T) {
	t.Parallel()
	got := referenceCell("a|b\\c`*_[]<&\r\nx\ry\nz")
	want := "a\\|b\\\\c\\`\\*\\_\\[\\]&lt;&amp;<br>x<br>y<br>z"
	if got != want {
		t.Errorf("cell = %q, want %q", got, want)
	}
}

func TestReferenceEnumLabels(t *testing.T) {
	t.Parallel()
	for value, want := range map[Outcome]string{OutcomeUnspecified: "Unspecified", OutcomeLegal: "Legal", OutcomeIllegal: "Illegal", OutcomeNoOp: "NoOp", Outcome(99): "Unknown (99)"} {
		if got := referenceOutcome(value); got != want {
			t.Errorf("outcome %d = %q, want %q", value, got, want)
		}
	}
	for value, want := range map[RejectionLayer]string{RejectionLayerNone: "none", RejectionLayerVerbTime: "verb-time", RejectionLayerCheckTime: "check-time", RejectionLayer(99): "unknown (99)"} {
		if got := referenceLayer(value); got != want {
			t.Errorf("layer %d = %q, want %q", value, got, want)
		}
	}
}

func TestReferenceMatchesCommittedDocument(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "..", "docs", "reference", "workflow-legality.md")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(committed, RenderReference()) {
		t.Fatal("workflow reference is stale; from the repository root run go run ./cmd/workflow-reference")
	}
}

func TestReferenceIncludesEveryDeclaration(t *testing.T) {
	t.Parallel()
	output := string(RenderReference())
	for section, want := range map[string]int{"Applicability": len(Applicabilities()), "Declared transitions": len(Rules()), "Global restrictions": len(GlobalRules())} {
		_, body, found := strings.Cut(output, "## "+section+"\n")
		if !found {
			t.Fatalf("missing %s", section)
		}
		body, _, _ = strings.Cut(body, "\n## ")
		if got := strings.Count(body, "\n| ") - 1; got != want {
			t.Errorf("%s rows = %d, want %d", section, got, want)
		}
	}
}

func TestReferencePredicatesDistinguishEmptyComparisonsFromUnaryOperators(t *testing.T) {
	t.Parallel()
	predicates := []Predicate{
		{Subject: "a", Op: "=="},
		{Subject: "b", Op: "!="},
		{Subject: "c", Op: "exists"},
		{Subject: "d", Op: "non-empty"},
		{Subject: "e", Op: "==", Value: "value"},
	}
	want := `a == "" AND b != "" AND c exists AND d non-empty AND e == value`
	if got := referencePredicates(predicates); got != want {
		t.Errorf("predicates = %q, want %q", got, want)
	}
}
