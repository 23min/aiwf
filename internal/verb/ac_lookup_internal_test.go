package verb

import (
	"strings"
	"testing"
)

// TestLookupAC_RefusesMalformedComposite: every exported caller routes only
// composite-shaped ids here, so the malformed arm is reached by calling the
// helper directly. A bare milestone id is refused before the tree is read,
// and the refusal names the id it was given.
func TestLookupAC_RefusesMalformedComposite(t *testing.T) {
	t.Parallel()
	_, _, err := lookupAC(nil, "M-0001")
	if err == nil {
		t.Fatal("lookupAC accepted a bare milestone id as a composite id")
	}
	if !strings.Contains(err.Error(), `"M-0001"`) {
		t.Errorf("refusal does not name the id it was given: %v", err)
	}
}
