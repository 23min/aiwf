package policies

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/workflows/spec"
)

func TestM0319_AC2_BranchPreflightUsesRungPair(t *testing.T) {
	t.Parallel()
	want := []spec.Predicate{
		{Subject: "target-agent-role", Op: "==", Value: "ai"},
		{Subject: "rung-pair-legal", Op: "==", Value: "false"},
		{Subject: "force", Op: "==", Value: "false"},
	}
	cell := indexBranchRulesByID(t)["branch-cell-2"]
	if cell.ExpectedErrorCode != "rung-pair-illegal" {
		t.Errorf("branch preflight code = %q, want rung-pair-illegal", cell.ExpectedErrorCode)
	}
	if diff := cmp.Diff(want, cell.Preconditions); diff != "" {
		t.Errorf("branch preflight predicates (-want +got):\n%s", diff)
	}
	count := 0
	for _, rule := range spec.GlobalRules() {
		for _, p := range rule.Preconditions {
			if p.Subject == "branch-flag-resolves" {
				t.Error("global preflight must accept a future branch when its rung pair is legal")
			}
		}
		if rule.Verb == "authorize" && rule.ExpectedErrorCode == "rung-pair-illegal" {
			count++
			if diff := cmp.Diff(want, rule.Preconditions); diff != "" {
				t.Errorf("global preflight predicates (-want +got):\n%s", diff)
			}
		}
	}
	if count != 1 {
		t.Errorf("got %d global rung-pair rules, want one", count)
	}
}
