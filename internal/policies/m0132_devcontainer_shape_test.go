package policies

import "testing"

func TestPolicy_M0132DevcontainerShape(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0132DevcontainerShape)
}
