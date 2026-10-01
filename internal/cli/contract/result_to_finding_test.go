package contract

import (
	"testing"

	"github.com/23min/aiwf/internal/contractverify"
)

// TestResultToFinding_NamesTheContractAtCanonicalWidth: a verify result
// for a binding stored at a legacy narrow width becomes a finding whose
// entity_id is canonical, as every rendered surface names an id.
func TestResultToFinding_NamesTheContractAtCanonicalWidth(t *testing.T) {
	t.Parallel()
	got := ResultToFinding(contractverify.Result{Code: contractverify.CodeValidatorUnavailable, EntityID: "C-001"}, false)
	if got.EntityID != "C-0001" {
		t.Errorf("EntityID = %q, want C-0001", got.EntityID)
	}
}
