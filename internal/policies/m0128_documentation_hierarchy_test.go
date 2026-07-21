package policies

import "testing"

func TestPolicy_M0128DocumentationHierarchy(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0128DocumentationHierarchy)
}
